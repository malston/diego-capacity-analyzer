// ABOUTME: CF API proxy handlers for BFF OAuth pattern
// ABOUTME: Proxies CF API calls using session-stored tokens (never exposed to frontend)

package handlers

import (
	"io"
	"log/slog"
	"net/http"
)

// getSessionToken retrieves the CF access token from the session cookie.
// Returns empty string and writes 401 response if session is invalid.
func (h *Handler) getSessionToken(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie("DIEGO_SESSION")
	if err != nil {
		slog.Debug("CF proxy: no session cookie", "path", r.URL.Path)
		h.writeError(w, "Authentication required", http.StatusUnauthorized)
		return ""
	}

	if h.sessionService == nil {
		slog.Error("CF proxy: session service not configured")
		h.writeError(w, "Server configuration error", http.StatusInternalServerError)
		return ""
	}

	session, err := h.sessionService.Get(cookie.Value)
	if err != nil {
		slog.Debug("CF proxy: invalid session", "path", r.URL.Path)
		h.writeError(w, "Invalid session", http.StatusUnauthorized)
		return ""
	}

	return session.AccessToken
}

// proxyCFRequest makes an authenticated request to the CF API and streams the response.
func (h *Handler) proxyCFRequest(w http.ResponseWriter, cfPath, token string) {
	cfURL := h.cfg.CFAPIUrl + cfPath

	req, err := http.NewRequest(http.MethodGet, cfURL, nil)
	if err != nil {
		slog.Error("CF proxy: failed to create request", "error", err)
		h.writeError(w, "Internal error", http.StatusInternalServerError)
		return
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("CF proxy: request failed", "url", cfURL, "error", err)
		h.writeError(w, "CF API request failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy CF API response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Stream the response
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// CFProxyIsolationSegments proxies GET /v3/isolation_segments to CF API.
func (h *Handler) CFProxyIsolationSegments(w http.ResponseWriter, r *http.Request) {
	token := h.getSessionToken(w, r)
	if token == "" {
		return
	}

	h.proxyCFRequest(w, "/v3/isolation_segments", token)
}

// CFProxyApps proxies GET /v3/apps to CF API.
func (h *Handler) CFProxyApps(w http.ResponseWriter, r *http.Request) {
	token := h.getSessionToken(w, r)
	if token == "" {
		return
	}

	// Preserve query string for pagination
	cfPath := "/v3/apps"
	if r.URL.RawQuery != "" {
		cfPath += "?" + r.URL.RawQuery
	}

	h.proxyCFRequest(w, cfPath, token)
}

// CFProxyAppProcesses proxies GET /v3/apps/{guid}/processes to CF API.
func (h *Handler) CFProxyAppProcesses(w http.ResponseWriter, r *http.Request) {
	token := h.getSessionToken(w, r)
	if token == "" {
		return
	}

	guid := r.PathValue("guid")
	if guid == "" {
		h.writeError(w, "Missing app GUID", http.StatusBadRequest)
		return
	}

	h.proxyCFRequest(w, "/v3/apps/"+guid+"/processes", token)
}

// CFProxyProcessStats proxies GET /v3/processes/{guid}/stats to CF API.
func (h *Handler) CFProxyProcessStats(w http.ResponseWriter, r *http.Request) {
	token := h.getSessionToken(w, r)
	if token == "" {
		return
	}

	guid := r.PathValue("guid")
	if guid == "" {
		h.writeError(w, "Missing process GUID", http.StatusBadRequest)
		return
	}

	h.proxyCFRequest(w, "/v3/processes/"+guid+"/stats", token)
}

// CFProxySpaces proxies GET /v3/spaces/{guid} to CF API.
func (h *Handler) CFProxySpaces(w http.ResponseWriter, r *http.Request) {
	token := h.getSessionToken(w, r)
	if token == "" {
		return
	}

	guid := r.PathValue("guid")
	if guid == "" {
		h.writeError(w, "Missing space GUID", http.StatusBadRequest)
		return
	}

	h.proxyCFRequest(w, "/v3/spaces/"+guid, token)
}

// CFProxyIsolationSegmentByGUID proxies GET /v3/isolation_segments/{guid} to CF API.
func (h *Handler) CFProxyIsolationSegmentByGUID(w http.ResponseWriter, r *http.Request) {
	token := h.getSessionToken(w, r)
	if token == "" {
		return
	}

	guid := r.PathValue("guid")
	if guid == "" {
		h.writeError(w, "Missing isolation segment GUID", http.StatusBadRequest)
		return
	}

	h.proxyCFRequest(w, "/v3/isolation_segments/"+guid, token)
}
