package main

import "Goauld/client/api"

type Socks struct {
	Target         string   `arg:"" help:"The target agent."`
	Socks          bool     `default:"${_socks_socks}" name:"socks" negatable:""  optional:"" help:"Forward the SOCKS ports on the local host."`
	LocalSocksPort int      `default:"${_socks_local_socks_port}" name:"socks-port" optional:"" help:"Local port to bind the SOCKS to."`
	Ssh            bool     `default:"${_socks_ssh}" name:"ssh" negatable:""  optional:"" help:"Connect to the agent SSHD service."`
	Print          bool     `default:"${_socks_print}" name:"print" negatable:""  optional:"" help:"Show the SSH command instead of executing it."`
	Proxy          bool     `default:"${_socks_proxy}" name:"proxy" optional:"" help:"Enable direct STDIN/STDOUT connections to Allow to use proxycommand."`
	SshArgs        []string `arg:"" default:"${_ssh_ssh_args}" passthrough:"" optional:"" help:"Additional args directly passed to the SSH command."`
}

func (s *Socks) Run(api *api.API, cfg ClientConfig) error {
	ssh := &Ssh{
		Target:         s.Target,
		Socks:          s.Socks,
		LocalSocksPort: s.LocalSocksPort,
		Ssh:            s.Ssh,
		Print:          s.Print,
		Proxy:          s.Proxy,
		SshArgs:        s.SshArgs,
	}
	return ssh.Run(api, cfg)
}
