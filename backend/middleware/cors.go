// ABOUTME: CORS middleware for API cross-origin requests
// ABOUTME: Handles preflight OPTIONS and adds required headers

package middleware

import "net/http"

// CORS returns middleware that adds CORS headers to responses.
// It handles OPTIONS preflight requests by returning 204 No Content
// without calling the wrapped handler.
//
// Deprecated: Use CORSWithConfig for production deployments. This function
// allows all origins (*) which is insecure for APIs with authentication.
func CORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

// CORSWithConfig returns a middleware factory that validates Origin headers
// against a whitelist of allowed origins. Only requests from allowed origins
// receive CORS headers; others are processed without CORS headers (browser
// will block cross-origin access).
func CORSWithConfig(allowedOrigins []string) func(http.HandlerFunc) http.HandlerFunc {
	// Build a set for O(1) lookup
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = true
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Only add CORS headers if origin is in whitelist
			if origin != "" && allowed[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Vary", "Origin")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next(w, r)
		}
	}
}
