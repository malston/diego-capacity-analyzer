// ABOUTME: JWKS (JSON Web Key Set) parsing and JWT verification for CF UAA
// ABOUTME: Converts UAA's JWKS endpoint response into RSA public keys and verifies JWT signatures

package services

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// jwksResponse represents the JSON structure returned by the UAA JWKS endpoint
type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

// jwkKey represents a single JSON Web Key in the JWKS response
type jwkKey struct {
	Kty string `json:"kty"` // Key type (must be "RSA" for our use)
	Kid string `json:"kid"` // Key ID
	N   string `json:"n"`   // RSA modulus (base64url encoded)
	E   string `json:"e"`   // RSA exponent (base64url encoded)
	Alg string `json:"alg"` // Algorithm (e.g., "RS256")
	Use string `json:"use"` // Key use (e.g., "sig" for signature)
}

// ErrUnknownKeyID indicates a JWT references a key ID not present in the JWKS key set.
var ErrUnknownKeyID = errors.New("unknown key ID")

// parseJWKS parses a JWKS JSON response and returns a map of key ID to RSA public key.
// Non-RSA keys are silently skipped.
func parseJWKS(data []byte) (map[string]*rsa.PublicKey, error) {
	var response jwksResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse JWKS JSON: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey)
	for _, jwk := range response.Keys {
		// Skip non-RSA keys
		if jwk.Kty != "RSA" {
			continue
		}

		pubKey, err := parseRSAPublicKey(jwk.N, jwk.E)
		if err != nil {
			return nil, fmt.Errorf("failed to parse RSA key %s: %w", jwk.Kid, err)
		}

		keys[jwk.Kid] = pubKey
	}

	return keys, nil
}

// parseRSAPublicKey decodes base64url-encoded modulus and exponent into an RSA public key
func parseRSAPublicKey(nB64, eB64 string) (*rsa.PublicKey, error) {
	// Decode the modulus (n)
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	// Decode the exponent (e)
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert exponent bytes to int
	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}

	// Create the RSA public key
	n := new(big.Int).SetBytes(nBytes)

	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}

// JWTClaims contains the extracted claims from a verified JWT
type JWTClaims struct {
	Username string
	UserID   string
	Scopes   []string
}

// jwtHeaderForVerification represents the header portion of a JWT for parsing
type jwtHeaderForVerification struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	Typ string `json:"typ"`
}

// jwtClaimsForVerification represents the claims portion of a JWT for parsing
type jwtClaimsForVerification struct {
	Sub      string   `json:"sub"`
	UserName string   `json:"user_name"`
	UserID   string   `json:"user_id"`
	ClientID string   `json:"client_id"`
	Exp      int64    `json:"exp"`
	Nbf      int64    `json:"nbf"`
	Iat      int64    `json:"iat"`
	Scope    []string `json:"scope"`
}

// supportedAlgorithms defines the only allowed signing algorithms (RS256/RS384/RS512)
// This prevents algorithm confusion attacks where an attacker might try to use
// symmetric algorithms (HS256) or "none"
var supportedAlgorithms = map[string]struct {
	hash       func() hash.Hash
	cryptoHash crypto.Hash
}{
	"RS256": {sha256.New, crypto.SHA256},
	"RS384": {sha512.New384, crypto.SHA384},
	"RS512": {sha512.New, crypto.SHA512},
}

