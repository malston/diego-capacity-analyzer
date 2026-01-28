// ABOUTME: Tests for auth request/response models
// ABOUTME: Verifies JSON serialization and field mappings

package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestLoginRequest_JSON(t *testing.T) {
	req := LoginRequest{
		Username: "admin",
		Password: "secret123",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal LoginRequest: %v", err)
	}

	// Verify JSON field names
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	if _, ok := jsonMap["username"]; !ok {
		t.Error("Expected 'username' field in JSON")
	}
	if _, ok := jsonMap["password"]; !ok {
		t.Error("Expected 'password' field in JSON")
	}

	// Verify round-trip
	var decoded LoginRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal LoginRequest: %v", err)
	}

	if decoded.Username != req.Username {
		t.Errorf("Expected Username %q, got %q", req.Username, decoded.Username)
	}
	if decoded.Password != req.Password {
		t.Errorf("Expected Password %q, got %q", req.Password, decoded.Password)
	}
}

func TestLoginResponse_JSON_Success(t *testing.T) {
	resp := LoginResponse{
		Success:  true,
		Username: "admin",
		UserID:   "user-guid-123",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal LoginResponse: %v", err)
	}

	var decoded LoginResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal LoginResponse: %v", err)
	}

	if decoded.Success != true {
		t.Error("Expected Success to be true")
	}
	if decoded.Username != "admin" {
		t.Errorf("Expected Username %q, got %q", "admin", decoded.Username)
	}
	if decoded.UserID != "user-guid-123" {
		t.Errorf("Expected UserID %q, got %q", "user-guid-123", decoded.UserID)
	}
	if decoded.Error != "" {
		t.Errorf("Expected Error to be empty, got %q", decoded.Error)
	}
}

func TestLoginResponse_JSON_Failure(t *testing.T) {
	resp := LoginResponse{
		Success: false,
		Error:   "Invalid credentials",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal LoginResponse: %v", err)
	}

	var decoded LoginResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal LoginResponse: %v", err)
	}

	if decoded.Success != false {
		t.Error("Expected Success to be false")
	}
	if decoded.Error != "Invalid credentials" {
		t.Errorf("Expected Error %q, got %q", "Invalid credentials", decoded.Error)
	}
}

func TestLoginResponse_OmitsEmptyFields(t *testing.T) {
	// Failure response should not include username/user_id
	resp := LoginResponse{
		Success: false,
		Error:   "Invalid credentials",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var jsonMap map[string]interface{}
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	if _, ok := jsonMap["username"]; ok {
		t.Error("Expected 'username' to be omitted when empty")
	}
	if _, ok := jsonMap["user_id"]; ok {
		t.Error("Expected 'user_id' to be omitted when empty")
	}
}

func TestUserInfoResponse_JSON_Authenticated(t *testing.T) {
	resp := UserInfoResponse{
		Authenticated: true,
		Username:      "admin",
		UserID:        "user-guid-123",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal UserInfoResponse: %v", err)
	}

	var decoded UserInfoResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal UserInfoResponse: %v", err)
	}

	if decoded.Authenticated != true {
		t.Error("Expected Authenticated to be true")
	}
	if decoded.Username != "admin" {
		t.Errorf("Expected Username %q, got %q", "admin", decoded.Username)
	}
}

func TestUserInfoResponse_JSON_NotAuthenticated(t *testing.T) {
	resp := UserInfoResponse{
		Authenticated: false,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal UserInfoResponse: %v", err)
	}

	var jsonMap map[string]interface{}
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	// When not authenticated, username and user_id should be omitted
	if _, ok := jsonMap["username"]; ok {
		t.Error("Expected 'username' to be omitted when not authenticated")
	}
	if _, ok := jsonMap["user_id"]; ok {
		t.Error("Expected 'user_id' to be omitted when not authenticated")
	}
}

func TestSession_Fields(t *testing.T) {
	now := time.Now()
	expiry := now.Add(time.Hour)

	session := Session{
		ID:           "session-id-abc123",
		Username:     "admin",
		UserID:       "user-guid-123",
		AccessToken:  "access-token-secret",
		RefreshToken: "refresh-token-secret",
		TokenExpiry:  expiry,
		CreatedAt:    now,
	}

	if session.ID != "session-id-abc123" {
		t.Errorf("Expected ID %q, got %q", "session-id-abc123", session.ID)
	}
	if session.Username != "admin" {
		t.Errorf("Expected Username %q, got %q", "admin", session.Username)
	}
	if session.UserID != "user-guid-123" {
		t.Errorf("Expected UserID %q, got %q", "user-guid-123", session.UserID)
	}
	if session.AccessToken != "access-token-secret" {
		t.Errorf("Expected AccessToken %q, got %q", "access-token-secret", session.AccessToken)
	}
	if session.RefreshToken != "refresh-token-secret" {
		t.Errorf("Expected RefreshToken %q, got %q", "refresh-token-secret", session.RefreshToken)
	}
	if !session.TokenExpiry.Equal(expiry) {
		t.Errorf("Expected TokenExpiry %v, got %v", expiry, session.TokenExpiry)
	}
	if !session.CreatedAt.Equal(now) {
		t.Errorf("Expected CreatedAt %v, got %v", now, session.CreatedAt)
	}
}

func TestSession_NotJSONSerializable(t *testing.T) {
	// Session should NOT be serialized to JSON (contains secrets)
	// This test documents the intentional design: Session is internal only
	session := Session{
		ID:           "session-id",
		Username:     "admin",
		AccessToken:  "secret-token",
		RefreshToken: "secret-refresh",
	}

	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("Marshal failed unexpectedly: %v", err)
	}

	var jsonMap map[string]interface{}
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify tokens are NOT exposed in JSON (using json:"-" tag)
	if _, ok := jsonMap["access_token"]; ok {
		t.Error("AccessToken MUST NOT be exposed in JSON serialization")
	}
	if _, ok := jsonMap["refresh_token"]; ok {
		t.Error("RefreshToken MUST NOT be exposed in JSON serialization")
	}
}
