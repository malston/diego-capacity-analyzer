// ABOUTME: Low-level SSE transport for the chat endpoint
// ABOUTME: Handles POST-based SSE with CSRF, chunk buffering, and abort support

import { withCSRFToken } from "../utils/csrf.js";

const API_URL = import.meta.env.VITE_API_URL || "";

/**
 * Typed error for chat transport failures.
 * The `type` field enables differentiated error messages in the UI.
 */
export class ChatError extends Error {
  constructor(message, type = "server") {
    super(message);
    this.name = "ChatError";
    this.type = type;
  }
}

/**
 * Parse a single SSE event text block into { type, data }.
 *
 * @param {string} raw - Raw SSE event text (lines between double newlines)
 * @returns {{ type: string, data: any } | null} Parsed event, or null if no
 *   data line is present or if the data line contains malformed JSON
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

  try {
    return { type, data: JSON.parse(dataLine) };
  } catch {
    console.warn("Skipping malformed SSE data:", dataLine);
    return null;
  }
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

  let response;
  try {
    response = await fetch(`${API_URL}/api/v1/chat`, {
      method: "POST",
      headers,
      credentials: "include",
      body: JSON.stringify({ messages }),
      signal,
    });
  } catch (err) {
    if (err instanceof TypeError) {
      throw new ChatError("Connection lost", "network");
    }
    throw err;
  }

  if (!response.ok) {
    let message;
    try {
      const body = await response.json();
      message = body.error;
    } catch {
      // Response body is not JSON
    }
    const type = response.status === 429 ? "rate_limit" : "server";
    throw new ChatError(
      message || `Chat request failed: ${response.status}`,
      type,
    );
  }

  if (!response.body) {
    throw new Error("Response body is not readable (streaming not supported)");
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

    // Flush any remaining multi-byte sequences from the decoder
    buffer += decoder.decode();
  } finally {
    reader.releaseLock();
  }
}
