package midleware

import (
	"net/http"

	"github.com/urfave/negroni"
)

// AuthMiddleware checks if the Authorization header matches a given token.
func AuthMiddleware(expectedAuthToken string) negroni.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		// Extract the Authorization header
		authHeader := r.Header.Get("Authorization")
		// Validate the Authorization header
		if authHeader != expectedAuthToken {
			// If the token is not correct, return 403 Forbidden
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// If the token is valid, continue to the next middleware/handler
		next(w, r)
	}
}
