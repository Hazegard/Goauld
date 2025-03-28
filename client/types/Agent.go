package types

import (
	"strconv"
	"strings"

	"Goauld/common/types"
)

type Agent struct {
	types.Agent
}

// GetSSHPort returns the SSHD forwarded port
func (a *Agent) GetSSHPort() string {
	for _, rpf := range a.RemotePortForwarding {
		if strings.EqualFold(rpf.Tag, "sshd") {
			return strconv.Itoa(rpf.ServerPort)
		}
	}
	return "/"
}

// GetOtherPort returns the forwarded port, excepted the SSHD or Socks ports
func (a *Agent) GetOtherPort() string {
	var ports []string
	for _, rpf := range a.RemotePortForwarding {
		if strings.EqualFold(rpf.Tag, "socks") || strings.EqualFold(rpf.Tag, "sshd") || strings.EqualFold(rpf.Tag, "http") || rpf.ServerPort == 0 {
			continue
		}
		ports = append(ports, rpf.String())
	}
	res := strings.Join(ports, ", ")
	if res == "" {
		res = "/"
	}
	return res
}

// GetSocksPort returns the socks forwarded port
func (a *Agent) GetSocksPort() string {
	for _, rpf := range a.RemotePortForwarding {
		if strings.EqualFold(rpf.Tag, "socks") {
			return strconv.Itoa(rpf.ServerPort)
		}
	}
	return "/"
}

// GetHttpPort returns the socks forwarded port
func (a *Agent) GetHttpPort() string {
	for _, rpf := range a.RemotePortForwarding {
		if strings.EqualFold(rpf.Tag, "http") {
			return strconv.Itoa(rpf.ServerPort)
		}
	}
	return "/"
}
