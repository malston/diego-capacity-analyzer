---
description: Generate implementation spec from GitHub issue
argument-hint: <issue-number>
---
# Analyze Issue #$ARGUMENTS

1. Fetch issue: `gh issue view $ARGUMENTS`
2. **Requirements Analysis**
   - Extract user story and acceptance criteria
   - List functional requirements
   - Note non-functional requirements (performance, security)
3. **Technical Specification**
   - Files to modify/create
   - API contracts (request/response schemas)
   - Database schema changes
   - External dependencies
4. **Implementation Plan**
   - Break into 5-7 sub-tasks with complexity estimates (1-5 scale)
   - Identify risks and implementation order
5. **Test Strategy**
   - Unit tests, integration tests, E2E scenarios
   - Edge cases to cover
6. **Definition of Done**
   - Functionality checklist, test coverage requirements
   - Documentation updates, performance benchmarks
Create `specs/issue-$ARGUMENTS-spec.md` with complete analysis.
