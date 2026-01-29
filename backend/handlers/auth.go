// ABOUTME: Auth handlers implementing BFF OAuth pattern
// ABOUTME: Handles login, logout, session management with httpOnly cookies

package handlers

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

const sessionCookieName = "DIEGO_SESSION"

// Login authenticates with CF UAA and creates a server-side session
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		h.writeError(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Authenticate with CF UAA
	tokenResp, err := h.authenticateWithCFUAA(req.Username, req.Password)
	if err != nil {
		slog.Warn("Authentication failed", "username", req.Username, "error", err)
		h.writeJSON(w, http.StatusUnauthorized, models.LoginResponse{
			Success: false,
			Error:   "Invalid credentials",
		})
		return
	}

	// Calculate token expiry
	expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	// Create session (stores tokens server-side)
	sessionID, err := h.sessionService.Create(
		req.Username,
		tokenResp.UserID,
		tokenResp.AccessToken,
		tokenResp.RefreshToken,
		expiry,
	)
	if err != nil {
		slog.Error("Failed to create session", "error", err)
		h.writeError(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set httpOnly cookie with session ID only
	h.setSessionCookie(w, sessionID)

	// Return success response (no tokens!)
	h.writeJSON(w, http.StatusOK, models.LoginResponse{
		Success:  true,
		Username: req.Username,
		UserID:   tokenResp.UserID,
	})
}

// Me returns the current user's authentication status
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	session := h.getSessionFromCookie(r)
	if session == nil {
		h.writeJSON(w, http.StatusOK, models.UserInfoResponse{
			Authenticated: false,
		})
		return
	}

	h.writeJSON(w, http.StatusOK, models.UserInfoResponse{
		Authenticated: true,
		Username:      session.Username,
		UserID:        session.UserID,
	})
}

// Logout clears the session and cookie
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get session ID from cookie
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil && cookie.Value != "" {
		// Delete session from cache (if sessionService is configured)
		if h.sessionService != nil {
			h.sessionService.Delete(cookie.Value)
		}
	}

	// Clear cookie
	h.clearSessionCookie(w)

	h.writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// Refresh refreshes the session token if needed
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	session := h.getSessionFromCookie(r)
	if session == nil {
		h.writeError(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	// Check if refresh is needed
	if !h.sessionService.NeedsRefresh(session) {
		h.writeJSON(w, http.StatusOK, map[string]bool{"refreshed": false})
		return
	}

	// Refresh the token with UAA
	tokenResp, err := h.refreshWithCFUAA(session.RefreshToken)
	if err != nil {
		slog.Warn("Token refresh failed", "error", err)
		h.writeError(w, "Token refresh failed", http.StatusUnauthorized)
		return
	}

	// Calculate new token expiry
	expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	// Update session with new tokens
	cookie, _ := r.Cookie(sessionCookieName)
	if err := h.sessionService.UpdateTokens(cookie.Value, tokenResp.AccessToken, tokenResp.RefreshToken, expiry); err != nil {
		slog.Error("Failed to update session tokens", "error", err)
		h.writeError(w, "Failed to update session", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]bool{"refreshed": true})
}

// uaaTokenResponse represents the OAuth token response from CF UAA
type uaaTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	UserID       string `json:"user_id"`
}

// refreshWithCFUAA performs OAuth2 refresh_token grant with CF UAA
func (h *Handler) refreshWithCFUAA(refreshToken string) (*uaaTokenResponse, error) {
	if h.cfg == nil || h.cfg.CFAPIUrl == "" {
		return nil, fmt.Errorf("CF API not configured")
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: h.cfg.CFSkipSSLValidation},
		},
	}

	// Get UAA URL from CF API info
	uaaURL, err := h.getUAAURL(client)
	if err != nil {
		return nil, fmt.Errorf("failed to get UAA URL: %w", err)
	}

	// Request new tokens using refresh_token grant
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequest("POST", uaaURL+"/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("cf", "")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp uaaTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokenResp, nil
}

// authenticateWithCFUAA performs OAuth2 password grant with CF UAA
func (h *Handler) authenticateWithCFUAA(username, password string) (*uaaTokenResponse, error) {
	if h.cfg == nil || h.cfg.CFAPIUrl == "" {
		return nil, fmt.Errorf("CF API not configured")
	}

	// Create HTTP client (reusing CF skip SSL validation setting)
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: h.cfg.CFSkipSSLValidation},
		},
	}

	// Get UAA URL from CF API info
	uaaURL, err := h.getUAAURL(client)
	if err != nil {
		return nil, fmt.Errorf("failed to get UAA URL: %w", err)
	}

	// Authenticate with UAA
	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", username)
	data.Set("password", password)

	req, err := http.NewRequest("POST", uaaURL+"/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("cf", "")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("authentication request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("authentication failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp uaaTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	// Extract user_id from JWT if not in response
	if tokenResp.UserID == "" {
		tokenResp.UserID = username // Fallback to username
	}

	return &tokenResp, nil
}

// getUAAURL discovers the UAA endpoint from CF API info
func (h *Handler) getUAAURL(client *http.Client) (string, error) {
	resp, err := client.Get(h.cfg.CFAPIUrl + "/v3/info")
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
		uaaURL = strings.Replace(h.cfg.CFAPIUrl, "://api.", "://login.", 1)
	}

	return uaaURL, nil
}

// getSessionFromCookie retrieves the session from the request cookie
func (h *Handler) getSessionFromCookie(r *http.Request) *models.Session {
	if h.sessionService == nil {
		return nil
	}

	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil
	}

	session, err := h.sessionService.Get(cookie.Value)
	if err != nil {
		return nil
	}

	return session
}

// setSessionCookie sets the httpOnly session cookie
func (h *Handler) setSessionCookie(w http.ResponseWriter, sessionID string) {
	secure := true
	if h.cfg != nil {
		secure = h.cfg.CookieSecure
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		MaxAge:   3600, // 1 hour
	})
}

// clearSessionCookie removes the session cookie
func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	secure := true
	if h.cfg != nil {
		secure = h.cfg.CookieSecure
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		MaxAge:   -1, // Delete cookie
	})
}
