// ABOUTME: Integration tests for JWT authentication flow with mock UAA server
// ABOUTME: Verifies Bearer token validation via JWKS and session cookie fallback

package e2e

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/middleware"
	"github.com/markalston/diego-capacity-analyzer/backend/services"
)

// jwtHeader represents the header portion of a JWT for test token creation
type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	Kid string `json:"kid,omitempty"`
}

// jwtClaims represents the payload/claims portion of a JWT for test token creation
type jwtClaims struct {
	Sub      string   `json:"sub,omitempty"`
	UserName string   `json:"user_name,omitempty"`
	UserID   string   `json:"user_id,omitempty"`
	ClientID string   `json:"client_id,omitempty"`
	Exp      int64    `json:"exp,omitempty"`
	Iat      int64    `json:"iat,omitempty"`
	Scope    []string `json:"scope,omitempty"`
}

// createTestJWT creates a signed JWT for testing with the given private key
func createTestJWT(t *testing.T, privateKey *rsa.PrivateKey, kid string, claims jwtClaims) string {
	t.Helper()

	header := jwtHeader{
		Alg: "RS256",
		Typ: "JWT",
		Kid: kid,
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("failed to marshal header: %v", err)
	}
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	signingInput := headerB64 + "." + payloadB64

	hash := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

	return signingInput + "." + signatureB64
}

// createMockUAAServer creates a mock UAA server that serves JWKS at /token_keys
func createMockUAAServer(t *testing.T, publicKey *rsa.PublicKey, keyID string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token_keys" {
			http.NotFound(w, r)
			return
		}

		// Convert public key to JWK format
		nBytes := publicKey.N.Bytes()
		nB64 := base64.RawURLEncoding.EncodeToString(nBytes)

		eBytes := make([]byte, 4)
		eBytes[0] = byte(publicKey.E >> 24)
		eBytes[1] = byte(publicKey.E >> 16)
		eBytes[2] = byte(publicKey.E >> 8)
		eBytes[3] = byte(publicKey.E)
		// Trim leading zeros
		for len(eBytes) > 1 && eBytes[0] == 0 {
			eBytes = eBytes[1:]
		}
		eB64 := base64.RawURLEncoding.EncodeToString(eBytes)

		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
				{
					"kty": "RSA",
					"kid": keyID,
					"n":   nB64,
					"e":   eB64,
					"alg": "RS256",
					"use": "sig",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(jwks); err != nil {
			t.Errorf("failed to encode JWKS: %v", err)
		}
	}))
}

