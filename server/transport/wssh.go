package transport

import (
	"Goauld/common/log"
	"Goauld/server/config"
	"Goauld/server/db"
	"Goauld/server/store"
	"context"
	"io"
	"net"
	"net/http"
	"nhooyr.io/websocket"
	"time"
)

func NewWSshHandler(agentStore *store.AgentStore, db *db.DB) *WSshHandler {
	return &WSshHandler{
		agentStore: agentStore,
		db:         db,
	}
}

type WSshHandler struct {
	agentStore *store.AgentStore
	db         *db.DB
}

func (wssh *WSshHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	id := r.PathValue("agentId")
	agent, err := wssh.db.FindAgent(id)
	if err != nil {
		log.Error().Err(err).Msgf("WSSH: unable to find agent %s", id)
		return
	}

	host := "127.0.0.1"
	port := config.Get().SshPort

	wsConn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msgf("WSSH: error initiating websocket connection %s", id)
		return
	}

	defer wsConn.CloseNow()

	targetConn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 10*time.Second)
	if err != nil {
		log.Error().Err(err).Msgf("WSSH: failed to connect to %q:%q (%s)", host, port, id)
		return
	}
	defer targetConn.Close()
	conn := websocket.NetConn(ctx, wsConn, websocket.MessageBinary)

	wssh.agentStore.WsshAddAgent(agent, id, conn, targetConn)
	errChan := make(chan error, 1)
	go func() {
		_, err := io.Copy(conn, targetConn)
		if err != nil {
			log.Error().Err(err).Msgf("WSSH: ws -> ssh connection failed (%s)", id)
			errChan <- err
		}
	}()
	go func() {
		_, err := io.Copy(targetConn, conn)
		if err != nil {
			log.Error().Err(err).Msgf("WSSH: ssh -> ws connection failed (%s)", id)
			errChan <- err
		}
	}()
	err = <-errChan
	if err != nil {
		log.Error().Err(err).Msgf("WSSH: error durring copy (%s)", id)
	}

	err = wssh.agentStore.WsshCloseAgent(id)
	if err != nil {
		log.Error().Err(err).Msgf("WSSH: error while closing websocket streams (%s)", id)
	}
}

func NewWSshRouter() {

}
