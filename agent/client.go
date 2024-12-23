package main

import (
	"Goauld/agent/agent"
	"Goauld/agent/control"
	"Goauld/agent/sshd"
	"context"
	"fmt"
)

func main() {
	controlErr := make(chan error)
	sshErr := make(chan error)

	_, err := agent.InitAgent()
	if err != nil {
		fmt.Println(err)
		return
	}

	ctx := context.Background()
	go func() {
		controlErr <- control.NewClient(ctx)
	}()

	go func() {
		sshErr <- sshd.StartSShd()
	}()

	select {}

}
