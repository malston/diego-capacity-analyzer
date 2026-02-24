// ABOUTME: ChatProvider interface and domain types for LLM provider abstraction
// ABOUTME: Defines Message, TokenEvent, Usage, ChatConfig, and Option types

package ai

import "context"

// Message represents a single conversation turn.
type Message struct {
	Role    string
	Content string
}

// TokenEvent carries a streaming token or terminal signal.
type TokenEvent struct {
	Text       string
	Done       bool
	StopReason string
	Usage      *Usage
	Err        error
}

// Usage tracks token consumption for cost monitoring.
type Usage struct {
	InputTokens  int64
	OutputTokens int64
}

// ChatProvider streams LLM responses token by token.
type ChatProvider interface {
	Chat(ctx context.Context, messages []Message, opts ...Option) <-chan TokenEvent
}

// Option configures a Chat request.
type Option func(*ChatConfig)

// ChatConfig holds resolved options for a single request.
type ChatConfig struct {
	MaxTokens   int64
	Temperature *float64
	System      string
	Model       string
}

// NewChatConfig creates a ChatConfig by applying all options in order.
func NewChatConfig(opts ...Option) ChatConfig {
	var cfg ChatConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}
