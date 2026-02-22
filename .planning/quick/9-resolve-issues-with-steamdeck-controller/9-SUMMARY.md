---
phase: quick-9
plan: 01
subsystem: launch
tags: [wine, dxvk, steam-deck, controller, sdl]

# Dependency graph
requires:
  - phase: quick-8
    provides: "Investigation of Steam Deck controller issues and identification of needed env vars"
provides:
  - "DXVK dll override (WINEDLLOVERRIDES=dxgi=n) for all launches"
  - "Steam Deck controller env vars (STEAM_INPUT_DISABLE, SDL_GAMECONTROLLERCONFIG, SDL_JOYSTICK_HIDAPI)"
affects: [launch, wine]

# Tech tracking
tech-stack:
  added: []
  patterns: ["isSteamDeck() detection via distro ID or /home/deck path"]

key-files:
  created: []
  modified: ["internal/launch/process.go"]

key-decisions:
  - "WINEDLLOVERRIDES=dxgi=n set unconditionally (matches POC behavior)"
  - "isSteamDeck() duplicates detection pattern from cli/steam.go rather than exporting it (keeps packages independent)"

patterns-established:
  - "Steam Deck detection: wine.DetectDistro() == steamos OR /home/deck exists"

requirements-completed: [QUICK-9]

# Metrics
duration: 1min
completed: 2026-02-22
---

# Quick Task 9: Restore DXVK Override and Steam Deck Controller Env Vars Summary

**Unconditional WINEDLLOVERRIDES=dxgi=n for DXVK plus conditional STEAM_INPUT_DISABLE=1 and SDL overrides on Steam Deck**

## Performance

- **Duration:** 1 min
- **Started:** 2026-02-22T21:00:21Z
- **Completed:** 2026-02-22T21:01:15Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments
- Restored WINEDLLOVERRIDES=dxgi=n unconditionally in the launch env block (required for DXVK, was in Python POC but missing after quick-8 revert)
- Added isSteamDeck() helper function detecting SteamOS distro or /home/deck directory
- Added conditional Steam Deck controller env vars: STEAM_INPUT_DISABLE=1, SDL_GAMECONTROLLERCONFIG= (empty), SDL_JOYSTICK_HIDAPI=0

## Task Commits

Each task was committed atomically:

1. **Task 1: Add DXVK override and Steam Deck controller env vars** - `30c217c` (feat)

## Files Created/Modified
- `internal/launch/process.go` - Added WINEDLLOVERRIDES=dxgi=n, isSteamDeck() helper, and conditional Steam Deck controller env vars

## Decisions Made
- WINEDLLOVERRIDES=dxgi=n is set unconditionally for all launches, matching the original Python POC behavior
- isSteamDeck() is a package-private function in process.go, duplicating the detection pattern from cli/steam.go rather than exporting it (keeps launch and cli packages independent)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- DXVK override and Steam Deck controller input routing are now active in the Go launcher
- Ready for on-device testing on Steam Deck to verify controller input works end-to-end

---
*Phase: quick-9*
*Completed: 2026-02-22*

## Self-Check: PASSED

- [x] internal/launch/process.go exists
- [x] 9-SUMMARY.md exists
- [x] Commit 30c217c exists in git log
