// ABOUTME: Session management service for BFF OAuth pattern
// ABOUTME: Stores and retrieves auth sessions using cache backend

package services

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/cache"
	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

// SessionService manages server-side authentication sessions
type SessionService struct {
	cache *cache.Cache
}

// NewSessionService creates a new session service
func NewSessionService(c *cache.Cache) *SessionService {
	return &SessionService{cache: c}
}

// Create generates a new session and stores it in the cache
// Returns the cryptographically secure session ID
func (s *SessionService) Create(username, userID, accessToken, refreshToken string, tokenExpiry time.Time) (string, error) {
	// Generate 32 bytes of cryptographically secure random data for session ID
	sessionIDBytes := make([]byte, 32)
	if _, err := rand.Read(sessionIDBytes); err != nil {
		return "", err
	}
	sessionID := base64.URLEncoding.EncodeToString(sessionIDBytes)

	// Generate 32 bytes of cryptographically secure random data for CSRF token
	csrfBytes := make([]byte, 32)
	if _, err := rand.Read(csrfBytes); err != nil {
		return "", err
	}
	csrfToken := base64.URLEncoding.EncodeToString(csrfBytes)

	session := &models.Session{
		ID:           sessionID,
		Username:     username,
		UserID:       userID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		CSRFToken:    csrfToken,
		TokenExpiry:  tokenExpiry,
		CreatedAt:    time.Now(),
	}

	// Store session with TTL matching token expiry (plus buffer for refresh)
	ttl := time.Until(tokenExpiry) + 10*time.Minute
	if ttl < time.Minute {
		ttl = time.Minute
	}
	s.cache.SetWithTTL(sessionKey(sessionID), session, ttl)

	return sessionID, nil
}

// Get retrieves a session by ID
func (s *SessionService) Get(sessionID string) (*models.Session, error) {
	val, ok := s.cache.Get(sessionKey(sessionID))
	if !ok {
		return nil, errors.New("session not found")
	}

	session, ok := val.(*models.Session)
	if !ok {
		return nil, errors.New("invalid session data")
	}

	return session, nil
}

// Delete removes a session from the cache
func (s *SessionService) Delete(sessionID string) {
	s.cache.Clear(sessionKey(sessionID))
}

// NeedsRefresh checks if the session's token is near expiry
// Returns true if token expires within 5 minutes or less
func (s *SessionService) NeedsRefresh(session *models.Session) bool {
	return time.Until(session.TokenExpiry) <= 5*time.Minute
}

// UpdateTokens updates the tokens for an existing session
func (s *SessionService) UpdateTokens(sessionID, accessToken, refreshToken string, tokenExpiry time.Time) error {
	session, err := s.Get(sessionID)
	if err != nil {
		return err
	}

	session.AccessToken = accessToken
	session.RefreshToken = refreshToken
	session.TokenExpiry = tokenExpiry

	// Update cache with new TTL
	ttl := time.Until(tokenExpiry) + 10*time.Minute
	if ttl < time.Minute {
		ttl = time.Minute
	}
	s.cache.SetWithTTL(sessionKey(sessionID), session, ttl)

	return nil
}

// sessionKey returns the cache key for a session ID
func sessionKey(sessionID string) string {
	return "session:" + sessionID
}
