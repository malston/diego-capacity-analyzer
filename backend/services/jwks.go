// ABOUTME: JWKS (JSON Web Key Set) parsing for CF UAA public keys
// ABOUTME: Converts UAA's JWKS endpoint response into RSA public keys for JWT verification

package services

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
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
