package router

import (
	"net/http"

	"Goauld/server/config"
	"Goauld/server/router/midleware"
	"github.com/urfave/negroni"
)

// StaticRouter is the router used to serve the agent binaries
type StaticRouter struct {
	staticRouter *http.ServeMux
}

// NewStaticRouter returns a new StaticRouter
func NewStaticRouter() *StaticRouter {
	r := &StaticRouter{
		staticRouter: http.NewServeMux(),
	}
	r.staticRouter.Handle("/", http.FileServer(http.Dir(config.Get().BinariesPathLocation)))
	return r
}

func (sr *StaticRouter) GetRouter() *negroni.Negroni {
	n := negroni.New()
	n.Use(midleware.BasicAuthMiddleware(config.Get().GetBinariesBasicAuth()))
	n.UseHandler(sr.staticRouter)
	return n
}
