// ABOUTME: Low-level SSE transport for the chat endpoint
// ABOUTME: Handles POST-based SSE with CSRF, chunk buffering, and abort support

import { withCSRFToken } from "../utils/csrf.js";

/**
 * Parse a single SSE event text block into { type, data }.
 *
 * @param {string} raw - Raw SSE event text (lines between double newlines)
 * @returns {{ type: string, data: any } | null} Parsed event or null if no data line
 */
export function parseSSEEvent(raw) {
  const lines = raw.split("\n");
  let type = "message";
  let dataLine = null;

  for (const line of lines) {
    if (line.startsWith("event:")) {
      type = line.slice("event:".length).trim();
    } else if (line.startsWith("data:")) {
      dataLine = line.slice("data:".length).trim();
    }
  }

  if (dataLine === null) {
    return null;
  }

  return { type, data: JSON.parse(dataLine) };
}

/**
 * Async generator that streams chat responses from the backend SSE endpoint.
 *
 * POST /api/v1/chat with { messages } body. Yields parsed SSE events as they
 * arrive, buffering incomplete chunks across reads.
 *
 * @param {Array<{ role: string, content: string }>} messages - Conversation history
 * @param {AbortSignal} [signal] - AbortSignal for cancellation
 * @yields {{ type: string, data: any }} Parsed SSE events
 */
export async function* streamChat(messages, signal) {
  const headers = withCSRFToken({
    "Content-Type": "application/json",
  });

  const response = await fetch("/api/v1/chat", {
    method: "POST",
    headers,
    credentials: "include",
    body: JSON.stringify({ messages }),
    signal,
  });

  if (!response.ok) {
    let message;
    try {
      const body = await response.json();
      message = body.error;
    } catch {
      // Response body is not JSON
    }
    throw new Error(message || `Chat request failed: ${response.status}`);
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });

      const parts = buffer.split("\n\n");
      // Keep the last part as buffer (may be incomplete)
      buffer = parts.pop();

      for (const part of parts) {
        if (part.trim() === "") continue;
        const event = parseSSEEvent(part);
        if (event !== null) {
          yield event;
        }
      }
    }
  } finally {
    reader.releaseLock();
  }
}
