// ABOUTME: Functional option constructors for ChatConfig
// ABOUTME: Provides WithMaxTokens, WithTemperature, WithSystem, WithModel

package ai

// WithMaxTokens sets the maximum number of output tokens for the request.
func WithMaxTokens(n int64) Option {
	return func(cfg *ChatConfig) {
		cfg.MaxTokens = n
	}
}

// WithTemperature sets the sampling temperature for the request.
func WithTemperature(t float64) Option {
	return func(cfg *ChatConfig) {
		cfg.Temperature = &t
	}
}

// WithSystem sets the system prompt for the request.
func WithSystem(s string) Option {
	return func(cfg *ChatConfig) {
		cfg.System = s
	}
}

// WithModel sets the model identifier for the request.
func WithModel(m string) Option {
	return func(cfg *ChatConfig) {
		cfg.Model = m
	}
}
