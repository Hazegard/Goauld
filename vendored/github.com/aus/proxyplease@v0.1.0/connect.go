package proxyplease

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ErrorConn struct {
	Conn   net.Conn
	Buffer *bytes.Buffer
}

func (e ErrorConn) Read(b []byte) (n int, err error) {
	return e.Buffer.Read(b)
}

func (e ErrorConn) Write(b []byte) (n int, err error) {
	return len(b), nil
}

func (e ErrorConn) Close() error {
	return e.Conn.Close()
}

func (e ErrorConn) LocalAddr() net.Addr {
	return e.Conn.LocalAddr()
}

func (e ErrorConn) RemoteAddr() net.Addr {
	return e.Conn.RemoteAddr()
}

func (e ErrorConn) SetDeadline(t time.Time) error {
	return e.Conn.SetDeadline(t)
}

func (e ErrorConn) SetReadDeadline(t time.Time) error {
	return e.Conn.SetReadDeadline(t)
}

func (e ErrorConn) SetWriteDeadline(t time.Time) error {
	return e.Conn.SetWriteDeadline(t)
}

func dialAndNegotiateHTTP(p Proxy, addr string, baseDial func() (net.Conn, error)) (net.Conn, error) {
	// establish TCP with proxy. baseDial will negoiate TLS if needed.
	conn, err := baseDial()
	if err != nil {
		debugf("connect> Could not call dial context with proxy: %s", err)
		return conn, err
	}

	// build and write first CONNECT request
	h := p.Headers.Clone()
	h.Set("Proxy-Connection", "Keep-Alive")
	connect := &http.Request{
		Method: "CONNECT",
		URL:    &url.URL{Opaque: addr},
		Host:   addr,
		Header: h,
	}
	if err := connect.Write(conn); err != nil {
		debugf("connect> CONNECT to proxy failed: %s", err)
		return conn, err
	}

	// read first response
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, connect)
	if err != nil {
		debugf("connect> Could not read response from proxy: %s", err)
		return conn, err
	}

	// if StatusOK, no auth is required and proxy is established
	if resp.StatusCode == http.StatusOK {
		debugf("connect> Proxy successfully established. No authentication was required.")
		return conn, nil
	}

	// if authentication is required
	if resp.StatusCode == http.StatusProxyAuthRequired {
		debugf("connect> Proxy authentication is required. Attempting to select a authentication scheme.")

		// read authentication scheme options
		schemes := resp.Header["Proxy-Authenticate"]
		for _, s := range schemes {
			// only test for first word in scheme
			trimmed := strings.Split(s, " ")[0]
			switch trimmed {
			case "NTLM":
				if !contains(p.AuthSchemeFilter, "NTLM") {
					debugf("connect> Skipping NTLM due to AuthSchemeFilter")
					continue
				}
				conn, err = dialNTLM(p, addr, baseDial)
				if err != nil {
					debugf("connect> NTLM authentication failed. Trying next available scheme.")
					continue
				}
				return conn, err
			case "Basic":
				if !contains(p.AuthSchemeFilter, "Basic") {
					debugf("connect> Skipping Basic due to AuthSchemeFilter")
					continue
				}
				conn, err = dialBasic(p, addr, baseDial)
				if err != nil {
					debugf("connect> Basic authentication failed. Trying next available scheme.")
					continue
				}
				return conn, err

			case "Negotiate":
				if !contains(p.AuthSchemeFilter, "Negotiate") {
					debugf("connect> Skipping Negotiate due to AuthSchemeFilter")
					continue
				}
				conn, err = dialNegotiate(p, addr, baseDial)
				if err != nil {
					debugf("connect> Negotiate authentication failed. Trying next available scheme.")
					continue
				}
				return conn, err

			case "Kerberos":
				debugf("connect> Kerberos not implemented yet. Trying next available scheme.")
				continue

			case "Digest":
				debugf("connect> Digest not implemented yet. Trying next available scheme.")
				continue

			default:
				debugf("connect> Unsupported proxy authentication scheme: '%s'. Trying next available scheme.", trimmed)
				continue
			}
		}

		debugf("connect> No proxy authentication completed successfully")
		return conn, err
	}

	debugf("connect> Unhandled HTTP status, got: %d", resp.StatusCode)
	//content, err := io.ReadAll(resp.Body)

	buf := bytes.NewBuffer(nil)
	_ = resp.Write(buf)

	return ErrorConn{Conn: conn, Buffer: buf}, nil
	//return conn, errors.New(string(buf.Bytes()))
}

func contains(s []string, e string) bool {
	// if no filter supplied, assume scheme is wanted
	if s == nil {
		return true
	}
	// otherwise, test if filter matches
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
