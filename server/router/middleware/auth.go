// Package middleware handle the custom middleware used by negroni
package middleware

import (
	"Goauld/common/log"
	"Goauld/common/net"
	"Goauld/common/utils"
	"net/http"
	"strings"

	"github.com/urfave/negroni"
)

// AuthMiddleware checks if the Authorization header matches a given token.
func AuthMiddleware(expectedAuthToken []string) negroni.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		// Extract the Authorization header
		authHeader := r.Header.Get("Authorization")
		// Validate the Authorization header
		authHeader = strings.Split(authHeader, ":")[0]

		if !utils.Contains(expectedAuthToken, authHeader) {
			// If the token is not correct, return 403 Forbidden
			http.Error(w, net.Unauthorized, http.StatusUnauthorized)
			log.Get().Trace().Str("Header", authHeader).Msg("Header not allowed")

			return
		}

		// If the token is valid, continue to the next middleware/handler
		next(w, r)
	}
}

// BasicAuthMiddleware validates the basic authentication header.
func BasicAuthMiddleware(username string, password string) negroni.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if username == "" && password == "" {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			w.WriteHeader(http.StatusUnauthorized)
			http.Error(w, net.Unauthorized, http.StatusUnauthorized)

			return
		}
		u, p, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			w.WriteHeader(http.StatusUnauthorized)
			http.Error(w, net.Unauthorized, http.StatusUnauthorized)

			return
		}
		if u != username || p != password {
			http.Error(w, net.Unauthorized, http.StatusUnauthorized)

			return
		}

		// If authentication passes, call the next handler
		next(w, r)
	}
}
