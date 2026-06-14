package router

import (
	"net/http"

	"Goauld/server/config"
	"Goauld/server/router/middleware"

	"github.com/urfave/negroni"
)

// StaticRouter is the router used to serve the agent binaries.
type StaticRouter struct {
	staticRouter *http.ServeMux
}

// NewStaticRouter returns a new StaticRouter.
func NewStaticRouter() *StaticRouter {
	r := &StaticRouter{
		staticRouter: http.NewServeMux(),
	}
	if config.Get().BinariesPathLocation != "" {
		r.staticRouter.Handle("/", http.FileServer(http.Dir(config.Get().BinariesPathLocation)))
	}

	return r
}

// GetRouter returns the router responsible for serving the agents.
func (sr *StaticRouter) GetRouter() *negroni.Negroni {
	n := negroni.New()
	u, p := config.Get().GetBinariesBasicAuth()
	if u == "" && p == "" {
		n.Use(middleware.BasicAuthMiddleware(config.Get().GetBinariesBasicAuth()))
	}
	n.UseHandler(sr.staticRouter)

	return n
}
