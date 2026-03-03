// ABOUTME: Unit tests for the chat stream React hook
// ABOUTME: Verifies message state, streaming lifecycle, error classification, reset, and retry

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useChatStream } from "./useChatStream";

// Mock the streamChat async generator and ChatError export
vi.mock("../services/chatApi", () => {
  class MockChatError extends Error {
    constructor(message, type = "server") {
      super(message);
      this.name = "ChatError";
      this.type = type;
    }
  }
  return {
    streamChat: vi.fn(),
    ChatError: MockChatError,
  };
});

import { streamChat, ChatError } from "../services/chatApi";

describe("useChatStream", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("starts with empty messages and not streaming", () => {
    const { result } = renderHook(() => useChatStream());

    expect(result.current.messages).toEqual([]);
    expect(result.current.isStreaming).toBe(false);
  });

  it("adds user and assistant messages on sendMessage", async () => {
    // Mock streamChat to yield done immediately
    streamChat.mockImplementation(async function* () {
      yield { type: "done", data: {} };
    });

    const { result } = renderHook(() => useChatStream());

    await act(async () => {
      await result.current.sendMessage("Hello");
    });

    expect(result.current.messages).toHaveLength(2);
    expect(result.current.messages[0]).toMatchObject({
      role: "user",
      content: "Hello",
    });
    expect(result.current.messages[1]).toMatchObject({
      role: "assistant",
      content: "",
    });
    expect(result.current.messages[0].timestamp).toBeTypeOf("number");
    expect(result.current.messages[0].id).toBeTypeOf("string");
    expect(result.current.messages[1].id).toBeTypeOf("string");
    expect(result.current.messages[0].id).not.toBe(
      result.current.messages[1].id,
    );
  });

  it("appends tokens to assistant message content", async () => {
    streamChat.mockImplementation(async function* () {
      yield { type: "token", data: { text: "Hi " } };
      yield { type: "token", data: { text: "there!" } };
      yield { type: "done", data: {} };
    });

    const { result } = renderHook(() => useChatStream());

    await act(async () => {
      await result.current.sendMessage("Hello");
    });

    expect(result.current.messages[1].content).toBe("Hi there!");
  });

  it("sets isStreaming to true during streaming and false after", async () => {
    let resolve;
    const gate = new Promise((r) => {
      resolve = r;
    });

    streamChat.mockImplementation(async function* () {
      yield { type: "token", data: { text: "tok" } };
      await gate;
      yield { type: "done", data: {} };
    });

    const { result } = renderHook(() => useChatStream());

    let sendPromise;
    act(() => {
      sendPromise = result.current.sendMessage("test");
    });

    // Give React time to process the state update
    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });

    expect(result.current.isStreaming).toBe(true);

    await act(async () => {
      resolve();
      await sendPromise;
    });

    expect(result.current.isStreaming).toBe(false);
  });

  it("accumulates messages across multiple sends", async () => {
    streamChat.mockImplementation(async function* () {
      yield { type: "token", data: { text: "Response" } };
      yield { type: "done", data: {} };
    });

    const { result } = renderHook(() => useChatStream());

    await act(async () => {
      await result.current.sendMessage("First");
    });

    expect(result.current.messages).toHaveLength(2);

    await act(async () => {
      await result.current.sendMessage("Second");
    });

    expect(result.current.messages).toHaveLength(4);
    expect(result.current.messages[0].content).toBe("First");
    expect(result.current.messages[1].content).toBe("Response");
    expect(result.current.messages[2].content).toBe("Second");
    expect(result.current.messages[3].content).toBe("Response");
  });

  it("silently catches AbortError", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});

    // eslint-disable-next-line require-yield
    streamChat.mockImplementation(async function* () {
      const error = new Error("aborted");
      error.name = "AbortError";
      throw error;
    });

    const { result } = renderHook(() => useChatStream());

    await act(async () => {
      await result.current.sendMessage("test");
    });

    expect(consoleSpy).not.toHaveBeenCalled();
    expect(result.current.isStreaming).toBe(false);
  });

  it("logs non-abort errors to console", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});

    // eslint-disable-next-line require-yield
    streamChat.mockImplementation(async function* () {
      throw new Error("network failure");
    });

    const { result } = renderHook(() => useChatStream());

    await act(async () => {
      await result.current.sendMessage("test");
    });

    expect(consoleSpy).toHaveBeenCalledWith(
      "Chat stream error:",
      expect.any(Error),
    );
    expect(result.current.isStreaming).toBe(false);
  });

  it("sets error state as { message, type } on ChatError", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {});

    // eslint-disable-next-line require-yield
    streamChat.mockImplementation(async function* () {
      throw new ChatError("Rate limit exceeded", "rate_limit");
    });

    const { result } = renderHook(() => useChatStream());

    expect(result.current.error).toBeNull();

    await act(async () => {
      await result.current.sendMessage("test");
    });

    expect(result.current.error).toEqual({
      message: "Rate limit exceeded",
      type: "rate_limit",
    });
  });

  it("sets error state as { message, type: 'server' } on non-ChatError", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {});

    // eslint-disable-next-line require-yield
    streamChat.mockImplementation(async function* () {
      throw new Error("server exploded");
    });

    const { result } = renderHook(() => useChatStream());

    expect(result.current.error).toBeNull();

    await act(async () => {
      await result.current.sendMessage("test");
    });

    expect(result.current.error).toEqual({
      message: "server exploded",
      type: "server",
    });
  });

  it("preserves partial content when SSE error event occurs mid-stream", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {});

    streamChat.mockImplementation(async function* () {
      yield { type: "token", data: { text: "partial " } };
      yield { type: "token", data: { text: "content" } };
      yield { type: "error", data: { message: "rate limited" } };
    });

    const { result } = renderHook(() => useChatStream());

    await act(async () => {
      await result.current.sendMessage("test");
    });

    // Partial content preserved in assistant message
    expect(result.current.messages[1].content).toBe("partial content");
    // Error state set as typed object from SSE error event
    expect(result.current.error).toEqual({
      message: "rate limited",
      type: "server",
    });
    expect(result.current.isStreaming).toBe(false);
  });

  it("clears error on next sendMessage", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {});

    // First call errors
    // eslint-disable-next-line require-yield
    streamChat.mockImplementationOnce(async function* () {
      throw new Error("fail");
    });

    const { result } = renderHook(() => useChatStream());

    await act(async () => {
      await result.current.sendMessage("first");
    });

    expect(result.current.error).toEqual({ message: "fail", type: "server" });

    // Second call succeeds
    streamChat.mockImplementationOnce(async function* () {
      yield { type: "done", data: {} };
    });

    await act(async () => {
      await result.current.sendMessage("second");
    });

    expect(result.current.error).toBeNull();
  });

  it("does not set error on AbortError", async () => {
    // eslint-disable-next-line require-yield
    streamChat.mockImplementation(async function* () {
      const error = new Error("aborted");
      error.name = "AbortError";
      throw error;
    });

    const { result } = renderHook(() => useChatStream());

    await act(async () => {
      await result.current.sendMessage("test");
    });

    expect(result.current.error).toBeNull();
  });

  it("maps mid-stream SSE timeout error to type 'timeout'", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {});

    streamChat.mockImplementation(async function* () {
      yield { type: "token", data: { text: "partial" } };
      yield {
        type: "error",
        data: { code: "timeout", message: "Response timed out" },
      };
    });

    const { result } = renderHook(() => useChatStream());

    await act(async () => {
      await result.current.sendMessage("test");
    });

    expect(result.current.error).toEqual({
      message: "Response timed out",
      type: "timeout",
    });
  });

  it("maps mid-stream SSE provider_error to type 'server'", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {});

    streamChat.mockImplementation(async function* () {
      yield {
        type: "error",
        data: { code: "provider_error", message: "Model overloaded" },
      };
    });

    const { result } = renderHook(() => useChatStream());

    await act(async () => {
      await result.current.sendMessage("test");
    });

    expect(result.current.error).toEqual({
      message: "Model overloaded",
      type: "server",
    });
  });

  describe("clearConversation", () => {
    it("resets messages to empty array", async () => {
      streamChat.mockImplementation(async function* () {
        yield { type: "token", data: { text: "Response" } };
        yield { type: "done", data: {} };
      });

      const { result } = renderHook(() => useChatStream());

      await act(async () => {
        await result.current.sendMessage("Hello");
      });

      expect(result.current.messages).toHaveLength(2);

      act(() => {
        result.current.clearConversation();
      });

      expect(result.current.messages).toEqual([]);
    });

    it("sets isStreaming to false", async () => {
      let resolve;
      const gate = new Promise((r) => {
        resolve = r;
      });

      streamChat.mockImplementation(async function* () {
        yield { type: "token", data: { text: "tok" } };
        await gate;
        yield { type: "done", data: {} };
      });

      const { result } = renderHook(() => useChatStream());

      let sendPromise;
      act(() => {
        sendPromise = result.current.sendMessage("test");
      });

      await act(async () => {
        await new Promise((r) => setTimeout(r, 0));
      });

      expect(result.current.isStreaming).toBe(true);

      act(() => {
        result.current.clearConversation();
      });

      expect(result.current.isStreaming).toBe(false);

      // Clean up the pending promise
      resolve();
      await act(async () => {
        await sendPromise.catch(() => {});
      });
    });

    it("sets error to null", async () => {
      vi.spyOn(console, "error").mockImplementation(() => {});

      // eslint-disable-next-line require-yield
      streamChat.mockImplementation(async function* () {
        throw new Error("fail");
      });

      const { result } = renderHook(() => useChatStream());

      await act(async () => {
        await result.current.sendMessage("test");
      });

      expect(result.current.error).not.toBeNull();

      act(() => {
        result.current.clearConversation();
      });

      expect(result.current.error).toBeNull();
    });

    it("aborts the active abort controller", async () => {
      let resolve;
      const gate = new Promise((r) => {
        resolve = r;
      });

      let capturedSignal;
      streamChat.mockImplementation(async function* (messages, signal) {
        capturedSignal = signal;
        yield { type: "token", data: { text: "tok" } };
        await gate;
        yield { type: "done", data: {} };
      });

      const { result } = renderHook(() => useChatStream());

      let sendPromise;
      act(() => {
        sendPromise = result.current.sendMessage("test");
      });

      await act(async () => {
        await new Promise((r) => setTimeout(r, 0));
      });

      expect(capturedSignal.aborted).toBe(false);

      act(() => {
        result.current.clearConversation();
      });

      expect(capturedSignal.aborted).toBe(true);

      // Clean up
      resolve();
      await act(async () => {
        await sendPromise.catch(() => {});
      });
    });
  });

  describe("retryLastMessage", () => {
    it("removes the last assistant message from messages", async () => {
      vi.spyOn(console, "error").mockImplementation(() => {});

      // First call errors after adding messages
      // eslint-disable-next-line require-yield
      streamChat.mockImplementationOnce(async function* () {
        throw new ChatError("fail", "server");
      });

      // Second call (retry) succeeds
      streamChat.mockImplementationOnce(async function* () {
        yield { type: "token", data: { text: "Success" } };
        yield { type: "done", data: {} };
      });

      const { result } = renderHook(() => useChatStream());

      await act(async () => {
        await result.current.sendMessage("Hello");
      });

      // Should have user + failed assistant
      expect(result.current.messages).toHaveLength(2);
      const failedAssistantId = result.current.messages[1].id;

      await act(async () => {
        await result.current.retryLastMessage();
      });

      // The failed assistant message should be gone;
      // new user + assistant pair from retry
      const messageIds = result.current.messages.map((m) => m.id);
      expect(messageIds).not.toContain(failedAssistantId);
    });

    it("clears the error state", async () => {
      vi.spyOn(console, "error").mockImplementation(() => {});

      // eslint-disable-next-line require-yield
      streamChat.mockImplementationOnce(async function* () {
        throw new ChatError("fail", "server");
      });

      streamChat.mockImplementationOnce(async function* () {
        yield { type: "done", data: {} };
      });

      const { result } = renderHook(() => useChatStream());

      await act(async () => {
        await result.current.sendMessage("Hello");
      });

      expect(result.current.error).not.toBeNull();

      await act(async () => {
        await result.current.retryLastMessage();
      });

      expect(result.current.error).toBeNull();
    });

    it("calls sendMessage with the last user message content", async () => {
      vi.spyOn(console, "error").mockImplementation(() => {});

      // eslint-disable-next-line require-yield
      streamChat.mockImplementationOnce(async function* () {
        throw new ChatError("fail", "server");
      });

      streamChat.mockImplementationOnce(async function* () {
        yield { type: "done", data: {} };
      });

      const { result } = renderHook(() => useChatStream());

      await act(async () => {
        await result.current.sendMessage("What is capacity?");
      });

      await act(async () => {
        await result.current.retryLastMessage();
      });

      // The second streamChat call should have included the retried user message
      const secondCall = streamChat.mock.calls[1];
      const conversationMessages = secondCall[0];
      const lastUserMessage = conversationMessages
        .filter((m) => m.role === "user")
        .pop();
      expect(lastUserMessage.content).toBe("What is capacity?");
    });

    it("is a no-op when messages is empty", async () => {
      const { result } = renderHook(() => useChatStream());

      expect(result.current.messages).toEqual([]);

      await act(async () => {
        await result.current.retryLastMessage();
      });

      // Still empty, no errors, streamChat not called
      expect(result.current.messages).toEqual([]);
      expect(streamChat).not.toHaveBeenCalled();
    });
  });
});
