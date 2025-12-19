// ABOUTME: Configuration loader for backend service
// ABOUTME: Loads settings from environment variables with defaults

package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	// Server
	Port     string
	CacheTTL int // seconds

	// CF API
	CFAPIUrl   string
	CFUsername string
	CFPassword string

	// BOSH API (optional)
	BOSHEnvironment string
	BOSHClient      string
	BOSHSecret      string
	BOSHCACert      string
	BOSHDeployment  string

	// CredHub (optional)
	CredHubURL    string
	CredHubClient string
	CredHubSecret string

	// vSphere (optional)
	VSphereHost       string
	VSphereUsername   string
	VSpherePassword   string
	VSphereDatacenter string
	VSphereInsecure   bool
	VSphereCacheTTL   int // seconds, default 300 (5 min)
}

// VSphereConfigured returns true if vSphere credentials are set
func (c *Config) VSphereConfigured() bool {
	return c.VSphereHost != "" && c.VSphereUsername != "" && c.VSpherePassword != "" && c.VSphereDatacenter != ""
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:     getEnv("PORT", "8080"),
		CacheTTL: getEnvInt("CACHE_TTL", 300),

		CFAPIUrl:   os.Getenv("CF_API_URL"),
		CFUsername: os.Getenv("CF_USERNAME"),
		CFPassword: os.Getenv("CF_PASSWORD"),

		BOSHEnvironment: os.Getenv("BOSH_ENVIRONMENT"),
		BOSHClient:      os.Getenv("BOSH_CLIENT"),
		BOSHSecret:      os.Getenv("BOSH_CLIENT_SECRET"),
		BOSHCACert:      os.Getenv("BOSH_CA_CERT"),
		BOSHDeployment:  os.Getenv("BOSH_DEPLOYMENT"),

		CredHubURL:    os.Getenv("CREDHUB_URL"),
		CredHubClient: os.Getenv("CREDHUB_CLIENT"),
		CredHubSecret: os.Getenv("CREDHUB_SECRET"),

		VSphereHost:       os.Getenv("VSPHERE_HOST"),
		VSphereUsername:   os.Getenv("VSPHERE_USERNAME"),
		VSpherePassword:   os.Getenv("VSPHERE_PASSWORD"),
		VSphereDatacenter: os.Getenv("VSPHERE_DATACENTER"),
		VSphereInsecure:   getEnvBool("VSPHERE_INSECURE", true),
		VSphereCacheTTL:   getEnvInt("VSPHERE_CACHE_TTL", 300),
	}

	// Validate required fields
	if cfg.CFAPIUrl == "" {
		return nil, fmt.Errorf("CF_API_URL is required")
	}
	if cfg.CFUsername == "" {
		return nil, fmt.Errorf("CF_USERNAME is required")
	}
	if cfg.CFPassword == "" {
		return nil, fmt.Errorf("CF_PASSWORD is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
