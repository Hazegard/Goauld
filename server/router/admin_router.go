package router

import (
	"Goauld/common/log"
	"Goauld/common/types"
	"Goauld/server/config"
	"Goauld/server/persistence"
	"Goauld/server/router/midleware"
	"Goauld/server/store"
	"encoding/json"
	"github.com/rs/zerolog"
	"github.com/urfave/negroni"
	"net/http"
)

// AdminRouter is the router used by the management API
type AdminRouter struct {
	userRouter *http.ServeMux
	db         *persistence.DB
	store      *store.AgentStore
}

// NewAdminRouter returns a new AdminRouter
func NewAdminRouter(_db *persistence.DB, store *store.AgentStore) *AdminRouter {
	r := &AdminRouter{
		db:         _db,
		userRouter: http.NewServeMux(),
		store:      store,
	}
	r.userRouter.HandleFunc("GET /dump/", r.DumpAll)
	r.userRouter.HandleFunc("GET /dump/{id}", r.Dump)
	r.userRouter.HandleFunc("POST /loglevel/{level}", r.UpdateLogLevel)
	return r
}

// GetRouter returns the router, with the middleware configures
// - Authentication middleware
// - IP whitelisting middleware
func (ur *AdminRouter) GetRouter() *negroni.Negroni {
	n := negroni.New()
	n.Use(midleware.AuthMiddleware(config.Get().AdminToken))
	n.Use(midleware.WhitelistMiddleware(config.Get().AllowedIPs))
	n.UseHandler(ur.userRouter)
	return n
}

func (ur *AdminRouter) Dump(w http.ResponseWriter, r *http.Request) {
	id := r.PostFormValue("id")
	dump := ur.store.GetState(id)
	agent, err := ur.db.FindAgentById(dump.Id)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("dump fail")
		http.NotFound(w, r)
		return
	}
	dump.Path = agent.Path
	dump.Name = agent.Name
	dump.Id = agent.Id
	dump.LastUpdated = agent.LastUpdated
	dump.Platform = agent.Platform
	dump.SSHMode = agent.SshMode
	dump.Architecture = agent.Architecture
	dump.Hostname = agent.Hostname
	dump.Username = agent.Username
	dump.UsedPorts = agent.UsedPorts
	dump.IPs = agent.IPs
	res, err := json.Marshal(dump)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("error generating response json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	_, err = w.Write(res)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("write response failed")
	}
}

func (ur *AdminRouter) DumpAll(w http.ResponseWriter, r *http.Request) {
	dump := ur.store.GetAllStates()
	outDump := []types.State{}
	for _, d := range dump {
		agent, err := ur.db.FindAgentById(d.Id)
		if err != nil {
			outDump = append(outDump, d)
			continue
		}
		d.Path = agent.Path
		d.Name = agent.Name
		d.Id = agent.Id
		d.LastUpdated = agent.LastUpdated
		d.Platform = agent.Platform
		d.SSHMode = agent.SshMode
		d.Architecture = agent.Architecture
		d.Hostname = agent.Hostname
		d.Username = agent.Username
		d.UsedPorts = agent.UsedPorts
		d.IPs = agent.IPs
		outDump = append(outDump, d)
	}
	res, err := json.Marshal(outDump)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("error generating response json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	_, err = w.Write(res)
	if err != nil {
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("write response failed")
	}
}

func (ur *AdminRouter) UpdateLogLevel(w http.ResponseWriter, r *http.Request) {
	l := r.PostFormValue("level")
	level, err := zerolog.ParseLevel(l)
	res := types.HttpResponse{}
	if err != nil {
		res = types.HttpResponse{
			Message: err.Error(),
			Success: true,
		}
		log.Warn().Err(err).Str("Path", r.URL.Path).Msg("unable to update log level")
	} else {
		log.UpdateLogLevel(level)
		res = types.HttpResponse{
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
