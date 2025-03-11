package midleware

import (
	"Goauld/common/log"
	"net/http"
	"strings"

	"Goauld/common/net"
	"github.com/urfave/negroni"
)

// WhitelistMiddleware checks if the request IP is in the allowed list.
func WhitelistMiddleware(allowedIPs []string) negroni.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		// Get the IP address from the request
		clientIP := getClientIP(r)

		// Check if the IP is in the whitelist
		if !net.IsIPAllowed(clientIP, allowedIPs) {
			// If not allowed, return 403 Forbidden
			log.Get().Trace().Str("ClientIP", clientIP).Msg("ClientIP not allowed")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Continue to the next middleware/handler
		next(w, r)
	}
}

// Helper function to retrieve the client IP address
func getClientIP(r *http.Request) string {
	return strings.Split(r.RemoteAddr, ":")[0]
}
