// ABOUTME: Unit tests for SSE transport, event parsing, and error classification
// ABOUTME: Verifies chunk buffering, event type parsing, ChatError types, and abort support

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { parseSSEEvent, streamChat, ChatError, sendFeedback } from "./chatApi";

describe("parseSSEEvent", () => {
  it("parses a token event", () => {
    const raw = 'event: token\ndata: {"text":"hello"}';
    const result = parseSSEEvent(raw);
    expect(result).toEqual({ type: "token", data: { text: "hello" } });
  });

  it("parses a done event", () => {
    const raw = 'event: done\ndata: {"usage":{"tokens":42}}';
    const result = parseSSEEvent(raw);
    expect(result).toEqual({ type: "done", data: { usage: { tokens: 42 } } });
  });

  it("parses an error event", () => {
    const raw = 'event: error\ndata: {"message":"rate limited"}';
    const result = parseSSEEvent(raw);
    expect(result).toEqual({
      type: "error",
      data: { message: "rate limited" },
    });
  });

  it("returns null when no data line is present", () => {
    const raw = "event: token";
    expect(parseSSEEvent(raw)).toBeNull();
  });

  it("defaults to message type when no event line is present", () => {
    const raw = 'data: {"text":"hi"}';
    const result = parseSSEEvent(raw);
    expect(result).toEqual({ type: "message", data: { text: "hi" } });
  });

  it("returns null on malformed JSON data", () => {
    const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

    const raw = "event: token\ndata: {not valid json}";
    const result = parseSSEEvent(raw);

    expect(result).toBeNull();
    expect(warnSpy).toHaveBeenCalledWith(
      "Skipping malformed SSE data:",
      "{not valid json}",
      expect.any(String),
    );

    warnSpy.mockRestore();
  });
});

describe("streamChat", () => {
  let originalFetch;

  beforeEach(() => {
    originalFetch = global.fetch;
  });

  afterEach(() => {
    global.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  /**
   * Create a mock ReadableStream from an array of string chunks.
   */
  function mockReadableStream(chunks) {
    let index = 0;
    return {
      getReader() {
        return {
          read() {
            if (index < chunks.length) {
              const value = new TextEncoder().encode(chunks[index]);
              index++;
              return Promise.resolve({ done: false, value });
            }
            return Promise.resolve({ done: true });
          },
          releaseLock() {},
        };
      },
    };
  }

  it("yields correct token events in order", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: mockReadableStream([
        'event: token\ndata: {"text":"Hello"}\n\nevent: token\ndata: {"text":" world"}\n\nevent: done\ndata: {}\n\n',
      ]),
    });

    const events = [];
    for await (const event of streamChat([{ role: "user", content: "hi" }])) {
      events.push(event);
      if (event.type === "done") break;
    }

    expect(events).toEqual([
      { type: "token", data: { text: "Hello" } },
      { type: "token", data: { text: " world" } },
      { type: "done", data: {} },
    ]);
  });

  it("handles chunk boundary splits", async () => {
    // Event split across two reads
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: mockReadableStream([
        'event: token\ndata: {"tex',
        't":"split"}\n\nevent: done\ndata: {}\n\n',
      ]),
    });

    const events = [];
    for await (const event of streamChat([{ role: "user", content: "test" }])) {
      events.push(event);
      if (event.type === "done") break;
    }

    expect(events).toEqual([
      { type: "token", data: { text: "split" } },
      { type: "done", data: {} },
    ]);
  });

  it("throws ChatError with type 'rate_limit' on HTTP 429", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 429,
      json: () => Promise.resolve({ error: "Rate limit exceeded" }),
    });

    const gen = streamChat([{ role: "user", content: "hi" }]);
    try {
      await gen.next();
      expect.fail("should have thrown");
    } catch (err) {
      expect(err).toBeInstanceOf(ChatError);
      expect(err.type).toBe("rate_limit");
      expect(err.message).toBe("Rate limit exceeded");
    }
  });

  it("throws ChatError with type 'network' when fetch throws TypeError", async () => {
    global.fetch = vi.fn().mockRejectedValue(new TypeError("Failed to fetch"));

    const gen = streamChat([{ role: "user", content: "hi" }]);
    try {
      await gen.next();
      expect.fail("should have thrown");
    } catch (err) {
      expect(err).toBeInstanceOf(ChatError);
      expect(err.type).toBe("network");
      expect(err.message).toBe("Connection lost");
    }
  });

  it("throws ChatError with type 'server' for non-429 HTTP errors", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 503,
      json: () => Promise.resolve({ error: "AI provider unavailable" }),
    });

    const gen = streamChat([{ role: "user", content: "hi" }]);
    try {
      await gen.next();
      expect.fail("should have thrown");
    } catch (err) {
      expect(err).toBeInstanceOf(ChatError);
      expect(err.type).toBe("server");
      expect(err.message).toBe("AI provider unavailable");
    }
  });

  it("throws ChatError with fallback message for non-429 when body has no error", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
      json: () => Promise.reject(new Error("not json")),
    });

    const gen = streamChat([{ role: "user", content: "hi" }]);
    try {
      await gen.next();
      expect.fail("should have thrown");
    } catch (err) {
      expect(err).toBeInstanceOf(ChatError);
      expect(err.type).toBe("server");
      expect(err.message).toBe("Chat request failed: 500");
    }
  });

  it("yields SSE error event with code for mid-stream error classification", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: mockReadableStream([
        'event: token\ndata: {"text":"partial"}\n\nevent: error\ndata: {"code":"timeout","message":"Response timed out"}\n\n',
      ]),
    });

    const events = [];
    for await (const event of streamChat([{ role: "user", content: "hi" }])) {
      events.push(event);
    }

    expect(events).toHaveLength(2);
    expect(events[1]).toEqual({
      type: "error",
      data: { code: "timeout", message: "Response timed out" },
    });
  });

  it("yields SSE error event with provider_error code", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: mockReadableStream([
        'event: error\ndata: {"code":"provider_error","message":"Model overloaded"}\n\n',
      ]),
    });

    const events = [];
    for await (const event of streamChat([{ role: "user", content: "hi" }])) {
      events.push(event);
    }

    expect(events).toHaveLength(1);
    expect(events[0]).toEqual({
      type: "error",
      data: { code: "provider_error", message: "Model overloaded" },
    });
  });

  it("discards trailing buffer without terminator", async () => {
    // Stream ends with data in buffer that never gets a \n\n terminator
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: mockReadableStream([
        'event: token\ndata: {"text":"ok"}\n\nevent: token\ndata: {"text":"trailing"}',
      ]),
    });

    const events = [];
    for await (const event of streamChat([{ role: "user", content: "hi" }])) {
      events.push(event);
    }

    // Only the first event (terminated by \n\n) should be yielded
    expect(events).toEqual([{ type: "token", data: { text: "ok" } }]);
  });

  it("throws ChatError when response.body is null", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: null,
    });

    const gen = streamChat([{ role: "user", content: "hi" }]);
    try {
      await gen.next();
      expect.fail("should have thrown");
    } catch (err) {
      expect(err).toBeInstanceOf(ChatError);
      expect(err.type).toBe("server");
    }
  });

  it("throws ChatError with type 'network' when reader.read() throws TypeError mid-stream", async () => {
    let readCount = 0;
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: {
        getReader() {
          return {
            read() {
              readCount++;
              if (readCount === 1) {
                const value = new TextEncoder().encode(
                  'event: token\ndata: {"text":"partial"}\n\n',
                );
                return Promise.resolve({ done: false, value });
              }
              // Simulate mid-stream network drop
              return Promise.reject(new TypeError("network error"));
            },
            releaseLock() {},
          };
        },
      },
    });

    const events = [];
    try {
      for await (const event of streamChat([{ role: "user", content: "hi" }])) {
        events.push(event);
      }
      expect.fail("should have thrown");
    } catch (err) {
      expect(err).toBeInstanceOf(ChatError);
      expect(err.type).toBe("network");
      expect(err.message).toBe("Connection lost");
    }

    // Should have yielded the first token before the error
    expect(events).toHaveLength(1);
  });

  it("ChatError has message and type properties", () => {
    const err = new ChatError("test message", "rate_limit");
    expect(err).toBeInstanceOf(Error);
    expect(err.message).toBe("test message");
    expect(err.type).toBe("rate_limit");
    expect(err.name).toBe("ChatError");
  });

  it("ChatError defaults to 'server' type", () => {
    const err = new ChatError("generic error");
    expect(err.type).toBe("server");
  });

  it("includes CSRF header and credentials in request", async () => {
    // Set up CSRF cookie
    Object.defineProperty(document, "cookie", {
      value: "DIEGO_CSRF=test-token-123",
      writable: true,
    });

    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: mockReadableStream(["event: done\ndata: {}\n\n"]),
    });

    const gen = streamChat([{ role: "user", content: "hi" }]);
    // Consume all events
    for await (const event of gen) {
      if (event.type === "done") break;
    }

    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/api/v1/chat"),
      expect.objectContaining({
        method: "POST",
        credentials: "include",
        headers: expect.objectContaining({
          "Content-Type": "application/json",
          "X-CSRF-Token": "test-token-123",
        }),
      }),
    );
  });
});

