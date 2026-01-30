// ABOUTME: Tests for JWKS parsing functionality
// ABOUTME: Verifies parsing of CF UAA's JWKS JSON into RSA public keys

package services

import (
	"testing"
)

// Test key modulus from RFC 7517 Appendix A.1 (example RSA public key)
const testKeyModulus = "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw"

func TestParseJWKS_ValidResponse(t *testing.T) {
	jwksJSON := `{
		"keys": [
			{
				"kty": "RSA",
				"kid": "test-key-id",
				"n": "` + testKeyModulus + `",
				"e": "AQAB",
				"alg": "RS256",
				"use": "sig"
			}
		]
	}`

	keys, err := parseJWKS([]byte(jwksJSON))
	if err != nil {
		t.Fatalf("parseJWKS returned error: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}

	key, ok := keys["test-key-id"]
	if !ok {
		t.Fatal("expected key with id 'test-key-id' to be present")
	}

	if key == nil {
		t.Fatal("expected key to be non-nil")
	}

	// Verify the exponent is correct (AQAB = 65537)
	if key.E != 65537 {
		t.Errorf("expected exponent 65537, got %d", key.E)
	}
}

func TestParseJWKS_MultipleKeys(t *testing.T) {
	jwksJSON := `{
		"keys": [
			{
				"kty": "RSA",
				"kid": "key-1",
				"n": "` + testKeyModulus + `",
				"e": "AQAB",
				"alg": "RS256",
				"use": "sig"
			},
			{
				"kty": "RSA",
				"kid": "key-2",
				"n": "` + testKeyModulus + `",
				"e": "AQAB",
				"alg": "RS256",
				"use": "sig"
			}
		]
	}`

	keys, err := parseJWKS([]byte(jwksJSON))
	if err != nil {
		t.Fatalf("parseJWKS returned error: %v", err)
	}

	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}

	if _, ok := keys["key-1"]; !ok {
		t.Error("expected key with id 'key-1' to be present")
	}

	if _, ok := keys["key-2"]; !ok {
		t.Error("expected key with id 'key-2' to be present")
	}
}

func TestParseJWKS_SkipsNonRSAKeys(t *testing.T) {
	jwksJSON := `{
		"keys": [
			{
				"kty": "RSA",
				"kid": "rsa-key",
				"n": "` + testKeyModulus + `",
				"e": "AQAB",
				"alg": "RS256",
				"use": "sig"
			},
			{
				"kty": "EC",
				"kid": "ec-key",
				"crv": "P-256",
				"x": "WbbxfFsQAIHdkp3zT-v-RhXfgG7W5XluomJVxJnJNNw",
				"y": "LGgr4sJEBB2YzJ95kmrCxiQ-1h2e3RWw8hnckP8MhEY",
				"alg": "ES256",
				"use": "sig"
			},
			{
				"kty": "oct",
				"kid": "symmetric-key",
				"k": "GawgguFyGrWKav7AX4VKUg",
				"alg": "HS256"
			}
		]
	}`

	keys, err := parseJWKS([]byte(jwksJSON))
	if err != nil {
		t.Fatalf("parseJWKS returned error: %v", err)
	}

	// Should only have the RSA key
	if len(keys) != 1 {
		t.Fatalf("expected 1 key (RSA only), got %d", len(keys))
	}

	if _, ok := keys["rsa-key"]; !ok {
		t.Error("expected RSA key with id 'rsa-key' to be present")
	}

	if _, ok := keys["ec-key"]; ok {
		t.Error("EC key should have been skipped")
	}

	if _, ok := keys["symmetric-key"]; ok {
		t.Error("symmetric key should have been skipped")
	}
}

func TestParseJWKS_InvalidJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"not json", "not json at all"},
		{"incomplete json", `{"keys": [`},
		{"wrong type", `{"keys": "not an array"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseJWKS([]byte(tt.input))
			if err == nil {
				t.Error("expected error for invalid JSON, got nil")
			}
		})
	}
}

func TestParseJWKS_NullKeys(t *testing.T) {
	// null keys is valid JSON and should return empty map (no keys)
	jwksJSON := `{"keys": null}`

	keys, err := parseJWKS([]byte(jwksJSON))
	if err != nil {
		t.Fatalf("parseJWKS returned error: %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("expected 0 keys for null keys, got %d", len(keys))
	}
}

func TestParseJWKS_InvalidBase64(t *testing.T) {
	tests := []struct {
		name string
		n    string
		e    string
	}{
		{"invalid modulus", "!!!invalid!!!", "AQAB"},
		{"invalid exponent", testKeyModulus, "!!!invalid!!!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jwksJSON := `{
				"keys": [
					{
						"kty": "RSA",
						"kid": "bad-key",
						"n": "` + tt.n + `",
						"e": "` + tt.e + `",
						"alg": "RS256"
					}
				]
			}`

			_, err := parseJWKS([]byte(jwksJSON))
			if err == nil {
				t.Error("expected error for invalid base64, got nil")
			}
		})
	}
}

func TestParseJWKS_EmptyKeys(t *testing.T) {
	jwksJSON := `{"keys": []}`

	keys, err := parseJWKS([]byte(jwksJSON))
	if err != nil {
		t.Fatalf("parseJWKS returned error: %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("expected 0 keys, got %d", len(keys))
	}
}
