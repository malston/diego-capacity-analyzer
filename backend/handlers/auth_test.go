// ABOUTME: Tests for auth handlers implementing BFF OAuth pattern
// ABOUTME: Verifies login, logout, session management, and cookie security

package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/cache"
	"github.com/markalston/diego-capacity-analyzer/backend/config"
	"github.com/markalston/diego-capacity-analyzer/backend/models"
	"github.com/markalston/diego-capacity-analyzer/backend/services"
)

// setupMockUAAServerWithRefresh creates a mock UAA server that handles both password and refresh_token grants
func setupMockUAAServerWithRefresh(validUser, validPass, validRefreshToken string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			if err := r.ParseForm(); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			grantType := r.FormValue("grant_type")

			// Handle refresh_token grant
			if grantType == "refresh_token" {
				refreshToken := r.FormValue("refresh_token")
				if validRefreshToken != "" && refreshToken == validRefreshToken {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"access_token":  "new-access-token-refreshed",
						"refresh_token": "new-refresh-token-refreshed",
						"token_type":    "bearer",
						"expires_in":    3600,
					})
					return
				}
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":             "invalid_grant",
					"error_description": "Invalid refresh token",
				})
				return
			}

			// Handle password grant
			username := r.FormValue("username")
			password := r.FormValue("password")

			if username == validUser && password == validPass {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"access_token":  "test-access-token-xyz",
					"refresh_token": "test-refresh-token-xyz",
					"token_type":    "bearer",
					"expires_in":    3600,
				})
				return
			}

			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":             "unauthorized",
				"error_description": "Bad credentials",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

// setupMockCFAndUAAServers creates mock CF API and UAA servers
func setupMockCFAndUAAServers(validUser, validPass string) (*httptest.Server, *httptest.Server) {
	return setupMockCFAndUAAServersWithRefresh(validUser, validPass, "")
}

// setupMockCFAndUAAServersWithRefresh creates mock CF API and UAA servers with refresh token support
func setupMockCFAndUAAServersWithRefresh(validUser, validPass, validRefreshToken string) (*httptest.Server, *httptest.Server) {
	uaaServer := setupMockUAAServerWithRefresh(validUser, validPass, validRefreshToken)

	cfServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v3/info" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"links": map[string]interface{}{
					"login": map[string]interface{}{
						"href": uaaServer.URL,
					},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))

	return cfServer, uaaServer
}

