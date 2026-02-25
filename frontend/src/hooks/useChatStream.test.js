// ABOUTME: Unit tests for the chat stream React hook
// ABOUTME: Verifies message state management, streaming lifecycle, and abort handling

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useChatStream } from "./useChatStream";

// Mock the streamChat async generator
vi.mock("../services/chatApi", () => ({
  streamChat: vi.fn(),
}));

import { streamChat } from "../services/chatApi";

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

  it("sets error state on non-abort errors", async () => {
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

    expect(result.current.error).toBe("server exploded");
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
    // Error state set from the thrown error
    expect(result.current.error).toBe("rate limited");
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

    expect(result.current.error).toBe("fail");

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
});
