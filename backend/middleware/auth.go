// ABOUTME: JWT authentication middleware for CF UAA tokens
// ABOUTME: Validates token structure and expiration, extracts user claims

package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/services"
)

// AuthMode defines how authentication is enforced
type AuthMode string

const (
	// AuthModeDisabled skips all authentication
	AuthModeDisabled AuthMode = "disabled"
	// AuthModeOptional validates tokens if present, allows anonymous
	AuthModeOptional AuthMode = "optional"
	// AuthModeRequired rejects requests without valid tokens
	AuthModeRequired AuthMode = "required"
)

// SessionValidatorFunc validates a session ID and returns user claims if valid
type SessionValidatorFunc func(sessionID string) *UserClaims

// AuthConfig holds authentication middleware settings
type AuthConfig struct {
	Mode             AuthMode
	SessionValidator SessionValidatorFunc // Optional: validates session cookies
	JWKSClient       *services.JWKSClient // Optional: validates Bearer token signatures
}

// ValidateAuthMode validates an auth mode string and returns the corresponding AuthMode.
// Empty string defaults to AuthModeOptional.
// Returns error for invalid mode values.
func ValidateAuthMode(mode string) (AuthMode, error) {
	switch mode {
	case "", "optional":
		return AuthModeOptional, nil
	case "disabled":
		return AuthModeDisabled, nil
	case "required":
		return AuthModeRequired, nil
	default:
		return "", fmt.Errorf("invalid auth mode: %q (must be disabled, optional, or required)", mode)
	}
}

// UserClaims contains extracted JWT claims
type UserClaims struct {
	Username string
	UserID   string
}

// contextKey is a private type for context keys to avoid collisions
type contextKey string

const userClaimsKey contextKey = "userClaims"

// Auth returns middleware that validates JWT tokens and/or session cookies.
// The middleware behavior depends on the configured mode:
//   - disabled: passes all requests through
//   - optional: validates auth if present, allows anonymous
//   - required: rejects requests without valid auth
//
// Authentication methods (checked in order):
//  1. Bearer token in Authorization header (takes precedence)
//  2. Session cookie (if SessionValidator is configured)
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

				// If JWKSClient is not configured, Bearer auth is unavailable
				if cfg.JWKSClient == nil {
					slog.Debug("Auth rejected: JWKSClient not configured", "path", r.URL.Path)
					http.Error(w, "Bearer authentication unavailable, please use web UI login", http.StatusUnauthorized)
					return
				}

				// Use JWKS client for cryptographic signature verification
				jwtClaims, err := cfg.JWKSClient.VerifyAndParse(token)
				if err != nil {
					slog.Debug("Auth rejected: invalid token", "path", r.URL.Path, "error", err.Error())
					http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
					return
				}

				// Convert services.JWTClaims to middleware.UserClaims
				claims := &UserClaims{
					Username: jwtClaims.Username,
					UserID:   jwtClaims.UserID,
				}

				slog.Debug("Auth: valid bearer token", "path", r.URL.Path, "user", claims.Username)
				ctx := context.WithValue(r.Context(), userClaimsKey, claims)
				next(w, r.WithContext(ctx))
				return
			}

			// Check session cookie second (if validator configured)
			if cfg.SessionValidator != nil {
				cookie, err := r.Cookie("DIEGO_SESSION")
				if err == nil && cookie.Value != "" {
					claims := cfg.SessionValidator(cookie.Value)
					if claims != nil {
						slog.Debug("Auth: valid session cookie", "path", r.URL.Path, "user", claims.Username)
						ctx := context.WithValue(r.Context(), userClaimsKey, claims)
						next(w, r.WithContext(ctx))
						return
					}
					// Session cookie present but invalid
					slog.Debug("Auth rejected: invalid session", "path", r.URL.Path)
					http.Error(w, "Invalid session", http.StatusUnauthorized)
					return
				}
			}

			// No auth provided
			if cfg.Mode == AuthModeRequired {
				slog.Debug("Auth rejected: no auth provided", "path", r.URL.Path, "mode", cfg.Mode)
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Optional mode with no auth: pass through
			slog.Debug("Auth: anonymous request allowed", "path", r.URL.Path, "mode", cfg.Mode)
			next(w, r)
		}
	}
}

// GetUserClaims extracts user claims from request context.
// Returns nil if no claims are present.
func GetUserClaims(r *http.Request) *UserClaims {
	claims, ok := r.Context().Value(userClaimsKey).(*UserClaims)
	if !ok {
		return nil
	}
	return claims
}

// parseJWT extracts claims from a JWT token.
// Note: This implementation validates structure and expiration but does not
// cryptographically verify the signature. For demo purposes, we trust tokens
// issued by CF UAA.
func parseJWT(token string) (*UserClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, &jwtError{"malformed token structure"}
	}

	payload, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, &jwtError{"invalid payload encoding"}
	}

	var claims jwtClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, &jwtError{"invalid payload format"}
	}

	// Check expiration
	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return nil, &jwtError{"token expired"}
	}

	// Validate required claims
	if claims.UserName == "" {
		return nil, &jwtError{"missing required claim: user_name"}
	}

	return &UserClaims{
		Username: claims.UserName,
		UserID:   claims.UserID,
	}, nil
}

// jwtClaims represents CF UAA JWT payload fields
type jwtClaims struct {
	UserName string `json:"user_name"`
	UserID   string `json:"user_id"`
	Exp      int64  `json:"exp"`
}

// jwtError represents a JWT validation error
type jwtError struct {
	msg string
}

func (e *jwtError) Error() string {
	return e.msg
}

// base64URLDecode decodes base64url encoded data (RFC 4648)
func base64URLDecode(s string) ([]byte, error) {
	// RawURLEncoding handles base64url without padding
	// Add padding if present in input (some JWTs include it)
	data, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		// Try with standard URL encoding (with padding) as fallback
		data, err = base64.URLEncoding.DecodeString(s)
		if err != nil {
			return nil, &jwtError{"invalid payload encoding"}
		}
	}
	return data, nil
}
