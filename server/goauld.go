package main

import (
	"Goauld/server/control"
	"Goauld/server/db"
	"Goauld/server/sshd"
	"context"
	"fmt"
)

func main() {
	initdb()
	// err := mock()
	// if err != nil {
	//	fmt.Println(err)
	// }

	fmt.Println("Hello World")
	ctx, cancel := context.WithCancel(context.Background())
	startSshd(ctx)
	startSocketIO(ctx)
	select {
	case <-ctx.Done():
		fmt.Println("Done")
		fmt.Println(ctx.Err())
		cancel()
	}
	<-ctx.Done()
	fmt.Println()
}

func initdb() {
	db.Get().Migrate()
}

func mock() error {
	agent := &db.Agent{
		Id:           "1",
		UsedPorts:    nil,
		PrivateKey:   "",
		PublicKey:    "",
		Source:       "",
		Connected:    false,
		SharedSecret: "",
	}
	err := db.Get().CreateAgent(agent)
	if err != nil {
		return err
	}
	return nil
}

func startSshd(ctx context.Context) {
	go sshd.StartSshd(ctx)
}

func startSocketIO(ctx context.Context) {
	go control.RunSocketIOServer()
}