// TestAuthIntegration_BearerTokenWithJWKS tests the full JWT authentication flow
// with a mock UAA server that serves JWKS for signature verification.
func TestAuthIntegration_BearerTokenWithJWKS(t *testing.T) {
	// Generate RSA key pair for signing test tokens
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Generate a different key pair for testing invalid signatures
	wrongPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate wrong RSA key: %v", err)
	}

	// Create mock UAA server
	keyID := "test-key-id"
	uaaServer := createMockUAAServer(t, &privateKey.PublicKey, keyID)
	defer uaaServer.Close()

	// Create JWKS client pointing to mock UAA
	jwksClient, err := services.NewJWKSClient(uaaServer.URL, nil)
	if err != nil {
		t.Fatalf("failed to create JWKS client: %v", err)
	}

	tests := []struct {
		name           string
		token          func() string
		expectedStatus int
		expectedUser   string
	}{
		{
			name: "valid user token",
			token: func() string {
				return createTestJWT(t, privateKey, keyID, jwtClaims{
					UserName: "test-user",
					UserID:   "user-123",
					Exp:      time.Now().Add(time.Hour).Unix(),
					Iat:      time.Now().Unix(),
				})
			},
			expectedStatus: http.StatusOK,
			expectedUser:   "test-user",
		},
		{
			name: "valid client credentials token",
			token: func() string {
				return createTestJWT(t, privateKey, keyID, jwtClaims{
					ClientID: "my-service-client",
					Sub:      "my-service-client",
					Exp:      time.Now().Add(time.Hour).Unix(),
					Iat:      time.Now().Unix(),
				})
			},
			expectedStatus: http.StatusOK,
			expectedUser:   "my-service-client",
		},
		{
			name: "expired token",
			token: func() string {
				return createTestJWT(t, privateKey, keyID, jwtClaims{
					UserName: "expired-user",
					UserID:   "user-456",
					Exp:      time.Now().Add(-time.Hour).Unix(), // Expired
					Iat:      time.Now().Add(-2 * time.Hour).Unix(),
				})
			},
			expectedStatus: http.StatusUnauthorized,
			expectedUser:   "",
		},
		{
			name: "wrong key ID",
			token: func() string {
				return createTestJWT(t, privateKey, "unknown-key-id", jwtClaims{
					UserName: "unknown-key-user",
					UserID:   "user-789",
					Exp:      time.Now().Add(time.Hour).Unix(),
					Iat:      time.Now().Unix(),
				})
			},
			expectedStatus: http.StatusUnauthorized,
			expectedUser:   "",
		},
		{
			name: "invalid signature (signed with different key)",
			token: func() string {
				// Sign with a different key, but use the correct key ID
				return createTestJWT(t, wrongPrivateKey, keyID, jwtClaims{
					UserName: "wrong-sig-user",
					UserID:   "user-bad",
					Exp:      time.Now().Add(time.Hour).Unix(),
					Iat:      time.Now().Unix(),
				})
			},
			expectedStatus: http.StatusUnauthorized,
			expectedUser:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create auth config with JWKS client
			cfg := middleware.AuthConfig{
				Mode:       middleware.AuthModeRequired,
				JWKSClient: jwksClient,
			}

			// Track extracted claims
			var extractedClaims *middleware.UserClaims
			handler := middleware.Auth(cfg)(func(w http.ResponseWriter, r *http.Request) {
				extractedClaims = middleware.GetUserClaims(r)
				w.WriteHeader(http.StatusOK)
			})

			// Create request with Bearer token
			req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
			req.Header.Set("Authorization", "Bearer "+tt.token())
			rec := httptest.NewRecorder()

			// Execute
			handler(rec, req)

			// Verify status
			if rec.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d. Body: %s", rec.Code, tt.expectedStatus, rec.Body.String())
			}

			// Verify claims if expected
			if tt.expectedUser != "" {
				if extractedClaims == nil {
					t.Fatal("expected claims to be extracted")
				}
				if extractedClaims.Username != tt.expectedUser {
					t.Errorf("username = %q, want %q", extractedClaims.Username, tt.expectedUser)
				}
			} else if tt.expectedStatus == http.StatusUnauthorized {
				// Handler should not have been called
				if extractedClaims != nil {
					t.Errorf("expected no claims for rejected request, got %+v", extractedClaims)
				}
			}
		})
	}
}

// TestAuthIntegration_SessionCookieFallback verifies that session cookies work
// when JWKSClient is nil (graceful degradation for backward compatibility).
func TestAuthIntegration_SessionCookieFallback(t *testing.T) {
	// Session validator that accepts specific session IDs
	validSessions := map[string]*middleware.UserClaims{
		"valid-session-abc": {Username: "session-user-1", UserID: "session-id-1"},
		"valid-session-xyz": {Username: "session-user-2", UserID: "session-id-2"},
	}

	sessionValidator := func(sessionID string) *middleware.UserClaims {
		return validSessions[sessionID]
	}

	tests := []struct {
		name           string
		sessionID      string
		jwksClient     *services.JWKSClient
		expectedStatus int
		expectedUser   string
	}{
		{
			name:           "valid session cookie with nil JWKSClient",
			sessionID:      "valid-session-abc",
			jwksClient:     nil, // No JWKS client
			expectedStatus: http.StatusOK,
			expectedUser:   "session-user-1",
		},
		{
			name:           "another valid session cookie",
			sessionID:      "valid-session-xyz",
			jwksClient:     nil,
			expectedStatus: http.StatusOK,
			expectedUser:   "session-user-2",
		},
		{
			name:           "invalid session cookie",
			sessionID:      "invalid-session-id",
			jwksClient:     nil,
			expectedStatus: http.StatusUnauthorized,
			expectedUser:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := middleware.AuthConfig{
				Mode:             middleware.AuthModeRequired,
				SessionValidator: sessionValidator,
				JWKSClient:       tt.jwksClient,
			}

			var extractedClaims *middleware.UserClaims
			handler := middleware.Auth(cfg)(func(w http.ResponseWriter, r *http.Request) {
				extractedClaims = middleware.GetUserClaims(r)
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
			req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: tt.sessionID})
			rec := httptest.NewRecorder()

			handler(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.expectedStatus)
			}

			if tt.expectedUser != "" {
				if extractedClaims == nil {
					t.Fatal("expected claims to be extracted from session")
				}
				if extractedClaims.Username != tt.expectedUser {
					t.Errorf("username = %q, want %q", extractedClaims.Username, tt.expectedUser)
				}
			}
		})
	}
}

