// ABOUTME: Integration tests for TLS security configuration
// ABOUTME: Verifies SSL validation settings and CA certificate handling

package e2e

import (
	"strings"
	"testing"

	"github.com/markalston/diego-capacity-analyzer/backend/config"
	"github.com/markalston/diego-capacity-analyzer/backend/services"
)

// TestTLS_DefaultSecureConfig verifies that with only required CF env vars set,
// TLS validation defaults to secure (skip=false).
func TestTLS_DefaultSecureConfig(t *testing.T) {
	t.Cleanup(withTestCFEnvAndExtra(t, map[string]string{
		"CF_SKIP_SSL_VALIDATION":   "",
		"BOSH_SKIP_SSL_VALIDATION": "",
	}))

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify secure defaults
	if cfg.CFSkipSSLValidation {
		t.Error("CFSkipSSLValidation should default to false (secure)")
	}
	if cfg.BOSHSkipSSLValidation {
		t.Error("BOSHSkipSSLValidation should default to false (secure)")
	}
}

// TestTLS_BOSHSkipSSLValidationTrue verifies that BOSH_SKIP_SSL_VALIDATION=true
// allows the BOSH client to be created without error. The actual TLS behavior
// is tested in services/boshapi_test.go with a mock TLS server.
func TestTLS_BOSHSkipSSLValidationTrue(t *testing.T) {
	// With skipSSLValidation=true, client creation should succeed
	client, err := services.NewBOSHClient("https://bosh.example.com:25555", "test-client", "test-secret", "", "cf-test", true)
	if err != nil {
		t.Fatalf("NewBOSHClient with skipSSLValidation=true should not fail: %v", err)
	}
	if client == nil {
		t.Error("Client should not be nil with skipSSLValidation=true")
	}
}

// TestTLS_BOSHSkipSSLValidationFalse verifies that BOSH_SKIP_SSL_VALIDATION=false
// (default, secure mode) allows client creation - TLS validation happens at runtime.
func TestTLS_BOSHSkipSSLValidationFalse(t *testing.T) {
	// With skipSSLValidation=false and no CA cert, client should still be created
	// (TLS validation uses system CA pool at connection time)
	client, err := services.NewBOSHClient("https://bosh.example.com:25555", "test-client", "test-secret", "", "cf-test", false)
	if err != nil {
		t.Fatalf("NewBOSHClient with skipSSLValidation=false should not fail at creation: %v", err)
	}
	if client == nil {
		t.Error("Client should not be nil with skipSSLValidation=false")
	}
}

// TestTLS_BOSHCACertMalformed verifies that malformed CA certificates are properly
// rejected when BOSH_SKIP_SSL_VALIDATION=false.
func TestTLS_BOSHCACertMalformed(t *testing.T) {
	malformedCert := "not-a-valid-certificate"

	// With skipSSLValidation=false and malformed cert, should fail
	_, err := services.NewBOSHClient("https://bosh.example.com", "test-client", "test-secret", malformedCert, "cf-test", false)
	if err == nil {
		t.Fatal("NewBOSHClient should fail with malformed CA cert when skipSSLValidation=false")
	}

	// Error message should indicate the CA cert issue
	if !strings.Contains(err.Error(), "BOSH_CA_CERT") {
		t.Errorf("Error should mention BOSH_CA_CERT, got: %v", err)
	}
}

// TestTLS_BOSHCACertMalformedWithSkipFallback verifies that malformed CA certs
// fall back to insecure mode when BOSH_SKIP_SSL_VALIDATION=true.
func TestTLS_BOSHCACertMalformedWithSkipFallback(t *testing.T) {
	malformedCert := "not-a-valid-certificate"

	// With skipSSLValidation=true, should fall back to insecure mode
	client, err := services.NewBOSHClient("https://bosh.example.com", "test-client", "test-secret", malformedCert, "cf-test", true)
	if err != nil {
		t.Errorf("NewBOSHClient should fall back to insecure mode with malformed CA cert when skipSSLValidation=true: %v", err)
	}
	if client == nil {
		t.Error("Client should not be nil when falling back to insecure mode")
	}
}

// TestTLS_CFSkipSSLValidation_EnvParsing verifies CF_SKIP_SSL_VALIDATION
// environment variable is correctly parsed.
func TestTLS_CFSkipSSLValidation_EnvParsing(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{
			name:     "true string",
			envValue: "true",
			expected: true,
		},
		{
			name:     "false string",
			envValue: "false",
			expected: false,
		},
		{
			name:     "1 is truthy",
			envValue: "1",
			expected: true,
		},
		{
			name:     "0 is falsy",
			envValue: "0",
			expected: false,
		},
		{
			name:     "empty defaults to false",
			envValue: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(withTestCFEnvAndExtra(t, map[string]string{
				"CF_SKIP_SSL_VALIDATION": tt.envValue,
			}))

			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			if cfg.CFSkipSSLValidation != tt.expected {
				t.Errorf("CFSkipSSLValidation = %v, want %v", cfg.CFSkipSSLValidation, tt.expected)
			}
		})
	}
}

// TestTLS_BOSHSkipSSLValidation_EnvParsing verifies BOSH_SKIP_SSL_VALIDATION
// environment variable is correctly parsed.
func TestTLS_BOSHSkipSSLValidation_EnvParsing(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{
			name:     "true enables skip",
			envValue: "true",
			expected: true,
		},
		{
			name:     "false disables skip",
			envValue: "false",
			expected: false,
		},
		{
			name:     "empty defaults to false (secure by default)",
			envValue: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(withTestCFEnvAndExtra(t, map[string]string{
				"BOSH_SKIP_SSL_VALIDATION": tt.envValue,
			}))

			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			if cfg.BOSHSkipSSLValidation != tt.expected {
				t.Errorf("BOSHSkipSSLValidation = %v, want %v", cfg.BOSHSkipSSLValidation, tt.expected)
			}
		})
	}
}

// TestTLS_BOSHCACert_EnvParsing verifies BOSH_CA_CERT environment variable
// is correctly loaded into config.
func TestTLS_BOSHCACert_EnvParsing(t *testing.T) {
	// Multi-line string to verify env var is loaded without truncation
	sampleCert := `-----BEGIN CERTIFICATE-----
MIIBkTCB+wIJAKHBfpj2S5JNMA0GCSqGSIb3DQEBCwUAMBExDzANBgNVBAMMBnRl
c3RjYTAeFw0yNDAxMDEwMDAwMDBaFw0yNTAxMDEwMDAwMDBaMBExDzANBgNVBAMM
BnRlc3RjYTBcMA0GCSqGSIb3DQEBAQUAA0sAMEgCQQC7o96HFLqGzOHHY+QLqJZT
-----END CERTIFICATE-----`

	t.Cleanup(withTestCFEnvAndExtra(t, map[string]string{
		"BOSH_CA_CERT": sampleCert,
	}))

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.BOSHCACert != sampleCert {
		t.Errorf("BOSHCACert not loaded correctly, got length %d, want %d", len(cfg.BOSHCACert), len(sampleCert))
	}
}
