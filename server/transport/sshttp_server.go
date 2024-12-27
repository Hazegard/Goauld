package transport

import (
	"Goauld/common/log"
	"Goauld/server/config"
	"Goauld/server/store"
	"io"
	"net"
	"net/http"
)

type SSHttpServer struct {
	store *store.AgentStore
}

func NewSSHHttpServer(store *store.AgentStore) *SSHttpServer {
	return &SSHttpServer{
		store: store,
	}
}

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
	return
}

func (s *SSHttpServer) StartSSH(id string, w http.ResponseWriter, r *http.Request) {
	// log.Trace()()().Msgf("[SSHTTP] HEAD Server %s", id)
	conn, err := net.Dial("tcp", config.Get().LocalSShServer())
	// log.Trace()()().Msgf("[SSHTTP] Connect to %s", config.Get().LocalSShServer())
	if err != nil {
		log.Error().Err(err).Msgf("[SSHTTP] Start SSH Server error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.store.SshttpAddAgent(conn, id)
	w.WriteHeader(http.StatusOK)
	// log.Trace()()().Msgf("[SSHTTP] DONE HEAD Server %s", id)

}

func (s *SSHttpServer) Get(id string, w http.ResponseWriter, r *http.Request) {
	// log.Trace()()().Msgf("[SSHTTP] GET Server %s", id)

	agent := s.store.SshttpGetAgent(id)
	if agent == nil {
		log.Error().Msgf("[SSHTTP] Get Agent Not Found")
		http.Error(w, "agent not fount", http.StatusNotFound)
		return
	}
	buffer := make([]byte, 10*1024*1024)
	n, err := agent.SshConn.Read(buffer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	// log.Trace()().Msgf("[SSHTTP] Read %d bytes from %s connection", n, id)
	n, err = w.Write(buffer[:n])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	// log.Trace()().Msgf("[SSHTTP] DONE GET Server %s (%d bytes)", id, n)
}

func (s *SSHttpServer) Post(id string, w http.ResponseWriter, r *http.Request) {
	// log.Trace()().Msgf("[SSHTTP] POST Server %s", id)
	agent := s.store.SshttpGetAgent(id)
	if agent == nil {
		log.Error().Msgf("[SSHTTP] Get Agent Not Found")
		http.Error(w, "agent not fount", http.StatusNotFound)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	r.Body.Close()
	// log.Trace()().Msgf("[SSHTTP] Received %d bytes from %s", len(body), id)
	_, err = agent.SshConn.Write(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	// log.Trace()().Msgf("[SSHTTP] Written %d bytes to %s ssh conn", n, id)
	w.WriteHeader(http.StatusOK)
	// log.Trace()().Msgf("[SSHTTP] DONE POST Server %s", id)
}

func (s *SSHttpServer) StopSSH(id string, w http.ResponseWriter, r *http.Request) {
	err := s.store.SshttpCloseAgent(id)
	if err != nil {
		log.Error().Err(err).Msgf("[SSHTTP] Stop SSH Server error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
