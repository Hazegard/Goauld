package transport

import (
	"Goauld/agent/config"
	"Goauld/agent/proxy"
	"context"
	"fmt"
	"io"
	"net"

	"github.com/coder/websocket"
)

// GetWebsocketConn returns a net.Conn wrapping a websocket connection.
func GetWebsocketConn(localCtx context.Context, globalCtx context.Context, id string) (net.Conn, error) {
	url := config.Get().WSshURL(id)

	httpclient := proxy.NewHTTPClientProxy(nil)

	// Attempt to connect to the websocket server
	wsConn, resp, err := websocket.Dial(localCtx, url, &websocket.DialOptions{
		HTTPClient: httpclient,
	})
	if err != nil {
		if wsConn != nil {
			_ = wsConn.Close(0, "timeout")
		}
		if resp != nil && resp.Body != nil {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()

			return nil, fmt.Errorf("websocket dial error, got response: %s: %w", body, err)
		}

		return nil, fmt.Errorf("websocket dial error: %w", err)
	}

	// Wraps the websocket connection to expose it as a raw net.Conn connection
	netConn := websocket.NetConn(globalCtx, wsConn, websocket.MessageBinary)

	return netConn, nil
}
