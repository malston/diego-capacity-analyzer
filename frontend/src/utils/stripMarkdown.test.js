// ABOUTME: Tests for Markdown-to-plain-text stripping utility
// ABOUTME: Verifies headers, bold, italic, links, images, code blocks, inline code,
// ABOUTME: blockquotes, lists, tables, horizontal rules, newline collapsing, edge cases

import { describe, it, expect } from "vitest";
import { stripMarkdown } from "./stripMarkdown";

describe("stripMarkdown", () => {
  it("strips h1 headers", () => {
    expect(stripMarkdown("# Hello World")).toBe("Hello World");
  });

  it("strips h2 headers", () => {
    expect(stripMarkdown("## Sub heading")).toBe("Sub heading");
  });

  it("strips h3 through h6 headers", () => {
    expect(stripMarkdown("### Level 3")).toBe("Level 3");
    expect(stripMarkdown("#### Level 4")).toBe("Level 4");
    expect(stripMarkdown("##### Level 5")).toBe("Level 5");
    expect(stripMarkdown("###### Level 6")).toBe("Level 6");
  });

  it("strips bold with double asterisks", () => {
    expect(stripMarkdown("This is **bold** text")).toBe("This is bold text");
  });

  it("strips bold with double underscores", () => {
    expect(stripMarkdown("This is __bold__ text")).toBe("This is bold text");
  });

  it("strips italic with single asterisks", () => {
    expect(stripMarkdown("This is *italic* text")).toBe("This is italic text");
  });

  it("strips italic with single underscores", () => {
    expect(stripMarkdown("This is _italic_ text")).toBe("This is italic text");
  });

  it("strips strikethrough", () => {
    expect(stripMarkdown("This is ~~deleted~~ text")).toBe(
      "This is deleted text",
    );
  });

  it("strips links but keeps link text", () => {
    expect(stripMarkdown("Visit [Google](https://google.com) now")).toBe(
      "Visit Google now",
    );
  });

  it("strips images but keeps alt text", () => {
    expect(stripMarkdown("![Alt text](image.png)")).toBe("Alt text");
  });

  it("strips fenced code block markers but keeps code content", () => {
    const input = "```javascript\nconsole.log('hello');\n```";
    expect(stripMarkdown(input)).toBe("console.log('hello');");
  });

  it("strips inline code backticks", () => {
    expect(stripMarkdown("Use `npm install` to install")).toBe(
      "Use npm install to install",
    );
  });

  it("strips blockquote markers", () => {
    expect(stripMarkdown("> This is a quote")).toBe("This is a quote");
  });

  it("strips unordered list markers entirely", () => {
    const input = "- Item 1\n- Item 2\n* Item 3";
    const result = stripMarkdown(input);
    expect(result).toBe("Item 1\nItem 2\nItem 3");
  });

  it("strips ordered list markers", () => {
    const input = "1. First\n2. Second\n3. Third";
    const result = stripMarkdown(input);
    expect(result).toContain("First");
    expect(result).toContain("Second");
    expect(result).toContain("Third");
  });

  it("strips table pipes and separator rows", () => {
    const input = "| Name | Value |\n|------|-------|\n| CPU | 80% |";
    const result = stripMarkdown(input);
    expect(result).toContain("Name");
    expect(result).toContain("Value");
    expect(result).toContain("CPU");
    expect(result).toContain("80%");
    expect(result).not.toContain("|");
    expect(result).not.toMatch(/^[-|]+$/m);
  });

  it("strips horizontal rules", () => {
    expect(stripMarkdown("Above\n---\nBelow")).toBe("Above\n\nBelow");
  });

  it("collapses excessive newlines to double newlines", () => {
    expect(stripMarkdown("Line 1\n\n\n\n\nLine 2")).toBe("Line 1\n\nLine 2");
  });

  it("returns trimmed output", () => {
    expect(stripMarkdown("  hello  ")).toBe("hello");
  });

  it("handles empty string", () => {
    expect(stripMarkdown("")).toBe("");
  });

  it("handles null input", () => {
    expect(stripMarkdown(null)).toBe("");
  });

  it("handles undefined input", () => {
    expect(stripMarkdown(undefined)).toBe("");
  });

  it("passes through plain text unchanged", () => {
    expect(stripMarkdown("Just plain text")).toBe("Just plain text");
  });

  it("preserves snake_case identifiers when stripping italic underscores", () => {
    expect(stripMarkdown("The diego_cell_count metric")).toBe(
      "The diego_cell_count metric",
    );
  });

  it("strips italic underscores at word boundaries", () => {
    expect(stripMarkdown("This is _italic_ text")).toBe("This is italic text");
  });

  it("preserves underscores in identifiers within sentences", () => {
    expect(stripMarkdown("Check max_memory_usage and cpu_percent values")).toBe(
      "Check max_memory_usage and cpu_percent values",
    );
  });

  it("preserves code content inside fenced code blocks from markdown stripping", () => {
    const input = "```python\ndef __init__(self, *args):\n    pass\n```";
    const result = stripMarkdown(input);
    expect(result).toContain("__init__");
    expect(result).toContain("*args");
  });

  it("preserves inline code content from markdown stripping", () => {
    expect(stripMarkdown("Use `**bold**` for emphasis")).toBe(
      "Use **bold** for emphasis",
    );
  });

  it("handles mixed markdown content", () => {
    const input =
      "# Title\n\nThis is **bold** and *italic* with a [link](url).\n\n> A quote\n\n- List item";
    const result = stripMarkdown(input);
    expect(result).toContain("Title");
    expect(result).toContain("bold");
    expect(result).toContain("italic");
    expect(result).toContain("link");
    expect(result).toContain("A quote");
    expect(result).toContain("List item");
    expect(result).not.toContain("#");
    expect(result).not.toContain("**");
    expect(result).not.toContain("*italic*");
    expect(result).not.toContain("[link]");
    expect(result).not.toContain(">");
  });
});
