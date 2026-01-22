---
description: Generate feature scaffold with tests, types, and docs
argument-hint: <feature-name>
---
# Scaffold Feature: $ARGUMENTS

1. Create feature directory: `src/features/$ARGUMENTS/`
2. **Generate Core Files:**

$ARGUMENTS.tsx

# Main component $ARGUMENTS.test.tsx

# Unit tests $ARGUMENTS.types.ts

# TypeScript interfaces $ARGUMENTS.styles.ts

# Styled components index.ts

# Barrel export

3. **Component Template:**

- Props interface with JSDoc
- Error boundary wrapper
- Loading and error states
- Accessibility attributes

4. **Test Template:**

- Render test
- User interaction tests
- Error state tests
- Accessibility tests (axe-core)

5. **Documentation:**

- Create `$ARGUMENTS/README.md` with:
  - Feature overview
  - Props documentation
  - Usage examples
  - Known limitations

6. **Git Integration:**

- Stage all files: `git add src/features/$ARGUMENTS/`
- Create feature branch: `git checkout -b feature/$ARGUMENTS`
