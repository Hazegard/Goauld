package transport

import (
	"io"
	"net"
	"net/http"

	"Goauld/common/log"
	"Goauld/server/config"
	"Goauld/server/persistence"
	"Goauld/server/store"
)

// SSHttpServer is the server allowing to perform SSH over HTTP
type SSHttpServer struct {
	store *store.AgentStore
	db    *persistence.DB
}

// NewSSHHttpServer returns a new server
func NewSSHHttpServer(store *store.AgentStore, db *persistence.DB) *SSHttpServer {
	return &SSHttpServer{
		store: store,
		db:    db,
	}
}

// ServeHTTP handles all the connections performed to the SSHTTP router
func (s *SSHttpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("agentId")
	// log.Trace()().Msgf("[SSHTTP] Received %s from %s", r.Method, id)
	switch r.Method {
	case http.MethodHead:
		s.StartSSH(id, w, r)
	case http.MethodGet:
		s.Get(id, w, r)
	case http.MethodPost:
		s.Post(id, w, r)
	case http.MethodDelete:
		s.StopSSH(id, w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// StartSSH handle the HEAD requests performed to router
// It initializes the connection to the SSH server
// And responds 200 OK of the connection succeeds
func (s *SSHttpServer) StartSSH(id string, w http.ResponseWriter, r *http.Request) {
	// Initialize the connection to the SSH server
	conn, err := net.Dial("tcp", config.Get().LocalSShAddr())
	// log.Trace()()().Msgf("[SSHTTP] Connect to %s", config.Get().LocalSShAddr())
	if err != nil {
		log.Error().Str("ID", id).Str("SSH Mode", "HTTP").Err(err).Msgf("[SSHTTP] Start SSH Server error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Adds the connections to the SSH HTTP store
	s.store.SshttpAddAgent(conn, id)
	// Adds the connection mode to the agent in the database
	err = s.db.SetAgentSshMode(id, "HTTP")
	if err != nil {
		log.Error().Str("ID", id).Str("SSH Mode", "HTTP").Err(err).Msgf("[SSHTTP] Unable to set agent SSH connection mode")
	}
	w.WriteHeader(http.StatusOK)
	// log.Trace()()().Msgf("[SSHTTP] DONE HEAD Server %s", id)
}

// Get reads from the SSH connections and returns its content as a response
func (s *SSHttpServer) Get(id string, w http.ResponseWriter, r *http.Request) {
	// log.Trace()()().Msgf("[SSHTTP] GET Server %s", id)

	// Get the agent connections from the store
	agent := s.store.SshttpGetAgent(id)
	if agent == nil {
		log.Error().Str("ID", id).Str("SSH Mode", "HTTP").Msg("Get Agent Not Found")
		http.Error(w, "agent not fount", http.StatusNotFound)
		return
	}
	// Reads from the SSH connection
	buffer := make([]byte, 10*1024*1024)
	n, err := agent.SshConn.Read(buffer)
	if err != nil {
		log.Error().Str("ID", id).Str("SSH Mode", "HTTP").Msg("SSH connection read error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	// Returns the read data to the caller
	_, err = w.Write(buffer[:n])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Post gets the data from the body of the request and sends it
// to the SSH connection
func (s *SSHttpServer) Post(id string, w http.ResponseWriter, r *http.Request) {
	// retrieve the SSH connection from the store
	agent := s.store.SshttpGetAgent(id)
	if agent == nil {
		log.Warn().Str("ID", id).Str("SSH Mode", "HTTP").Msgf("Agent Not Found")
		http.Error(w, "agent not fount", http.StatusNotFound)
		return
	}
	// read the body data of the request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Warn().Str("ID", id).Str("SSH Mode", "HTTP").Err(err).Msgf("[SSHTTP] Post Agent Read Body error")
		return
	}
	err = r.Body.Close()
	if err != nil {
		log.Warn().Err(err).Str("ID", id).Str("SSH Mode", "error closing HTTP body")
	}
	// Write the received data to the SSH connection
	_, err = agent.SshConn.Write(body)
	if err != nil {
		log.Warn().Str("ID", id).Str("SSH Mode", "HTTP").Msgf("[SSHTTP] Post Agent Write Error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
}

// StopSSH stops the SSH connection
func (s *SSHttpServer) StopSSH(id string, w http.ResponseWriter, r *http.Request) {
	// Closes the SSH connection
	err := s.store.SshttpCloseAgent(id)
	if err != nil {
		log.Warn().Str("ID", id).Str("SSH Mode", "HTTP").Err(err).Msgf("[SSHTTP] Stop SSH Server error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	// Updates the agent in the database to remove the SSH connection mode
	err = s.db.SetAgentSshMode(id, "OFF")
	if err != nil {
		log.Warn().Str("ID", id).Str("SSH Mode", "HTTP").Err(err).Msgf("Unable to update SSH agent mode to [OFF]")
	}
}
