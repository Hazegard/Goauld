//go:build !mini

package monkeypatch

func SetReadLimit(conn *websocket.Conn) *websocket.Conn {
	return conn
}
