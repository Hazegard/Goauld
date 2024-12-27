package transport

import (
	"Goauld/common/log"
	socketio "Goauld/common/socket.io"
	"Goauld/server/config"
	"Goauld/server/store"
	gosio "github.com/karagenc/socket.io-go"
	"net"
	"sync"
)

type SSHTTP struct {
	ssHttpAgentStore *store.AgentStore
	server           *gosio.Server
}

func InitSSHTTPServer(agentStore *store.AgentStore) *gosio.Server {

	io := gosio.NewServer(&gosio.ServerConfig{})
	socketIO := &SSHTTP{
		ssHttpAgentStore: agentStore,
	}
	socketIO.Setup(io.Of("/"))
	err := io.Run()
	if err != nil {
	}

	return io
}

func (sio *SSHTTP) Setup(root *gosio.Namespace) {
	root.OnConnection(func(socket gosio.ServerSocket) {

		sentCounter := int64(0)
		receivedCounter := int64(0)
		queueMutex := sync.Mutex{}
		nextPackets := map[int64]socketio.SsHttp{}
		sshConn, err := net.Dial("tcp", config.Get().LocalSShServer())
		if err != nil {
			log.Error().Err(err).Msgf("[SSHTTP] error connecting to ssh server")
		}
		sio.ssHttpAgentStore.SshttpAddAgent(sshConn, socket)

		socket.OnEvent(socketio.SSHTTPEvent, func(data socketio.SsHttp) {

			//defer log.Trace().Msg("Receive packet start")
			queueMutex.Lock()
			defer queueMutex.Unlock()
			if data.Num == receivedCounter {
				//log.Trace().Msgf("writing... %s", strconv.Itoa(int(receivedCounter)))
				_, err := sshConn.Write(data.Data)
				if err != nil {
					log.Error().Err(err).Msgf("[SSHTTP] error writing to sshs server")
				}
				receivedCounter++
				for {
					if next, ok := nextPackets[receivedCounter]; ok {
						//log.Trace().Msgf("writing... %s", strconv.Itoa(int(receivedCounter)))
						_, err := sshConn.Write(next.Data)
						if err != nil {
							log.Error().Err(err).Msgf("[SSHTTP] error writing to sshs server")
						}
						delete(nextPackets, receivedCounter)
						receivedCounter++
					} else {
						break
					}
				}
			} else if data.Num > receivedCounter {
				nextPackets[data.Num] = data
			}
			//log.Trace().Msgf("Received packet %d (%d)", data.Num, len(data.Data))
			//defer log.Trace().Msg("Receive packet end")

		})

		socket.OnError(func(err error) {
			log.Error().Err(err).Msgf("[SSHTTP] error from server")
		})

		socket.OnDisconnect(func(reason gosio.Reason) {
			//agent := sio.ssHttpAgentStore.SshttpGetAgent(socket)
			log.Debug().Msgf("[SSHTTP] socketio.Disconnect: %s !", reason)
			_ = sio.ssHttpAgentStore.SshttpCloseAgent(socket)
		})

		for {
			//log.Trace().Msg("Read start")
			buf := make([]byte, 10*1024)
			n, err := sshConn.Read(buf)
			if err != nil {
				log.Error().Err(err).Msgf("[SSHTTP] error reading from SSH server (%s)")
				return
			}
			data := socketio.SsHttp{Data: buf[:n], Id: "todo", Num: sentCounter}
			//log.Trace().Msgf("[SSHTTP] Sent packet N°%d %d", data.Num, len(data.Data))
			sentCounter++
			socket.Emit(socketio.SSHTTPEvent, data)
			//log.Trace().Msg("Read end")
		}
	})
}
