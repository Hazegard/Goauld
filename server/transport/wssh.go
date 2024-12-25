package transport

import (
	"Goauld/common/log"
	"Goauld/server/config"
	"Goauld/server/db"
	"Goauld/server/store"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"nhooyr.io/websocket"
	"strconv"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	id := r.PathValue("agentId")
	agent, err := wssh.db.FindAgent(id)
	if err != nil {
		log.Error().Err(err).Msgf("WSSH: unable to find agent %s", id)
		return
	}

	host := "127.0.0.1"
	port := config.Get().SshPort

	wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
		OriginPatterns:     []string{"*"},
	})
	if err != nil {
		log.Error().Err(err).Msgf("WSSH: error initiating websocket connection %s", id)
		return
	}

	defer wsConn.CloseNow()
	ssh := net.JoinHostPort(host, strconv.Itoa(port))
	fmt.Printf("WSSH: connecting to %s\n", ssh)
	targetConn, err := net.Dial("tcp", ssh)
	if err != nil {
		log.Error().Err(err).Msgf("WSSH: failed to connect to %q:%q (%s)", host, port, id)
		return
	}
	defer targetConn.Close()
	conn := websocket.NetConn(ctx, wsConn, websocket.MessageBinary)
	defer conn.Close()
	wssh.agentStore.WsshAddAgent(agent, id, conn, targetConn)
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

func NewWSshRouter() {

}
