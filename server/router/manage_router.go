package router

import (
	"Goauld/common/log"
	"Goauld/server/config"
	"Goauld/server/persistence"
	"Goauld/server/router/midleware"
	"Goauld/server/store"
	"encoding/json"
	"fmt"
	"github.com/urfave/negroni"
	"net/http"
	"strconv"
)

// ManageRouter is the router used by the management API
type ManageRouter struct {
	userRouter *http.ServeMux
	db         *persistence.DB
	store      *store.AgentStore
}

// NewManageRouter returns a new ManageRouter
func NewManageRouter(_db *persistence.DB, store *store.AgentStore) *ManageRouter {
	r := &ManageRouter{
		db:         _db,
		userRouter: http.NewServeMux(),
		store:      store,
	}
	r.userRouter.HandleFunc("/agent/{id}", r.GetAGentById)
	r.userRouter.HandleFunc("/agent/", r.GetAgents)
	r.userRouter.HandleFunc("POST /clearport/", r.ClearPort)
	return r
}

// GetRouter returns the router, with the middleware configures
// - Authentication middleware
// - IP whitelisting middleware
func (ur *ManageRouter) GetRouter() *negroni.Negroni {
	n := negroni.New()
	n.Use(midleware.AuthMiddleware(config.Get().AccessToken))
	n.Use(midleware.WhitelistMiddleware(config.Get().AllowedIPs))
	return n
}

// GetAGentById handles the /agent/{id} endpoints
// it returns to the caller the associated agent
func (ur *ManageRouter) GetAGentById(w http.ResponseWriter, r *http.Request) {
	// Find the agent corresponding to the id
	id := r.PostFormValue("id")
	agent, err := ur.db.FindAgent(id)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Str("ID", id).Msg("find agent failed")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if agent == nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Str("ID", id).Msg("No agent found with corresponding id")
		http.NotFound(w, r)
		return
	}

	// Generate a JSON of the agent
	jsonAgent, err := agent.JSON()
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Str("ID", id).Msg("error generating json response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	// return the json agent to the caller
	_, err = w.Write(jsonAgent)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Str("ID", id).Msg("error returning response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetAgents return all agents stored in the database
func (ur *ManageRouter) GetAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := ur.db.GetAllAgents()
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("find all agents failed")
		return
	}
	jsonAgent, err := json.Marshal(agents)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("error generating json response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	_, err = w.Write(jsonAgent)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("error returning response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// ClearPortData is the data type used to retrieve the clearPort endpoint
type ClearPortData struct {
	AgentId string `json:"agentId"`
	Port    string `json:"port"`
}

// ClearPort delete all the remaining connections related to the agent and the port
func (ur *ManageRouter) ClearPort(w http.ResponseWriter, r *http.Request) {
	var data ClearPortData
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse JSON: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	if data.AgentId != "" {
		ur.ClearPortsByAgentId(data.AgentId, w, r)
		return
	}
	if data.Port != "" {
		ur.ClearPortByPortNumber(data.Port, w, r)
		return
	}
	http.Error(w, fmt.Sprintf("No agent id or port specified"), http.StatusBadRequest)
	log.Warn().Str("Path", r.URL.Path).Msg("No agent id or port specified")
}

// ClearPortByPortNumber clears all the remaining connections associated to the port
func (ur *ManageRouter) ClearPortByPortNumber(p string, w http.ResponseWriter, r *http.Request) {
	port, err := strconv.Atoi(p)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Str("Port", p).Msg("error converting port")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = ur.store.ClearByPort(port)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Str("Port", p).Msg("error clearing port")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ClearPortsByAgentId clears all the remaining connections associated to the agent
func (ur *ManageRouter) ClearPortsByAgentId(agentId string, w http.ResponseWriter, r *http.Request) {
	err := ur.store.ClearById(agentId)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Str("AgentId", agentId).Msg("error clearing port")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