describe("sendFeedback", () => {
  let originalFetch;

  beforeEach(() => {
    originalFetch = global.fetch;
    Object.defineProperty(document, "cookie", {
      value: "DIEGO_CSRF=feedback-csrf-token",
      writable: true,
    });
  });

  afterEach(() => {
    global.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it("sends POST with CSRF header, credentials, and mapped field names", async () => {
    global.fetch = vi.fn().mockResolvedValue({ ok: true });

    await sendFeedback({
      messageIndex: 2,
      rating: "up",
      truncatedQuestion: "What is capacity?",
    });

    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/api/v1/chat/feedback"),
      expect.objectContaining({
        method: "POST",
        credentials: "include",
        headers: expect.objectContaining({
          "Content-Type": "application/json",
          "X-CSRF-Token": "feedback-csrf-token",
        }),
        body: JSON.stringify({
          message_index: 2,
          rating: "up",
          truncated_question: "What is capacity?",
        }),
      }),
    );
  });

  it("does not throw on non-OK response (fire-and-forget)", async () => {
    global.fetch = vi.fn().mockResolvedValue({ ok: false, status: 500 });
    const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

    await expect(
      sendFeedback({ messageIndex: 0, rating: "down", truncatedQuestion: "" }),
    ).resolves.toBeUndefined();

    expect(warnSpy).toHaveBeenCalledWith("Feedback submission returned 500");
    warnSpy.mockRestore();
  });

  it("does not throw on network error (fire-and-forget)", async () => {
    global.fetch = vi.fn().mockRejectedValue(new TypeError("Failed to fetch"));
    const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

    await expect(
      sendFeedback({ messageIndex: 0, rating: "up", truncatedQuestion: "" }),
    ).resolves.toBeUndefined();

    expect(warnSpy).toHaveBeenCalledWith(
      "Feedback submission failed:",
      expect.any(TypeError),
    );
    warnSpy.mockRestore();
  });
});
