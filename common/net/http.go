package net

import (
	"errors"
	"net/http"
	"strings"
)

// Http10ToHttp11FakeUpgrader perform a fake upgrade from HTTP/1.0 to HTTP/1.1
// If the agent runs through a proxy forcing HTTP/1.0.
// Even though HTTP/1.0 does not really support websockets, it sometimes helps as the websocket library used
// rejects HTTP/1.0 connection.
// See: https://github.com/coder/websocket/blob/246891f172ef96b0b5681c8e4d59dfd32ad1b091/accept.go#L186
func Http10ToHttp11FakeUpgrader(r *http.Request) *http.Request {
	if r.ProtoAtLeast(1, 1) {
		return r
	}
	if r.Proto == "HTTP/1.0" {
		r.Proto = "HTTP/1.1"
		r.ProtoMajor = 1
		r.ProtoMinor = 1
	}
	if r.Header.Get("Connection") == "" || strings.EqualFold(r.Header.Get("Connection"), "close") {
		r.Header.Set("Connection", "Upgrade")
	}
	if r.Header.Get("Upgrade") == "" {
		r.Header.Set("Upgrade", "WebSocket")
	}
	return r
}

const Forbidden = "Forbidden"
const Unauthorized = "Unauthorized"

var ForbiddenErr = errors.New("Forbidden")
var UnauthorizedErr = errors.New("Unauthorized")
