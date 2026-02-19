# Configurable OAuth Client Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace hardcoded `"cf"` OAuth client in UAA password/refresh grants with configurable `OAUTH_CLIENT_ID` / `OAUTH_CLIENT_SECRET` env vars.

**Architecture:** Add two config fields with backward-compatible defaults (`"cf"` / `""`), wire them into the two `SetBasicAuth` calls in `auth.go`, and validate with tests that the mock UAA receives the correct client credentials.

**Tech Stack:** Go 1.23+, standard library `net/http`, `encoding/base64`

**Design:** [docs/plans/2026-02-18-oauth-client-config-design.md](2026-02-18-oauth-client-config-design.md)

---

### Task 1: Add Config Fields

**Files:**

- Modify: `backend/config/config.go:13-55` (Config struct)
- Modify: `backend/config/config.go:62-99` (Load function)
- Test: `backend/config/config_test.go`
- Reference: `backend/config/helpers_test.go` (test helpers: `withCleanCFEnv`, `withCleanCFEnvAndExtra`)

**Step 1: Write the failing test**

Add tests in `backend/config/config_test.go`. Use the existing `withCleanCFEnv` / `withCleanCFEnvAndExtra` helpers from `helpers_test.go` -- these clear the env, set required CF vars, and restore on cleanup.

```go
func TestLoad_OAuthClientDefaults(t *testing.T) {
	t.Cleanup(withCleanCFEnv(t))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.OAuthClientID != "cf" {
		t.Errorf("OAuthClientID = %q, want %q", cfg.OAuthClientID, "cf")
	}
	if cfg.OAuthClientSecret != "" {
		t.Errorf("OAuthClientSecret = %q, want %q", cfg.OAuthClientSecret, "")
	}
}

func TestLoad_OAuthClientFromEnv(t *testing.T) {
	t.Cleanup(withCleanCFEnvAndExtra(t, map[string]string{
		"OAUTH_CLIENT_ID":     "diego-analyzer",
		"OAUTH_CLIENT_SECRET": "my-secret",
	}))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.OAuthClientID != "diego-analyzer" {
		t.Errorf("OAuthClientID = %q, want %q", cfg.OAuthClientID, "diego-analyzer")
	}
	if cfg.OAuthClientSecret != "my-secret" {
		t.Errorf("OAuthClientSecret = %q, want %q", cfg.OAuthClientSecret, "my-secret")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./config/ -run TestLoad_OAuth -v`
Expected: FAIL -- `cfg.OAuthClientID` field does not exist

**Step 3: Write minimal implementation**

In `backend/config/config.go`, add to the `Config` struct (after line 20, the `CookieSecure` field):

```go
// OAuth Client (for UAA password/refresh grants)
OAuthClientID     string
OAuthClientSecret string
```

In the `Load()` function, add after line 69 (`CookieSecure`):

```go
OAuthClientID:     getEnv("OAUTH_CLIENT_ID", "cf"),
OAuthClientSecret: os.Getenv("OAUTH_CLIENT_SECRET"),
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./config/ -run TestLoad_OAuth -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/config/config.go backend/config/config_test.go
git commit -m "feat(config): add OAuthClientID and OAuthClientSecret fields (GH-108)"
```

---

### Task 2: Update Mock UAA to Validate Client Credentials

**Files:**

- Modify: `backend/handlers/auth_test.go:100-163` (mock UAA helpers)

**Step 1: Update mock UAA server to accept client credentials parameters**

Change the `setupMockUAAServerWithRefresh` signature to accept `clientID` and `clientSecret`:

```go
func setupMockUAAServerWithRefresh(validUser, validPass, validRefreshToken, clientID, clientSecret string) *httptest.Server {
```

At the top of the handler (before grant_type checks), validate Basic Auth:

```go
reqClientID, reqClientSecret, ok := r.BasicAuth()
if !ok || reqClientID != clientID || reqClientSecret != clientSecret {
    w.WriteHeader(http.StatusUnauthorized)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "error":             "unauthorized",
        "error_description": "Bad client credentials",
    })
    return
}
```

**Step 2: Update convenience helpers**

Update `setupMockCFAndUAAServers` to default to `"cf"`, `""`:

```go
func setupMockCFAndUAAServers(validUser, validPass string) (*httptest.Server, *httptest.Server) {
    return setupMockCFAndUAAServersWithRefresh(validUser, validPass, "")
}
```

Update `setupMockCFAndUAAServersWithRefresh` to pass `"cf"`, `""` through:

```go
func setupMockCFAndUAAServersWithRefresh(validUser, validPass, validRefreshToken string) (*httptest.Server, *httptest.Server) {
    uaaServer := setupMockUAAServerWithRefresh(validUser, validPass, validRefreshToken, "cf", "")
    // ... rest unchanged
}
```

Add a new helper for custom client credentials:

```go
func setupMockCFAndUAAServersWithClient(validUser, validPass, validRefreshToken, clientID, clientSecret string) (*httptest.Server, *httptest.Server) {
    uaaServer := setupMockUAAServerWithRefresh(validUser, validPass, validRefreshToken, clientID, clientSecret)
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
```

**Step 3: Run all existing tests to verify they still pass**

Run: `cd backend && go test ./handlers/ -v`
Expected: All existing tests PASS (they use `"cf"` / `""` which matches the default)

