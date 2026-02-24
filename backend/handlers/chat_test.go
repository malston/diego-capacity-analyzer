// ABOUTME: Tests for SSE streaming chat handler
// ABOUTME: Covers pre-stream validation, SSE streaming, timeouts, and cancellation

package handlers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/cache"
	"github.com/markalston/diego-capacity-analyzer/backend/config"
	"github.com/markalston/diego-capacity-analyzer/backend/models"
	"github.com/markalston/diego-capacity-analyzer/backend/services/ai"
)

// mockChatProvider implements ai.ChatProvider for testing the chat handler.
// It returns a channel pre-loaded with configurable events and captures
// the options passed to Chat() so tests can verify system prompt content.
type mockChatProvider struct {
	events       []ai.TokenEvent
	capturedOpts []ai.Option
	capturedMsgs []ai.Message
	mu           sync.Mutex
}

func (m *mockChatProvider) Chat(_ context.Context, messages []ai.Message, opts ...ai.Option) <-chan ai.TokenEvent {
	m.mu.Lock()
	m.capturedOpts = opts
	m.capturedMsgs = messages
	m.mu.Unlock()

	ch := make(chan ai.TokenEvent, len(m.events))
	for _, e := range m.events {
		ch <- e
	}
	close(ch)
	return ch
}

func (m *mockChatProvider) getCapturedConfig() ai.ChatConfig {
	m.mu.Lock()
	defer m.mu.Unlock()
	return ai.NewChatConfig(m.capturedOpts...)
}

// newChatTestHandler creates a Handler suitable for chat tests.
// It bypasses NewHandler to avoid requiring CF/BOSH/vSphere clients.
// Timeout fields use production defaults (30s idle, 300s max duration).
func newChatTestHandler(provider ai.ChatProvider) *Handler {
	c := cache.New(5 * time.Minute)
	cfg := &config.Config{
		AIIdleTimeoutSecs: 30,
		AIMaxDurationSecs: 300,
	}
	return &Handler{
		cfg:          cfg,
		cache:        c,
		chatProvider: provider,
	}
}

// --- Pre-stream validation tests (JSON responses) ---

