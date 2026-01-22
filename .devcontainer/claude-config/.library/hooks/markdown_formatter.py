#!/usr/bin/env python3
# ABOUTME: PostToolUse hook that auto-formats markdown files after Claude edits them.
# ABOUTME: Adds missing language tags to code fences and fixes excessive blank lines.
"""
Markdown formatter for Claude Code output.
Fixes missing language tags and spacing issues while preserving code content.

USAGE:
    As a Claude Code hook (automatic):
        Triggered automatically after Edit|Write operations on .md/.mdx files.

    Manual invocation:
        echo '{"tool_input": {"file_path": "/path/to/file.md"}}' | ./markdown_formatter.py

    Test language detection:
        python3 -c "from markdown_formatter import detect_language; print(detect_language('def foo(): pass'))"

CONFIGURATION:
    Add to ~/.claude/settings.json:

    {
      "hooks": {
        "PostToolUse": [
          {
            "matcher": "Edit|Write",
            "hooks": [
              {
                "type": "command",
                "command": "~/.claude/hooks/markdown_formatter.py"
              }
            ]
          }
        ]
      }
    }

INPUT FORMAT (stdin):
    {
      "tool_input": {
        "file_path": "/path/to/document.md"
      }
    }

FEATURES:
    - Detects unlabeled code fences and adds language tags (python, javascript, bash, json, sql, text)
    - Reduces excessive blank lines (3+ newlines → 2 newlines)
    - Only processes .md and .mdx files
    - Preserves code content exactly

EXIT CODES:
    0 - Success (file formatted or skipped if not markdown)
    1 - Error (JSON parse error, file read/write error, encoding error)
"""
import json
import sys
import re
import os

def detect_language(code):
    """Best-effort language detection from code content."""
    s = code.strip()

    # JSON detection
    if re.search(r'^\s*[{\[]', s):
        try:
            json.loads(s)
            return 'json'
        except (json.JSONDecodeError, ValueError):
            pass

    # Python detection
    if re.search(r'^\s*def\s+\w+\s*\(', s, re.M) or \
       re.search(r'^\s*(import|from)\s+\w+', s, re.M):
        return 'python'

    # JavaScript detection
    if re.search(r'\b(function\s+\w+\s*\(|const\s+\w+\s*=)', s) or \
       re.search(r'=>|console\.(log|error)', s):
        return 'javascript'

    # Bash detection
    if re.search(r'^#!.*\b(bash|sh)\b', s, re.M) or \
       re.search(r'\b(if|then|fi|for|in|do|done)\b', s):
        return 'bash'

    # SQL detection
    if re.search(r'\b(SELECT|INSERT|UPDATE|DELETE|CREATE)\s+', s, re.I):
        return 'sql'

    return 'text'

def format_markdown(text):
    """Format markdown content with language detection."""
    # Fix unlabeled code fences
    def add_lang_to_fence(match):
        indent, info, body, closing = match.groups()
        if not info.strip():
            lang = detect_language(body)
            return f"{indent}```{lang}\n{body}{closing}\n"
        return match.group(0)

    fence_pattern = r'(?ms)^([ \t]{0,3})```([^\n]*)\n(.*?)(\n\1```)\s*$'
    formatted_content = re.sub(fence_pattern, add_lang_to_fence, text)

    # Fix excessive blank lines (only outside code fences)
    formatted_content = re.sub(r'\n{3,}', '\n\n', formatted_content)

    return formatted_content.rstrip() + '\n'

# Main execution
try:
    input_data = json.load(sys.stdin)
    file_path = input_data.get('tool_input', {}).get('file_path', '')

    if not file_path.endswith(('.md', '.mdx')):
        sys.exit(0)  # Not a markdown file

    if os.path.exists(file_path):
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()

        formatted = format_markdown(content)

        if formatted != content:
            with open(file_path, 'w', encoding='utf-8') as f:
                f.write(formatted)
            print(f"✓ Fixed markdown formatting in {file_path}")

except (json.JSONDecodeError, OSError, UnicodeDecodeError) as e:
    print(f"Error formatting markdown: {e}", file=sys.stderr)
    sys.exit(1)