func TestLogin_Success(t *testing.T) {
	cfServer, uaaServer := setupMockCFAndUAAServers("admin", "secret")
	defer cfServer.Close()
	defer uaaServer.Close()

	c := cache.New(5 * time.Minute)
	sessionSvc := services.NewSessionService(c)
	cfg := &config.Config{
		CFAPIUrl:     cfServer.URL,
		CookieSecure: false, // false for test (http)
	}

	h := NewHandler(cfg, c)
	h.SetSessionService(sessionSvc)

	body := `{"username":"admin","password":"secret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Login(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Check response body
	var loginResp models.LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !loginResp.Success {
		t.Error("Expected Success to be true")
	}
	if loginResp.Username != "admin" {
		t.Errorf("Username = %q, want %q", loginResp.Username, "admin")
	}

	// Check that cookie is set
	cookies := resp.Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "DIEGO_SESSION" {
			sessionCookie = c
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("Expected DIEGO_SESSION cookie to be set")
	}

	// Verify cookie security attributes
	if !sessionCookie.HttpOnly {
		t.Error("Cookie should be HttpOnly")
	}
	if sessionCookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("Cookie SameSite = %v, want Strict", sessionCookie.SameSite)
	}
	if sessionCookie.Value == "" {
		t.Error("Cookie value should not be empty")
	}

	// Response should NOT contain tokens
	body2, _ := json.Marshal(loginResp)
	if strings.Contains(string(body2), "access_token") {
		t.Error("Response should NOT contain access_token")
	}
	if strings.Contains(string(body2), "refresh_token") {
		t.Error("Response should NOT contain refresh_token")
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	cfServer, uaaServer := setupMockCFAndUAAServers("admin", "secret")
	defer cfServer.Close()
	defer uaaServer.Close()

	c := cache.New(5 * time.Minute)
	sessionSvc := services.NewSessionService(c)
	cfg := &config.Config{
		CFAPIUrl:     cfServer.URL,
		CookieSecure: false,
	}

	h := NewHandler(cfg, c)
	h.SetSessionService(sessionSvc)

	body := `{"username":"admin","password":"wrongpassword"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Login(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	var loginResp models.LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if loginResp.Success {
		t.Error("Expected Success to be false")
	}
	if loginResp.Error == "" {
		t.Error("Expected Error to be set")
	}
}

func TestLogin_MissingCredentials(t *testing.T) {
	c := cache.New(5 * time.Minute)
	sessionSvc := services.NewSessionService(c)
	cfg := &config.Config{CookieSecure: false}

	h := NewHandler(cfg, c)
	h.SetSessionService(sessionSvc)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Login(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestMe_Authenticated(t *testing.T) {
	c := cache.New(5 * time.Minute)
	sessionSvc := services.NewSessionService(c)

	// Create a session
	sessionID, err := sessionSvc.Create("testuser", "user-123", "access", "refresh", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	cfg := &config.Config{CookieSecure: false}
	h := NewHandler(cfg, c)
	h.SetSessionService(sessionSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: sessionID})
	w := httptest.NewRecorder()

	h.Me(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var userInfo models.UserInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !userInfo.Authenticated {
		t.Error("Expected Authenticated to be true")
	}
	if userInfo.Username != "testuser" {
		t.Errorf("Username = %q, want %q", userInfo.Username, "testuser")
	}
	if userInfo.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", userInfo.UserID, "user-123")
	}
}

func TestMe_NotAuthenticated(t *testing.T) {
	c := cache.New(5 * time.Minute)
	sessionSvc := services.NewSessionService(c)
	cfg := &config.Config{CookieSecure: false}

	h := NewHandler(cfg, c)
	h.SetSessionService(sessionSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()

	h.Me(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var userInfo models.UserInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if userInfo.Authenticated {
		t.Error("Expected Authenticated to be false")
	}
}

func TestMe_InvalidSession(t *testing.T) {
	c := cache.New(5 * time.Minute)
	sessionSvc := services.NewSessionService(c)
	cfg := &config.Config{CookieSecure: false}

	h := NewHandler(cfg, c)
	h.SetSessionService(sessionSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: "invalid-session-id"})
	w := httptest.NewRecorder()

	h.Me(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var userInfo models.UserInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if userInfo.Authenticated {
		t.Error("Expected Authenticated to be false for invalid session")
	}
}

func TestLogout_Success(t *testing.T) {
	c := cache.New(5 * time.Minute)
	sessionSvc := services.NewSessionService(c)

	// Create a session
	sessionID, err := sessionSvc.Create("testuser", "user-123", "access", "refresh", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	cfg := &config.Config{CookieSecure: false}
	h := NewHandler(cfg, c)
	h.SetSessionService(sessionSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: sessionID})
	w := httptest.NewRecorder()

	h.Logout(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Check that cookie is cleared
	cookies := resp.Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "DIEGO_SESSION" {
			sessionCookie = c
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("Expected DIEGO_SESSION cookie to be set (for clearing)")
	}
	if sessionCookie.MaxAge != -1 {
		t.Errorf("Cookie MaxAge = %d, want -1 (expired)", sessionCookie.MaxAge)
	}

	// Verify session is deleted from cache
	_, err = sessionSvc.Get(sessionID)
	if err == nil {
		t.Error("Session should be deleted after logout")
	}
}

func TestLogout_NoSession(t *testing.T) {
	c := cache.New(5 * time.Minute)
	sessionSvc := services.NewSessionService(c)
	cfg := &config.Config{CookieSecure: false}

	h := NewHandler(cfg, c)
	h.SetSessionService(sessionSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	w := httptest.NewRecorder()

	h.Logout(w, req)

	// Logout should succeed even without a session
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestLogin_CookieSecureFlag(t *testing.T) {
	cfServer, uaaServer := setupMockCFAndUAAServers("admin", "secret")
	defer cfServer.Close()
	defer uaaServer.Close()

	c := cache.New(5 * time.Minute)
	sessionSvc := services.NewSessionService(c)
	cfg := &config.Config{
		CFAPIUrl:     cfServer.URL,
		CookieSecure: true, // Production setting
	}

	h := NewHandler(cfg, c)
	h.SetSessionService(sessionSvc)

	body := `{"username":"admin","password":"secret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Login(w, req)

	resp := w.Result()
	cookies := resp.Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "DIEGO_SESSION" {
			sessionCookie = c
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("Expected DIEGO_SESSION cookie")
	}

	if !sessionCookie.Secure {
		t.Error("Cookie should be Secure when CookieSecure=true")
	}
}

func TestRefresh_TokensUpdated(t *testing.T) {
	// Set up mock servers that accept our known refresh token
	knownRefreshToken := "known-refresh-token-abc123"
	cfServer, uaaServer := setupMockCFAndUAAServersWithRefresh("admin", "secret", knownRefreshToken)
	defer cfServer.Close()
	defer uaaServer.Close()

	c := cache.New(5 * time.Minute)
	sessionSvc := services.NewSessionService(c)

	// Create a session with token expiring within 5 minutes (triggers refresh)
	sessionID, err := sessionSvc.Create(
		"testuser",
		"user-123",
		"old-access-token",
		knownRefreshToken,             // Must match what mock UAA expects
		time.Now().Add(2*time.Minute), // Expires in 2 min, within 5-min threshold
	)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	cfg := &config.Config{
		CFAPIUrl:     cfServer.URL,
		CookieSecure: false,
	}
	h := NewHandler(cfg, c)
	h.SetSessionService(sessionSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: sessionID})
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Verify response indicates refresh happened
	var refreshResp map[string]bool
	if err := json.NewDecoder(resp.Body).Decode(&refreshResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !refreshResp["refreshed"] {
		t.Error("Expected refreshed=true, got false")
	}

	// Verify session tokens were updated
	session, err := sessionSvc.Get(sessionID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if session.AccessToken != "new-access-token-refreshed" {
		t.Errorf("AccessToken = %q, want %q", session.AccessToken, "new-access-token-refreshed")
	}
	if session.RefreshToken != "new-refresh-token-refreshed" {
		t.Errorf("RefreshToken = %q, want %q", session.RefreshToken, "new-refresh-token-refreshed")
	}

	// Token expiry should be updated (approximately 1 hour from now)
	expectedExpiry := time.Now().Add(55 * time.Minute) // Give some buffer
	if session.TokenExpiry.Before(expectedExpiry) {
		t.Errorf("TokenExpiry = %v, expected after %v", session.TokenExpiry, expectedExpiry)
	}
}

func TestRefresh_NotNeeded(t *testing.T) {
	c := cache.New(5 * time.Minute)
	sessionSvc := services.NewSessionService(c)

	// Create a session with token NOT expiring soon (more than 5 min out)
	sessionID, err := sessionSvc.Create(
		"testuser",
		"user-123",
		"valid-access-token",
		"valid-refresh-token",
		time.Now().Add(30*time.Minute), // Expires in 30 min, no refresh needed
	)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	cfg := &config.Config{CookieSecure: false}
	h := NewHandler(cfg, c)
	h.SetSessionService(sessionSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: sessionID})
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Should return refreshed=false since not needed
	var refreshResp map[string]bool
	if err := json.NewDecoder(resp.Body).Decode(&refreshResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if refreshResp["refreshed"] {
		t.Error("Expected refreshed=false when token not near expiry")
	}
}

func TestRefresh_InvalidRefreshToken(t *testing.T) {
	// Set up mock servers that only accept a specific refresh token
	cfServer, uaaServer := setupMockCFAndUAAServersWithRefresh("admin", "secret", "valid-token-only")
	defer cfServer.Close()
	defer uaaServer.Close()

	c := cache.New(5 * time.Minute)
	sessionSvc := services.NewSessionService(c)

	// Create a session with an INVALID refresh token
	sessionID, err := sessionSvc.Create(
		"testuser",
		"user-123",
		"old-access-token",
		"invalid-refresh-token",       // Won't be accepted by mock UAA
		time.Now().Add(2*time.Minute), // Expires soon, triggers refresh
	)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	cfg := &config.Config{
		CFAPIUrl:     cfServer.URL,
		CookieSecure: false,
	}
	h := NewHandler(cfg, c)
	h.SetSessionService(sessionSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "DIEGO_SESSION", Value: sessionID})
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	// Should return error when UAA rejects the refresh token
	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	// Session should be deleted to force re-login
	_, err = sessionSvc.Get(sessionID)
	if err == nil {
		t.Error("Session should be deleted after failed refresh")
	}

	// Cookie should be cleared
	cookies := resp.Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "DIEGO_SESSION" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("Expected DIEGO_SESSION cookie to be set (for clearing)")
	}
	if sessionCookie.MaxAge != -1 {
		t.Errorf("Cookie MaxAge = %d, want -1 (expired)", sessionCookie.MaxAge)
	}
}

func TestRefresh_NoSession(t *testing.T) {
	c := cache.New(5 * time.Minute)
	sessionSvc := services.NewSessionService(c)
	cfg := &config.Config{CookieSecure: false}

	h := NewHandler(cfg, c)
	h.SetSessionService(sessionSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}
