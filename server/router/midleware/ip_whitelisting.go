package midleware

import (
	"github.com/urfave/negroni"
	"net/http"
)

// WhitelistMiddleware checks if the request IP is in the allowed list.
func WhitelistMiddleware(allowedIPs []string) negroni.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		// Get the IP address from the request
		clientIP := getClientIP(r)

		// Check if the IP is in the whitelist
		if !isIPAllowed(clientIP, allowedIPs) {
			// If not allowed, return 403 Forbidden
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Continue to the next middleware/handler
		next(w, r)
	}
}

// Helper function to retrieve the client IP address
func getClientIP(r *http.Request) string {
	return r.RemoteAddr
}

// Check if the IP is in the allowed list
func isIPAllowed(ip string, allowedIPs []string) bool {
	for _, allowedIP := range allowedIPs {
		if ip == allowedIP {
			return true
		}
	}
	return false
}
