// ABOUTME: Auth request/response models for BFF OAuth pattern
// ABOUTME: Defines session structure and login/logout API contracts

package models

import "time"

// LoginRequest represents credentials for authentication
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the result of a login attempt
type LoginResponse struct {
	Success  bool   `json:"success"`
	Username string `json:"username,omitempty"`
	UserID   string `json:"user_id,omitempty"`
	Error    string `json:"error,omitempty"`
}

// UserInfoResponse represents the current user's authentication state
type UserInfoResponse struct {
	Authenticated bool   `json:"authenticated"`
	Username      string `json:"username,omitempty"`
	UserID        string `json:"user_id,omitempty"`
}

// Session stores server-side authentication state
// Tokens are never exposed to the client
type Session struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	UserID       string    `json:"user_id"`
	AccessToken  string    `json:"-"` // Never expose to client
	RefreshToken string    `json:"-"` // Never expose to client
	TokenExpiry  time.Time `json:"token_expiry"`
	CreatedAt    time.Time `json:"created_at"`
}