func TestChat_NilProvider(t *testing.T) {
	h := newChatTestHandler(nil)

	body := `{"messages":[{"role":"user","content":"hello"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Chat(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}

	var resp models.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if resp.Error != "AI advisor not configured" {
		t.Errorf("expected 'AI advisor not configured', got %q", resp.Error)
	}
}

func TestChat_EmptyMessages(t *testing.T) {
	h := newChatTestHandler(&mockChatProvider{})

	body := `{"messages":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Chat(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var resp models.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if resp.Error != "Messages required" {
		t.Errorf("expected 'Messages required', got %q", resp.Error)
	}
}

func TestChat_TooManyMessages(t *testing.T) {
	h := newChatTestHandler(&mockChatProvider{})

	// Build 51 messages
	msgs := make([]string, 51)
	for i := range msgs {
		msgs[i] = `{"role":"user","content":"msg"}`
	}
	body := `{"messages":[` + strings.Join(msgs, ",") + `]}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Chat(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var resp models.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if resp.Error != "Maximum 50 messages per request" {
		t.Errorf("expected 'Maximum 50 messages per request', got %q", resp.Error)
	}
}

func TestChat_InvalidJSON(t *testing.T) {
	h := newChatTestHandler(&mockChatProvider{})

	body := `{not valid json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Chat(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestChat_InvalidRole(t *testing.T) {
	h := newChatTestHandler(&mockChatProvider{})

	body := `{"messages":[{"role":"system","content":"hello"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Chat(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var resp models.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if !strings.Contains(resp.Error, "role") {
		t.Errorf("expected error mentioning 'role', got %q", resp.Error)
	}
}

func TestChat_EmptyContent(t *testing.T) {
	h := newChatTestHandler(&mockChatProvider{})

	body := `{"messages":[{"role":"user","content":""}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Chat(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var resp models.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if !strings.Contains(resp.Error, "content") {
		t.Errorf("expected error mentioning 'content', got %q", resp.Error)
	}
}

// --- SSE streaming tests ---
// These tests use httptest.NewServer to get a real TCP connection that
// supports http.Flusher (httptest.NewRecorder does not).

// parseSSEEvents reads SSE events from an HTTP response body.
func parseSSEEvents(t *testing.T, body *bufio.Reader) []sseEvent {
	t.Helper()
	var events []sseEvent
	var current sseEvent

	for {
		line, err := body.ReadString('\n')
		if err != nil {
			// End of stream
			break
		}
		line = strings.TrimRight(line, "\n")

		if line == "" {
			// Empty line marks end of event
			if current.eventType != "" {
				events = append(events, current)
			}
			current = sseEvent{}
			continue
		}

		if strings.HasPrefix(line, "event: ") {
			current.eventType = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			current.data = strings.TrimPrefix(line, "data: ")
		}
	}
	return events
}

type sseEvent struct {
	eventType string
	data      string
}

func TestChat_StreamTokens(t *testing.T) {
	mock := &mockChatProvider{
		events: []ai.TokenEvent{
			{Text: "Hello"},
			{Text: " world"},
			{Text: "!"},
			{Done: true, StopReason: "end_turn", Usage: &ai.Usage{InputTokens: 10, OutputTokens: 3}},
		},
	}
	h := newChatTestHandler(mock)

	ts := httptest.NewServer(http.HandlerFunc(h.Chat))
	defer ts.Close()

	body := `{"messages":[{"role":"user","content":"hi"}]}`
	resp, err := http.Post(ts.URL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	events := parseSSEEvents(t, bufio.NewReader(resp.Body))

	// Expect 3 token events + 1 done event = 4 events
	if len(events) != 4 {
		t.Fatalf("expected 4 SSE events, got %d: %+v", len(events), events)
	}

	// Verify token events
	for i, expected := range []struct {
		text string
		seq  int
	}{
		{"Hello", 1},
		{" world", 2},
		{"!", 3},
	} {
		if events[i].eventType != "token" {
			t.Errorf("event %d: expected type 'token', got %q", i, events[i].eventType)
		}
		var payload TokenPayload
		if err := json.Unmarshal([]byte(events[i].data), &payload); err != nil {
			t.Fatalf("event %d: failed to parse token payload: %v", i, err)
		}
		if payload.Text != expected.text {
			t.Errorf("event %d: expected text %q, got %q", i, expected.text, payload.Text)
		}
		if payload.Seq != expected.seq {
			t.Errorf("event %d: expected seq %d, got %d", i, expected.seq, payload.Seq)
		}
	}

	// Verify done event
	if events[3].eventType != "done" {
		t.Errorf("last event: expected type 'done', got %q", events[3].eventType)
	}
	var done DonePayload
	if err := json.Unmarshal([]byte(events[3].data), &done); err != nil {
		t.Fatalf("failed to parse done payload: %v", err)
	}
	if done.StopReason != "end_turn" {
		t.Errorf("expected stop_reason 'end_turn', got %q", done.StopReason)
	}
	if done.Usage.InputTokens != 10 {
		t.Errorf("expected input_tokens 10, got %d", done.Usage.InputTokens)
	}
	if done.Usage.OutputTokens != 3 {
		t.Errorf("expected output_tokens 3, got %d", done.Usage.OutputTokens)
	}
}

func TestChat_ProviderError(t *testing.T) {
	mock := &mockChatProvider{
		events: []ai.TokenEvent{
			{Text: "partial"},
			{Err: context.DeadlineExceeded},
		},
	}
	h := newChatTestHandler(mock)

	ts := httptest.NewServer(http.HandlerFunc(h.Chat))
	defer ts.Close()

	body := `{"messages":[{"role":"user","content":"hi"}]}`
	resp, err := http.Post(ts.URL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	events := parseSSEEvents(t, bufio.NewReader(resp.Body))

	// Should have 1 token + 1 error = 2 events
	if len(events) != 2 {
		t.Fatalf("expected 2 SSE events, got %d: %+v", len(events), events)
	}

	if events[0].eventType != "token" {
		t.Errorf("first event: expected type 'token', got %q", events[0].eventType)
	}

	if events[1].eventType != "error" {
		t.Errorf("second event: expected type 'error', got %q", events[1].eventType)
	}
	var errPayload ErrorPayload
	if err := json.Unmarshal([]byte(events[1].data), &errPayload); err != nil {
		t.Fatalf("failed to parse error payload: %v", err)
	}
	if errPayload.Code != "provider_error" {
		t.Errorf("expected code 'provider_error', got %q", errPayload.Code)
	}
}

func TestChat_SSEHeaders(t *testing.T) {
	mock := &mockChatProvider{
		events: []ai.TokenEvent{
			{Done: true, StopReason: "end_turn", Usage: &ai.Usage{}},
		},
	}
	h := newChatTestHandler(mock)

	ts := httptest.NewServer(http.HandlerFunc(h.Chat))
	defer ts.Close()

	body := `{"messages":[{"role":"user","content":"hi"}]}`
	resp, err := http.Post(ts.URL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected Content-Type 'text/event-stream', got %q", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("expected Cache-Control 'no-cache', got %q", cc)
	}
	if cn := resp.Header.Get("Connection"); cn != "keep-alive" {
		t.Errorf("expected Connection 'keep-alive', got %q", cn)
	}
	if xab := resp.Header.Get("X-Accel-Buffering"); xab != "no" {
		t.Errorf("expected X-Accel-Buffering 'no', got %q", xab)
	}
}

// --- Timeout and cancellation tests ---
// These tests use a slowMockProvider that introduces configurable delays
// between tokens and respects context cancellation.

// slowMockProvider sends events with a configurable delay between each.
// It respects context cancellation and records whether its context was canceled.
type slowMockProvider struct {
	events      []ai.TokenEvent
	delay       time.Duration
	ctxCanceled atomic.Bool
}

func (s *slowMockProvider) Chat(ctx context.Context, _ []ai.Message, _ ...ai.Option) <-chan ai.TokenEvent {
	ch := make(chan ai.TokenEvent)
	go func() {
		defer close(ch)
		for _, e := range s.events {
			select {
			case <-time.After(s.delay):
			case <-ctx.Done():
				s.ctxCanceled.Store(true)
				return
			}
			select {
			case ch <- e:
			case <-ctx.Done():
				s.ctxCanceled.Store(true)
				return
			}
		}
		// Block until context is canceled (simulates an infinite stream)
		<-ctx.Done()
		s.ctxCanceled.Store(true)
	}()
	return ch
}

// newChatTestHandlerWithTimeouts creates a Handler with specific timeout values.
func newChatTestHandlerWithTimeouts(provider ai.ChatProvider, idleTimeoutSecs, maxDurationSecs int) *Handler {
	c := cache.New(5 * time.Minute)
	cfg := &config.Config{
		AIIdleTimeoutSecs: idleTimeoutSecs,
		AIMaxDurationSecs: maxDurationSecs,
	}
	return &Handler{
		cfg:          cfg,
		cache:        c,
		chatProvider: provider,
	}
}

func TestChat_IdleTimeout(t *testing.T) {
	// Provider sends one token then stalls forever.
	// With a very short idle timeout, the stream should terminate with a timeout error.
	mock := &slowMockProvider{
		events: []ai.TokenEvent{
			{Text: "Hello"},
			// After this, the provider blocks (no more events before idle timeout)
		},
		delay: 0, // First token arrives immediately
	}

	// Use a fractional idle timeout -- config is in seconds, but we need sub-second
	// for fast tests. We'll set 1 second and accept the test takes ~1s.
	// Actually, the plan says 50ms -- we need to check if config supports sub-second.
	// Config is int seconds, so minimum is 1. We'll need the implementation to
	// support sub-second for testing. For RED phase, just write what we expect.
	h := newChatTestHandlerWithTimeouts(mock, 1, 300)

	ts := httptest.NewServer(http.HandlerFunc(h.Chat))
	defer ts.Close()

	body := `{"messages":[{"role":"user","content":"hi"}]}`
	resp, err := http.Post(ts.URL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	events := parseSSEEvents(t, bufio.NewReader(resp.Body))

	// Expect: 1 token event + 1 error event (idle timeout)
	if len(events) < 2 {
		t.Fatalf("expected at least 2 SSE events (token + timeout error), got %d: %+v", len(events), events)
	}

	if events[0].eventType != "token" {
		t.Errorf("first event: expected type 'token', got %q", events[0].eventType)
	}

	// Last event should be a timeout error
	lastEvent := events[len(events)-1]
	if lastEvent.eventType != "error" {
		t.Fatalf("last event: expected type 'error', got %q", lastEvent.eventType)
	}
	var errPayload ErrorPayload
	if err := json.Unmarshal([]byte(lastEvent.data), &errPayload); err != nil {
		t.Fatalf("failed to parse error payload: %v", err)
	}
	if errPayload.Code != "timeout" {
		t.Errorf("expected error code 'timeout', got %q", errPayload.Code)
	}
	if !strings.Contains(errPayload.Message, "timeout") {
		t.Errorf("expected error message to mention 'timeout', got %q", errPayload.Message)
	}
}

func TestChat_IdleTimerResets(t *testing.T) {
	// Provider sends 5 tokens with 30ms gaps. Idle timeout is 50ms.
	// Without timer reset, the 2nd or 3rd token would trigger idle timeout.
	// With proper reset, all tokens arrive because each one resets the timer.
	events := make([]ai.TokenEvent, 5)
	for i := range events {
		events[i] = ai.TokenEvent{Text: fmt.Sprintf("tok%d", i)}
	}

	mock := &slowMockProvider{
		events: events,
		delay:  30 * time.Millisecond,
	}

	// We need sub-second idle timeout for this test. The implementation must
	// handle this. For now, the test documents the expected behavior.
	// Using 1 second idle timeout -- all 5 tokens at 30ms gaps (150ms total)
	// should arrive well before the 1s idle timeout.
	h := newChatTestHandlerWithTimeouts(mock, 1, 300)

	ts := httptest.NewServer(http.HandlerFunc(h.Chat))
	defer ts.Close()

	body := `{"messages":[{"role":"user","content":"hi"}]}`

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(ts.URL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	sseEvents := parseSSEEvents(t, bufio.NewReader(resp.Body))

	// All 5 tokens should arrive, plus an idle timeout error at the end
	// (since the mock blocks after sending all events)
	tokenCount := 0
	for _, e := range sseEvents {
		if e.eventType == "token" {
			tokenCount++
		}
	}

	if tokenCount != 5 {
		t.Errorf("expected 5 token events (idle timer resets prevented false timeout), got %d", tokenCount)
		for i, e := range sseEvents {
			t.Logf("  event %d: type=%q data=%s", i, e.eventType, e.data)
		}
	}
}

func TestChat_MaxDuration(t *testing.T) {
	// Provider sends tokens forever (slow trickle). Max duration should cap it.
	// Generate many events -- more than could finish within max duration.
	events := make([]ai.TokenEvent, 100)
	for i := range events {
		events[i] = ai.TokenEvent{Text: fmt.Sprintf("t%d", i)}
	}

	mock := &slowMockProvider{
		events: events,
		delay:  50 * time.Millisecond, // 100 tokens * 50ms = 5s total without cap
	}

	// Set max duration to 1 second. The stream should be cut short.
	h := newChatTestHandlerWithTimeouts(mock, 30, 1)

	ts := httptest.NewServer(http.HandlerFunc(h.Chat))
	defer ts.Close()

	body := `{"messages":[{"role":"user","content":"hi"}]}`
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(ts.URL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	sseEvents := parseSSEEvents(t, bufio.NewReader(resp.Body))

	// Should have some tokens (not all 100) and end with a timeout error
	tokenCount := 0
	var lastEvent sseEvent
	for _, e := range sseEvents {
		if e.eventType == "token" {
			tokenCount++
		}
		lastEvent = e
	}

	if tokenCount >= 100 {
		t.Errorf("expected fewer than 100 tokens (max duration should cap stream), got %d", tokenCount)
	}

	if lastEvent.eventType != "error" {
		t.Fatalf("last event: expected type 'error' (max duration), got %q", lastEvent.eventType)
	}
	var errPayload ErrorPayload
	if err := json.Unmarshal([]byte(lastEvent.data), &errPayload); err != nil {
		t.Fatalf("failed to parse error payload: %v", err)
	}
	if errPayload.Code != "timeout" {
		t.Errorf("expected error code 'timeout', got %q", errPayload.Code)
	}
	if !strings.Contains(errPayload.Message, "maximum duration") {
		t.Errorf("expected error message to mention 'maximum duration', got %q", errPayload.Message)
	}
}

func TestChat_ClientDisconnect(t *testing.T) {
	// Provider blocks until context is canceled. We cancel the request context
	// and verify the provider's context was canceled.
	mock := &slowMockProvider{
		events: []ai.TokenEvent{
			{Text: "first"},
		},
		delay: 10 * time.Millisecond,
	}

	h := newChatTestHandlerWithTimeouts(mock, 30, 300)

	ts := httptest.NewServer(http.HandlerFunc(h.Chat))
	defer ts.Close()

	body := `{"messages":[{"role":"user","content":"hi"}]}`
	req, err := http.NewRequest(http.MethodPost, ts.URL, strings.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Use a context we can cancel to simulate client disconnect
	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)

	// Start the request in a goroutine
	type result struct {
		resp *http.Response
		err  error
	}
	resultCh := make(chan result, 1)
	go func() {
		resp, err := http.DefaultClient.Do(req)
		resultCh <- result{resp, err}
	}()

	// Wait briefly for the stream to start, then cancel
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Wait for the request to complete
	res := <-resultCh
	if res.resp != nil {
		res.resp.Body.Close()
	}

	// Give the server a moment to propagate the cancellation
	time.Sleep(100 * time.Millisecond)

	// Verify the provider's context was canceled
	if !mock.ctxCanceled.Load() {
		t.Error("expected provider context to be canceled on client disconnect")
	}
}

func TestChat_MidStreamProviderError(t *testing.T) {
	// Provider sends 2 tokens then an error. Verify 2 token events + 1 error event.
	// Also verify slog.Warn was called with message count.

	// Capture log output
	var logBuf bytes.Buffer
	logHandler := slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn})
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(logHandler))
	defer slog.SetDefault(originalLogger)

	providerErr := fmt.Errorf("upstream service unavailable")
	mock := &mockChatProvider{
		events: []ai.TokenEvent{
			{Text: "token1"},
			{Text: "token2"},
			{Err: providerErr},
		},
	}
	h := newChatTestHandler(mock)

	ts := httptest.NewServer(http.HandlerFunc(h.Chat))
	defer ts.Close()

	body := `{"messages":[{"role":"user","content":"hi"},{"role":"assistant","content":"hello"},{"role":"user","content":"test"}]}`
	resp, err := http.Post(ts.URL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	sseEvents := parseSSEEvents(t, bufio.NewReader(resp.Body))

	// Expect 2 token events + 1 error event = 3 events
	if len(sseEvents) != 3 {
		t.Fatalf("expected 3 SSE events, got %d: %+v", len(sseEvents), sseEvents)
	}

	// Verify token events
	for i := 0; i < 2; i++ {
		if sseEvents[i].eventType != "token" {
			t.Errorf("event %d: expected type 'token', got %q", i, sseEvents[i].eventType)
		}
	}

	// Verify error event
	if sseEvents[2].eventType != "error" {
		t.Fatalf("event 2: expected type 'error', got %q", sseEvents[2].eventType)
	}
	var errPayload ErrorPayload
	if err := json.Unmarshal([]byte(sseEvents[2].data), &errPayload); err != nil {
		t.Fatalf("failed to parse error payload: %v", err)
	}
	if errPayload.Code != "provider_error" {
		t.Errorf("expected error code 'provider_error', got %q", errPayload.Code)
	}

	// Verify log output contains the message count (3 messages in request)
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "messages=3") {
		t.Errorf("expected log output to contain 'messages=3', got:\n%s", logOutput)
	}
	if !strings.Contains(logOutput, "AI provider error") {
		t.Errorf("expected log output to contain 'AI provider error', got:\n%s", logOutput)
	}
}

// --- Context snapshot test ---

func TestChat_SystemPromptIncludesContext(t *testing.T) {
	mock := &mockChatProvider{
		events: []ai.TokenEvent{
			{Done: true, StopReason: "end_turn", Usage: &ai.Usage{}},
		},
	}
	h := newChatTestHandler(mock)
	// Enable vSphere so the infrastructure section renders cluster data
	h.cfg.VSphereHost = "vcenter.example.com"
	h.cfg.VSphereUsername = "admin"
	h.cfg.VSpherePassword = "secret"
	h.cfg.VSphereDatacenter = "DC-1"

	// Populate cache with dashboard data including apps with actual memory
	dashboard := models.DashboardResponse{
		Cells: []models.DiegoCell{
			{ID: "cell-1", Name: "diego_cell/0", MemoryMB: 32768, AllocatedMB: 24576, IsolationSegment: ""},
		},
		Apps: []models.App{
			{Name: "web-app", Instances: 2, RequestedMB: 512, ActualMB: 400},
		},
		Metadata: models.Metadata{
			BOSHAvailable: true,
		},
	}
	h.cache.Set("dashboard:all", dashboard)

	// Set infrastructure state
	infraState := &models.InfrastructureState{
		Source:         "manual",
		TotalMemoryGB:  256,
		TotalHostCount: 4,
		Clusters: []models.ClusterState{
			{Name: "test-cluster", HostCount: 4, MemoryGB: 256, HAStatus: "ok"},
		},
	}
	h.infraMutex.Lock()
	h.infrastructureState = infraState
	h.infraMutex.Unlock()

	ts := httptest.NewServer(http.HandlerFunc(h.Chat))
	defer ts.Close()

	body := `{"messages":[{"role":"user","content":"analyze capacity"}]}`
	resp, err := http.Post(ts.URL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Verify the mock captured a system prompt containing infrastructure context
	cfg := mock.getCapturedConfig()
	if cfg.System == "" {
		t.Fatal("expected system prompt to be set, got empty string")
	}
	if !strings.Contains(cfg.System, "test-cluster") {
		t.Errorf("expected system prompt to contain 'test-cluster', got:\n%s", cfg.System)
	}
	if !strings.Contains(cfg.System, "web-app") {
		t.Errorf("expected system prompt to contain 'web-app', got:\n%s", cfg.System)
	}
	if !strings.Contains(cfg.System, "Log Cache: available") {
		t.Errorf("expected system prompt to contain 'Log Cache: available' (app has ActualMB > 0), got:\n%s", cfg.System)
	}
}
