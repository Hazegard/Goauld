//go:build !windows

package proxy

import (
	"errors"
	"net/http"

	"github.com/elazarl/goproxy"
)

// MITMHTTPProxy holds the HTTP proxy that performs mitm to inject NTLM/Kerberos authentication.
type MITMHTTPProxy struct {
	Proxy    *goproxy.ProxyHttpServer
	Dialer   *ProxyDialer
	Server   *http.Server
	Username string
	Password string
	Domain   string
}

func InitMITMHTTPProxy(_ string, _ string, _ string) (*MITMHTTPProxy, error) {
	return nil, errors.New("not implemented")
}
