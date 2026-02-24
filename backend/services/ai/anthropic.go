// ABOUTME: Anthropic Claude provider implementing the ChatProvider interface
// ABOUTME: Streams token-by-token responses via the anthropic-sdk-go SDK

package ai

import (
	"context"
	"log/slog"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
)

// AnthropicProvider implements ChatProvider using the Anthropic Messages API.
type AnthropicProvider struct {
	client   anthropic.Client
	defaults ChatConfig
}

// NewAnthropicProvider creates an Anthropic provider with the given API key and
// default configuration. The defaults are used when Chat options omit a value.
func NewAnthropicProvider(apiKey string, defaults ChatConfig) *AnthropicProvider {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &AnthropicProvider{
		client:   client,
		defaults: defaults,
	}
}

// Chat starts a streaming conversation with the Anthropic API. It returns a
// channel immediately; a goroutine fills it with TokenEvent values as tokens
// arrive from the API. The final event has Done=true with StopReason and Usage.
func (p *AnthropicProvider) Chat(ctx context.Context, messages []Message, opts ...Option) <-chan TokenEvent {
	cfg := p.resolveConfig(opts)
	ch := make(chan TokenEvent, 1)

	go func() {
		defer close(ch)

		params := anthropic.MessageNewParams{
			Model:     anthropic.Model(cfg.Model),
			MaxTokens: cfg.MaxTokens,
			Messages:  toSDKMessages(messages),
		}

		if cfg.Temperature != nil {
			params.Temperature = param.NewOpt(*cfg.Temperature)
		}

		if cfg.System != "" {
			params.System = []anthropic.TextBlockParam{
				{Text: cfg.System},
			}
		}

		stream := p.client.Messages.NewStreaming(ctx, params)
		defer stream.Close()

		var accumulated anthropic.Message
		for stream.Next() {
			event := stream.Current()
			if err := accumulated.Accumulate(event); err != nil {
				p.send(ctx, ch, TokenEvent{Err: err, Done: true})
				return
			}

			switch variant := event.AsAny().(type) {
			case anthropic.ContentBlockDeltaEvent:
				switch delta := variant.Delta.AsAny().(type) {
				case anthropic.TextDelta:
					p.send(ctx, ch, TokenEvent{Text: delta.Text})
				}
			}
		}

		if err := stream.Err(); err != nil {
			p.send(ctx, ch, TokenEvent{Err: err, Done: true})
			return
		}

		usage := &Usage{
			InputTokens:  accumulated.Usage.InputTokens,
			OutputTokens: accumulated.Usage.OutputTokens,
		}
		slog.Info("chat completed",
			"input_tokens", usage.InputTokens,
			"output_tokens", usage.OutputTokens,
			"model", cfg.Model,
		)
		p.send(ctx, ch, TokenEvent{
			Done:       true,
			StopReason: string(accumulated.StopReason),
			Usage:      usage,
		})
	}()

	return ch
}

// toSDKMessages converts domain Messages to SDK MessageParams. System and
// unknown roles are skipped; system prompts are passed separately via the
// System field on MessageNewParams.
func toSDKMessages(msgs []Message) []anthropic.MessageParam {
	sdkMsgs := make([]anthropic.MessageParam, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case "user":
			sdkMsgs = append(sdkMsgs, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
		case "assistant":
			sdkMsgs = append(sdkMsgs, anthropic.NewAssistantMessage(anthropic.NewTextBlock(m.Content)))
		case "system":
			// System prompt is passed via MessageNewParams.System, not as a message.
		default:
			slog.Warn("skipping message with unknown role", "role", m.Role)
		}
	}
	return sdkMsgs
}

// resolveConfig merges user-provided options with provider defaults. User options
// take precedence; zero-value fields are filled from the provider defaults.
func (p *AnthropicProvider) resolveConfig(opts []Option) ChatConfig {
	cfg := NewChatConfig(opts...)

	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = p.defaults.MaxTokens
	}
	if cfg.Temperature == nil {
		cfg.Temperature = p.defaults.Temperature
	}
	if cfg.Model == "" {
		cfg.Model = p.defaults.Model
	}
	// System is not defaulted -- it is per-request only.

	return cfg
}

// send writes an event to the channel, respecting context cancellation to
// prevent goroutine leaks.
func (p *AnthropicProvider) send(ctx context.Context, ch chan<- TokenEvent, event TokenEvent) {
	select {
	case ch <- event:
	case <-ctx.Done():
	}
}
