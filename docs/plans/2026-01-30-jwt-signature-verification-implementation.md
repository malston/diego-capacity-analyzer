# JWT Signature Verification Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add cryptographic JWT signature verification for Bearer tokens using CF UAA's JWKS endpoint.

**Architecture:** New JWKS client fetches and caches UAA public keys. Auth middleware uses JWKS client to verify Bearer token signatures. Session cookie auth remains unchanged.

**Tech Stack:** Go standard library (crypto/rsa, encoding/json, math/big, sync/singleflight), no external JWT libraries.

**Review Feedback Incorporated:**
- Signature verification BEFORE expiration check (prevents timing attacks)
- singleflight.Group for thundering herd prevention on JWKS refresh
- nbf (not before) claim validation for RFC 7519 compliance
- Missing kid header rejection with clear error
- HTTP timeout configuration (30s startup, 10s refresh)
- Improved error message for JWKS unavailable

---

## Task 1: JWKS Response Parsing

**Files:**

- Create: `backend/services/jwks.go`
- Create: `backend/services/jwks_test.go`

### Step 1.1: Write test for parsing JWKS response

```go
// backend/services/jwks_test.go
package services

import (
	"testing"
)

func TestParseJWKS_ValidResponse(t *testing.T) {
	// Sample JWKS response from CF UAA
	jwksJSON := `{
		"keys": [
			{
				"kty": "RSA",
				"kid": "key-1",
				"n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
				"e": "AQAB",
				"alg": "RS256",
				"use": "sig"
			}
		]
	}`

	keys, err := parseJWKS([]byte(jwksJSON))
	if err != nil {
		t.Fatalf("parseJWKS failed: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}

	key, ok := keys["key-1"]
	if !ok {
		t.Fatal("key-1 not found in parsed keys")
	}

	if key == nil {
		t.Fatal("key-1 is nil")
	}

	// Verify it's a valid RSA public key by checking the exponent
	if key.E != 65537 {
		t.Errorf("expected exponent 65537, got %d", key.E)
	}
}
```

### Step 1.2: Run test to verify it fails

Run: `cd backend && go test ./services -run TestParseJWKS_ValidResponse -v`
Expected: FAIL with "undefined: parseJWKS"

### Step 1.3: Write minimal implementation

```go
// backend/services/jwks.go
// ABOUTME: JWKS client for fetching and caching CF UAA public keys
// ABOUTME: Verifies JWT signatures for Bearer token authentication

package services

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
)

// jwksResponse represents the JSON Web Key Set response from UAA
type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

// jwkKey represents a single JSON Web Key
type jwkKey struct {
	Kty string `json:"kty"` // Key type (RSA)
	Kid string `json:"kid"` // Key ID
	N   string `json:"n"`   // RSA modulus (base64url)
	E   string `json:"e"`   // RSA exponent (base64url)
	Alg string `json:"alg"` // Algorithm (RS256)
	Use string `json:"use"` // Usage (sig)
}

// parseJWKS parses a JWKS JSON response and returns a map of key ID to RSA public key
func parseJWKS(data []byte) (map[string]*rsa.PublicKey, error) {
	var resp jwksResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey)
	for _, k := range resp.Keys {
		if k.Kty != "RSA" {
			continue // Skip non-RSA keys
		}

		pubKey, err := parseRSAPublicKey(k.N, k.E)
		if err != nil {
			return nil, fmt.Errorf("failed to parse key %s: %w", k.Kid, err)
		}

		keys[k.Kid] = pubKey
	}

	return keys, nil
}

// parseRSAPublicKey converts base64url-encoded modulus and exponent to RSA public key
func parseRSAPublicKey(nB64, eB64 string) (*rsa.PublicKey, error) {
	// Decode modulus
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	// Decode exponent
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert to big integers
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}
```

### Step 1.4: Run test to verify it passes

Run: `cd backend && go test ./services -run TestParseJWKS_ValidResponse -v`
Expected: PASS

### Step 1.5: Add test for multiple keys

