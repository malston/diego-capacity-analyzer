// ABOUTME: Converts Markdown-formatted text to plain text for clipboard operations
// ABOUTME: Strips headers, bold, italic, links, images, code blocks, blockquotes, lists, tables

/**
 * Strip Markdown formatting from a string, returning plain text.
 * Handles: headers, bold, italic, strikethrough, links, images,
 * fenced code blocks, inline code, blockquotes, list markers,
 * horizontal rules, tables, and excessive newlines.
 *
 * Code blocks (fenced and inline) are preserved verbatim by extracting
 * them to placeholders before other transformations run.
 *
 * @param {string} text - Markdown-formatted text
 * @returns {string} Plain text with formatting removed
 */
export function stripMarkdown(text) {
  if (!text) return "";

  // Extract code blocks to placeholders so their content is not altered
  const codePlaceholders = [];
  let placeholderIndex = 0;

  const savePlaceholder = (content) => {
    const key = `\x00CODE${placeholderIndex++}\x00`;
    codePlaceholders.push({ key, content });
    return key;
  };

  // Extract fenced code blocks: ```lang\ncode\n```
  let result = text.replace(/```[\s\S]*?```/g, (match) => {
    const inner = match.replace(/```\w*\n?/g, "").trim();
    return savePlaceholder(inner);
  });

  // Extract inline code spans: `code`
  result = result.replace(/`([^`]+?)`/g, (_m, inner) => savePlaceholder(inner));

  result = result
    // Headers
    .replace(/^#{1,6}\s+/gm, "")
    // Bold (double asterisks and double underscores)
    .replace(/\*\*(.+?)\*\*/g, "$1")
    .replace(/__(.+?)__/g, "$1")
    // Italic (single asterisks)
    .replace(/\*(.+?)\*/g, "$1")
    // Italic (single underscores) -- only at word boundaries to preserve snake_case
    .replace(/(^|\s)_([^_]+?)_(?=\s|[.,;:!?]|$)/gm, "$1$2")
    // Strikethrough
    .replace(/~~(.+?)~~/g, "$1")
    // Images (before links, since images start with !)
    .replace(/!\[([^\]]*)\]\([^)]+\)/g, "$1")
    // Links
    .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1")
    // Blockquotes
    .replace(/^>\s?/gm, "")
    // Unordered list markers
    .replace(/^\s*[-*+]\s+/gm, "")
    // Ordered list markers
    .replace(/^\s*\d+\.\s+/gm, "")
    // Horizontal rules
    .replace(/^---+$/gm, "")
    // Table pipes
    .replace(/\|/g, " ")
    // Table separator rows
    .replace(/^[-: ]+$/gm, "")
    // Collapse excessive newlines
    .replace(/\n{3,}/g, "\n\n");

  // Restore code content from placeholders
  for (const { key, content } of codePlaceholders) {
    result = result.split(key).join(content);
  }

  return result.trim();
}
