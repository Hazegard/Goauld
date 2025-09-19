package router

import (
	"Goauld/common"
	"Goauld/server/control"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"Goauld/common/log"
	"Goauld/server/config"
	"Goauld/server/persistence"
	"Goauld/server/router/midleware"
	"Goauld/server/store"

	socket_io "Goauld/common/socket.io"

	"github.com/urfave/negroni"
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
	r.userRouter.HandleFunc("POST /agent/{id}/kill", r.KillAgent)
	r.userRouter.HandleFunc("GET /agent/{id}", r.GetAgentById)
	r.userRouter.HandleFunc("DELETE /agent/{id}", r.DeleteAgentById)
	r.userRouter.HandleFunc("GET /agent/by_name/{name}", r.GetAgentByName)
	r.userRouter.HandleFunc("GET /agent/{$}", r.GetAgents)
	r.userRouter.HandleFunc("POST /clearport/{$}", r.ClearPort)
	r.userRouter.HandleFunc("GET /version/", r.Version)
	return r
}

// GetRouter returns the router, with the configured middleware
// - Authentication middleware
// - IP allowlisting middleware
func (mr *ManageRouter) GetRouter() *negroni.Negroni {
	n := negroni.New()
	n.Use(midleware.AuthMiddleware(config.Get().AccessToken))
	n.Use(midleware.WhitelistMiddleware(config.Get().AllowedIPs))
	n.UseHandler(mr.userRouter)
	return n
}

func (mr *ManageRouter) Version(w http.ResponseWriter, r *http.Request) {
	res, err := json.Marshal(common.JsonVersion())
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("error generating response json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	_, err = w.Write(res)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("write response failed")
	}
}

// GetAgentByName return the agent information
func (mr *ManageRouter) GetAgentByName(w http.ResponseWriter, r *http.Request) {
	// Find the agent corresponding to the name
	name := r.PathValue("name")
	agent, err := mr.db.FindAgentByName(name)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Str("Name", name).Msg("find agent failed")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if agent == nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Str("Name", name).Msg("No agent found with corresponding name")
		http.NotFound(w, r)
		return
	}
	agent.SharedSecret = ""
	agent.PrivateKey = ""
	agent.PublicKey = ""

	// Generate a JSON of the agent
	jsonAgent, err := agent.JSON()
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Str("Name", name).Msg("error generating json response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	// return the JSON agent to the caller
	_, err = w.Write(jsonAgent)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Str("ID", name).Msg("error returning response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetAgentById handles the /agent/{id} endpoints
// it returns to the caller the associated agent
func (mr *ManageRouter) GetAgentById(w http.ResponseWriter, r *http.Request) {
	// Find the agent corresponding to the id
	id := r.PathValue("id")
	agent, err := mr.db.FindAgentById(id)
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

	agent.SharedSecret = ""
	agent.PrivateKey = ""
	agent.PublicKey = ""

	// Generate a JSON of the agent
	jsonAgent, err := agent.JSON()
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Str("ID", id).Msg("error generating json response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	// return the JSON agent to the caller
	_, err = w.Write(jsonAgent)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Str("ID", id).Msg("error returning response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// DeleteAgentById kills the agent and deletes the remaining connections
func (mr *ManageRouter) DeleteAgentById(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := mr.store.KillAGent(id, true)
	if err != nil {
		// http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Warn().Err(err).Str("Path", r.URL.Path).Str("ID", id).Msg("error killing agent")
	}
	err = mr.db.DeleteAgentById(id)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Str("ID", id).Msg("delete agent failed")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Info().Str("Path", r.URL.Path).Str("ID", id).Msg("delete agent success")
	http.Redirect(w, r, "/", http.StatusNoContent)
}

// GetAgents return all agents stored in the database
func (mr *ManageRouter) GetAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := mr.db.GetAllAgentsSanitized()
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("find all agents failed")
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
func (mr *ManageRouter) ClearPort(w http.ResponseWriter, r *http.Request) {
	var data ClearPortData
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse JSON: %v", err), http.StatusBadRequest)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Warn().Err(err).Str("SSH Mode", "error closing HTTP body")
		}
	}(r.Body)

	if data.AgentId != "" {
		mr.ClearPortsByAgentId(data.AgentId, w, r)
		return
	}
	if data.Port != "" {
		mr.ClearPortByPortNumber(data.Port, w, r)
		return
	}
	http.Error(w, "No agent id or port specified", http.StatusBadRequest)
	log.Warn().Str("Path", r.URL.Path).Msg("No agent id or port specified")
}

// ClearPortByPortNumber clears all the remaining connections associated to the port
func (mr *ManageRouter) ClearPortByPortNumber(p string, w http.ResponseWriter, r *http.Request) {
	port, err := strconv.Atoi(p)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Str("Port", p).Msg("error converting port")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = mr.store.ClearByPort(port)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Str("Port", p).Msg("error clearing port")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ClearPortsByAgentId clears all the remaining connections associated to the agent
func (mr *ManageRouter) ClearPortsByAgentId(agentId string, w http.ResponseWriter, r *http.Request) {
	err := mr.store.ClearById(agentId)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Str("AgentId", agentId).Msg("error clearing port")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// KillAgent kill the agent
func (mr *ManageRouter) KillAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Str("ID", id).Msg("error reading body")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonBody := socket_io.ExitRequest{}
	err = json.Unmarshal(body, &jsonBody)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Str("ID", id).Msg("error unmarshalling json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	doRequest := HasAdminToken(r)

	if !doRequest {
		socket := mr.store.SioGetSocket(id)
		agent := mr.store.SioGetAgent(socket)

		checkPwd := agent.HasStaticPassword

		if jsonBody.Delete && !agent.Connected {
			checkPwd = false
		}

		if checkPwd {
			isStaticPwdValid := control.ValidateStaticPassword(agent, socket, string(jsonBody.HashedPassword))
			if isStaticPwdValid {
				doRequest = true
			}
		} else {
			doRequest = true
		}
	}
	if doRequest {
		err = mr.store.KillAGent(id, jsonBody.Kill)
		if err != nil {
			log.Warn().Err(err).Str("Path", r.URL.Path).Str("ID", id).Msg("error killing agent")
		}
		err = mr.store.CloseAgentConnections(id)
		if err != nil {
			//http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Warn().Err(err).Str("Path", r.URL.Path).Str("ID", id).Msg("error killing agent")
			//return
		}
		if jsonBody.Delete {
			err = mr.db.DeleteAgentById(id)
			if err != nil {
				log.Warn().Err(err).Str("Path", r.URL.Path).Str("ID", id).Msg("delete agent failed")
				//http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	}
	http.Error(w, "Invalid password", http.StatusBadRequest)
}

func HasAdminToken(r *http.Request) bool {
	// Extract the Authorization header
	authHeader := r.Header.Get("Authorization")
	// Validate the Authorization header
	if len(authHeader) < 2 {
		return false
	}
	authHeader = strings.Split(authHeader, ":")[1]
	return authHeader == config.Get().AdminToken
}