```go
// Add to backend/services/jwks_test.go

func TestParseJWKS_MultipleKeys(t *testing.T) {
	jwksJSON := `{
		"keys": [
			{"kty": "RSA", "kid": "key-1", "n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw", "e": "AQAB", "alg": "RS256", "use": "sig"},
			{"kty": "RSA", "kid": "key-2", "n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw", "e": "AQAB", "alg": "RS256", "use": "sig"}
		]
	}`

	keys, err := parseJWKS([]byte(jwksJSON))
	if err != nil {
		t.Fatalf("parseJWKS failed: %v", err)
	}

	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}

	if _, ok := keys["key-1"]; !ok {
		t.Error("key-1 not found")
	}
	if _, ok := keys["key-2"]; !ok {
		t.Error("key-2 not found")
	}
}

func TestParseJWKS_SkipsNonRSAKeys(t *testing.T) {
	jwksJSON := `{
		"keys": [
			{"kty": "RSA", "kid": "rsa-key", "n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw", "e": "AQAB", "alg": "RS256", "use": "sig"},
			{"kty": "EC", "kid": "ec-key", "crv": "P-256", "x": "test", "y": "test"}
		]
	}`

	keys, err := parseJWKS([]byte(jwksJSON))
	if err != nil {
		t.Fatalf("parseJWKS failed: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("expected 1 RSA key, got %d", len(keys))
	}

	if _, ok := keys["rsa-key"]; !ok {
		t.Error("rsa-key not found")
	}
}

func TestParseJWKS_InvalidJSON(t *testing.T) {
	_, err := parseJWKS([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
```

### Step 1.6: Run all JWKS parsing tests

Run: `cd backend && go test ./services -run TestParseJWKS -v`
Expected: All PASS

### Step 1.7: Commit

```bash
git add backend/services/jwks.go backend/services/jwks_test.go
git commit -m "feat(auth): add JWKS response parsing for UAA public keys"
```

---

## Task 2: JWT Signature Verification

**Files:**

- Modify: `backend/services/jwks.go`
- Modify: `backend/services/jwks_test.go`
- Create: `backend/services/testdata/rsa_test_key.pem` (test fixture)

### Step 2.1: Generate test RSA key pair

Run: `mkdir -p backend/services/testdata && openssl genrsa -out backend/services/testdata/rsa_test_private.pem 2048 && openssl rsa -in backend/services/testdata/rsa_test_private.pem -pubout -out backend/services/testdata/rsa_test_public.pem`

### Step 2.2: Write test for signature verification

```go
// Add to backend/services/jwks_test.go

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
	"strings"
	"testing"
	"time"
)

// Helper to load test private key
func loadTestPrivateKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	data, err := os.ReadFile("testdata/rsa_test_private.pem")
	if err != nil {
		t.Fatalf("failed to read test private key: %v", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatal("failed to decode PEM block")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse private key: %v", err)
	}
	return key
}

// Helper to create a signed test JWT
func createTestJWT(t *testing.T, privateKey *rsa.PrivateKey, kid string, claims map[string]interface{}) string {
	t.Helper()

	// Header
	header := map[string]string{
		"alg": "RS256",
		"typ": "JWT",
		"kid": kid,
	}
	headerJSON, _ := json.Marshal(header)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	// Payload
	payloadJSON, _ := json.Marshal(claims)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	// Sign
	signingInput := headerB64 + "." + payloadB64
	hash := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		t.Fatalf("failed to sign JWT: %v", err)
	}
	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

	return signingInput + "." + signatureB64
}

func TestVerifyJWT_ValidSignature(t *testing.T) {
	privateKey := loadTestPrivateKey(t)
	publicKey := &privateKey.PublicKey

	keys := map[string]*rsa.PublicKey{
		"test-key": publicKey,
	}

	claims := map[string]interface{}{
		"user_name": "testuser",
		"user_id":   "user-123",
		"exp":       time.Now().Add(time.Hour).Unix(),
	}

	token := createTestJWT(t, privateKey, "test-key", claims)

	result, err := verifyJWT(token, keys)
	if err != nil {
		t.Fatalf("verifyJWT failed: %v", err)
	}

	if result.Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", result.Username)
	}
	if result.UserID != "user-123" {
		t.Errorf("expected user_id 'user-123', got '%s'", result.UserID)
	}
}

func TestVerifyJWT_ExpiredToken(t *testing.T) {
	privateKey := loadTestPrivateKey(t)
	publicKey := &privateKey.PublicKey

	keys := map[string]*rsa.PublicKey{
		"test-key": publicKey,
	}

	claims := map[string]interface{}{
		"user_name": "testuser",
		"user_id":   "user-123",
		"exp":       time.Now().Add(-time.Hour).Unix(), // Expired
	}

	token := createTestJWT(t, privateKey, "test-key", claims)

	_, err := verifyJWT(token, keys)
	if err == nil {
		t.Error("expected error for expired token")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("expected 'expired' in error, got: %v", err)
	}
}

func TestVerifyJWT_InvalidSignature(t *testing.T) {
	privateKey := loadTestPrivateKey(t)

	// Use a different key for verification
	otherKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	keys := map[string]*rsa.PublicKey{
		"test-key": &otherKey.PublicKey, // Wrong key
	}

	claims := map[string]interface{}{
		"user_name": "testuser",
		"exp":       time.Now().Add(time.Hour).Unix(),
	}

	token := createTestJWT(t, privateKey, "test-key", claims)

	_, err := verifyJWT(token, keys)
	if err == nil {
		t.Error("expected error for invalid signature")
	}
}

func TestVerifyJWT_UnknownKeyID(t *testing.T) {
	privateKey := loadTestPrivateKey(t)

	keys := map[string]*rsa.PublicKey{
		"different-key": &privateKey.PublicKey,
	}

	claims := map[string]interface{}{
		"user_name": "testuser",
		"exp":       time.Now().Add(time.Hour).Unix(),
	}

	token := createTestJWT(t, privateKey, "unknown-key", claims)

	_, err := verifyJWT(token, keys)
	if err == nil {
		t.Error("expected error for unknown key ID")
	}
	if !strings.Contains(err.Error(), "unknown key") {
		t.Errorf("expected 'unknown key' in error, got: %v", err)
	}
}

func TestVerifyJWT_MissingKid(t *testing.T) {
	privateKey := loadTestPrivateKey(t)
	publicKey := &privateKey.PublicKey

	keys := map[string]*rsa.PublicKey{
		"test-key": publicKey,
	}

	// Create token with empty kid
	claims := map[string]interface{}{
		"user_name": "testuser",
		"exp":       time.Now().Add(time.Hour).Unix(),
	}

	token := createTestJWT(t, privateKey, "", claims) // Empty kid

	_, err := verifyJWT(token, keys)
	if err == nil {
		t.Error("expected error for missing kid")
	}
	if !strings.Contains(err.Error(), "missing kid") {
		t.Errorf("expected 'missing kid' in error, got: %v", err)
	}
}

func TestVerifyJWT_NotYetValid(t *testing.T) {
	privateKey := loadTestPrivateKey(t)
	publicKey := &privateKey.PublicKey

	keys := map[string]*rsa.PublicKey{
		"test-key": publicKey,
	}

	claims := map[string]interface{}{
		"user_name": "testuser",
		"user_id":   "user-123",
		"exp":       time.Now().Add(time.Hour).Unix(),
		"nbf":       time.Now().Add(time.Hour).Unix(), // Not valid until 1 hour from now
	}

	token := createTestJWT(t, privateKey, "test-key", claims)

	_, err := verifyJWT(token, keys)
	if err == nil {
		t.Error("expected error for not yet valid token")
	}
	if !strings.Contains(err.Error(), "not yet valid") {
		t.Errorf("expected 'not yet valid' in error, got: %v", err)
	}
}
```

