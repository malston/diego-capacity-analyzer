// ABOUTME: React hook managing chat conversation state and SSE streaming lifecycle
// ABOUTME: Handles message accumulation, token appending, abort on unmount, and multi-turn history

import { useState, useCallback, useRef, useEffect } from "react";
import { streamChat } from "../services/chatApi.js";

/**
 * Custom hook for managing chat conversation state and streaming.
 *
 * @returns {{
 *   messages: Array<{ role: string, content: string, timestamp: number }>,
 *   isStreaming: boolean,
 *   sendMessage: (text: string) => Promise<void>
 * }}
 */
export function useChatStream() {
  const [messages, setMessages] = useState([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const abortRef = useRef(null);

  const sendMessage = useCallback(
    async (text) => {
      const userMessage = {
        role: "user",
        content: text,
        timestamp: Date.now(),
      };
      const assistantMessage = {
        role: "assistant",
        content: "",
        timestamp: Date.now(),
      };

      setMessages((prev) => [...prev, userMessage, assistantMessage]);
      setIsStreaming(true);

      const controller = new AbortController();
      abortRef.current = controller;

      // Build conversation array for backend: strip timestamps
      const conversation = [...messages, userMessage].map(
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
        }
      } finally {
        setIsStreaming(false);
        abortRef.current = null;
      }
    },
    [messages],
  );

  // Abort on unmount
  useEffect(() => {
    return () => {
      if (abortRef.current) {
        abortRef.current.abort();
      }
    };
  }, []);

  return { messages, isStreaming, sendMessage };
}
