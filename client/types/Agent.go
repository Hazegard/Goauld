package types

import (
	"Goauld/common/ssh"
	"Goauld/common/types"
	"strconv"
	"strings"
)

type Agent struct {
	types.Agent
}

func (a *Agent) GetSSHPort() string {
	for _, rpf := range a.Rpf {
		if strings.EqualFold(rpf.Tag, "sshd") {
			return strconv.Itoa(rpf.ServerPort)
		}
	}
	return "/"
}

func (a *Agent) GetOtherPort() string {
	var ports []string
	for _, rpf := range a.Rpf {
		if strings.EqualFold(rpf.Tag, "socks") || strings.EqualFold(rpf.Tag, "sshd") || rpf.ServerPort == 0 {
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

func (a *Agent) GetSocksPort() string {
	for _, rpf := range a.Rpf {
		if strings.EqualFold(rpf.Tag, "socks") {
			return strconv.Itoa(rpf.ServerPort)
		}
	}
	return "/"
}

func (a *Agent) ParseFPR() {
	rpfs := strings.Split(a.RemotePortForwarding, ",")
	for _, rpf := range rpfs {
		_rpf := ssh.RemotePortForwarding{}
		err := _rpf.UnmarshalText([]byte(rpf))
		if err != nil {
			continue
		}
		a.Rpf = append(a.Rpf, _rpf)
	}
}