### Step 2.3: Run test to verify it fails

Run: `cd backend && go test ./services -run TestVerifyJWT -v`
Expected: FAIL with "undefined: verifyJWT"

### Step 2.4: Write verifyJWT implementation

```go
// Add to backend/services/jwks.go

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"hash"
	"strings"
	"time"
)

// JWTClaims contains extracted and verified JWT claims
type JWTClaims struct {
	Username string
	UserID   string
}

// jwtHeader represents the JWT header
type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	Typ string `json:"typ"`
}

// jwtPayload represents the JWT payload with UAA claims
type jwtPayload struct {
	UserName string `json:"user_name"`
	UserID   string `json:"user_id"`
	ClientID string `json:"client_id"`
	Sub      string `json:"sub"`
	Exp      int64  `json:"exp"`
	Nbf      int64  `json:"nbf"` // Not before (RFC 7519)
}

// verifyJWT verifies a JWT signature and returns the claims
func verifyJWT(token string, keys map[string]*rsa.PublicKey) (*JWTClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed token: expected 3 parts, got %d", len(parts))
	}

	// Parse header
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to decode header: %w", err)
	}

	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("failed to parse header: %w", err)
	}

	// Validate algorithm (prevent algorithm confusion attacks)
	var hashFunc hash.Hash
	var cryptoHash crypto.Hash
	switch header.Alg {
	case "RS256":
		hashFunc = sha256.New()
		cryptoHash = crypto.SHA256
	case "RS384":
		hashFunc = sha512.New384()
		cryptoHash = crypto.SHA384
	case "RS512":
		hashFunc = sha512.New()
		cryptoHash = crypto.SHA512
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s (only RS256/RS384/RS512 allowed)", header.Alg)
	}

	// Reject tokens with missing kid header
	if header.Kid == "" {
		return nil, fmt.Errorf("missing kid (key ID) in token header")
	}

	// Get public key
	pubKey, ok := keys[header.Kid]
	if !ok {
		return nil, fmt.Errorf("unknown key ID: %s", header.Kid)
	}

	// CRITICAL: Verify signature BEFORE checking expiration
	// This prevents timing attacks where attackers can determine if
	// a forged token's structure is valid by observing error types
	signingInput := parts[0] + "." + parts[1]
	hashFunc.Write([]byte(signingInput))
	hashed := hashFunc.Sum(nil)

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %w", err)
	}

	if err := rsa.VerifyPKCS1v15(pubKey, cryptoHash, hashed, signature); err != nil {
		return nil, fmt.Errorf("invalid signature: %w", err)
	}

	// Parse payload (after signature is verified)
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	var payload jwtPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse payload: %w", err)
	}

	// Check nbf (not before) - RFC 7519 compliance
	if payload.Nbf > 0 && time.Now().Unix() < payload.Nbf {
		return nil, fmt.Errorf("token not yet valid (nbf)")
	}

	// Check expiration
	if payload.Exp > 0 && time.Now().Unix() > payload.Exp {
		return nil, fmt.Errorf("token expired")
	}

	// Extract claims (support both user tokens and client credentials)
	claims := &JWTClaims{}
	if payload.UserName != "" {
		claims.Username = payload.UserName
		claims.UserID = payload.UserID
	} else if payload.ClientID != "" {
		claims.Username = payload.ClientID
		claims.UserID = payload.Sub
	} else {
		return nil, fmt.Errorf("token missing required claims (user_name or client_id)")
	}

	return claims, nil
}
```

