package transport

import (
	"Goauld/agent/agent"
	"Goauld/agent/proxy"
	"context"
	"fmt"
	"io"
	"net"
	"nhooyr.io/websocket"
)

func GetWebsocketConn(ctx context.Context) (net.Conn, error) {
	url := agent.Get().WSshUrl()

	httpclient := proxy.NewHttpClientProxy()

	wsConn, resp, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		HTTPClient: httpclient,
	})
	if err != nil {
		if resp != nil && resp.Body != nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("websocket dial error, got response: %s: %s", body, err)
		}
		return nil, fmt.Errorf("websocket dial error: %v", err)
	}

	netConn := websocket.NetConn(ctx, wsConn, websocket.MessageBinary)

	return netConn, nil
}
