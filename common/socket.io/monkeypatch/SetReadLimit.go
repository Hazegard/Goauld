//go:build !mini

package monkeypatch

import "github.com/coder/websocket"

func SetReadLimit(conn *websocket.Conn) *websocket.Conn {
	return conn
}
