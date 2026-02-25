// ABOUTME: React hook managing chat conversation state and SSE streaming lifecycle
// ABOUTME: Handles message accumulation, token appending, abort on unmount, and multi-turn history

import { useState, useCallback, useRef, useEffect } from "react";
import { streamChat } from "../services/chatApi.js";

let nextMessageId = 0;

/**
 * Custom hook for managing chat conversation state and streaming.
 *
 * @returns {{
 *   messages: Array<{ id: string, role: string, content: string, timestamp: number }>,
 *   isStreaming: boolean,
 *   error: string | null,
 *   sendMessage: (text: string) => Promise<void>
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
          throw new Error(event.data.message);
        }
      }
    } catch (err) {
      if (err.name !== "AbortError") {
        console.error("Chat stream error:", err);
        setError(err.message);
      }
    } finally {
      setIsStreaming(false);
      abortRef.current = null;
    }
  }, []);

  // Abort on unmount
  useEffect(() => {
    return () => {
      if (abortRef.current) {
        abortRef.current.abort();
      }
    };
  }, []);

  return { messages, isStreaming, error, sendMessage };
}
