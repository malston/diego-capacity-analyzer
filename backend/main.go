// ABOUTME: Entry point for Diego Capacity Analyzer backend service
// ABOUTME: Provides HTTP API for CF app and BOSH Diego cell metrics

package main

import (
	"log"
	"net/http"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/cache"
	"github.com/markalston/diego-capacity-analyzer/backend/config"
	"github.com/markalston/diego-capacity-analyzer/backend/handlers"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting Diego Capacity Analyzer Backend")
	log.Printf("CF API: %s", cfg.CFAPIUrl)
	if cfg.BOSHEnvironment != "" {
		log.Printf("BOSH: %s", cfg.BOSHEnvironment)
	} else {
		log.Printf("BOSH: not configured (degraded mode)")
	}

	// Initialize cache
	cacheTTL := time.Duration(cfg.CacheTTL) * time.Second
	c := cache.New(cacheTTL)
	log.Printf("Cache TTL: %v", cacheTTL)

	// Initialize handlers
	h := handlers.NewHandler(cfg, c)

	// Register routes
	http.HandleFunc("/api/health", h.EnableCORS(h.Health))
	http.HandleFunc("/api/dashboard", h.EnableCORS(h.Dashboard))
	http.HandleFunc("/api/infrastructure/manual", h.EnableCORS(h.HandleManualInfrastructure))
	http.HandleFunc("/api/scenario/compare", h.EnableCORS(h.HandleScenarioCompare))

	// Start server
	addr := ":" + cfg.Port
	log.Printf("Server listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