### Step 2.5: Run tests to verify they pass

Run: `cd backend && go test ./services -run TestVerifyJWT -v`
Expected: All PASS

### Step 2.6: Commit

```bash
git add backend/services/jwks.go backend/services/jwks_test.go backend/services/testdata/
git commit -m "feat(auth): add JWT signature verification with RS256/384/512 support"
```

---

## Task 3: JWKS Client with HTTP Fetching

**Files:**

- Modify: `backend/services/jwks.go`
- Modify: `backend/services/jwks_test.go`

### Step 3.1: Write test for JWKS client initialization

```go
// Add to backend/services/jwks_test.go

import (
	"net/http"
	"net/http/httptest"
	"sync"
)

func TestJWKSClient_FetchKeys(t *testing.T) {
	// Create mock UAA server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token_keys" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"keys": [
				{"kty": "RSA", "kid": "key-1", "n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw", "e": "AQAB", "alg": "RS256", "use": "sig"}
			]
		}`))
	}))
	defer server.Close()

	client, err := NewJWKSClient(server.URL, nil)
	if err != nil {
		t.Fatalf("NewJWKSClient failed: %v", err)
	}

	key := client.GetKey("key-1")
	if key == nil {
		t.Fatal("expected key-1 to be cached")
	}
}

func TestJWKSClient_RefreshOnUnknownKey(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			// First call: return key-1
			w.Write([]byte(`{"keys": [{"kty": "RSA", "kid": "key-1", "n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw", "e": "AQAB", "alg": "RS256", "use": "sig"}]}`))
		} else {
			// Subsequent calls: return key-1 and key-2
			w.Write([]byte(`{"keys": [
				{"kty": "RSA", "kid": "key-1", "n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw", "e": "AQAB", "alg": "RS256", "use": "sig"},
				{"kty": "RSA", "kid": "key-2", "n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw", "e": "AQAB", "alg": "RS256", "use": "sig"}
			]}`))
		}
	}))
	defer server.Close()

	client, err := NewJWKSClient(server.URL, nil)
	if err != nil {
		t.Fatalf("NewJWKSClient failed: %v", err)
	}

	// key-1 should be available
	if client.GetKey("key-1") == nil {
		t.Fatal("key-1 should be cached")
	}

	// key-2 not yet available, triggers refresh
	key2 := client.GetKey("key-2")
	if key2 == nil {
		t.Fatal("key-2 should be available after refresh")
	}

	if callCount != 2 {
		t.Errorf("expected 2 fetch calls, got %d", callCount)
	}
}

func TestJWKSClient_ConcurrentRefresh_ThunderingHerd(t *testing.T) {
	// Test that concurrent requests for unknown key only trigger one refresh
	callCount := 0
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		mu.Unlock()

		// Simulate slow UAA response
		time.Sleep(100 * time.Millisecond)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"keys": [{"kty": "RSA", "kid": "key-1", "n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw", "e": "AQAB", "alg": "RS256", "use": "sig"}]}`))
	}))
	defer server.Close()

	// Create client but clear keys to force refresh
	client, err := NewJWKSClient(server.URL, nil)
	if err != nil {
		t.Fatalf("NewJWKSClient failed: %v", err)
	}

	// Clear keys to simulate unknown key scenario
	client.mu.Lock()
	client.keys = make(map[string]*rsa.PublicKey)
	client.mu.Unlock()

	// Reset call count after initial fetch
	mu.Lock()
	callCount = 0
	mu.Unlock()

	// Spawn multiple concurrent requests for the same unknown key
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client.GetKey("key-1")
		}()
	}
	wg.Wait()

	// Should only have made ONE refresh call due to singleflight
	mu.Lock()
	finalCount := callCount
	mu.Unlock()

	if finalCount != 1 {
		t.Errorf("expected 1 refresh call (singleflight), got %d", finalCount)
	}
}
```

