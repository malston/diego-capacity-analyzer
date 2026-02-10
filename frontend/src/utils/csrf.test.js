// ABOUTME: Tests for CSRF token utility functions
// ABOUTME: Validates cookie parsing and header construction for double-submit pattern

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { getCSRFToken, withCSRFToken } from "./csrf";

describe("getCSRFToken", () => {
  afterEach(() => {
    // Clear all cookies
    document.cookie.split(";").forEach((c) => {
      const name = c.trim().split("=")[0];
      document.cookie = `${name}=;expires=Thu, 01 Jan 1970 00:00:00 GMT;path=/`;
    });
  });

  it("returns token when DIEGO_CSRF cookie is present", () => {
    document.cookie = "DIEGO_CSRF=test-csrf-token-abc123";
    expect(getCSRFToken()).toBe("test-csrf-token-abc123");
  });

  it("returns null when DIEGO_CSRF cookie is missing", () => {
    expect(getCSRFToken()).toBeNull();
  });

  it("returns null when other cookies exist but not DIEGO_CSRF", () => {
    document.cookie = "OTHER_COOKIE=some-value";
    document.cookie = "DIEGO_SESSION=session-id";
    expect(getCSRFToken()).toBeNull();
  });

  it("extracts token when multiple cookies are present", () => {
    document.cookie = "DIEGO_SESSION=session-abc";
    document.cookie = "DIEGO_CSRF=my-csrf-token";
    document.cookie = "OTHER=value";
    expect(getCSRFToken()).toBe("my-csrf-token");
  });

  it("handles URL-encoded token values", () => {
    document.cookie =
      "DIEGO_CSRF=" + encodeURIComponent("token/with+special=chars");
    expect(getCSRFToken()).toBe("token/with+special=chars");
  });
});

describe("withCSRFToken", () => {
  afterEach(() => {
    document.cookie.split(";").forEach((c) => {
      const name = c.trim().split("=")[0];
      document.cookie = `${name}=;expires=Thu, 01 Jan 1970 00:00:00 GMT;path=/`;
    });
  });

  it("adds X-CSRF-Token header when cookie is present", () => {
    document.cookie = "DIEGO_CSRF=token123";
    const headers = withCSRFToken();
    expect(headers["X-CSRF-Token"]).toBe("token123");
  });

  it("returns empty object when no CSRF cookie", () => {
    const headers = withCSRFToken();
    expect(headers).toEqual({});
    expect(headers["X-CSRF-Token"]).toBeUndefined();
  });

  it("preserves existing headers when adding CSRF token", () => {
    document.cookie = "DIEGO_CSRF=token456";
    const headers = withCSRFToken({ "Content-Type": "application/json" });
    expect(headers["Content-Type"]).toBe("application/json");
    expect(headers["X-CSRF-Token"]).toBe("token456");
  });

  it("preserves existing headers when no CSRF cookie", () => {
    const headers = withCSRFToken({ "Content-Type": "application/json" });
    expect(headers["Content-Type"]).toBe("application/json");
    expect(headers["X-CSRF-Token"]).toBeUndefined();
  });

  it("returns passed headers unchanged when no cookie (same reference)", () => {
    const original = { "Content-Type": "text/plain" };
    const result = withCSRFToken(original);
    expect(result).toBe(original);
  });

  it("returns new object when CSRF token is added (does not mutate input)", () => {
    document.cookie = "DIEGO_CSRF=token789";
    const original = { "Content-Type": "application/json" };
    const result = withCSRFToken(original);
    expect(result).not.toBe(original);
    expect(original["X-CSRF-Token"]).toBeUndefined();
  });
});
