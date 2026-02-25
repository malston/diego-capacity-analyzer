// ABOUTME: Unit tests for relative time formatting utility
// ABOUTME: Verifies all time brackets from "just now" through locale fallback

import { describe, it, expect, vi, afterEach } from "vitest";
import { formatRelativeTime } from "./formatRelativeTime";

describe("formatRelativeTime", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  function mockNow(timestamp) {
    vi.useFakeTimers();
    vi.setSystemTime(timestamp);
  }

  it('returns "just now" for timestamps less than 10 seconds ago', () => {
    const now = 1700000000000;
    mockNow(now);

    expect(formatRelativeTime(now - 5000)).toBe("just now");
    expect(formatRelativeTime(now - 0)).toBe("just now");
    expect(formatRelativeTime(now - 9999)).toBe("just now");
  });

  it('returns "{N}s ago" for timestamps 10-59 seconds ago', () => {
    const now = 1700000000000;
    mockNow(now);

    expect(formatRelativeTime(now - 30000)).toBe("30s ago");
    expect(formatRelativeTime(now - 10000)).toBe("10s ago");
    expect(formatRelativeTime(now - 59000)).toBe("59s ago");
  });

  it('returns "{N} min ago" for timestamps 1-59 minutes ago', () => {
    const now = 1700000000000;
    mockNow(now);

    expect(formatRelativeTime(now - 5 * 60 * 1000)).toBe("5 min ago");
    expect(formatRelativeTime(now - 1 * 60 * 1000)).toBe("1 min ago");
    expect(formatRelativeTime(now - 59 * 60 * 1000)).toBe("59 min ago");
  });

  it('returns "{N} hr ago" for timestamps 1-23 hours ago', () => {
    const now = 1700000000000;
    mockNow(now);

    expect(formatRelativeTime(now - 3 * 60 * 60 * 1000)).toBe("3 hr ago");
    expect(formatRelativeTime(now - 1 * 60 * 60 * 1000)).toBe("1 hr ago");
    expect(formatRelativeTime(now - 23 * 60 * 60 * 1000)).toBe("23 hr ago");
  });

  it("falls back to toLocaleTimeString for timestamps beyond 24 hours", () => {
    const now = 1700000000000;
    mockNow(now);

    const oldTimestamp = now - 25 * 60 * 60 * 1000;
    const result = formatRelativeTime(oldTimestamp);

    // Should match the output of toLocaleTimeString for that timestamp
    expect(result).toBe(new Date(oldTimestamp).toLocaleTimeString());
  });
});
