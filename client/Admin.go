package main

import (
	"Goauld/client/api"
	colorYaml "Goauld/common/yaml"
	"gopkg.in/yaml.v3"
)

type Admin struct {
	Dump     Dump     `cmd:""`
	Loglevel Loglevel `cmd:""`
	Config   Config   `cmd:""`
}

type Dump struct{}
type Config struct{}

type Loglevel struct {
	Level string `arg:"" help:"Log level"`
}

func (d *Dump) Run(_ *api.API, cfg ClientConfig) error {

	adminApi := api.NewAPI(cfg.ServerUrl(), cfg.AdminToken, cfg.Insecure)

	err, res := adminApi.DumpAll()
	if err != nil {
		return err
	}

	y, err := yaml.Marshal(res)
	if err != nil {
		return err
	}
	colorYaml.PrintColorizedYAML(string(y))
	return nil
}

func (l *Loglevel) Run(_ *api.API, cfg ClientConfig) error {
	adminApi := api.NewAPI(cfg.ServerUrl(), cfg.AdminToken, cfg.Insecure)
	err, res := adminApi.UpdateLogLevel(cfg.Admin.Loglevel.Level)
	if err != nil {
		return err
	}
	y, err := yaml.Marshal(res)
	if err != nil {
		return err
	}
	colorYaml.PrintColorizedYAML(string(y))
	return nil
}

func (c *Config) Run(_ *api.API, cfg ClientConfig) error {

	adminApi := api.NewAPI(cfg.ServerUrl(), cfg.AdminToken, cfg.Insecure)

	err, res := adminApi.GetConfig()
	if err != nil {
		return err
	}

	colorYaml.PrintColorizedYAML(res)
	return nil
}

/*func JsonColor(data any) {

	var enc *json.Encoder
	// Note: this check will fail if running inside Goland (and
	// other IDEs?) as IsColorTerminal will return false.
	if json.IsColorTerminal(os.Stdout) {
		// Safe to use color
		out := colorable.NewColorable(os.Stdout) // needed for Windows
		enc = json.NewEncoder(out)

		// DefaultColors are similar to jq
		clrs := json.DefaultColors()

		// Change some values, just for fun
		clrs.Bool = json.Color("\x1b[36m") // Change the bool color
		clrs.String = json.Color{}         // Disable the string color

		enc.SetColors(clrs)
	} else {
		// Can't use color; but the encoder will still work
		enc = json.NewEncoder(os.Stdout)
	}
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}*/
