package transport

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"

	"Goauld/common/log"
	net2 "Goauld/common/net"
	"Goauld/server/config"
	"Goauld/server/persistence"
	"Goauld/server/store"

	"github.com/coder/websocket"
)

// NewWSshHandler returns a new WSshHandler
func NewWSshHandler(agentStore *store.AgentStore, db *persistence.DB) *WSshHandler {
	return &WSshHandler{
		agentStore: agentStore,
		db:         db,
	}
}

// WSshHandler handles the SSH over Websockets connections
type WSshHandler struct {
	agentStore *store.AgentStore
	db         *persistence.DB
}

// ServeHTTP handle the SSH over Websockets connections
func (wssh *WSshHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	id := r.PathValue("agentId")

	r = net2.Http10ToHttp11FakeUpgrader(r)

	// Handle the websocket connection
	wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
		OriginPatterns:     []string{"*"},
	})
	if err != nil {
		log.Error().Err(err).Str("ID", id).Str("Mode", "WSSH").Msg("error initiating websocket connection")
		return
	}

	defer func(wsConn *websocket.Conn) {
		err := wsConn.CloseNow()
		if err != nil {
			log.Warn().Err(err).Str("ID", id).Str("Mode", "WSSH").Msg("error closing connection")
		}
	}(wsConn)
	log.Info().Str("ID", id).Err(err).Str("Mode", "WS").Msgf("connecting to agent SSH server %s", config.Get().LocalSShAddr())
	// Initializes the connection to the SSH server
	targetConn, err := net.Dial("tcp", config.Get().LocalSShAddr())
	if err != nil {
		log.Error().Err(err).Str("ID", id).Err(err).Str("Mode", "WS").Msgf("failed to connect to %s", config.Get().LocalSShAddr())
		return
	}
	defer func(targetConn net.Conn) {
		err := targetConn.Close()
		if err != nil {
			log.Warn().Err(err).Str("ID", id).Err(err).Str("Mode", "WS").Msg("failed to close SSH connection")
		}
	}(targetConn)

	// Convert the websocket connection to a raw net.Conn connection
	conn := websocket.NetConn(ctx, wsConn, websocket.MessageBinary)
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Warn().Err(err).Str("ID", id).Err(err).Str("Mode", "WS").Msg("failed to close websocket connection")
		}
	}(conn)

	// Adds the agent to the websocket store
	wssh.agentStore.WsshAddAgent(id, conn, targetConn)
	errChan := make(chan error, 1)

	// Initialize the Websocket -> SSH connection
	go func() {
		_, err := io.Copy(conn, targetConn)
		if err != nil && !errors.Is(err, io.EOF) {
			log.Error().Err(err).Str("ID", id).Err(err).Str("Mode", "WS").Msg("ws -> ssh connection failed (%s)")
			errChan <- err
		}
	}()

	// Initialize the SSH -> Websocket connection
	go func() {
		_, err := io.Copy(targetConn, conn)
		if err != nil && !errors.Is(err, io.EOF) {
			log.Error().Err(err).Str("ID", id).Err(err).Str("Mode", "WS").Msg("ssh -> ws connection failed")
			errChan <- err
		}
	}()
	// Updates the database to add the Websocket over SSH as the connection mode
	err = wssh.db.SetAgentSshMode(id, "WS", r.RemoteAddr)
	if err != nil {
		log.Warn().Str("ID", id).Err(err).Str("Mode", "WS").Msg("error setting agent mode to WS")
	}

	// Waits for an error to occur, either in the
	// SSH -> Websocket connection or in the Websocket -> SSH connection
	err = <-errChan
	if err != nil {
		log.Error().Err(err).Str("ID", id).Err(err).Str("Mode", "WS").Msg("error during copy")
	}

	// Closes all remaining connections of the agent
	err = wssh.agentStore.WsshCloseAgent(id)
	if err != nil {
		log.Error().Err(err).Str("ID", id).Err(err).Str("Mode", "WS").Msg("error while closing websocket streams")
	}

	// Updates the database to set the agent mode as disconnected
	err = wssh.db.SetAgentSshMode(id, "OFF", "")
	if err != nil {
		log.Warn().Str("ID", id).Err(err).Str("Mode", "WS").Msg("error setting agent mode to [OFF]")
	}
}
