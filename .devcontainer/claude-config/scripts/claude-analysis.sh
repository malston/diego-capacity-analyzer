#!/usr/bin/env bash

# Get detailed breakdown of what's in projects
find ~/.claude/projects -maxdepth 2 -type d -exec du -sh {} \; | sort -rh | head -20

# Get the actual projects structure
# tree -L 2 ~/.claude/projects 2>/dev/null || find ~/.claude/projects -maxdepth 2 -type f -name '*.json' -o -name 'README*' | head -20
