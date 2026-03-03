// ABOUTME: React hook managing chat conversation state and SSE streaming lifecycle
// ABOUTME: Handles message accumulation, token appending, abort on unmount, and multi-turn history

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

    setMessages((prev) => [...prev, userMessage, assistantMessage]);
    setIsStreaming(true);
    setError(null);

    const controller = new AbortController();
    abortRef.current = controller;

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
          const mappedType = SSE_ERROR_TYPE_MAP[event.data.code] || "server";
          throw new ChatError(event.data.message, mappedType);
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
      setIsStreaming(false);
      abortRef.current = null;
    }
  }, []);

  const clearConversation = useCallback(() => {
    if (abortRef.current) {
      abortRef.current.abort();
      abortRef.current = null;
    }
    setMessages([]);
    setIsStreaming(false);
    setError(null);
  }, []);

  const retryLastMessage = useCallback(async () => {
    const currentMessages = messagesRef.current;
    if (currentMessages.length === 0) return;

    // Find the last user message
    let lastUserContent = null;
    for (let i = currentMessages.length - 1; i >= 0; i--) {
      if (currentMessages[i].role === "user") {
        lastUserContent = currentMessages[i].content;
        break;
      }
    }
    if (lastUserContent === null) return;

    // Remove the last assistant message
    setMessages((prev) => {
      const updated = [...prev];
      for (let i = updated.length - 1; i >= 0; i--) {
        if (updated[i].role === "assistant") {
          updated.splice(i, 1);
          break;
        }
      }
      return updated;
    });
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
