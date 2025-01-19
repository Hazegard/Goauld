package net

import (
	"net/http"
	"strings"
)

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