### Step 3.2: Run test to verify it fails

Run: `cd backend && go test ./services -run TestJWKSClient -v`
Expected: FAIL with "undefined: NewJWKSClient"

### Step 3.3: Write JWKSClient implementation

```go
// Add to backend/services/jwks.go

import (
	"io"
	"log/slog"
	"net/http"
	"sync"

	"golang.org/x/sync/singleflight"
)

// JWKSClient fetches and caches public keys from CF UAA
type JWKSClient struct {
	uaaURL     string
	httpClient *http.Client
	keys       map[string]*rsa.PublicKey
	mu         sync.RWMutex
	sfGroup    singleflight.Group // Prevents thundering herd on refresh
}

// NewJWKSClient creates a JWKS client and fetches initial keys
// Uses 30 second timeout for initial fetch (startup), 10 second for runtime refresh
func NewJWKSClient(uaaURL string, httpClient *http.Client) (*JWKSClient, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second} // Startup timeout
	}

	client := &JWKSClient{
		uaaURL:     uaaURL,
		httpClient: httpClient,
		keys:       make(map[string]*rsa.PublicKey),
	}

	// Fetch initial keys
	if err := client.refresh(); err != nil {
		return nil, fmt.Errorf("failed to fetch initial JWKS: %w", err)
	}

	return client, nil
}

// GetKey returns the public key for the given key ID
// If not found, attempts one refresh before returning nil
// Uses singleflight to prevent thundering herd on concurrent refresh requests
func (c *JWKSClient) GetKey(kid string) *rsa.PublicKey {
	c.mu.RLock()
	key, ok := c.keys[kid]
	c.mu.RUnlock()

	if ok {
		return key
	}

	// Key not found, try refreshing with singleflight to prevent thundering herd
	slog.Debug("JWKS key not found, refreshing", "kid", kid)
	_, err, _ := c.sfGroup.Do("refresh", func() (interface{}, error) {
		return nil, c.refresh()
	})
	if err != nil {
		slog.Error("JWKS refresh failed", "error", err)
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.keys[kid]
}

// refresh fetches the latest keys from UAA
func (c *JWKSClient) refresh() error {
	resp, err := c.httpClient.Get(c.uaaURL + "/token_keys")
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS fetch returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read JWKS response: %w", err)
	}

	keys, err := parseJWKS(body)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.keys = keys
	c.mu.Unlock()

	slog.Info("JWKS keys refreshed", "count", len(keys))
	return nil
}

// VerifyAndParse verifies a JWT and returns the claims
func (c *JWKSClient) VerifyAndParse(token string) (*JWTClaims, error) {
	// First attempt with cached keys
	c.mu.RLock()
	keys := c.keys
	c.mu.RUnlock()

	claims, err := verifyJWT(token, keys)
	if err != nil {
		// Check if it's an unknown key error
		if strings.Contains(err.Error(), "unknown key ID") {
			// Try refreshing keys once
			if refreshErr := c.refresh(); refreshErr != nil {
				return nil, err // Return original error
			}
			c.mu.RLock()
			keys = c.keys
			c.mu.RUnlock()
			return verifyJWT(token, keys)
		}
		return nil, err
	}
	return claims, nil
}
```

### Step 3.4: Run tests to verify they pass

Run: `cd backend && go test ./services -run TestJWKSClient -v`
Expected: All PASS

### Step 3.5: Commit

```bash
git add backend/services/jwks.go backend/services/jwks_test.go
git commit -m "feat(auth): add JWKS client with HTTP fetching and key caching"
```

---

## Task 4: Integrate JWKS Client with Auth Middleware

**Files:**

- Modify: `backend/middleware/auth.go`
- Modify: `backend/middleware/auth_test.go`

### Step 4.1: Write integration test for Bearer token with JWKS

