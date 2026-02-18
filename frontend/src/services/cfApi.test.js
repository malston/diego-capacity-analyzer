// ABOUTME: Unit tests for Cloud Foundry API service
// ABOUTME: Verifies error handling, proxy path mapping, and authentication flows

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cfApi } from "./cfApi";
import { ApiConnectionError, ApiPermissionError } from "./apiClient";

describe("cfApi.request", () => {
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
      await expect(cfApi.request("/v3/apps")).rejects.toThrow(
        ApiConnectionError,
      );
    });

    it("re-throws non-TypeError errors unchanged", async () => {
      const original = new Error("some other error");
      global.fetch = vi.fn().mockRejectedValue(original);
      await expect(cfApi.request("/v3/apps")).rejects.toBe(original);
    });
  });

  describe("authentication errors", () => {
    it("throws descriptive error on 401 response", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        statusText: "Unauthorized",
        json: () => Promise.resolve({}),
      });
      await expect(cfApi.request("/v3/apps")).rejects.toThrow(
        "Authentication required. Please login.",
      );
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
      await expect(cfApi.request("/v3/apps")).rejects.toThrow(
        ApiPermissionError,
      );
    });

    it("includes user-friendly message and setup guidance", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 403,
        statusText: "Forbidden",
        json: () => Promise.resolve({}),
      });
      const err = await cfApi.request("/v3/apps").catch((e) => e);
      expect(err).toBeInstanceOf(ApiPermissionError);
      expect(err.message).toBe(
        "You don't have permission to perform this action",
      );
      expect(err.detail).toContain("diego-analyzer.operator");
    });
  });

  describe("HTTP errors", () => {
    it("extracts error message from JSON response body", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        statusText: "Internal Server Error",
        json: () => Promise.resolve({ error: "database connection failed" }),
      });
      await expect(cfApi.request("/v3/apps")).rejects.toThrow(
        "database connection failed",
      );
    });

    it("extracts description field from CF API error format", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 422,
        statusText: "Unprocessable Entity",
        json: () => Promise.resolve({ description: "Name must be unique" }),
      });
      await expect(cfApi.request("/v3/apps")).rejects.toThrow(
        "Name must be unique",
      );
    });

    it("falls back to status code when body is not JSON", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 502,
        statusText: "Bad Gateway",
        json: () => Promise.reject(new Error("not json")),
      });
      await expect(cfApi.request("/v3/apps")).rejects.toThrow("API Error: 502");
    });
  });

  describe("successful responses", () => {
    it("returns parsed JSON on success", async () => {
      const payload = { resources: [{ name: "my-app" }] };
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(payload),
      });
      const result = await cfApi.request("/v3/apps");
      expect(result).toEqual(payload);
    });

    it("maps CF API paths to backend proxy paths", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({}),
      });
      await cfApi.request("/v3/apps");
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/api/v1/cf/apps"),
        expect.any(Object),
      );
    });

    it("includes credentials in requests", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({}),
      });
      await cfApi.request("/v3/apps");
      expect(global.fetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({ credentials: "include" }),
      );
    });
  });
});
