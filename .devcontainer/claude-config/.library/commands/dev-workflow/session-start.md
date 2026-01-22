---
description: Start tracked development session with auto-documentation
argument-hint: <task-description>
---
# Start Session: $ARGUMENTS

1. **Session Initialization:**
   - Create session log: `.sessions/session-$(date +%Y%m%d-%H%M%S).md`
   - Log start time and task description
2. **Context Capture:**
   - Current branch: `git branch --show-current`
   - Recent commits: `git log -3 --oneline`
   - Open files in editor workspace
   - Relevant documentation links
3. **Task Breakdown:**
   - Break $ARGUMENTS into 3-5 concrete subtasks
   - Estimate each subtask (S/M/L complexity)
   - Identify dependencies and blockers
4. **Checkpoint System:**
   - Auto-commit every 30 minutes with: `git add -A && git commit -m "checkpoint: [progress-description]"`
   - Log decisions and discoveries in session file
5. **Handoff Template:**

   ```markdown
   ## Progress Summary
   [Completed tasks]

   ## Current State
   [What's working, what's blocked]

   ## Next Steps
   1. [Immediate priority]
   2. [Secondary task]
   3. [Future consideration]

   ## Questions for Team
   - [Specific question 1]
   - [Specific question 2]
