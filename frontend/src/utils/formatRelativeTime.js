// ABOUTME: Formats timestamps as human-readable relative strings
// ABOUTME: Used by chat messages to display when each message was sent

/**
 * Convert a timestamp to a human-readable relative time string.
 *
 * @param {number} timestamp - Unix timestamp in milliseconds
 * @returns {string} Relative time string (e.g., "just now", "5 min ago")
 */
export function formatRelativeTime(timestamp) {
  const seconds = Math.floor((Date.now() - timestamp) / 1000);

  if (seconds < 10) {
    return "just now";
  }
  if (seconds < 60) {
    return `${seconds}s ago`;
  }

  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) {
    return `${minutes} min ago`;
  }

  const hours = Math.floor(minutes / 60);
  if (hours < 24) {
    return `${hours} hr ago`;
  }

  return new Date(timestamp).toLocaleTimeString();
}