```go
// Add to backend/middleware/auth_test.go

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/services"
)

// Helper to create test JWT (same as in services tests)
func createTestJWTForMiddleware(t *testing.T, privateKey *rsa.PrivateKey, kid string, claims map[string]interface{}) string {
	t.Helper()

	header := map[string]string{"alg": "RS256", "typ": "JWT", "kid": kid}
	headerJSON, _ := json.Marshal(header)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	payloadJSON, _ := json.Marshal(claims)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	signingInput := headerB64 + "." + payloadB64
	hash := sha256.Sum256([]byte(signingInput))
	signature, _ := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

	return signingInput + "." + signatureB64
}

func TestAuth_BearerWithJWKS(t *testing.T) {
	// Generate test key pair
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	// Create mock UAA server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Convert public key to JWK format
		n := base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes())
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"keys": [{"kty": "RSA", "kid": "test-key", "n": "` + n + `", "e": "AQAB", "alg": "RS256", "use": "sig"}]}`))
	}))
	defer server.Close()

	// Create JWKS client
	jwksClient, err := services.NewJWKSClient(server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create JWKS client: %v", err)
	}

	// Create valid token
	claims := map[string]interface{}{
		"user_name": "testuser",
		"user_id":   "user-123",
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	token := createTestJWTForMiddleware(t, privateKey, "test-key", claims)

	// Create middleware
	cfg := AuthConfig{
		Mode:       AuthModeRequired,
		JWKSClient: jwksClient,
	}
	middleware := Auth(cfg)

	// Create handler that checks claims
	var extractedClaims *UserClaims
	handler := middleware(func(w http.ResponseWriter, r *http.Request) {
		extractedClaims = GetUserClaims(r)
		w.WriteHeader(http.StatusOK)
	})

	// Test request
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if extractedClaims == nil {
		t.Fatal("expected claims to be extracted")
	}
	if extractedClaims.Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", extractedClaims.Username)
	}
}

func TestAuth_BearerWithoutJWKSClient(t *testing.T) {
	cfg := AuthConfig{
		Mode:       AuthModeRequired,
		JWKSClient: nil, // No JWKS client
	}
	middleware := Auth(cfg)

	handler := middleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
```

### Step 4.2: Run test to verify it fails

Run: `cd backend && go test ./middleware -run TestAuth_Bearer -v`
Expected: FAIL (AuthConfig doesn't have JWKSClient field yet)

### Step 4.3: Update AuthConfig and Auth middleware

```go
// Modify backend/middleware/auth.go

// Add import
import "github.com/markalston/diego-capacity-analyzer/backend/services"

// Update AuthConfig struct
type AuthConfig struct {
	Mode             AuthMode
	SessionValidator SessionValidatorFunc
	JWKSClient       *services.JWKSClient // For Bearer token signature verification
}

// Update Auth function - replace the Bearer token handling section
func Auth(cfg AuthConfig) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Disabled mode: pass through
			if cfg.Mode == AuthModeDisabled {
				next(w, r)
				return
			}

			// Check Bearer token first (takes precedence)
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				// Validate Bearer format
				if !strings.HasPrefix(authHeader, "Bearer ") {
					slog.Debug("Auth rejected: invalid format", "path", r.URL.Path)
					http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
					return
				}

				token := strings.TrimPrefix(authHeader, "Bearer ")

				// JWKS client required for Bearer token verification
				if cfg.JWKSClient == nil {
					slog.Debug("Auth rejected: JWKS client not configured", "path", r.URL.Path)
					http.Error(w, "Bearer authentication unavailable, please use web UI login", http.StatusUnauthorized)
					return
				}

				// Verify signature and parse claims
				jwtClaims, err := cfg.JWKSClient.VerifyAndParse(token)
				if err != nil {
					slog.Debug("Auth rejected: invalid token", "path", r.URL.Path, "error", err.Error())
					http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
					return
				}

				claims := &UserClaims{
					Username: jwtClaims.Username,
					UserID:   jwtClaims.UserID,
				}

				slog.Debug("Auth: valid bearer token", "path", r.URL.Path, "user", claims.Username)
				ctx := context.WithValue(r.Context(), userClaimsKey, claims)
				next(w, r.WithContext(ctx))
				return
			}

			// Rest of the function remains the same (session cookie handling)
			// ... existing session cookie code ...
		}
	}
}
```

### Step 4.4: Run tests to verify they pass

Run: `cd backend && go test ./middleware -run TestAuth -v`
Expected: All PASS

### Step 4.5: Run all middleware tests

Run: `cd backend && go test ./middleware -v`
Expected: All PASS

### Step 4.6: Commit

```bash
git add backend/middleware/auth.go backend/middleware/auth_test.go
git commit -m "feat(auth): integrate JWKS client with auth middleware for signature verification"
```

---

## Task 5: Wire JWKS Client in main.go

**Files:**

- Modify: `backend/main.go`

### Step 5.1: Update main.go to initialize JWKS client

The JWKS client needs to be created after we discover the UAA URL. We'll reuse the getUAAURL logic from handlers/auth.go.

```go
// Modify backend/main.go

// Add to imports
import (
	"crypto/tls"
)

// After session service initialization, add JWKS client initialization:

