// ABOUTME: CORS middleware for API cross-origin requests
// ABOUTME: Handles preflight OPTIONS and adds required headers

package middleware

import "net/http"

// CORS returns middleware that adds CORS headers to responses.
// It handles OPTIONS preflight requests by returning 200 OK without
// calling the wrapped handler.
func CORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}
