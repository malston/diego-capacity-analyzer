// ABOUTME: React hook managing chat conversation state and SSE streaming lifecycle
// ABOUTME: Handles message accumulation, token appending, abort, error propagation, reset, and retry

import { useState, useCallback, useRef, useEffect } from "react";
import { streamChat, ChatError } from "../services/chatApi.js";

let nextMessageId = 0;

/**
 * Maps SSE error event codes to error types for the UI.
 */
const SSE_ERROR_TYPE_MAP = {
  timeout: "timeout",
  provider_error: "server",
};

/**
 * Custom hook for managing chat conversation state and streaming.
 *
 * @returns {{
 *   messages: Array<{ id: string, role: string, content: string, timestamp: number }>,
 *   isStreaming: boolean,
 *   error: { message: string, type: string } | null,
 *   sendMessage: (text: string) => Promise<void>,
 *   clearConversation: () => void,
 *   retryLastMessage: () => Promise<void>
 * }}
 */
export function useChatStream() {
  const [messages, setMessages] = useState([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [error, setError] = useState(null);
  const abortRef = useRef(null);
  const messagesRef = useRef(messages);
  messagesRef.current = messages;

  const sendMessage = useCallback(async (text) => {
    const now = Date.now();
    const userMessage = {
      id: `msg-${now}-${nextMessageId++}`,
      role: "user",
      content: text,
      timestamp: now,
    };
    const assistantMessage = {
      id: `msg-${now}-${nextMessageId++}`,
      role: "assistant",
      content: "",
      timestamp: now,
    };

    if (abortRef.current) abortRef.current.abort();
    const controller = new AbortController();
    abortRef.current = controller;

    setMessages((prev) => [...prev, userMessage, assistantMessage]);
    setIsStreaming(true);
    setError(null);

    // Build conversation array for backend: strip timestamps
    const conversation = [...messagesRef.current, userMessage].map(
      ({ role, content }) => ({ role, content }),
    );

    try {
      for await (const event of streamChat(conversation, controller.signal)) {
        if (event.type === "token") {
          setMessages((prev) => {
            const updated = [...prev];
            const last = updated[updated.length - 1];
            updated[updated.length - 1] = {
              ...last,
              content: last.content + event.data.text,
            };
            return updated;
          });
        } else if (event.type === "done") {
          break;
        } else if (event.type === "error") {
          const mappedType = SSE_ERROR_TYPE_MAP[event.data.code];
          if (!mappedType && event.data.code) {
            console.warn("Unmapped SSE error code:", event.data.code);
          }
          throw new ChatError(event.data.message, mappedType || "server");
        }
      }
    } catch (err) {
      if (err.name !== "AbortError") {
        console.error("Chat stream error:", err);
        if (err instanceof ChatError) {
          setError({ message: err.message, type: err.type });
        } else {
          setError({ message: err.message, type: "server" });
        }
      }
    } finally {
      if (abortRef.current === controller) {
        setIsStreaming(false);
        abortRef.current = null;
      }
    }
  }, []);

  const clearConversation = useCallback(() => {
    if (abortRef.current) {
      abortRef.current.abort();
      abortRef.current = null;
    }
    setMessages([]);
    messagesRef.current = [];
    setIsStreaming(false);
    setError(null);
  }, []);

  const retryLastMessage = useCallback(async () => {
    const currentMessages = messagesRef.current;
    if (currentMessages.length === 0) return;

    const lastUserIndex = currentMessages.findLastIndex(
      (m) => m.role === "user",
    );
    if (lastUserIndex === -1) return;

    const lastUserContent = currentMessages[lastUserIndex].content;

    // Remove both the last user message and any assistant message after it
    const cleaned = currentMessages.slice(0, lastUserIndex);
    setMessages(cleaned);
    // Sync ref so sendMessage reads the cleaned conversation (not stale state)
    messagesRef.current = cleaned;
    setError(null);

    await sendMessage(lastUserContent);
  }, [sendMessage]);

  // Abort on unmount
  useEffect(() => {
    return () => {
      if (abortRef.current) {
        abortRef.current.abort();
      }
    };
  }, []);

  return {
    messages,
    isStreaming,
    error,
    sendMessage,
    clearConversation,
    retryLastMessage,
  };
}
