---
phase: 04-cross-platform-gui
plan: 01
subsystem: ui
tags: [fyne, gui, build-tags, headless-detection, dark-theme, steam-deck]

# Dependency graph
requires:
  - phase: 01-foundation
    provides: "CLI framework (cobra/viper), config package, root command structure"
provides:
  - "Fyne v2.7.3 dependency and GUI build infrastructure"
  - "Build tag separation: gui vs CLI-only binary variants"
  - "Headless detection (DISPLAY/WAYLAND_DISPLAY on Linux, always true on Windows)"
  - "Custom dark theme (#1E1E23 bg, #4CAF50 green accent)"
  - "GUI app skeleton with placeholder content (logo + title)"
  - "Steam Deck detection for fullscreen mode"
  - "Embedded placeholder logo PNG resource"
affects: [04-02, 04-03, 04-04, 04-05, ci-cd]

# Tech tracking
tech-stack:
  added: [fyne.io/fyne/v2 v2.7.3]
  patterns: [gui-build-tags, headless-detection, fyne-theme-override, platform-specific-deck-detection]

key-files:
  created:
    - internal/gui/app.go
    - internal/gui/detect.go
    - internal/gui/detect_linux.go
    - internal/gui/detect_windows.go
    - internal/gui/theme.go
    - internal/gui/deck_linux.go
    - internal/gui/deck_windows.go
    - internal/gui/assets/embed.go
    - internal/gui/assets/cluckers_logo.png
    - internal/cli/root_gui.go
    - internal/cli/root_nogui.go
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "All GUI package files use //go:build gui tag to keep CLI-only build path clean"
  - "Steam Deck detection uses DMI board_vendor + os-release + /home/deck fallback (independent of wine package)"
  - "GUI binary requires CGO_ENABLED=1 on Linux; CLI-only binary unchanged at CGO_ENABLED=0"
  - "Placeholder logo is a 256x256 green circle with dark C letter (to be replaced with real logo)"

patterns-established:
  - "Build tag pattern: //go:build gui for GUI code, //go:build !gui for CLI-only fallback"
  - "Platform split: detect_linux.go / detect_windows.go / deck_linux.go / deck_windows.go"
  - "Root command wiring: root_gui.go init() sets RunE, root_nogui.go leaves Cobra default"
  - "GUI assets embedding: gui/assets/ package with //go:build gui tag and //go:embed"

requirements-completed: [GUI-01, GUI-06]

# Metrics
duration: 5min
completed: 2026-02-24
---

# Phase 04 Plan 01: Fyne GUI Foundation Summary

**Fyne v2.7.3 GUI skeleton with build-tag separation, headless detection, custom dark theme, and placeholder window**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-24T15:46:36Z
- **Completed:** 2026-02-24T15:51:58Z
- **Tasks:** 1
- **Files modified:** 13

## Accomplishments
- Added Fyne v2.7.3 as a dependency with clean build-tag separation (gui vs non-gui builds)
- Created complete GUI package skeleton: app, theme, headless detection, Steam Deck detection, embedded assets
- Wired root Cobra command to launch GUI (with gui tag) or show help (without gui tag)
- CLI-only binary (13MB, CGO_ENABLED=0) contains zero Fyne symbols; GUI binary (37MB, CGO_ENABLED=1) includes full Fyne stack
- Headless environments (no DISPLAY/WAYLAND_DISPLAY) correctly fall back to CLI help output

## Task Commits

Each task was committed atomically:

1. **Task 1: Add Fyne dependency and create GUI package skeleton with build tags** - `b25b267` (feat)

## Files Created/Modified
- `go.mod` / `go.sum` - Added fyne.io/fyne/v2 v2.7.3 and transitive dependencies
- `internal/gui/app.go` - Fyne app initialization, window creation, placeholder content (logo + title)
- `internal/gui/detect.go` - GUI package declaration (gui build tag)
- `internal/gui/detect_linux.go` - Linux headless detection via DISPLAY/WAYLAND_DISPLAY
- `internal/gui/detect_windows.go` - Windows always returns true for GUI capability
- `internal/gui/theme.go` - Custom dark theme (#1E1E23 bg, #3C3C46 buttons, #4CAF50 green accent)
- `internal/gui/deck_linux.go` - Steam Deck detection via DMI board vendor / os-release / /home/deck
- `internal/gui/deck_windows.go` - Windows always returns false for Steam Deck
- `internal/gui/assets/embed.go` - Embedded logo PNG resource with LogoResource() helper
- `internal/gui/assets/cluckers_logo.png` - Placeholder 256x256 logo (green circle with C letter)
- `internal/cli/root_gui.go` - GUI build: root command checks CanShowGUI() then calls gui.Run()
- `internal/cli/root_nogui.go` - Non-GUI build: Cobra default behavior (show help)

## Decisions Made
- **All files in gui/ use `//go:build gui` tag**: Ensures zero Fyne code compiles into CLI-only binary. Verified by checking binary symbols (0 fyne references in CLI binary vs 3,927 in GUI binary).
- **Separate Steam Deck detection from wine package**: The existing `wine.IsSteamDeck()` has `//go:build linux` and imports the wine package. Created independent `isSteamDeck()` in gui package using DMI board vendor check to avoid coupling GUI to Wine.
- **Custom theme delegates to DefaultTheme with VariantDark**: Only overrides background, button, primary, overlay, and input background colors. All other theme values (fonts, icons, sizes) delegate to Fyne defaults for stability.
- **Window size 480x640 for desktop**: Portrait orientation matching mobile/launcher aesthetic per research recommendation.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added platform-specific Steam Deck detection files**
- **Found during:** Task 1 (app.go implementation)
- **Issue:** Plan referenced `isSteamDeck()` but didn't specify platform-specific files for it. The existing `wine.IsSteamDeck()` is Linux-only and importing the wine package from the GUI would create unnecessary coupling.
- **Fix:** Created `deck_linux.go` and `deck_windows.go` with independent Steam Deck detection using DMI board vendor, os-release, and /home/deck fallback.
- **Files modified:** internal/gui/deck_linux.go, internal/gui/deck_windows.go
- **Verification:** Builds succeed on both platforms, `go vet -tags gui` passes
- **Committed in:** b25b267 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical functionality)
**Impact on plan:** Essential for cross-platform compilation. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- GUI skeleton is ready for login screen implementation (plan 02)
- `gui.Run()` currently shows placeholder content that will be replaced by login/main screens
- Theme and headless detection are available for all future GUI plans
- Build commands: `CGO_ENABLED=1 go build -tags gui` (GUI) or `CGO_ENABLED=0 go build` (CLI-only)

## Self-Check: PASSED

All 12 claimed files verified present. Task commit b25b267 verified in git log.

---
*Phase: 04-cross-platform-gui*
*Completed: 2026-02-24*
