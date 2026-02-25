// ABOUTME: Contract tests for the ChatProvider interface and domain types
// ABOUTME: Verifies Message, TokenEvent, Usage struct fields and ChatConfig resolution

package ai

import (
	"context"
	"testing"
)

func TestMessage_HasRoleAndContent(t *testing.T) {
	msg := Message{Role: "user", Content: "hello"}

	if msg.Role != "user" {
		t.Errorf("Role = %q, want %q", msg.Role, "user")
	}
	if msg.Content != "hello" {
		t.Errorf("Content = %q, want %q", msg.Content, "hello")
	}
}

func TestTokenEvent_CarriesAllFields(t *testing.T) {
	usage := &Usage{InputTokens: 10, OutputTokens: 20}
	event := TokenEvent{
		Text:       "token",
		Done:       true,
		StopReason: "end_turn",
		Usage:      usage,
		Err:        nil,
	}

	if event.Text != "token" {
		t.Errorf("Text = %q, want %q", event.Text, "token")
	}
	if !event.Done {
		t.Error("Done = false, want true")
	}
	if event.StopReason != "end_turn" {
		t.Errorf("StopReason = %q, want %q", event.StopReason, "end_turn")
	}
	if event.Usage == nil {
		t.Fatal("Usage = nil, want non-nil")
	}
	if event.Usage.InputTokens != 10 {
		t.Errorf("InputTokens = %d, want %d", event.Usage.InputTokens, 10)
	}
	if event.Usage.OutputTokens != 20 {
		t.Errorf("OutputTokens = %d, want %d", event.Usage.OutputTokens, 20)
	}
	if event.Err != nil {
		t.Errorf("Err = %v, want nil", event.Err)
	}
}

func TestUsage_TracksInputAndOutputTokens(t *testing.T) {
	u := Usage{InputTokens: 150, OutputTokens: 300}

	if u.InputTokens != 150 {
		t.Errorf("InputTokens = %d, want %d", u.InputTokens, 150)
	}
	if u.OutputTokens != 300 {
		t.Errorf("OutputTokens = %d, want %d", u.OutputTokens, 300)
	}
}

func TestNewChatConfig_NoOptions_ReturnsZeroValues(t *testing.T) {
	cfg := NewChatConfig()

	if cfg.MaxTokens != 0 {
		t.Errorf("MaxTokens = %d, want 0", cfg.MaxTokens)
	}
	if cfg.Temperature != nil {
		t.Errorf("Temperature = %v, want nil", cfg.Temperature)
	}
	if cfg.System != "" {
		t.Errorf("System = %q, want empty", cfg.System)
	}
	if cfg.Model != "" {
		t.Errorf("Model = %q, want empty", cfg.Model)
	}
}

func TestNewChatConfig_AppliesOptions(t *testing.T) {
	cfg := NewChatConfig(
		WithMaxTokens(4096),
		WithTemperature(0.3),
		WithSystem("You are an assistant"),
		WithModel("claude-3"),
	)

	if cfg.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %d, want %d", cfg.MaxTokens, 4096)
	}
	if cfg.Temperature == nil {
		t.Fatal("Temperature = nil, want non-nil")
	}
	if *cfg.Temperature != 0.3 {
		t.Errorf("Temperature = %f, want %f", *cfg.Temperature, 0.3)
	}
	if cfg.System != "You are an assistant" {
		t.Errorf("System = %q, want %q", cfg.System, "You are an assistant")
	}
	if cfg.Model != "claude-3" {
		t.Errorf("Model = %q, want %q", cfg.Model, "claude-3")
	}
}

func TestChatProvider_InterfaceSatisfaction(t *testing.T) {
	// Compile-time check that a concrete type can satisfy ChatProvider.
	var _ ChatProvider = (*mockProvider)(nil)
}

// mockProvider verifies the ChatProvider interface is implementable.
type mockProvider struct{}

func (m *mockProvider) Chat(_ context.Context, _ []Message, _ ...Option) <-chan TokenEvent {
	return nil
}
