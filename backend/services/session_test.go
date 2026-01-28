// ABOUTME: Tests for session management service
// ABOUTME: Verifies secure session ID generation and CRUD operations

package services

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/cache"
	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

func TestNewSessionService(t *testing.T) {
	c := cache.New(5 * time.Minute)
	svc := NewSessionService(c)

	if svc == nil {
		t.Fatal("NewSessionService returned nil")
	}
}

func TestSessionService_Create(t *testing.T) {
	c := cache.New(5 * time.Minute)
	svc := NewSessionService(c)

	expiry := time.Now().Add(time.Hour)
	sessionID, err := svc.Create("testuser", "user-123", "access-token", "refresh-token", expiry)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if sessionID == "" {
		t.Error("Create returned empty session ID")
	}

	// Session ID should be base64-encoded
	decoded, err := base64.URLEncoding.DecodeString(sessionID)
	if err != nil {
		t.Errorf("Session ID is not valid base64: %v", err)
	}

	// Session ID should be 32 bytes when decoded
	if len(decoded) != 32 {
		t.Errorf("Session ID decoded length = %d, want 32", len(decoded))
	}
}

func TestSessionService_Create_UniqueIDs(t *testing.T) {
	c := cache.New(5 * time.Minute)
	svc := NewSessionService(c)

	expiry := time.Now().Add(time.Hour)
	ids := make(map[string]bool)

	// Create 100 sessions and verify all IDs are unique
	for i := 0; i < 100; i++ {
		sessionID, err := svc.Create("testuser", "user-123", "access", "refresh", expiry)
		if err != nil {
			t.Fatalf("Create failed at iteration %d: %v", i, err)
		}
		if ids[sessionID] {
			t.Errorf("Duplicate session ID generated: %s", sessionID)
		}
		ids[sessionID] = true
	}
}

func TestSessionService_Get(t *testing.T) {
	c := cache.New(5 * time.Minute)
	svc := NewSessionService(c)

	expiry := time.Now().Add(time.Hour)
	sessionID, err := svc.Create("testuser", "user-123", "access-token", "refresh-token", expiry)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	session, err := svc.Get(sessionID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if session == nil {
		t.Fatal("Get returned nil session")
	}

	if session.ID != sessionID {
		t.Errorf("Session ID = %q, want %q", session.ID, sessionID)
	}
	if session.Username != "testuser" {
		t.Errorf("Username = %q, want %q", session.Username, "testuser")
	}
	if session.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", session.UserID, "user-123")
	}
	if session.AccessToken != "access-token" {
		t.Errorf("AccessToken = %q, want %q", session.AccessToken, "access-token")
	}
	if session.RefreshToken != "refresh-token" {
		t.Errorf("RefreshToken = %q, want %q", session.RefreshToken, "refresh-token")
	}
}

func TestSessionService_Get_NotFound(t *testing.T) {
	c := cache.New(5 * time.Minute)
	svc := NewSessionService(c)

	session, err := svc.Get("nonexistent-session-id")
	if err == nil {
		t.Error("Get should return error for nonexistent session")
	}
	if session != nil {
		t.Error("Get should return nil session for nonexistent session")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error should contain 'not found', got: %v", err)
	}
}

func TestSessionService_Delete(t *testing.T) {
	c := cache.New(5 * time.Minute)
	svc := NewSessionService(c)

	expiry := time.Now().Add(time.Hour)
	sessionID, err := svc.Create("testuser", "user-123", "access", "refresh", expiry)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify session exists
	_, err = svc.Get(sessionID)
	if err != nil {
		t.Fatalf("Session should exist before delete: %v", err)
	}

	// Delete session
	svc.Delete(sessionID)

	// Verify session is gone
	_, err = svc.Get(sessionID)
	if err == nil {
		t.Error("Get should return error after Delete")
	}
}

func TestSessionService_NeedsRefresh(t *testing.T) {
	c := cache.New(5 * time.Minute)
	svc := NewSessionService(c)

	tests := []struct {
		name        string
		tokenExpiry time.Duration
		wantRefresh bool
	}{
		{"token expires in 10 minutes", 10 * time.Minute, false},
		{"token expires in 5 minutes", 5 * time.Minute, true},
		{"token expires in 1 minute", 1 * time.Minute, true},
		{"token expired", -1 * time.Minute, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &models.Session{
				ID:          "test-session",
				TokenExpiry: time.Now().Add(tt.tokenExpiry),
			}

			needsRefresh := svc.NeedsRefresh(session)
			if needsRefresh != tt.wantRefresh {
				t.Errorf("NeedsRefresh() = %v, want %v", needsRefresh, tt.wantRefresh)
			}
		})
	}
}

func TestSessionService_UpdateTokens(t *testing.T) {
	c := cache.New(5 * time.Minute)
	svc := NewSessionService(c)

	expiry := time.Now().Add(time.Hour)
	sessionID, err := svc.Create("testuser", "user-123", "old-access", "old-refresh", expiry)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	newExpiry := time.Now().Add(2 * time.Hour)
	err = svc.UpdateTokens(sessionID, "new-access", "new-refresh", newExpiry)
	if err != nil {
		t.Fatalf("UpdateTokens failed: %v", err)
	}

	session, err := svc.Get(sessionID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if session.AccessToken != "new-access" {
		t.Errorf("AccessToken = %q, want %q", session.AccessToken, "new-access")
	}
	if session.RefreshToken != "new-refresh" {
		t.Errorf("RefreshToken = %q, want %q", session.RefreshToken, "new-refresh")
	}
	if !session.TokenExpiry.Equal(newExpiry) {
		t.Errorf("TokenExpiry = %v, want %v", session.TokenExpiry, newExpiry)
	}
}

func TestSessionService_UpdateTokens_NotFound(t *testing.T) {
	c := cache.New(5 * time.Minute)
	svc := NewSessionService(c)

	err := svc.UpdateTokens("nonexistent", "access", "refresh", time.Now().Add(time.Hour))
	if err == nil {
		t.Error("UpdateTokens should return error for nonexistent session")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error should contain 'not found', got: %v", err)
	}
}

func TestSessionService_SessionIDNotPredictable(t *testing.T) {
	c := cache.New(5 * time.Minute)
	svc := NewSessionService(c)

	expiry := time.Now().Add(time.Hour)

	// Create two sessions with same data
	id1, _ := svc.Create("testuser", "user-123", "access", "refresh", expiry)
	id2, _ := svc.Create("testuser", "user-123", "access", "refresh", expiry)

	// Session IDs should be different (not derived from input)
	if id1 == id2 {
		t.Error("Session IDs should not be predictable/deterministic")
	}
}

func TestSessionService_ConcurrentAccess(t *testing.T) {
	c := cache.New(5 * time.Minute)
	svc := NewSessionService(c)

	expiry := time.Now().Add(time.Hour)

	// Create a session
	sessionID, err := svc.Create("testuser", "user-123", "access", "refresh", expiry)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Concurrent reads
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := svc.Get(sessionID)
			if err != nil {
				t.Errorf("Concurrent Get failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