// TestAuthIntegration_BearerTakesPrecedenceOverSession verifies that when both
// Bearer token and session cookie are provided, Bearer token takes precedence.
func TestAuthIntegration_BearerTakesPrecedenceOverSession(t *testing.T) {
	// Generate key pair for JWT
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Create mock UAA server
	keyID := "test-key"
	uaaServer := createMockUAAServer(t, &privateKey.PublicKey, keyID)
	defer uaaServer.Close()

	// Create JWKS client
	jwksClient, err := services.NewJWKSClient(uaaServer.URL, nil)
	if err != nil {
		t.Fatalf("failed to create JWKS client: %v", err)
	}

	// Session validator
	sessionValidator := func(sessionID string) *middleware.UserClaims {
		if sessionID == "valid-session" {
			return &middleware.UserClaims{Username: "session-user", UserID: "session-id"}
		}
		return nil
	}

	cfg := middleware.AuthConfig{
		Mode:             middleware.AuthModeRequired,
		SessionValidator: sessionValidator,
		JWKSClient:       jwksClient,
	}

	var extractedClaims *middleware.UserClaims
	handler := middleware.Auth(cfg)(func(w http.ResponseWriter, r *http.Request) {
		extractedClaims = middleware.GetUserClaims(r)
		w.WriteHeader(http.StatusOK)
	})

	// Create request with both Bearer token and session cookie
	token := createTestJWT(t, privateKey, keyID, jwtClaims{
		UserName: "bearer-user",
		UserID:   "bearer-id",
		Exp:      time.Now().Add(time.Hour).Unix(),
		Iat:      time.Now().Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "valid-session"})
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Bearer token should take precedence
	if extractedClaims == nil {
		t.Fatal("expected claims to be extracted")
	}
	if extractedClaims.Username != "bearer-user" {
		t.Errorf("username = %q, want %q (Bearer should take precedence over session)", extractedClaims.Username, "bearer-user")
	}
}

// TestAuthIntegration_JWKSKeyRefresh verifies that the JWKS client can handle
// key rotation by refreshing keys when an unknown key ID is encountered.
func TestAuthIntegration_JWKSKeyRefresh(t *testing.T) {
	// Generate key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Track current key ID served by mock UAA
	currentKeyID := "old-key-id"

	// Create mock UAA server that can rotate keys
	uaaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token_keys" {
			http.NotFound(w, r)
			return
		}

		nBytes := privateKey.PublicKey.N.Bytes()
		nB64 := base64.RawURLEncoding.EncodeToString(nBytes)

		eBytes := make([]byte, 4)
		eBytes[0] = byte(privateKey.PublicKey.E >> 24)
		eBytes[1] = byte(privateKey.PublicKey.E >> 16)
		eBytes[2] = byte(privateKey.PublicKey.E >> 8)
		eBytes[3] = byte(privateKey.PublicKey.E)
		for len(eBytes) > 1 && eBytes[0] == 0 {
			eBytes = eBytes[1:]
		}
		eB64 := base64.RawURLEncoding.EncodeToString(eBytes)

		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
				{
					"kty": "RSA",
					"kid": currentKeyID, // Serve the current key ID
					"n":   nB64,
					"e":   eB64,
					"alg": "RS256",
					"use": "sig",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))
	defer uaaServer.Close()

	// Create JWKS client - will fetch "old-key-id" initially
	jwksClient, err := services.NewJWKSClient(uaaServer.URL, nil)
	if err != nil {
		t.Fatalf("failed to create JWKS client: %v", err)
	}

	cfg := middleware.AuthConfig{
		Mode:       middleware.AuthModeRequired,
		JWKSClient: jwksClient,
	}

	var extractedClaims *middleware.UserClaims
	handler := middleware.Auth(cfg)(func(w http.ResponseWriter, r *http.Request) {
		extractedClaims = middleware.GetUserClaims(r)
		w.WriteHeader(http.StatusOK)
	})

	// Simulate key rotation: UAA now serves "new-key-id"
	currentKeyID = "new-key-id"

	// Create token signed with "new-key-id" (unknown to client at this point)
	token := createTestJWT(t, privateKey, "new-key-id", jwtClaims{
		UserName: "rotated-key-user",
		UserID:   "rotated-id",
		Exp:      time.Now().Add(time.Hour).Unix(),
		Iat:      time.Now().Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler(rec, req)

	// Should succeed after JWKS refresh fetches the new key
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d. Body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	if extractedClaims == nil {
		t.Fatal("expected claims after key refresh")
	}
	if extractedClaims.Username != "rotated-key-user" {
		t.Errorf("username = %q, want %q", extractedClaims.Username, "rotated-key-user")
	}
}
