package main

import (
	"Goauld/client/api"
)

type Socks struct {
	Target         string   `arg:"" name:"agent" help:"The target agent."`
	Socks          bool     `default:"${_socks_socks}" name:"socks" yaml:"socks" negatable:""  optional:"" help:"Forward the SOCKS ports on the local host."`
	HTTP           bool     `default:"${_ssh_http}" name:"http" yaml:"http" negatable:"" optional:"" help:"Forward the HTTP proxy ports on the local host."`
	LocalSocksPort int      `default:"${_socks_local_socks_port}" name:"socks-port" yaml:"socks-port" optional:"" help:"Local port to bind the SOCKS to."`
	LocalHTTPPort  int      `default:"${_ssh_local_http_port}" name:"http-port" yaml:"http-port" optional:"" help:"Local port to bind the SOCKS to."`
	SSH            bool     `default:"${_socks_ssh}" name:"ssh" yaml:"ssh" negatable:"" optional:"" help:"Connect to the agent SSHD service."`
	Print          bool     `default:"${_socks_print}" name:"print" yaml:"print" negatable:""  optional:"" help:"Show the SSH command instead of executing it."`
	Proxy          bool     `default:"${_socks_proxy}" name:"proxy" yaml:"proxy" optional:"" help:"Enable direct STDIN/STDOUT connections to Allow to use proxycommand."`
	SSHArgs        []string `arg:"" passthrough:"" optional:"" help:"Additional args directly passed to the SSH command."`
}

// Run execute the socks command.
func (s *Socks) Run(clientAPI *api.API, cfg ClientConfig) error {
	ssh := &SSH{
		Target:         s.Target,
		Socks:          s.Socks,
		HTTP:           s.HTTP,
		LocalSocksPort: s.LocalSocksPort,
		LocalHTTPPort:  s.LocalHTTPPort,
		SSH:            s.SSH,
		Print:          s.Print,
		Proxy:          s.Proxy,
		SSHArgs:        s.SSHArgs,
	}

	return ssh.Run(clientAPI, cfg)
}
