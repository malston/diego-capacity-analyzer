// ABOUTME: JSON error response helper for middleware
// ABOUTME: Ensures middleware error responses match the API's JSON format

package middleware

import (
	"encoding/json"
	"net/http"
)

// writeJSONError writes an error response as JSON with the given status code.
// Matches the format used by handlers.writeError for consistency.
func writeJSONError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
		Code  int    `json:"code"`
	}{
		Error: message,
		Code:  code,
	})
}
