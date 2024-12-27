package agent

import (
	"Goauld/common/cli"
	"Goauld/common/log"
	"Goauld/common/utils"
	"github.com/alecthomas/kong"
	"path/filepath"
	"strings"
)

var (
	_localSshPassword = ""
	_name             = "user@hostname"
	_server           = "localhost:3000"
	_ssh_server       = "localhost:61160"
	_sshd_port        = "0"
	_rssh_port        = "0"
	_rssh_order       = "SSH,TLS,WS,HTTP"

	defaultValues = kong.Vars{
		"_localSshPassword": _localSshPassword,
		"_name":             _name,
		"_server":           _server,
		"_sshd_port":        _sshd_port,
		"_rssh_port":        _rssh_port,
		"_ssh_server":       _ssh_server,
		"_rssh_order":       _rssh_order,
	}
)

const APP_NAME = "Goa'uld"

type Config struct {
	LocalSshPassword string   `default:"${_localSshPassword}" short:"p" name:"password" optional:"" help:"SSH password to access the agent."`
	Name             string   `default:"user@hostname" name:"name" optional:"" help:"Nice name to identify the agent."`
	Server           string   `default:"${_server}" short:"s" name:"server" optional:"" help:"HTTP Server to connect to."`
	SshServer        string   `default:"${_ssh_server}" short:"S" name:"sshserver" optional:"" help:"SSH Server to connect to."`
	SshdPort         int      `default:"${_sshd_port}"  name:"sshd-port" optional:"" help:"Local port to listen to, 0 => Random."`
	RsshPort         int      `default:"${_rssh_port}"  name:"rssh-port" optional:"" help:"Remote port to bind to, 0 => Random."`
	RsshOrder        []string `default:"${_rssh_order}" short:"O"  name:"rssh_order" optional:"" help:"Order of ssh connection attempts."`
	//RemoteDynamicPortForwarding []string `name:"R" optional:"" help:"Ports to forward to the server."`
	//RemotePortForwarding        []string `name:"L" optional:"" help:"Ports to forward to the server."`
}

func parse() (*kong.Context, *Config, error) {
	cfg := &Config{}
	dir, err := utils.GetCurrentDirectory()
	if err != nil {
		return nil, cfg, err
	}
	app := kong.Parse(cfg,
		kong.Name(APP_NAME),
		kong.Description("TODO"),
		kong.UsageOnError(),
		kong.Configuration(cli.YAML, filepath.Join(dir, strings.ToLower(APP_NAME)+".yaml")),
		kong.DefaultEnvars(strings.ToUpper(APP_NAME)),
		defaultValues,
	)
	cfg.RsshOrder = utils.ToLower(utils.Unique(cfg.RsshOrder))
	log.Info().Msgf("%+v", cfg)
	return app, cfg, nil
}
