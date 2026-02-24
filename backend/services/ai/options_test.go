// ABOUTME: Unit tests for functional option constructors
// ABOUTME: Verifies WithMaxTokens, WithTemperature, WithSystem, WithModel modify ChatConfig correctly

package ai

import (
	"testing"
)

func TestWithMaxTokens(t *testing.T) {
	tests := []struct {
		name  string
		value int64
	}{
		{"standard value", 4096},
		{"small value", 100},
		{"large value", 100000},
		{"zero value", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewChatConfig(WithMaxTokens(tt.value))
			if cfg.MaxTokens != tt.value {
				t.Errorf("MaxTokens = %d, want %d", cfg.MaxTokens, tt.value)
			}
		})
	}
}

func TestWithTemperature(t *testing.T) {
	tests := []struct {
		name  string
		value float64
	}{
		{"low temperature", 0.0},
		{"default-like temperature", 0.3},
		{"mid temperature", 0.5},
		{"high temperature", 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewChatConfig(WithTemperature(tt.value))
			if cfg.Temperature == nil {
				t.Fatal("Temperature = nil, want non-nil")
			}
			if *cfg.Temperature != tt.value {
				t.Errorf("Temperature = %f, want %f", *cfg.Temperature, tt.value)
			}
		})
	}
}

func TestWithSystem(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"simple prompt", "You are a helpful assistant"},
		{"empty prompt", ""},
		{"multiline prompt", "Line one.\nLine two."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewChatConfig(WithSystem(tt.value))
			if cfg.System != tt.value {
				t.Errorf("System = %q, want %q", cfg.System, tt.value)
			}
		})
	}
}

func TestWithModel(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"claude model", "claude-3-7-sonnet-latest"},
		{"custom model", "my-custom-model"},
		{"empty model", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewChatConfig(WithModel(tt.value))
			if cfg.Model != tt.value {
				t.Errorf("Model = %q, want %q", cfg.Model, tt.value)
			}
		})
	}
}

func TestOptions_Compose(t *testing.T) {
	cfg := NewChatConfig(
		WithMaxTokens(1024),
		WithTemperature(0.5),
	)

	if cfg.MaxTokens != 1024 {
		t.Errorf("MaxTokens = %d, want %d", cfg.MaxTokens, 1024)
	}
	if cfg.Temperature == nil {
		t.Fatal("Temperature = nil, want non-nil")
	}
	if *cfg.Temperature != 0.5 {
		t.Errorf("Temperature = %f, want %f", *cfg.Temperature, 0.5)
	}
}

func TestOptions_LastWins(t *testing.T) {
	cfg := NewChatConfig(
		WithMaxTokens(1024),
		WithMaxTokens(2048),
	)

	if cfg.MaxTokens != 2048 {
		t.Errorf("MaxTokens = %d, want %d (last option should win)", cfg.MaxTokens, 2048)
	}
}
