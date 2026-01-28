// ABOUTME: Tests for BFF OAuth authentication service
// ABOUTME: Verifies secure auth via backend endpoints without sessionStorage

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cfAuth } from "./cfAuth";

describe("CFAuthService (BFF Pattern)", () => {
  let originalFetch;

  beforeEach(() => {
    originalFetch = global.fetch;
    // Clear any cached state
    cfAuth._cachedUser = null;
  });

  afterEach(() => {
    global.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  describe("login", () => {
    it("calls backend /api/v1/auth/login with credentials", async () => {
      const mockResponse = {
        success: true,
        username: "testuser",
        user_id: "user-123",
      };

      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      });

      const result = await cfAuth.login("testuser", "password123");

      expect(global.fetch).toHaveBeenCalledWith("/api/v1/auth/login", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          username: "testuser",
          password: "password123",
        }),
      });

      expect(result).toEqual({
        success: true,
        username: "testuser",
        userId: "user-123",
      });
    });

    it("throws error on login failure", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        json: () => Promise.resolve({ error: "Invalid credentials" }),
      });

      await expect(cfAuth.login("bad", "creds")).rejects.toThrow(
        "Invalid credentials",
      );
    });

    it("does not use sessionStorage", async () => {
      const setItemSpy = vi.spyOn(Storage.prototype, "setItem");

      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({ success: true, username: "test", user_id: "123" }),
      });

      await cfAuth.login("test", "pass");

      expect(setItemSpy).not.toHaveBeenCalled();
    });
  });

  describe("logout", () => {
    it("calls backend /api/v1/auth/logout", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: true }),
      });

      await cfAuth.logout();

      expect(global.fetch).toHaveBeenCalledWith("/api/v1/auth/logout", {
        method: "POST",
        credentials: "include",
      });
    });

    it("clears cached user on logout", async () => {
      cfAuth._cachedUser = { username: "test" };

      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: true }),
      });

      await cfAuth.logout();

      expect(cfAuth._cachedUser).toBeNull();
    });

    it("does not use sessionStorage", async () => {
      const removeItemSpy = vi.spyOn(Storage.prototype, "removeItem");

      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: true }),
      });

      await cfAuth.logout();

      expect(removeItemSpy).not.toHaveBeenCalled();
    });
  });

  describe("isAuthenticated", () => {
    it("calls backend /api/v1/auth/me to check session", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            authenticated: true,
            username: "testuser",
            user_id: "user-123",
          }),
      });

      const result = await cfAuth.isAuthenticated();

      expect(global.fetch).toHaveBeenCalledWith("/api/v1/auth/me", {
        method: "GET",
        credentials: "include",
      });

      expect(result).toBe(true);
    });

    it("returns false when not authenticated", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ authenticated: false }),
      });

      const result = await cfAuth.isAuthenticated();

      expect(result).toBe(false);
    });

    it("returns false on network error", async () => {
      global.fetch = vi.fn().mockRejectedValue(new Error("Network error"));

      const result = await cfAuth.isAuthenticated();

      expect(result).toBe(false);
    });

    it("does not read from sessionStorage", async () => {
      const getItemSpy = vi.spyOn(Storage.prototype, "getItem");

      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ authenticated: false }),
      });

      await cfAuth.isAuthenticated();

      expect(getItemSpy).not.toHaveBeenCalled();
    });
  });

  describe("getUserInfo", () => {
    it("returns user info from /api/v1/auth/me", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            authenticated: true,
            username: "testuser",
            user_id: "user-123",
          }),
      });

      const result = await cfAuth.getUserInfo();

      expect(result).toEqual({
        username: "testuser",
        userId: "user-123",
      });
    });

    it("returns null when not authenticated", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ authenticated: false }),
      });

      const result = await cfAuth.getUserInfo();

      expect(result).toBeNull();
    });
  });

  describe("no token exposure", () => {
    it("does not expose getToken method", () => {
      // getToken should not exist - tokens are never exposed to JS
      expect(cfAuth.getToken).toBeUndefined();
    });

    it("does not expose refreshAccessToken method", () => {
      // refreshAccessToken should not exist - backend handles refresh
      expect(cfAuth.refreshAccessToken).toBeUndefined();
    });

    it("does not have token property", () => {
      expect(cfAuth.token).toBeUndefined();
    });

    it("does not have refreshToken property", () => {
      expect(cfAuth.refreshToken).toBeUndefined();
    });
  });
});