// Discover UAA URL for JWKS client
var jwksClient *services.JWKSClient
if cfg.CFAPIUrl != "" {
	uaaURL, err := discoverUAAURL(cfg)
	if err != nil {
		slog.Warn("Failed to discover UAA URL, Bearer auth disabled", "error", err)
	} else {
		// Create HTTP client with same TLS settings as CF client
		httpClient := &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.CFSkipSSLValidation},
			},
		}
		jwksClient, err = services.NewJWKSClient(uaaURL, httpClient)
		if err != nil {
			slog.Warn("Failed to initialize JWKS client, Bearer auth disabled", "error", err)
		} else {
			slog.Info("JWKS client initialized", "uaaURL", uaaURL)
		}
	}
}

// Update authCfg to include JWKS client
authCfg := middleware.AuthConfig{
	Mode:             authMode,
	SessionValidator: sessionValidator,
	JWKSClient:       jwksClient, // May be nil for graceful degradation
}

// Add helper function at end of file
func discoverUAAURL(cfg *config.Config) (string, error) {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.CFSkipSSLValidation},
		},
	}

	resp, err := httpClient.Get(cfg.CFAPIUrl + "/v3/info")
	if err != nil {
		return "", fmt.Errorf("failed to get CF info: %w", err)
	}
	defer resp.Body.Close()

	var info struct {
		Links struct {
			Login struct {
				Href string `json:"href"`
			} `json:"login"`
		} `json:"links"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", fmt.Errorf("failed to parse CF info: %w", err)
	}

	uaaURL := info.Links.Login.Href
	if uaaURL == "" {
		// Fallback: construct from API URL
		uaaURL = strings.Replace(cfg.CFAPIUrl, "://api.", "://login.", 1)
	}

	return uaaURL, nil
}
```

### Step 5.2: Verify the code compiles

Run: `cd backend && go build -o /dev/null .`
Expected: Build succeeds

### Step 5.3: Run all tests

Run: `cd backend && go test ./... -v`
Expected: All PASS

### Step 5.4: Commit

```bash
git add backend/main.go
git commit -m "feat(auth): wire JWKS client into main.go with UAA discovery"
```

---

## Task 6: Add Integration Test with Mock UAA

**Files:**

- Create: `backend/e2e/auth_test.go`

### Step 6.1: Write integration test

```go
// backend/e2e/auth_test.go
// ABOUTME: Integration tests for authentication with JWKS verification
// ABOUTME: Tests full auth flow with mock UAA server

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

func createTestJWT(t *testing.T, privateKey *rsa.PrivateKey, kid string, claims map[string]interface{}) string {
	t.Helper()

	header := map[string]string{"alg": "RS256", "typ": "JWT", "kid": kid}
	headerJSON, _ := json.Marshal(header)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	payloadJSON, _ := json.Marshal(claims)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	signingInput := headerB64 + "." + payloadB64
	hash := sha256.Sum256([]byte(signingInput))
	signature, _ := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

	return signingInput + "." + signatureB64
}

func TestAuthIntegration_BearerTokenWithJWKS(t *testing.T) {
	// Generate test key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Create mock UAA server serving JWKS
	uaaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token_keys" {
			n := base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes())
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"keys": []map[string]string{
					{"kty": "RSA", "kid": "test-key-1", "n": n, "e": "AQAB", "alg": "RS256", "use": "sig"},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer uaaServer.Close()

	// Initialize JWKS client
	jwksClient, err := services.NewJWKSClient(uaaServer.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create JWKS client: %v", err)
	}

	// Create auth middleware
	authCfg := middleware.AuthConfig{
		Mode:       middleware.AuthModeRequired,
		JWKSClient: jwksClient,
	}
	authMiddleware := middleware.Auth(authCfg)

	// Create protected endpoint
	handler := authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		claims := middleware.GetUserClaims(r)
		json.NewEncoder(w).Encode(map[string]string{
			"username": claims.Username,
			"user_id":  claims.UserID,
		})
	})

	tests := []struct {
		name       string
		token      func() string
		wantStatus int
	}{
		{
			name: "valid user token",
			token: func() string {
				return createTestJWT(t, privateKey, "test-key-1", map[string]interface{}{
					"user_name": "admin@example.com",
					"user_id":   "user-abc-123",
					"exp":       time.Now().Add(time.Hour).Unix(),
				})
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "valid client credentials token",
			token: func() string {
				return createTestJWT(t, privateKey, "test-key-1", map[string]interface{}{
					"client_id": "ci-automation",
					"sub":       "client-xyz-789",
					"exp":       time.Now().Add(time.Hour).Unix(),
				})
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "expired token",
			token: func() string {
				return createTestJWT(t, privateKey, "test-key-1", map[string]interface{}{
					"user_name": "admin@example.com",
					"exp":       time.Now().Add(-time.Hour).Unix(),
				})
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "wrong key id",
			token: func() string {
				return createTestJWT(t, privateKey, "unknown-key", map[string]interface{}{
					"user_name": "admin@example.com",
					"exp":       time.Now().Add(time.Hour).Unix(),
				})
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid signature",
			token: func() string {
				// Create token signed with different key
				otherKey, _ := rsa.GenerateKey(rand.Reader, 2048)
				return createTestJWT(t, otherKey, "test-key-1", map[string]interface{}{
					"user_name": "admin@example.com",
					"exp":       time.Now().Add(time.Hour).Unix(),
				})
			},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/dashboard", nil)
			req.Header.Set("Authorization", "Bearer "+tt.token())
			rec := httptest.NewRecorder()

			handler(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestAuthIntegration_SessionCookieFallback(t *testing.T) {
	// Test that session cookies still work when no Bearer token provided
	sessionValidator := func(sessionID string) *middleware.UserClaims {
		if sessionID == "valid-session-123" {
			return &middleware.UserClaims{
				Username: "session-user",
				UserID:   "session-user-id",
			}
		}
		return nil
	}

	authCfg := middleware.AuthConfig{
		Mode:             middleware.AuthModeRequired,
		SessionValidator: sessionValidator,
		JWKSClient:       nil, // No JWKS client
	}
	authMiddleware := middleware.Auth(authCfg)

	handler := authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		claims := middleware.GetUserClaims(r)
		json.NewEncoder(w).Encode(map[string]string{
			"username": claims.Username,
		})
	})

	req := httptest.NewRequest("GET", "/api/v1/dashboard", nil)
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "valid-session-123"})
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want 200: %s", rec.Code, rec.Body.String())
	}
}
```

### Step 6.2: Run integration tests

Run: `cd backend && go test ./e2e -run TestAuthIntegration -v`
Expected: All PASS

### Step 6.3: Run all tests

Run: `cd backend && go test ./... -v`
Expected: All PASS

### Step 6.4: Commit

```bash
git add backend/e2e/auth_test.go
git commit -m "test(auth): add integration tests for JWKS-based JWT verification"
```

---

## Task 7: Update OpenAPI Documentation

**Files:**

- Modify: `backend/handlers/openapi.go` (or wherever OpenAPI spec is defined)

### Step 7.1: Find and update OpenAPI spec

Run: `grep -r "securitySchemes\|bearerAuth" backend/` to find where security is documented.

Add Bearer token documentation to the OpenAPI spec:

```yaml
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: |
        JWT token from CF UAA. Obtain via:
        - `cf oauth-token` for user tokens
        - OAuth2 client_credentials grant for automation
    cookieAuth:
      type: apiKey
      in: cookie
      name: DIEGO_SESSION
      description: Session cookie from /api/v1/auth/login

security:
  - bearerAuth: []
  - cookieAuth: []
```

### Step 7.2: Commit documentation update

```bash
git add backend/handlers/openapi.go  # or relevant file
git commit -m "docs(api): add Bearer token authentication to OpenAPI spec"
```

---

## Task 8: Final Verification and Cleanup

### Step 8.1: Run linter

Run: `cd backend && golangci-lint run ./...`
Expected: No errors

### Step 8.2: Run all tests with race detection

Run: `cd backend && go test -race ./...`
Expected: All PASS, no race conditions

### Step 8.3: Verify test coverage

Run: `cd backend && go test -cover ./services ./middleware`
Expected: Coverage report shows >80% for new code

### Step 8.4: Final commit if any cleanup needed

```bash
git add -A
git commit -m "chore: cleanup and formatting"
```

### Step 8.5: Push branch

```bash
git push -u origin feature/issue-44-jwt-signature-verification
```

---

## Summary

| Task | Description                    | Tests                                |
| ---- | ------------------------------ | ------------------------------------ |
| 1    | JWKS response parsing          | 4 tests                              |
| 2    | JWT signature verification     | 7 tests (+3 for nbf, missing kid)    |
| 3    | JWKS client with HTTP fetching | 3 tests (+1 for thundering herd)     |
| 4    | Auth middleware integration    | 2 tests                              |
| 5    | main.go wiring                 | Build verification                   |
| 6    | Integration tests              | 5 tests                              |
| 7    | OpenAPI documentation          | Manual verification                  |
| 8    | Final verification             | Lint, race, coverage                 |

**Total new tests:** ~21 tests covering JWKS parsing, signature verification, nbf validation, missing kid handling, thundering herd prevention, key rotation, and full auth flow.

**Review feedback addressed:**
- ✅ Signature verification before expiration check
- ✅ singleflight for thundering herd prevention
- ✅ nbf (not before) claim validation
- ✅ Missing kid header rejection
- ✅ HTTP timeout configuration
- ✅ Improved error message for JWKS unavailable
