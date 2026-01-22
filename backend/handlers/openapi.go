// ABOUTME: Handler for serving OpenAPI specification
// ABOUTME: Embeds openapi.yaml at compile time for Swagger UI consumption

package handlers

import (
	_ "embed"
	"net/http"
)

//go:embed openapi.yaml
var openapiSpec []byte

// OpenAPISpec serves the embedded OpenAPI specification.
func (h *Handler) OpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(openapiSpec)
}
