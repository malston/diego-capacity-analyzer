# TUI Menu Integration Design

**Feature:** Integrate menu into bubbletea app with JSON file selection
**Date:** 2026-01-16
**Status:** Approved

---

## Overview

Replace the standalone huh menu with an integrated bubbletea model. Add a file picker screen that shows recent files, allows path input, and provides access to sample infrastructure files.

## Architecture

### Screen States

```
ScreenMenu → ScreenFilePicker → ScreenDashboard → ScreenComparison
     ↓              ↓
  (vSphere)    (JSON selected)
     ↓              ↓
     └──────────────┴──→ loadInfrastructure()
```

### Key Changes

1. **Remove separate `menu.Run()` call** - Menu becomes a bubbletea model in the app's `Update()` loop
2. **Add `ScreenFilePicker` state** - Shows recent files, path input, and sample files
3. **Add `recentfiles` package** - Manages recent files list in XDG config
4. **Add `samples` package** - Discovers sample JSON files

## Component Structure

```
cli/internal/tui/
├── app.go                    # Add ScreenFilePicker, integrate menu model
├── menu/
│   └── menu.go               # Convert to bubbletea.Model
├── filepicker/               # NEW
│   ├── filepicker.go         # File selection screen
│   └── filepicker_test.go
├── recentfiles/              # NEW
│   ├── recentfiles.go        # Load/save recent files
│   └── recentfiles_test.go
└── samples/                  # NEW
    ├── samples.go            # Discover sample files
    └── samples_test.go
```

## File Picker Screen

### Layout

```
Select JSON file:

  Recent files:
  > /Users/mark/infra/prod.json
    /Users/mark/infra/staging.json

  ─────────────────────────────
  > Enter path...
  > Load sample file...
```

### Sample Selection (sub-screen)

```
Select sample:

  > small-footprint-4-hosts.json
    medium-foundation-8-hosts.json
    large-foundation-16-hosts.json
    cpu-constrained-scenario.json
    [back]
```

## Recent Files Storage

**Location:** `~/.config/diego-capacity/recent.json` (XDG with fallback)

**Format:**
```json
{
  "files": [
    "/Users/mark/infra/prod.json",
    "/Users/mark/infra/staging.json"
  ]
}
```

**Behavior:**
- Maximum 5 entries
- Most recently used at front
- Duplicate paths move to front (not added twice)
- Stale paths (file no longer exists) removed on load
- Directory created on first write

## Samples Discovery

**Search locations (in order):**
1. `./frontend/public/samples/` (running from repo)
2. `$DIEGO_SAMPLES_PATH` environment variable
3. Hidden if neither found

## Data Flow

1. User selects "Load JSON file" in menu
2. Transition to `ScreenFilePicker`
3. User picks file (recent, typed path, or sample)
4. Read JSON file from disk
5. Return `fileSelectedMsg{path, data}` to app
6. POST data to `/api/infrastructure/state`
7. On success: add path to recent files, transition to `ScreenDashboard`
8. On error: show inline error, stay on filepicker

## Error Handling

| Error | Handling |
|-------|----------|
| File not found | Show inline error, stay on filepicker |
| Invalid JSON | Show "Invalid JSON format" error |
| Permission denied | Show "Cannot read file" error |
| Backend rejects data | Show backend error message |
| Config dir not writable | Skip recent files (no crash) |
| Samples dir not found | Hide "Load sample file" option |

## Navigation

| Key | Action |
|-----|--------|
| `↑/↓` | Navigate options |
| `Enter` | Select option |
| `Esc` or `b` | Back to previous screen |
| `q` or `Ctrl+C` | Quit |

## Testing

### Unit Tests

| Package | Coverage |
|---------|----------|
| `recentfiles` | Load/save, max limit, move-to-front, missing file, create directory |
| `samples` | Discover files, missing directory, filter non-JSON |
| `filepicker` | Model init, navigation, selection messages, error states |
| `menu` | Model init, option selection, transition messages |

### Manual Testing

- [ ] Launch TUI, menu appears integrated (no flash)
- [ ] Select "Load JSON file", see recent files or empty state
- [ ] Type a path, loads successfully
- [ ] Select sample file, loads successfully
- [ ] Bad path shows error inline
- [ ] `Esc` returns to menu
- [ ] Recent files persist after quit/relaunch
