// ABOUTME: Unit tests for the Anthropic provider implementation
// ABOUTME: Verifies message mapping, config resolution, send helper, and interface compliance

package ai

import (
	"context"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

// Compile-time check that AnthropicProvider satisfies ChatProvider.
var _ ChatProvider = (*AnthropicProvider)(nil)

func TestToSDKMessages_MapsUserAndAssistant(t *testing.T) {
	msgs := []Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
		{Role: "user", Content: "how are you?"},
	}

	result := toSDKMessages(msgs)

	if len(result) != 3 {
		t.Fatalf("len(result) = %d, want 3", len(result))
	}

	// Verify user message
	if result[0].Role != "user" {
		t.Errorf("result[0].Role = %q, want %q", result[0].Role, "user")
	}

	// Verify assistant message
	if result[1].Role != "assistant" {
		t.Errorf("result[1].Role = %q, want %q", result[1].Role, "assistant")
	}

	// Verify second user message
	if result[2].Role != "user" {
		t.Errorf("result[2].Role = %q, want %q", result[2].Role, "user")
	}
}

func TestToSDKMessages_SkipsSystemRole(t *testing.T) {
	msgs := []Message{
		{Role: "system", Content: "you are helpful"},
		{Role: "user", Content: "hello"},
	}

	result := toSDKMessages(msgs)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1 (system should be skipped)", len(result))
	}
	if result[0].Role != "user" {
		t.Errorf("result[0].Role = %q, want %q", result[0].Role, "user")
	}
}

func TestToSDKMessages_SkipsUnknownRole(t *testing.T) {
	msgs := []Message{
		{Role: "tool", Content: "some tool result"},
		{Role: "user", Content: "hello"},
	}

	result := toSDKMessages(msgs)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1 (unknown role should be skipped)", len(result))
	}
	if result[0].Role != "user" {
		t.Errorf("result[0].Role = %q, want %q", result[0].Role, "user")
	}
}

func TestToSDKMessages_EmptyInput(t *testing.T) {
	result := toSDKMessages(nil)

	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0", len(result))
	}
}

func TestResolveConfig_UserOptionsOverrideDefaults(t *testing.T) {
	p := NewAnthropicProvider("test-key", ChatConfig{
		MaxTokens:   4096,
		Temperature: floatPtr(0.3),
		Model:       string(anthropic.ModelClaudeSonnet4_5),
	})

	cfg := p.resolveConfig([]Option{
		WithMaxTokens(8192),
		WithTemperature(0.7),
		WithModel("custom-model"),
		WithSystem("custom system"),
	})

	if cfg.MaxTokens != 8192 {
		t.Errorf("MaxTokens = %d, want %d", cfg.MaxTokens, 8192)
	}
	if cfg.Temperature == nil {
		t.Fatal("Temperature = nil, want non-nil")
	}
	if *cfg.Temperature != 0.7 {
		t.Errorf("Temperature = %f, want %f", *cfg.Temperature, 0.7)
	}
	if cfg.Model != "custom-model" {
		t.Errorf("Model = %q, want %q", cfg.Model, "custom-model")
	}
	if cfg.System != "custom system" {
		t.Errorf("System = %q, want %q", cfg.System, "custom system")
	}
}

func TestResolveConfig_ZeroValuesFilled(t *testing.T) {
	p := NewAnthropicProvider("test-key", ChatConfig{
		MaxTokens:   4096,
		Temperature: floatPtr(0.3),
		Model:       string(anthropic.ModelClaudeSonnet4_5),
	})

	// No options: zero-value ChatConfig should be filled from defaults.
	cfg := p.resolveConfig(nil)

	if cfg.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %d, want %d (default)", cfg.MaxTokens, 4096)
	}
	if cfg.Temperature == nil {
		t.Fatal("Temperature = nil, want non-nil (default)")
	}
	if *cfg.Temperature != 0.3 {
		t.Errorf("Temperature = %f, want %f (default)", *cfg.Temperature, 0.3)
	}
	if cfg.Model != string(anthropic.ModelClaudeSonnet4_5) {
		t.Errorf("Model = %q, want %q (default)", cfg.Model, string(anthropic.ModelClaudeSonnet4_5))
	}
}

