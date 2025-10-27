package main

import (
	"fmt"
	"github.com/hazegard/togettyc/ttycommon"
	"github.com/hazegard/togettyc/ttyplay"
	"github.com/hazegard/togettyc/ttyprint"
	"github.com/hazegard/togettyc/ttyrec"
	"github.com/hazegard/togettyc/ttytime"
	"github.com/alecthomas/kong"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	Play  ttyplay.Config  `cmd:"" help:"Play the record"`
	Print ttyprint.Config `cmd:"" help:"Render the record"`
	Rec   ttyrec.Config   `cmd:"" help:"Record" default:"withargs"`
	Time  ttytime.Config  `cmd:"" help:"Time to record"`
}

func (c *Config) Run() error {
	return nil
}

func main() {

	name := filepath.Base(os.Args[0])
	var ctx *kong.Context
	//fmt.Println(ttycommon.GetBanner())
	kongOptions := []kong.Option{
		kong.Name(name),
		kong.Description("Cross-platform reimplementation of ttyrec"),
		kong.UsageOnError(),
		kong.Help(func(options kong.HelpOptions, ctx *kong.Context) error {
			if ctx.Error == nil {
				fmt.Println(ttycommon.GetBanner())
			}
			return kong.DefaultHelpPrinter(options, ctx)
		}),
	}
	switch name {
	case "ttyplay":
		ctx = kong.Parse(&ttyplay.Config{}, kongOptions...)
	case "ttyrec":
		ctx = kong.Parse(&ttyrec.Config{}, kongOptions...)
	case "ttyprint":
		ctx = kong.Parse(&ttyprint.Config{}, kongOptions...)
	case "ttytime":
		ctx = kong.Parse(&ttytime.Config{}, kongOptions...)
	default:
		ctx = kong.Parse(&Config{}, kongOptions...)
	}

	err := ctx.Run()
	if err != nil {
		log.Fatal(err)
	}
}
