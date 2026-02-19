// ABOUTME: Unit tests for shared API client wrapper
// ABOUTME: Verifies network error classification, HTTP error handling, and JSON parsing

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { apiFetch, ApiConnectionError, ApiPermissionError } from "./apiClient";

describe("apiFetch", () => {
  let originalFetch;

  beforeEach(() => {
    originalFetch = global.fetch;
  });

  afterEach(() => {
    global.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  describe("network errors", () => {
    it("throws ApiConnectionError on TypeError", async () => {
      global.fetch = vi
        .fn()
        .mockRejectedValue(new TypeError("Failed to fetch"));

      await expect(apiFetch("/api/v1/health")).rejects.toThrow(
        ApiConnectionError,
      );
    });

    it("includes user-friendly summary in message", async () => {
      global.fetch = vi
        .fn()
        .mockRejectedValue(new TypeError("Failed to fetch"));

      await expect(apiFetch("/api/v1/health")).rejects.toThrow(
        "Unable to reach the server",
      );
    });

    it("includes diagnostic detail", async () => {
      global.fetch = vi
        .fn()
        .mockRejectedValue(new TypeError("Failed to fetch"));

      const err = await apiFetch("/api/v1/health").catch((e) => e);
      expect(err).toBeInstanceOf(ApiConnectionError);
      expect(err.detail).toContain("not responding");
      expect(err.detail).toContain("CORS");
    });

    it("re-throws non-TypeError errors unchanged", async () => {
      const original = new Error("some other error");
      global.fetch = vi.fn().mockRejectedValue(original);

      await expect(apiFetch("/api/v1/health")).rejects.toBe(original);
    });
  });

  describe("permission errors", () => {
    it("throws ApiPermissionError on 403 response", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 403,
        statusText: "Forbidden",
        json: () => Promise.resolve({ error: "Insufficient permissions" }),
      });

      await expect(apiFetch("/api/v1/infrastructure/manual")).rejects.toThrow(
        ApiPermissionError,
      );
    });

    it("includes user-friendly message", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 403,
        statusText: "Forbidden",
        json: () => Promise.resolve({ error: "Insufficient permissions" }),
      });

      await expect(apiFetch("/api/v1/infrastructure/manual")).rejects.toThrow(
        "You don't have permission to perform this action",
      );
    });

    it("includes UAA setup guidance in detail", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 403,
        statusText: "Forbidden",
        json: () => Promise.resolve({ error: "Insufficient permissions" }),
      });

      const err = await apiFetch("/api/v1/infrastructure/manual").catch(
        (e) => e,
      );
      expect(err).toBeInstanceOf(ApiPermissionError);
      expect(err.detail).toContain("diego-analyzer.operator");
      expect(err.detail).toContain("AUTHENTICATION.md");
    });

    it("throws ApiPermissionError even when 403 body is not JSON", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 403,
        statusText: "Forbidden",
        json: () => Promise.reject(new Error("not json")),
      });

      await expect(apiFetch("/api/v1/infrastructure/manual")).rejects.toThrow(
        ApiPermissionError,
      );
    });
  });

  describe("HTTP errors", () => {
    it("throws with server error message from JSON body", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        statusText: "Internal Server Error",
        json: () => Promise.resolve({ error: "database connection failed" }),
      });

      await expect(apiFetch("/api/v1/dashboard")).rejects.toThrow(
        "database connection failed",
      );
    });

    it("falls back to status text when body is not JSON", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 502,
        statusText: "Bad Gateway",
        json: () => Promise.reject(new Error("not json")),
      });

      await expect(apiFetch("/api/v1/dashboard")).rejects.toThrow(
        "Server error: 502 Bad Gateway",
      );
    });
  });

  describe("successful responses", () => {
    it("returns parsed JSON on success", async () => {
      const payload = { cells: [], apps: [] };
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(payload),
      });

      const result = await apiFetch("/api/v1/dashboard");
      expect(result).toEqual(payload);
    });

    it("passes url and options to fetch", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({}),
      });

      await apiFetch("/api/v1/test", { method: "POST", body: "{}" });

      expect(global.fetch).toHaveBeenCalledWith("/api/v1/test", {
        method: "POST",
        body: "{}",
      });
    });
  });
});
