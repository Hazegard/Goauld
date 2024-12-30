package router

import (
	"Goauld/common/log"
	"Goauld/server/persistence"
	"Goauld/server/store"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type UserRouter struct {
	userRouter *http.ServeMux
	db         *persistence.DB
	store      *store.AgentStore
}

func NewUserRouter(_db *persistence.DB, store *store.AgentStore) *UserRouter {
	r := &UserRouter{
		db:         _db,
		userRouter: http.NewServeMux(),
		store:      store,
	}
	r.userRouter.HandleFunc("/agent/{id}", r.GetAGentById)
	r.userRouter.HandleFunc("/agent/", r.GetAgents)
	r.userRouter.HandleFunc("POST /clearport/", r.ClearPort)
	return r
}

func (ur *UserRouter) GetAGentById(w http.ResponseWriter, r *http.Request) {
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

	jsonAgent, err := agent.JSON()
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Str("ID", id).Msg("error generating json response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	_, err = w.Write(jsonAgent)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Str("ID", id).Msg("error returning response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (ur *UserRouter) GetAgents(w http.ResponseWriter, r *http.Request) {
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

type ClearPortData struct {
	AgentId string `json:"agentId"`
	Port    string `json:"port"`
}

func (ur *UserRouter) ClearPort(w http.ResponseWriter, r *http.Request) {
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

func (ur *UserRouter) ClearPortByPortNumber(p string, w http.ResponseWriter, r *http.Request) {
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

func (ur *UserRouter) ClearPortsByAgentId(agentId string, w http.ResponseWriter, r *http.Request) {
	err := ur.store.ClearById(agentId)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Str("AgentId", agentId).Msg("error clearing port")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
