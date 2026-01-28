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

// setupMockUAAServer creates a mock UAA server for authentication
func setupMockUAAServer(validUser, validPass string) *httptest.Server {
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
	uaaServer := setupMockUAAServer(validUser, validPass)

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

func TestRefresh_Success(t *testing.T) {
	c := cache.New(5 * time.Minute)
	sessionSvc := services.NewSessionService(c)

	// Create a session with token expiring soon
	sessionID, err := sessionSvc.Create("testuser", "user-123", "old-access", "refresh-token", time.Now().Add(2*time.Minute))
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
