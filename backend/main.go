// ABOUTME: Entry point for Diego Capacity Analyzer backend service
// ABOUTME: Provides HTTP API for CF app and BOSH Diego cell metrics

package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	port := "8080"

	http.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	log.Printf("Starting capacity analyzer backend on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
