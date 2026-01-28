// ABOUTME: Tests for JWT authentication middleware
// ABOUTME: Verifies token validation, expiration, and claims extraction

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAuth_RequiredMode_NoHeader_Returns401(t *testing.T) {
	cfg := AuthConfig{Mode: AuthModeRequired}
	handler := Auth(cfg)(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called without auth header in required mode")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuth_OptionalMode_NoHeader_PassesThrough(t *testing.T) {
	cfg := AuthConfig{Mode: AuthModeOptional}
	handlerCalled := false
	handler := Auth(cfg)(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if !handlerCalled {
		t.Error("Handler should be called in optional mode without auth header")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuth_DisabledMode_NoHeader_PassesThrough(t *testing.T) {
	cfg := AuthConfig{Mode: AuthModeDisabled}
	handlerCalled := false
	handler := Auth(cfg)(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if !handlerCalled {
		t.Error("Handler should be called in disabled mode")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuth_ValidToken_ExtractsClaims(t *testing.T) {
	cfg := AuthConfig{Mode: AuthModeRequired}
	var extractedClaims *UserClaims
	handler := Auth(cfg)(func(w http.ResponseWriter, r *http.Request) {
		extractedClaims = GetUserClaims(r)
		w.WriteHeader(http.StatusOK)
	})

	// Create a valid JWT token (not expired)
	token := createTestToken(t, "test-user", "test-user-id", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
	if extractedClaims == nil {
		t.Fatal("Expected claims to be extracted")
	}
	if extractedClaims.Username != "test-user" {
		t.Errorf("Username = %q, want %q", extractedClaims.Username, "test-user")
	}
	if extractedClaims.UserID != "test-user-id" {
		t.Errorf("UserID = %q, want %q", extractedClaims.UserID, "test-user-id")
	}
}

func TestAuth_ExpiredToken_Returns401(t *testing.T) {
	cfg := AuthConfig{Mode: AuthModeRequired}
	handler := Auth(cfg)(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with expired token")
	})

	// Create an expired JWT token
	token := createTestToken(t, "test-user", "test-user-id", time.Now().Add(-time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuth_MalformedToken_Returns401(t *testing.T) {
	cfg := AuthConfig{Mode: AuthModeRequired}
	handler := Auth(cfg)(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with malformed token")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer not.a.valid.jwt.token")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuth_InvalidBearerFormat_Returns401(t *testing.T) {
	cfg := AuthConfig{Mode: AuthModeRequired}
	handler := Auth(cfg)(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with invalid bearer format")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Basic sometoken") // Wrong auth type
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuth_OptionalMode_ValidToken_ExtractsClaims(t *testing.T) {
	cfg := AuthConfig{Mode: AuthModeOptional}
	var extractedClaims *UserClaims
	handler := Auth(cfg)(func(w http.ResponseWriter, r *http.Request) {
		extractedClaims = GetUserClaims(r)
		w.WriteHeader(http.StatusOK)
	})

	token := createTestToken(t, "optional-user", "optional-id", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
	if extractedClaims == nil {
		t.Fatal("Expected claims to be extracted in optional mode with valid token")
	}
	if extractedClaims.Username != "optional-user" {
		t.Errorf("Username = %q, want %q", extractedClaims.Username, "optional-user")
	}
}

func TestAuth_OptionalMode_InvalidToken_Returns401(t *testing.T) {
	cfg := AuthConfig{Mode: AuthModeOptional}
	handler := Auth(cfg)(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with invalid token even in optional mode")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.token")
	rec := httptest.NewRecorder()
	handler(rec, req)

	// In optional mode, if a token IS provided but invalid, should reject
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestGetUserClaims_NoClaimsInContext_ReturnsNil(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	claims := GetUserClaims(req)
	if claims != nil {
		t.Errorf("Expected nil claims for request without context, got %+v", claims)
	}
}

func TestGetUserClaims_WithClaimsInContext_ReturnsClaims(t *testing.T) {
	expectedClaims := &UserClaims{
		Username: "ctx-user",
		UserID:   "ctx-id",
	}
	ctx := context.WithValue(context.Background(), userClaimsKey, expectedClaims)
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)

	claims := GetUserClaims(req)
	if claims == nil {
		t.Fatal("Expected claims from context")
	}
	if claims.Username != expectedClaims.Username {
		t.Errorf("Username = %q, want %q", claims.Username, expectedClaims.Username)
	}
}

func TestValidateAuthMode_ValidModes(t *testing.T) {
	tests := []struct {
		mode string
		want AuthMode
	}{
		{"disabled", AuthModeDisabled},
		{"optional", AuthModeOptional},
		{"required", AuthModeRequired},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			got, err := ValidateAuthMode(tt.mode)
			if err != nil {
				t.Errorf("ValidateAuthMode(%q) error = %v", tt.mode, err)
			}
			if got != tt.want {
				t.Errorf("ValidateAuthMode(%q) = %v, want %v", tt.mode, got, tt.want)
			}
		})
	}
}

func TestValidateAuthMode_InvalidMode(t *testing.T) {
	_, err := ValidateAuthMode("invalid")
	if err == nil {
		t.Error("ValidateAuthMode(\"invalid\") should return error")
	}
}

func TestValidateAuthMode_EmptyMode(t *testing.T) {
	// Empty string should default to optional
	got, err := ValidateAuthMode("")
	if err != nil {
		t.Errorf("ValidateAuthMode(\"\") error = %v", err)
	}
	if got != AuthModeOptional {
		t.Errorf("ValidateAuthMode(\"\") = %v, want %v", got, AuthModeOptional)
	}
}

func TestAuth_TokenWithEmptyUsername_Returns401(t *testing.T) {
	cfg := AuthConfig{Mode: AuthModeRequired}
	handler := Auth(cfg)(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with empty username")
	})

	// Create token with empty username
	token := createTestToken(t, "", "user-id", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// Tests for session cookie authentication

func TestAuthWithSession_ValidCookie_ExtractsClaims(t *testing.T) {
	// Mock session validator that returns claims for valid session ID
	sessionValidator := func(sessionID string) *UserClaims {
		if sessionID == "valid-session-123" {
			return &UserClaims{Username: "session-user", UserID: "session-user-id"}
		}
		return nil
	}

	cfg := AuthConfig{
		Mode:             AuthModeRequired,
		SessionValidator: sessionValidator,
	}

	var extractedClaims *UserClaims
	handler := Auth(cfg)(func(w http.ResponseWriter, r *http.Request) {
		extractedClaims = GetUserClaims(r)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "valid-session-123"})
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
	if extractedClaims == nil {
		t.Fatal("Expected claims to be extracted from session")
	}
	if extractedClaims.Username != "session-user" {
		t.Errorf("Username = %q, want %q", extractedClaims.Username, "session-user")
	}
	if extractedClaims.UserID != "session-user-id" {
		t.Errorf("UserID = %q, want %q", extractedClaims.UserID, "session-user-id")
	}
}

func TestAuthWithSession_InvalidCookie_Returns401(t *testing.T) {
	sessionValidator := func(sessionID string) *UserClaims {
		return nil // All sessions invalid
	}

	cfg := AuthConfig{
		Mode:             AuthModeRequired,
		SessionValidator: sessionValidator,
	}

	handler := Auth(cfg)(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with invalid session")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "invalid-session"})
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuthWithSession_BearerTakesPrecedence(t *testing.T) {
	sessionValidator := func(sessionID string) *UserClaims {
		return &UserClaims{Username: "session-user", UserID: "session-id"}
	}

	cfg := AuthConfig{
		Mode:             AuthModeRequired,
		SessionValidator: sessionValidator,
	}

	var extractedClaims *UserClaims
	handler := Auth(cfg)(func(w http.ResponseWriter, r *http.Request) {
		extractedClaims = GetUserClaims(r)
		w.WriteHeader(http.StatusOK)
	})

	// Valid JWT token
	token := createTestToken(t, "bearer-user", "bearer-id", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "valid-session"})
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
	// Bearer token should take precedence over session cookie
	if extractedClaims == nil {
		t.Fatal("Expected claims")
	}
	if extractedClaims.Username != "bearer-user" {
		t.Errorf("Username = %q, want %q (Bearer should take precedence)", extractedClaims.Username, "bearer-user")
	}
}

func TestAuthWithSession_OptionalMode_NoCookie_PassesThrough(t *testing.T) {
	sessionValidator := func(sessionID string) *UserClaims {
		return nil
	}

	cfg := AuthConfig{
		Mode:             AuthModeOptional,
		SessionValidator: sessionValidator,
	}

	handlerCalled := false
	handler := Auth(cfg)(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if !handlerCalled {
		t.Error("Handler should be called in optional mode without auth")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthWithSession_NoValidator_FallsBackToToken(t *testing.T) {
	// If SessionValidator is nil, session cookies should be ignored
	cfg := AuthConfig{
		Mode:             AuthModeRequired,
		SessionValidator: nil, // No session support
	}

	handler := Auth(cfg)(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called without valid auth")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "some-session"})
	rec := httptest.NewRecorder()
	handler(rec, req)

	// Should reject because no Bearer token and no session validator configured
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// createTestToken creates a simple JWT token for testing.
// Note: This creates a real JWT structure but doesn't sign with a real key.
// For this demo-level implementation, we only check structure and expiration.
func createTestToken(t *testing.T, username, userID string, exp time.Time) string {
	t.Helper()
	// JWT format: header.payload.signature (all base64url encoded)
	// We create a minimal valid JWT structure

	// Header: {"alg":"none","typ":"JWT"}
	header := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0"

	// Payload with exp, user_name, and user_id claims
	// We'll use a helper to create the payload
	payload := encodeJWTPayload(username, userID, exp.Unix())

	// No signature for testing (alg: none)
	return header + "." + payload + "."
}

// encodeJWTPayload creates a base64url encoded JWT payload
func encodeJWTPayload(username, userID string, exp int64) string {
	// Create JSON payload manually to avoid import cycles
	// {"user_name":"xxx","user_id":"xxx","exp":123}
	payload := `{"user_name":"` + username + `","user_id":"` + userID + `","exp":` + formatInt64(exp) + `}`
	return base64URLEncode([]byte(payload))
}

func formatInt64(n int64) string {
	// Simple int64 to string conversion without strconv to keep test self-contained
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf) - 1
	for n > 0 {
		buf[i] = byte(n%10) + '0'
		n /= 10
		i--
	}
	if neg {
		buf[i] = '-'
		i--
	}
	return string(buf[i+1:])
}

func base64URLEncode(data []byte) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	result := make([]byte, 0, (len(data)*4+2)/3)
	for i := 0; i < len(data); i += 3 {
		var n uint32
		remaining := len(data) - i
		switch remaining {
		case 1:
			n = uint32(data[i]) << 16
			result = append(result, alphabet[n>>18], alphabet[(n>>12)&0x3f])
		case 2:
			n = uint32(data[i])<<16 | uint32(data[i+1])<<8
			result = append(result, alphabet[n>>18], alphabet[(n>>12)&0x3f], alphabet[(n>>6)&0x3f])
		default:
			n = uint32(data[i])<<16 | uint32(data[i+1])<<8 | uint32(data[i+2])
			result = append(result, alphabet[n>>18], alphabet[(n>>12)&0x3f], alphabet[(n>>6)&0x3f], alphabet[n&0x3f])
		}
	}
	return string(result)
}
