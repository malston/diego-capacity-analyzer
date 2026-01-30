// ABOUTME: Generates signed JWT tokens for testing
// ABOUTME: Used by demo-jwt-auth.sh to create valid and expired tokens

package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <private-key-path> <token-type>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Token types: valid, expired, client\n")
		os.Exit(1)
	}

	keyPath := os.Args[1]
	tokenType := os.Args[2]

	// Load private key
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read key: %v\n", err)
		os.Exit(1)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		fmt.Fprintf(os.Stderr, "Failed to decode PEM\n")
		os.Exit(1)
	}

	var privateKey *rsa.PrivateKey
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse key: %v\n", err)
			os.Exit(1)
		}
	} else {
		privateKey = key.(*rsa.PrivateKey)
	}

	// Create header
	header := map[string]string{
		"alg": "RS256",
		"typ": "JWT",
		"kid": "demo-key-1",
	}

	// Create claims based on token type
	var claims map[string]interface{}
	switch tokenType {
	case "valid":
		claims = map[string]interface{}{
			"sub":       "demo-user-123",
			"user_name": "demo-user",
			"user_id":   "demo-user-123",
			"exp":       time.Now().Add(time.Hour).Unix(),
			"iat":       time.Now().Unix(),
			"iss":       "http://localhost:19999",
		}
	case "expired":
		claims = map[string]interface{}{
			"sub":       "expired-user",
			"user_name": "expired-user",
			"user_id":   "expired-user-123",
			"exp":       time.Now().Add(-time.Hour).Unix(),
			"iat":       time.Now().Add(-2 * time.Hour).Unix(),
			"iss":       "http://localhost:19999",
		}
	case "client":
		claims = map[string]interface{}{
			"sub":       "automation-client",
			"client_id": "automation-client",
			"exp":       time.Now().Add(time.Hour).Unix(),
			"iat":       time.Now().Unix(),
			"iss":       "http://localhost:19999",
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown token type: %s\n", tokenType)
		os.Exit(1)
	}

	// Encode header and claims
	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	// Sign
	signingInput := headerB64 + "." + claimsB64
	h := sha256.New()
	h.Write([]byte(signingInput))
	hashed := h.Sum(nil)

	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to sign: %v\n", err)
		os.Exit(1)
	}

	sigB64 := base64.RawURLEncoding.EncodeToString(signature)
	fmt.Print(signingInput + "." + sigB64)
}
