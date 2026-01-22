---
allowed-tools: Bash(git add:*), Bash(git status:*), Bash(git commit:*), Bash(git pull:*), Bash(gh:*)
description: Create a git pull request
---

## Context

- Recent tags: !`git tag --sort=-version:refname | head -10`
- Current git status: !`git status`
- Current git diff (staged and unstaged changes): !`git diff HEAD`
- Current branch: !`git branch --show-current`
- Recent commits: !`git log --oneline -10`

## Your task

When the all the checks pass in CI workflow, I want you to:

- Merge the pull request (if I haven't already merged it)
- Create and push new release tag with the latest commits since last release
- Wait for release workflow to complete
- Update release notes
