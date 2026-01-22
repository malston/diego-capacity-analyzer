// ABOUTME: Handler for serving OpenAPI specification
// ABOUTME: Embeds openapi.yaml at compile time for Swagger UI consumption

package handlers

import (
	_ "embed"
	"log/slog"
	"net/http"
)

//go:embed openapi.yaml
var openapiSpec []byte

// OpenAPISpec serves the embedded OpenAPI specification.
func (h *Handler) OpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	if _, err := w.Write(openapiSpec); err != nil {
		slog.Error("Failed to write OpenAPI spec response", "error", err)
	}
}
