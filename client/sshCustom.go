package main

import (
	"Goauld/client/api"
	"Goauld/client/customssh"
	_ssh "Goauld/common/ssh"
	"errors"
	"fmt"
	"net"

	"golang.org/x/crypto/ssh"
)

func NewCustomSSH(apiClient *api.API, cfg ClientConfig, target string) (*CustomSSH, error) {
	customSSH := &CustomSSH{}
	err := customSSH.CustomSSH(apiClient, cfg, target)
	if err != nil {
		return nil, err
	}

	return customSSH, nil
}

type CustomSSH struct {
	ProxyConn   net.Conn
	ProxyClient *ssh.Client
	Conn        ssh.Conn
	SSHClient   *ssh.Client
}

func (c *CustomSSH) Request() {

}

func (c *CustomSSH) Copy() ([]byte, error) {
	// Open the custom control channel
	ch, _, err := c.SSHClient.OpenChannel(_ssh.Copy, nil)
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	cc := _ssh.NewChannel(ch)

	return cc.ReadResponse()
}

func (c *CustomSSH) Paste(data []byte) error {
	// Open the custom control channel
	ch, _, err := c.SSHClient.OpenChannel(_ssh.Paste, nil)
	if err != nil {
		return err
	}
	defer ch.Close()

	cc := _ssh.NewChannel(ch)

	return cc.WriteResponse(true, data)
}

func (c *CustomSSH) Close() error {
	errs := []error{}
	if c.SSHClient != nil {
		errs = append(errs, c.SSHClient.Close())
	}

	if c.Conn != nil {
		errs = append(errs, c.Conn.Close())
	}

	if c.ProxyClient != nil {
		errs = append(errs, c.ProxyClient.Close())
	}

	if c.ProxyConn != nil {
		errs = append(errs, c.ProxyConn.Close())
	}

	return errors.Join(errs...)
}

func (c *CustomSSH) CustomSSH(clientAPI *api.API, cfg ClientConfig, agentName string) error {
	agent, err := clientAPI.GetAgentByName(agentName)
	if err != nil {
		return err
	}
	if cfg.ShouldPrompt(agent) {
		_ = cfg.Prompt(agentName)
	}
	if cfg.ControlMaster {
		target := GetSockDir(agentName)
		conn, err := net.Dial("unix", target)
		if err != nil {
			return fmt.Errorf("error dialing to %s: %w", target, err)
		}
		ncc, chans, reqs, err := customssh.NewControlClientConn(conn)
		if err != nil {
			return fmt.Errorf("error dialing new ssh client to %s: %w", target, err)
		}
		client := ssh.NewClient(ncc, chans, reqs)

		c.SSHClient = client
	} else {
		proxyUser := agent.Name

		proxyPass := GenerateServerPassword(cfg.GetStaticPassword(), agent.OneTimePassword)

		proxyHost := cfg.GetSshdHost()
		proxyPort := cfg.GetSshdPort()
		proxy, err := proxyCommand(proxyUser, proxyPass, proxyHost, proxyPort)
		if err != nil {
			return err
		}

		c.ProxyClient = proxy

		agentPort := agent.GetSSHPort()

		pxyConn, err := proxy.Dial("tcp", fmt.Sprintf("%s:%s", "127.0.0.1", agentPort))
		if err != nil {
			return err
		}
		c.ProxyConn = pxyConn

		sshConfig := &ssh.ClientConfig{
			User:            fmt.Sprintf("%s@%s", agent.Name, agent.ID),
			Auth:            []ssh.AuthMethod{ssh.Password(agent.SSHPasswd)},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec
		}

		conn, chn, req, err := ssh.NewClientConn(pxyConn, "127.0.0.1", sshConfig)
		if err != nil {
			return err
		}
		c.Conn = conn

		c.SSHClient = ssh.NewClient(conn, chn, req)
	}

	return nil
}

func proxyCommand(user string, password string, host string, port string) (*ssh.Client, error) {
	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec
	}

	return ssh.Dial("tcp", fmt.Sprintf("%s:%s", host, port), cfg)
}