// verifyJWT verifies a JWT signature and extracts claims.
// Security: Signature is verified BEFORE checking expiration to prevent timing attacks.
// Only RS256/RS384/RS512 algorithms are allowed to prevent algorithm confusion attacks.
func verifyJWT(token string, keys map[string]*rsa.PublicKey) (*JWTClaims, error) {
	// Split token into parts
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed JWT: expected 3 parts, got %d", len(parts))
	}

	headerB64, payloadB64, signatureB64 := parts[0], parts[1], parts[2]

	// Decode and parse header
	headerJSON, err := base64.RawURLEncoding.DecodeString(headerB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT header: %w", err)
	}

	var header jwtHeaderForVerification
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, fmt.Errorf("failed to parse JWT header: %w", err)
	}

	// Validate algorithm (prevent algorithm confusion attacks)
	algInfo, ok := supportedAlgorithms[header.Alg]
	if !ok {
		return nil, fmt.Errorf("unsupported algorithm %q: only RS256, RS384, RS512 are allowed", header.Alg)
	}

	// Validate kid is present
	if header.Kid == "" {
		return nil, fmt.Errorf("JWT missing required kid header")
	}

	// Look up public key by kid
	publicKey, ok := keys[header.Kid]
	if !ok {
		return nil, fmt.Errorf("%w: %q not found in JWKS", ErrUnknownKeyID, header.Kid)
	}

	// Decode signature
	signature, err := base64.RawURLEncoding.DecodeString(signatureB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT signature: %w", err)
	}

	// SECURITY: Verify signature BEFORE checking expiration (prevents timing attacks)
	signingInput := headerB64 + "." + payloadB64
	hashFunc := algInfo.hash()
	hashFunc.Write([]byte(signingInput))
	hashed := hashFunc.Sum(nil)

	if err := rsa.VerifyPKCS1v15(publicKey, algInfo.cryptoHash, hashed, signature); err != nil {
		return nil, fmt.Errorf("invalid JWT signature: %w", err)
	}

	// Decode and parse payload (only after signature is verified)
	payloadJSON, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var claims jwtClaimsForVerification
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse JWT claims: %w", err)
	}

	now := time.Now().Unix()

	// Check nbf (not before) - RFC 7519 compliance
	if claims.Nbf > 0 && now < claims.Nbf {
		return nil, fmt.Errorf("token not valid yet (nbf: %d, now: %d)", claims.Nbf, now)
	}

	// Check expiration (RFC 7519: token is valid AT the exp second, expires after)
	if claims.Exp > 0 && now > claims.Exp {
		return nil, fmt.Errorf("token expired (exp: %d, now: %d)", claims.Exp, now)
	}

	// Extract username and userID
	// User tokens have user_name and user_id
	// Client credentials tokens have client_id and sub
	username := claims.UserName
	if username == "" {
		username = claims.ClientID
	}

	userID := claims.UserID
	if userID == "" {
		userID = claims.Sub
	}

	// Reject tokens without identity information
	if username == "" && userID == "" {
		return nil, fmt.Errorf("token missing required identity claims (user_name/client_id or user_id/sub)")
	}

	return &JWTClaims{
		Username: username,
		UserID:   userID,
		Scopes:   claims.Scope,
	}, nil
}

// JWKSClient fetches and caches JWKS keys from a UAA server.
// Uses singleflight to prevent thundering herd when refreshing keys.
type JWKSClient struct {
	uaaURL     string
	httpClient *http.Client
	keys       map[string]*rsa.PublicKey
	mu         sync.RWMutex
	sfGroup    singleflight.Group
}

// NewJWKSClient creates a new JWKS client and fetches initial keys.
// If httpClient is nil, a default client with 30s timeout is used.
// Returns an error if the initial key fetch fails.
func NewJWKSClient(uaaURL string, httpClient *http.Client) (*JWKSClient, error) {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
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

// GetKey returns the RSA public key for the given key ID.
// If the key is not found, it triggers a refresh and tries again.
// Returns nil if the key is still not found after refresh.
func (c *JWKSClient) GetKey(kid string) *rsa.PublicKey {
	// First check with read lock
	c.mu.RLock()
	key, ok := c.keys[kid]
	c.mu.RUnlock()

	if ok {
		return key
	}

	// Key not found, try refreshing (with singleflight to prevent thundering herd)
	_, _, _ = c.sfGroup.Do("refresh", func() (interface{}, error) {
		return nil, c.refresh()
	})

	// Check again after refresh
	c.mu.RLock()
	key = c.keys[kid]
	c.mu.RUnlock()

	return key
}

// refresh fetches the JWKS from the UAA server and updates the keys map.
func (c *JWKSClient) refresh() error {
	url := c.uaaURL + "/token_keys"

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS from %s: %w", url, err)
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
		return fmt.Errorf("failed to parse JWKS: %w", err)
	}

	c.mu.Lock()
	c.keys = keys
	c.mu.Unlock()

	return nil
}

// ClearKeysForTesting clears all cached keys. This is only for testing purposes
// to force a refresh on the next verification attempt.
func (c *JWKSClient) ClearKeysForTesting() {
	c.mu.Lock()
	c.keys = make(map[string]*rsa.PublicKey)
	c.mu.Unlock()
}

// SetKeysForTesting sets the cached keys directly. This is only for testing purposes
// to avoid requiring a mock HTTP server for every test.
func (c *JWKSClient) SetKeysForTesting(keys map[string]*rsa.PublicKey) {
	c.mu.Lock()
	c.keys = keys
	c.mu.Unlock()
}

// VerifyAndParse verifies a JWT signature and extracts claims.
// If the key ID is unknown, it refreshes the keys and retries once.
func (c *JWKSClient) VerifyAndParse(token string) (*JWTClaims, error) {
	// Verify with read lock held to avoid map copy overhead
	c.mu.RLock()
	claims, err := verifyJWT(token, c.keys)
	c.mu.RUnlock()

	if err != nil {
		if errors.Is(err, ErrUnknownKeyID) {
			// Refresh keys using singleflight
			_, _, _ = c.sfGroup.Do("refresh", func() (interface{}, error) {
				return nil, c.refresh()
			})

			// Retry with fresh keys (read lock held during verification)
			c.mu.RLock()
			claims, err = verifyJWT(token, c.keys)
			c.mu.RUnlock()

			return claims, err
		}
		return nil, err
	}

	return claims, nil
}