**Step 4: Commit**

```bash
git add backend/handlers/auth_test.go
git commit -m "test: validate OAuth client credentials in mock UAA (GH-108)"
```

---

### Task 3: Wire Config into Auth Handlers (TDD)

**Files:**

- Modify: `backend/handlers/auth.go:214,268` (two SetBasicAuth calls)
- Test: `backend/handlers/auth_test.go` (add custom client test)

**Step 1: Write the failing test**

Add a test that uses a custom client ID/secret and verifies login succeeds:

```go
func TestLogin_UsesConfiguredOAuthClient(t *testing.T) {
    cfServer, uaaServer := setupMockCFAndUAAServersWithClient(
        "admin", "secret", "", "diego-analyzer", "client-secret-123",
    )
    defer cfServer.Close()
    defer uaaServer.Close()

    c := cache.New(5 * time.Minute)
    sessionSvc := services.NewSessionService(c)
    cfg := &config.Config{
        CFAPIUrl:          cfServer.URL,
        CookieSecure:      false,
        OAuthClientID:     "diego-analyzer",
        OAuthClientSecret: "client-secret-123",
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

    var loginResp models.LoginResponse
    if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
        t.Fatalf("Failed to decode response: %v", err)
    }
    if !loginResp.Success {
        t.Errorf("Expected login success with custom OAuth client")
    }
}
```

Add a test that verifies refresh also uses the custom client:

```go
func TestRefresh_UsesConfiguredOAuthClient(t *testing.T) {
    knownRefreshToken := "refresh-token-for-custom-client"
    cfServer, uaaServer := setupMockCFAndUAAServersWithClient(
        "admin", "secret", knownRefreshToken, "diego-analyzer", "client-secret-123",
    )
    defer cfServer.Close()
    defer uaaServer.Close()

    c := cache.New(5 * time.Minute)
    sessionSvc := services.NewSessionService(c)

    sessionID, err := sessionSvc.Create(
        "testuser", "user-123", "old-access-token",
        knownRefreshToken, nil,
        time.Now().Add(2*time.Minute),
    )
    if err != nil {
        t.Fatalf("Failed to create session: %v", err)
    }

    cfg := &config.Config{
        CFAPIUrl:          cfServer.URL,
        CookieSecure:      false,
        OAuthClientID:     "diego-analyzer",
        OAuthClientSecret: "client-secret-123",
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

    var refreshResp map[string]bool
    if err := json.NewDecoder(resp.Body).Decode(&refreshResp); err != nil {
        t.Fatalf("Failed to decode response: %v", err)
    }
    if !refreshResp["refreshed"] {
        t.Error("Expected refreshed=true with custom OAuth client")
    }
}
```

**Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./handlers/ -run "TestLogin_UsesConfiguredOAuthClient|TestRefresh_UsesConfiguredOAuthClient" -v`
Expected: FAIL -- mock UAA rejects `"cf"` / `""` because it expects `"diego-analyzer"` / `"client-secret-123"`

**Step 3: Write minimal implementation**

In `backend/handlers/auth.go`, change line 214:

```go
// Before:
req.SetBasicAuth("cf", "")
// After:
req.SetBasicAuth(h.cfg.OAuthClientID, h.cfg.OAuthClientSecret)
```

And line 268:

```go
// Before:
req.SetBasicAuth("cf", "")
// After:
req.SetBasicAuth(h.cfg.OAuthClientID, h.cfg.OAuthClientSecret)
```

**Step 4: Run all tests to verify they pass**

Run: `cd backend && go test ./handlers/ -v`
Expected: ALL tests PASS (existing tests use default `"cf"` / `""`, new tests use custom client)

**Step 5: Commit**

```bash
git add backend/handlers/auth.go backend/handlers/auth_test.go
git commit -m "feat(auth): use configurable OAuth client for UAA grants (GH-108)"
```

---

### Task 4: Update .env.example and Documentation

**Files:**

- Modify: `.env.example` (add OAuth section after Authentication section)
- Modify: `docs/AUTHENTICATION.md:21-29` (config table)

**Step 1: Add OAuth Client section to `.env.example`**

After the Authentication section (after line 81, `CORS_ALLOWED_ORIGINS`), add:

```
# =============================================================================
# OAuth Client (Optional - for dedicated UAA client)
# =============================================================================
# Create a dedicated UAA client instead of using the shared "cf" client.
# See docs/AUTHENTICATION.md for setup instructions.
# OAUTH_CLIENT_ID=diego-analyzer
# OAUTH_CLIENT_SECRET=
```

**Step 2: Add to AUTHENTICATION.md config table**

Add two rows to the table at `docs/AUTHENTICATION.md:21-29`:

```
| `OAUTH_CLIENT_ID`       | `cf`       | OAuth client ID for UAA password grants      |
| `OAUTH_CLIENT_SECRET`   | (empty)    | OAuth client secret                          |
```

**Step 3: Run full test suite**

Run: `make check`
Expected: All backend and frontend tests pass, linting clean

**Step 4: Commit**

```bash
git add .env.example docs/AUTHENTICATION.md
git commit -m "docs: add OAUTH_CLIENT_ID/SECRET to env example and auth docs (GH-108)"
```
