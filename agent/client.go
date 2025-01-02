package main

import (
	"Goauld/agent/agent"
	"Goauld/agent/control"
	"Goauld/agent/socks"
	"Goauld/agent/ssh"
	"Goauld/agent/sshd"
	"Goauld/common/log"
	"context"
	"strconv"
)

func main() {

	controlErr := make(chan error)
	sshdErr := make(chan error)
	sshErr := make(chan error)

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

	ctx := context.Background()

	// Initialize the control socket.io
	controlPlanClient := control.NewControlPlanClient(ctx, configDone)
	err = controlPlanClient.Init()
	if err != nil {
		log.Error().Err(err).Msg("error initializing the control plan")
		return
	}

	// Create the client SSH
	sshAgent := ssh.NewSSHAgent()
	//defer sshAgent.Close()

	// Start the control socket.io
	go func() {
		controlErr <- controlPlanClient.Start()
	}()

	// If the SSHD server is enabled, start it
	if agent.Get().SshdEnabled() {
		go func() {
			sshdErr <- sshd.StartSShd()
		}()
	}

	go func() {
		// Waiting for the configuration to be completed
		<-configDone
		// Initialize the client SSH
		err = sshAgent.Init()
		if err != nil {
			log.Error().Err(err).Msg("error initializing the SSH")
			return
		}

		// IF the SSHD server is enabled, we need to forward its port to the remote server
		// so we add it to the ports to forward
		if agent.Get().SshdEnabled() {
			agent.Get().AddSshdToRpf()
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
			socks5.Serve(rListener)
			log.Info().Str("Remote port", strconv.Itoa(rPort)).Msg("Remote Socks5 server started")
		}

		// For all porte forwards, launch the forwarding
		rpf := agent.Get().GetRemotePortForwarding()
		for i := range rpf {
			port, err := sshAgent.RemoteForward(rpf[i], ctx)
			if err != nil {
				log.Error().Err(err).Str("Local", rpf[i].GetLocal()).Str("Remote", rpf[i].GetRemote()).Msg("error initializing the port forwarding")
				continue
			}
			rpf[i].RemotePort = port

			log.Info().Str("Local", rpf[i].GetLocal()).Str("Remote", rpf[i].GetRemote()).Msg("Port forwarding started")

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
	}
	//ctx.Done()

}
