//go:build mini
// +build mini

package proxyplease

import (
	"errors"
	"net"
	"net/url"
)

func dialAndNegotiateSOCKS(u *url.URL, user, pass, addr string) (net.Conn, error) {
	return nil, errors.New("Not supported")
}
