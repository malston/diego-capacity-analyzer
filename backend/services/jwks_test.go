// ABOUTME: Tests for JWKS parsing and JWT verification functionality
// ABOUTME: Verifies parsing of CF UAA's JWKS JSON into RSA public keys and JWT signature verification

package services

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"hash"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
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

// -----------------------------------------------------------------------------
// JWT Verification Tests
// -----------------------------------------------------------------------------

// testdataPath returns the absolute path to the testdata directory
func testdataPath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata")
}

// loadTestPrivateKey loads the test RSA private key from testdata/
func loadTestPrivateKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	keyPath := filepath.Join(testdataPath(), "rsa_test_private.pem")
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("failed to read test private key: %v", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		t.Fatal("failed to decode PEM block from private key")
	}

	// Try parsing as PKCS#8 first (OpenSSL 3.x default)
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Fall back to PKCS#1
		rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			t.Fatalf("failed to parse private key: %v", err)
		}
		return rsaKey
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		t.Fatal("private key is not RSA")
	}
	return rsaKey
}

// loadTestPublicKey loads the test RSA public key from testdata/
func loadTestPublicKey(t *testing.T) *rsa.PublicKey {
	t.Helper()

	keyPath := filepath.Join(testdataPath(), "rsa_test_public.pem")
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("failed to read test public key: %v", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		t.Fatal("failed to decode PEM block from public key")
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse public key: %v", err)
	}

	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		t.Fatal("public key is not RSA")
	}
	return rsaKey
}

// jwtHeader represents the header portion of a JWT
type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	Kid string `json:"kid,omitempty"`
}

// jwtPayload represents the payload/claims portion of a JWT for test purposes
type jwtPayload struct {
	Sub      string `json:"sub,omitempty"`
	UserName string `json:"user_name,omitempty"`
	UserID   string `json:"user_id,omitempty"`
	ClientID string `json:"client_id,omitempty"`
	Exp      int64  `json:"exp,omitempty"`
	Nbf      int64  `json:"nbf,omitempty"`
	Iat      int64  `json:"iat,omitempty"`
	Iss      string `json:"iss,omitempty"`
}

// createTestJWT creates a signed JWT for testing
func createTestJWT(t *testing.T, privateKey *rsa.PrivateKey, kid string, alg string, claims jwtPayload) string {
	t.Helper()

	header := jwtHeader{
		Alg: alg,
		Typ: "JWT",
		Kid: kid,
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("failed to marshal header: %v", err)
	}

	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	signingInput := headerB64 + "." + payloadB64

	// Select hash based on algorithm
	var hashFunc hash.Hash
	var cryptoHash crypto.Hash
	switch alg {
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
		t.Fatalf("unsupported algorithm for test: %s", alg)
	}

	hashFunc.Write([]byte(signingInput))
	hashed := hashFunc.Sum(nil)

	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, cryptoHash, hashed)
	if err != nil {
		t.Fatalf("failed to sign JWT: %v", err)
	}

	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

	return signingInput + "." + signatureB64
}

// createTestJWTWithoutKid creates a JWT without a kid header for testing
func createTestJWTWithoutKid(t *testing.T, privateKey *rsa.PrivateKey, alg string, claims jwtPayload) string {
	t.Helper()

	// Create header without kid
	header := struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}{
		Alg: alg,
		Typ: "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("failed to marshal header: %v", err)
	}

	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	signingInput := headerB64 + "." + payloadB64

	hashFunc := sha256.New()
	hashFunc.Write([]byte(signingInput))
	hashed := hashFunc.Sum(nil)

	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed)
	if err != nil {
		t.Fatalf("failed to sign JWT: %v", err)
	}

	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

	return signingInput + "." + signatureB64
}

func TestVerifyJWT_ValidSignature(t *testing.T) {
	privateKey := loadTestPrivateKey(t)
	publicKey := loadTestPublicKey(t)

	keys := map[string]*rsa.PublicKey{
		"test-key-1": publicKey,
	}

	claims := jwtPayload{
		Sub:      "user-123",
		UserName: "testuser",
		UserID:   "user-123",
		Exp:      time.Now().Add(1 * time.Hour).Unix(),
		Iat:      time.Now().Unix(),
		Iss:      "https://uaa.example.com",
	}

	token := createTestJWT(t, privateKey, "test-key-1", "RS256", claims)

	result, err := verifyJWT(token, keys)
	if err != nil {
		t.Fatalf("verifyJWT returned error: %v", err)
	}

	if result.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %q", result.Username)
	}

	if result.UserID != "user-123" {
		t.Errorf("expected userID 'user-123', got %q", result.UserID)
	}
}

