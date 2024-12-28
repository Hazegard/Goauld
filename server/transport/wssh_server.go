package transport

import (
	"Goauld/common/log"
	"Goauld/server/config"
	"Goauld/server/store"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"nhooyr.io/websocket"
)

func NewWSshHandler(agentStore *store.AgentStore) *WSshHandler {
	return &WSshHandler{
		agentStore: agentStore,
	}
}

type WSshHandler struct {
	agentStore *store.AgentStore
}

func (wssh *WSshHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	id := r.PathValue("agentId")

	wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
		OriginPatterns:     []string{"*"},
	})
	if err != nil {
		log.Error().Err(err).Msgf("WSSH: error initiating websocket connection %s", id)
		return
	}

	defer wsConn.CloseNow()
	log.Info().Msgf("WSSH: connecting to agent SSH server %s", config.Get().LocalSShServer())
	targetConn, err := net.Dial("tcp", config.Get().LocalSShServer())
	if err != nil {
		log.Error().Err(err).Msgf("WSSH: failed to connect to %s (%s)", config.Get().LocalSShServer(), id)
		return
	}
	defer targetConn.Close()
	conn := websocket.NetConn(ctx, wsConn, websocket.MessageBinary)
	defer conn.Close()
	wssh.agentStore.WsshAddAgent(id, conn, targetConn)
	errChan := make(chan error, 1)
	go func() {
		_, err := io.Copy(conn, targetConn)
		if err != nil && !errors.Is(err, io.EOF) {
			log.Error().Err(err).Msgf("WSSH: ws -> ssh connection failed (%s)", id)
			errChan <- err
		}
	}()
	go func() {
		_, err := io.Copy(targetConn, conn)
		if err != nil && !errors.Is(err, io.EOF) {
			log.Error().Err(err).Msgf("WSSH: ssh -> ws connection failed (%s)", id)
			errChan <- err
		}
	}()
	err = <-errChan
	if err != nil {
		log.Error().Err(err).Msgf("WSSH: error during copy (%s)", id)
	}

	err = wssh.agentStore.WsshCloseAgent(id)
	if err != nil {
		log.Error().Err(err).Msgf("WSSH: error while closing websocket streams (%s)", id)
	}
}