func TestResolveConfig_PartialOverride(t *testing.T) {
	p := NewAnthropicProvider("test-key", ChatConfig{
		MaxTokens:   4096,
		Temperature: floatPtr(0.3),
		Model:       string(anthropic.ModelClaudeSonnet4_5),
	})

	// Only override MaxTokens; Temperature and Model should come from defaults.
	cfg := p.resolveConfig([]Option{WithMaxTokens(2048)})

	if cfg.MaxTokens != 2048 {
		t.Errorf("MaxTokens = %d, want %d", cfg.MaxTokens, 2048)
	}
	if cfg.Temperature == nil {
		t.Fatal("Temperature = nil, want non-nil (default)")
	}
	if *cfg.Temperature != 0.3 {
		t.Errorf("Temperature = %f, want %f (default)", *cfg.Temperature, 0.3)
	}
	if cfg.Model != string(anthropic.ModelClaudeSonnet4_5) {
		t.Errorf("Model = %q, want %q (default)", cfg.Model, string(anthropic.ModelClaudeSonnet4_5))
	}
}

func TestResolveConfig_ZeroTemperaturePreserved(t *testing.T) {
	p := NewAnthropicProvider("test-key", ChatConfig{
		MaxTokens:   4096,
		Temperature: floatPtr(0.3),
		Model:       "some-model",
	})

	// Explicitly set temperature to 0.0 -- should NOT be overridden by default 0.3.
	cfg := p.resolveConfig([]Option{WithTemperature(0.0)})

	if cfg.Temperature == nil {
		t.Fatal("Temperature = nil, want non-nil")
	}
	if *cfg.Temperature != 0.0 {
		t.Errorf("Temperature = %f, want 0.0 (explicit zero should be preserved)", *cfg.Temperature)
	}
}

func TestSend_DeliversEvent(t *testing.T) {
	p := &AnthropicProvider{}
	ch := make(chan TokenEvent, 1)
	ctx := context.Background()

	event := TokenEvent{Text: "hello"}
	p.send(ctx, ch, event)

	received := <-ch
	if received.Text != "hello" {
		t.Errorf("Text = %q, want %q", received.Text, "hello")
	}
}

func TestSend_RespectsContextCancellation(t *testing.T) {
	p := &AnthropicProvider{}
	ch := make(chan TokenEvent) // unbuffered -- will block without receiver
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before sending

	// send should not block because context is cancelled.
	done := make(chan struct{})
	go func() {
		p.send(ctx, ch, TokenEvent{Text: "should not block"})
		close(done)
	}()

	<-done // If this hangs, send is blocking.
}

func TestNewAnthropicProvider_StoresDefaults(t *testing.T) {
	defaults := ChatConfig{
		MaxTokens:   2048,
		Temperature: floatPtr(0.5),
		Model:       "test-model",
	}

	p := NewAnthropicProvider("test-key", defaults)

	if p == nil {
		t.Fatal("NewAnthropicProvider returned nil")
	}
	if p.defaults.MaxTokens != 2048 {
		t.Errorf("defaults.MaxTokens = %d, want %d", p.defaults.MaxTokens, 2048)
	}
	if p.defaults.Temperature == nil {
		t.Fatal("defaults.Temperature = nil, want non-nil")
	}
	if *p.defaults.Temperature != 0.5 {
		t.Errorf("defaults.Temperature = %f, want %f", *p.defaults.Temperature, 0.5)
	}
	if p.defaults.Model != "test-model" {
		t.Errorf("defaults.Model = %q, want %q", p.defaults.Model, "test-model")
	}
}

// floatPtr returns a pointer to a float64 value.
func floatPtr(f float64) *float64 {
	return &f
}