func TestVerifyJWT_ValidSignature_RS384(t *testing.T) {
	privateKey := loadTestPrivateKey(t)
	publicKey := loadTestPublicKey(t)

	keys := map[string]*rsa.PublicKey{
		"test-key-1": publicKey,
	}

	claims := jwtPayload{
		Sub:      "user-456",
		UserName: "testuser384",
		UserID:   "user-456",
		Exp:      time.Now().Add(1 * time.Hour).Unix(),
		Iat:      time.Now().Unix(),
	}

	token := createTestJWT(t, privateKey, "test-key-1", "RS384", claims)

	result, err := verifyJWT(token, keys)
	if err != nil {
		t.Fatalf("verifyJWT returned error for RS384: %v", err)
	}

	if result.Username != "testuser384" {
		t.Errorf("expected username 'testuser384', got %q", result.Username)
	}
}

func TestVerifyJWT_ValidSignature_RS512(t *testing.T) {
	privateKey := loadTestPrivateKey(t)
	publicKey := loadTestPublicKey(t)

	keys := map[string]*rsa.PublicKey{
		"test-key-1": publicKey,
	}

	claims := jwtPayload{
		Sub:      "user-789",
		UserName: "testuser512",
		UserID:   "user-789",
		Exp:      time.Now().Add(1 * time.Hour).Unix(),
		Iat:      time.Now().Unix(),
	}

	token := createTestJWT(t, privateKey, "test-key-1", "RS512", claims)

	result, err := verifyJWT(token, keys)
	if err != nil {
		t.Fatalf("verifyJWT returned error for RS512: %v", err)
	}

	if result.Username != "testuser512" {
		t.Errorf("expected username 'testuser512', got %q", result.Username)
	}
}

func TestVerifyJWT_ClientCredentials(t *testing.T) {
	// Client credentials tokens have client_id and sub, but no user_name/user_id
	privateKey := loadTestPrivateKey(t)
	publicKey := loadTestPublicKey(t)

	keys := map[string]*rsa.PublicKey{
		"test-key-1": publicKey,
	}

	claims := jwtPayload{
		Sub:      "my-service-client",
		ClientID: "my-service-client",
		Exp:      time.Now().Add(1 * time.Hour).Unix(),
		Iat:      time.Now().Unix(),
	}

	token := createTestJWT(t, privateKey, "test-key-1", "RS256", claims)

	result, err := verifyJWT(token, keys)
	if err != nil {
		t.Fatalf("verifyJWT returned error: %v", err)
	}

	// For client credentials, Username should come from client_id
	if result.Username != "my-service-client" {
		t.Errorf("expected username 'my-service-client', got %q", result.Username)
	}

	// UserID should come from sub
	if result.UserID != "my-service-client" {
		t.Errorf("expected userID 'my-service-client', got %q", result.UserID)
	}
}

func TestVerifyJWT_ExpiredToken(t *testing.T) {
	privateKey := loadTestPrivateKey(t)
	publicKey := loadTestPublicKey(t)

	keys := map[string]*rsa.PublicKey{
		"test-key-1": publicKey,
	}

	claims := jwtPayload{
		Sub:      "user-123",
		UserName: "testuser",
		UserID:   "user-123",
		Exp:      time.Now().Add(-1 * time.Hour).Unix(), // Expired 1 hour ago
		Iat:      time.Now().Add(-2 * time.Hour).Unix(),
	}

	token := createTestJWT(t, privateKey, "test-key-1", "RS256", claims)

	_, err := verifyJWT(token, keys)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}

	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("expected error to mention 'expired', got: %v", err)
	}
}

func TestVerifyJWT_InvalidSignature(t *testing.T) {
	privateKey := loadTestPrivateKey(t)
	publicKey := loadTestPublicKey(t)

	// Generate a different key pair for creating a token with wrong signature
	wrongKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate wrong key: %v", err)
	}

	keys := map[string]*rsa.PublicKey{
		"test-key-1": publicKey, // Using original public key
	}

	claims := jwtPayload{
		Sub:      "user-123",
		UserName: "testuser",
		UserID:   "user-123",
		Exp:      time.Now().Add(1 * time.Hour).Unix(),
		Iat:      time.Now().Unix(),
	}

	// Sign with wrong private key
	token := createTestJWT(t, wrongKey, "test-key-1", "RS256", claims)

	_, err = verifyJWT(token, keys)
	if err == nil {
		t.Fatal("expected error for invalid signature, got nil")
	}

	if !strings.Contains(err.Error(), "signature") {
		t.Errorf("expected error to mention 'signature', got: %v", err)
	}

	// Ensure we don't accidentally use the unused variable
	_ = privateKey
}

