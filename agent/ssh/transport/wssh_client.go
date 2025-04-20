package transport

import (
	"context"
	"fmt"
	"io"
	"net"

	"Goauld/agent/config"
	"Goauld/agent/proxy"

	"nhooyr.io/websocket"
)

// GetWebsocketConn returns a net.Conn wrapping a websocket connection
func GetWebsocketConn(ctx context.Context) (net.Conn, error) {
	url := config.Get().WSshUrl()

	httpclient := proxy.NewHttpClientProxy()

	// Attempt to connect to the websocket server
	wsConn, resp, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		HTTPClient: httpclient,
	})
	if err != nil {
		if wsConn != nil {
			_ = wsConn.Close(0, "timeout")
		}
		if resp != nil && resp.Body != nil {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			return nil, fmt.Errorf("websocket dial error, got response: %s: %s", body, err)
		}
		return nil, fmt.Errorf("websocket dial error: %v", err)
	}

	// Wraps the websocket connection to expose it as a raw net.Conn connection
	netConn := websocket.NetConn(ctx, wsConn, websocket.MessageBinary)

	return netConn, nil
}
