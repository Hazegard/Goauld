// Package router holds the admin routes
package router

import (
	"Goauld/common"
	"Goauld/common/log"
	"Goauld/common/types"
	"Goauld/server/config"
	"Goauld/server/persistence"
	"Goauld/server/router/middleware"
	"Goauld/server/store"
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"
	"github.com/urfave/negroni"
)

// AdminRouter is the router used by the management API.
type AdminRouter struct {
	userRouter *http.ServeMux
	db         *persistence.DB
	store      *store.AgentStore
}

// NewAdminRouter returns a new AdminRouter.
func NewAdminRouter(_db *persistence.DB, agentStore *store.AgentStore) *AdminRouter {
	r := &AdminRouter{
		db:         _db,
		userRouter: http.NewServeMux(),
		store:      agentStore,
	}
	r.userRouter.HandleFunc("GET /config/", r.GetConfig)
	r.userRouter.HandleFunc("GET /dump/", r.DumpAll)
	r.userRouter.HandleFunc("GET /state/", r.State)
	r.userRouter.HandleFunc("GET /dump/{id}", r.Dump)
	r.userRouter.HandleFunc("POST /loglevel/{level}", r.UpdateLogLevel)

	return r
}

// GetRouter returns the router, with the middleware configured
// - Authentication middleware
// - IP allowlisting middleware.
func (ur *AdminRouter) GetRouter() *negroni.Negroni {
	n := negroni.New()
	n.Use(middleware.AuthMiddleware(config.Get().AdminToken))
	n.Use(middleware.WhitelistMiddleware(config.Get().AllowedIPs))
	n.UseHandler(ur.userRouter)

	return n
}

// Version returns the server version (Version, Commit and Commit date).
func (ur *AdminRouter) Version(w http.ResponseWriter, r *http.Request) {
	res, err := json.Marshal(common.JSONVersion())
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("error generating response json")
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
	_, err = w.Write(res)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("write response failed")
	}
}

func addAgentInfoToDump(dump *types.State, agent *persistence.Agent) {
	dump.Path = agent.Path
	dump.Name = agent.Name
	dump.ID = agent.ID
	dump.LastUpdated = agent.LastUpdated
	dump.LastPing = agent.LastPing
	dump.Platform = agent.Platform
	dump.SSHMode = agent.SSHMode
	dump.Architecture = agent.Architecture
	dump.RemoteAddr = agent.RemoteAddr
	dump.Hostname = agent.Hostname
	dump.Username = agent.Username
	dump.UsedPorts = agent.UsedPorts
	dump.IPs = agent.IPs
}

// Dump return all the information stored regarding the agent.
func (ur *AdminRouter) Dump(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	dump := ur.store.GetState(id)
	agent, err := ur.db.FindAgentByID(dump.ID)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("dump fail")
		http.NotFound(w, r)

		return
	}
	addAgentInfoToDump(&dump, agent)
	res, err := json.Marshal(dump)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("error generating response json")
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
	_, err = w.Write(res)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("write response failed")
	}
}

// dumpAllAgents return all the information stored regarding all the agents.
func (ur *AdminRouter) dumpAllAgents() []types.State {
	dump := ur.store.GetAllStates()
	var outDump []types.State
	for i := range dump {
		agent, err := ur.db.FindAgentByID(dump[i].ID)
		if err != nil {
			// outDump = append(outDump, d)
			continue
		}
		addAgentInfoToDump(&dump[i], agent)
		outDump = append(outDump, dump[i])
	}

	return outDump
}

// DumpAll return all the information stored regarding all the agents.
func (ur *AdminRouter) DumpAll(w http.ResponseWriter, r *http.Request) {
	outDump := ur.dumpAllAgents()
	res, err := json.Marshal(outDump)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("error generating response json")
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
	_, err = w.Write(res)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("write response failed")
	}
}

// UpdateLogLevel updates the server log level.
func (ur *AdminRouter) UpdateLogLevel(w http.ResponseWriter, r *http.Request) {
	l := r.PathValue("level")
	level, err := zerolog.ParseLevel(l)
	res := types.HTTPResponse{}
	if err != nil {
		res = types.HTTPResponse{
			Message: err.Error(),
			Success: true,
		}
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("unable to update log level")
	} else {
		log.UpdateLogLevel(level)
		res = types.HTTPResponse{
			Message: level.String(),
			Success: true,
		}
		log.Info().Str("Path", r.URL.Path).Str("Level", level.String()).Msg("update log level")
	}
	_, err = w.Write(res.Bytes())
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("write response failed")
	}
}

// GetConfig return the currently running configuration
// It returns a sanitized configuration to remove sensitive information (secret keys, etc.)
func (ur *AdminRouter) GetConfig(w http.ResponseWriter, r *http.Request) {
	c, err := config.Get().GenerateSafeYAMLConfig()

	res := types.HTTPResponse{}
	if err != nil {
		res = types.HTTPResponse{
			Message: err.Error(),
			Success: true,
		}
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("error generating yaml config")
	} else {
		res = types.HTTPResponse{
			Success: true,
			Message: c,
		}
	}

	_, err = w.Write(res.Bytes())
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("write response failed")
	}
}

// State return all the server information (configuration, agent connected or not).
func (ur *AdminRouter) State(w http.ResponseWriter, r *http.Request) {
	agents, _ := ur.db.GetAllAgentsSanitized()

	var dbAgents []types.DbAgent
	for _, a := range agents {
		agent := types.DbAgent{
			CreatedAt: a.CreatedAt,
			UpdatedAt: a.UpdatedAt,
			DeletedAt: a.DeletedAt,
			SocketID:  a.SocketID,
			Agent:     a.Agent,
		}
		dbAgents = append(dbAgents, agent)
	}
	c := config.Get().CopySanitized()

	activeAgents := ur.dumpAllAgents()
	fullState := types.Status{
		Version:      common.GetVersion(),
		ActiveAgents: activeAgents,
		AllAgents:    dbAgents,
		Config:       c,
	}
	res, err := json.Marshal(fullState)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("error generating response json")
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
	_, err = w.Write(res)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("write response failed")
	}
}