func TestVerifyJWT_UnknownKeyID(t *testing.T) {
	privateKey := loadTestPrivateKey(t)
	publicKey := loadTestPublicKey(t)

	keys := map[string]*rsa.PublicKey{
		"known-key": publicKey,
	}

	claims := jwtPayload{
		Sub:      "user-123",
		UserName: "testuser",
		UserID:   "user-123",
		Exp:      time.Now().Add(1 * time.Hour).Unix(),
		Iat:      time.Now().Unix(),
	}

	// Create token with unknown key ID
	token := createTestJWT(t, privateKey, "unknown-key-id", "RS256", claims)

	_, err := verifyJWT(token, keys)
	if err == nil {
		t.Fatal("expected error for unknown key ID, got nil")
	}

	if !strings.Contains(err.Error(), "unknown-key-id") || !strings.Contains(err.Error(), "key") {
		t.Errorf("expected error to mention unknown key ID, got: %v", err)
	}
}

func TestVerifyJWT_MissingKid(t *testing.T) {
	privateKey := loadTestPrivateKey(t)
	publicKey := loadTestPublicKey(t)

	keys := map[string]*rsa.PublicKey{
		"test-key-1": publicKey,
	}

	claims := jwtPayload{
		Sub:      "user-123",
		UserName: "testuser",
		UserID:   "user-123",
		Exp:      time.Now().Add(1 * time.Hour).Unix(),
		Iat:      time.Now().Unix(),
	}

	// Create token without kid header
	token := createTestJWTWithoutKid(t, privateKey, "RS256", claims)

	_, err := verifyJWT(token, keys)
	if err == nil {
		t.Fatal("expected error for missing kid, got nil")
	}

	if !strings.Contains(err.Error(), "kid") {
		t.Errorf("expected error to mention 'kid', got: %v", err)
	}
}

func TestVerifyJWT_NotYetValid(t *testing.T) {
	privateKey := loadTestPrivateKey(t)
	publicKey := loadTestPublicKey(t)

	keys := map[string]*rsa.PublicKey{
		"test-key-1": publicKey,
	}

	claims := jwtPayload{
		Sub:      "user-123",
		UserName: "testuser",
		UserID:   "user-123",
		Exp:      time.Now().Add(2 * time.Hour).Unix(),
		Nbf:      time.Now().Add(1 * time.Hour).Unix(), // Not valid for another hour
		Iat:      time.Now().Unix(),
	}

	token := createTestJWT(t, privateKey, "test-key-1", "RS256", claims)

	_, err := verifyJWT(token, keys)
	if err == nil {
		t.Fatal("expected error for not-yet-valid token, got nil")
	}

	if !strings.Contains(err.Error(), "not valid yet") && !strings.Contains(err.Error(), "nbf") {
		t.Errorf("expected error to mention 'not valid yet' or 'nbf', got: %v", err)
	}
}

func TestVerifyJWT_UnsupportedAlgorithm(t *testing.T) {
	// Create a token with HS256 algorithm (symmetric, not allowed)
	header := `{"alg":"HS256","typ":"JWT"}`
	payload := `{"sub":"user-123","exp":` + string(rune(time.Now().Add(1*time.Hour).Unix())) + `}`

	headerB64 := base64.RawURLEncoding.EncodeToString([]byte(header))
	payloadB64 := base64.RawURLEncoding.EncodeToString([]byte(payload))

	// Fake signature (doesn't matter, should be rejected before verification)
	signatureB64 := base64.RawURLEncoding.EncodeToString([]byte("fake-signature"))

	token := headerB64 + "." + payloadB64 + "." + signatureB64

	publicKey := loadTestPublicKey(t)
	keys := map[string]*rsa.PublicKey{
		"test-key-1": publicKey,
	}

	_, err := verifyJWT(token, keys)
	if err == nil {
		t.Fatal("expected error for unsupported algorithm, got nil")
	}

	if !strings.Contains(err.Error(), "algorithm") && !strings.Contains(err.Error(), "HS256") {
		t.Errorf("expected error to mention unsupported algorithm, got: %v", err)
	}
}

func TestVerifyJWT_MalformedToken(t *testing.T) {
	publicKey := loadTestPublicKey(t)
	keys := map[string]*rsa.PublicKey{
		"test-key-1": publicKey,
	}

	tests := []struct {
		name  string
		token string
	}{
		{"empty string", ""},
		{"no dots", "nodots"},
		{"one dot", "one.dot"},
		{"too many dots", "too.many.dots.here"},
		{"invalid base64 header", "!!!.payload.signature"},
		{"invalid base64 payload", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InRlc3Qta2V5LTEifQ.!!!.signature"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := verifyJWT(tt.token, keys)
			if err == nil {
				t.Error("expected error for malformed token, got nil")
			}
		})
	}
}
