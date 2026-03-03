// ABOUTME: Converts Markdown-formatted text to plain text for clipboard operations
// ABOUTME: Strips headers, bold, italic, links, images, code blocks, blockquotes, lists, tables

/**
 * Strip Markdown formatting from a string, returning plain text.
 * Handles: headers, bold, italic, strikethrough, links, images,
 * fenced code blocks, inline code, blockquotes, list markers,
 * horizontal rules, tables, and excessive newlines.
 *
 * @param {string} text - Markdown-formatted text
 * @returns {string} Plain text with formatting removed
 */
export function stripMarkdown(text) {
  if (!text) return "";

  return (
    text
      // Fenced code blocks: remove markers, keep content
      .replace(/```[\s\S]*?```/g, (match) =>
        match.replace(/```\w*\n?/g, "").trim(),
      )
      // Headers
      .replace(/^#{1,6}\s+/gm, "")
      // Bold (double asterisks and double underscores)
      .replace(/\*\*(.+?)\*\*/g, "$1")
      .replace(/__(.+?)__/g, "$1")
      // Italic (single asterisks and single underscores)
      .replace(/\*(.+?)\*/g, "$1")
      .replace(/_(.+?)_/g, "$1")
      // Strikethrough
      .replace(/~~(.+?)~~/g, "$1")
      // Inline code
      .replace(/`(.+?)`/g, "$1")
      // Images (before links, since images start with !)
      .replace(/!\[([^\]]*)\]\([^)]+\)/g, "$1")
      // Links
      .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1")
      // Blockquotes
      .replace(/^>\s?/gm, "")
      // Unordered list markers
      .replace(/^\s*[-*+]\s+/gm, "- ")
      // Ordered list markers
      .replace(/^\s*\d+\.\s+/gm, "")
      // Horizontal rules
      .replace(/^---+$/gm, "")
      // Table pipes
      .replace(/\|/g, " ")
      // Table separator rows
      .replace(/^[-: ]+$/gm, "")
      // Collapse excessive newlines
      .replace(/\n{3,}/g, "\n\n")
      .trim()
  );
}
