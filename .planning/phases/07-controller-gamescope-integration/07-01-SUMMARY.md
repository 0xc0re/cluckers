---
phase: 07-controller-gamescope-integration
plan: 01
subsystem: wine
tags: [steam, proton, detection, linux, tdd]

# Dependency graph
requires:
  - phase: 06-core-proton-launch-pipeline
    provides: FindProtonGE and resolveReal utilities in internal/wine/detect.go
provides:
  - FindSteamInstall() function for Steam root directory detection
  - isSteamDir() validation for Steam installation markers
affects: [07-02, 07-03, steam-integration, gamescope]

# Tech tracking
tech-stack:
  added: []
  patterns: [public-wrapper-internal-impl testability pattern]

key-files:
  created:
    - internal/wine/steamdir.go
    - internal/wine/steamdir_test.go
  modified: []

key-decisions:
  - "Reuse resolveReal() and userHome() from detect.go for consistency"
  - "Priority order: native > symlink > Flatpak > Snap (most common first)"
  - "Two marker files: steam.sh and ubuntu12_32/steamclient.so cover all installation types"

patterns-established:
  - "FindSteamInstall/findSteamInstall: same public-wrapper/internal-impl pattern as FindProton/findProton for testability"

requirements-completed: [CTRL-02]

# Metrics
duration: 1min
completed: 2026-02-25
---

# Phase 07 Plan 01: Steam Installation Detection Summary

**FindSteamInstall with native/Flatpak/Snap detection, symlink dedup, and 9 TDD test cases**

## Performance

- **Duration:** 1 min
- **Started:** 2026-02-25T09:43:49Z
- **Completed:** 2026-02-25T09:45:17Z
- **Tasks:** 2 (TDD: RED + GREEN)
- **Files modified:** 2

## Accomplishments
- FindSteamInstall detects Steam across native, Flatpak, and Snap installations
- Symlink deduplication prevents double-detection of the same directory
- 9 test cases covering all installation types, not-found, dedup, and priority ordering
- Reuses existing resolveReal() and userHome() from detect.go for consistency

## Task Commits

Each task was committed atomically:

1. **RED: Failing tests** - `8daf35a` (test)
2. **GREEN: Implementation** - `0072bbd` (feat)

_No refactor step needed -- implementation is minimal and follows established patterns._

## Files Created/Modified
- `internal/wine/steamdir.go` - FindSteamInstall, findSteamInstall, isSteamDir, steamInstallDirs
- `internal/wine/steamdir_test.go` - 9 test cases for all detection scenarios

## Decisions Made
- Reused resolveReal() and userHome() from detect.go rather than duplicating
- Priority order: native ~/.local/share/Steam first (most common), then symlinks, Flatpak, Snap
- Two marker files (steam.sh, ubuntu12_32/steamclient.so) sufficient to validate any Steam installation type

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- FindSteamInstall() is ready for use by STEAM_COMPAT_CLIENT_INSTALL_PATH environment variable setup (07-02)
- Foundation for SteamGameId resolution via shortcuts.vdf (07-03)

## Self-Check: PASSED

- [x] internal/wine/steamdir.go exists
- [x] internal/wine/steamdir_test.go exists
- [x] 07-01-SUMMARY.md exists
- [x] Commit 8daf35a (RED) exists
- [x] Commit 0072bbd (GREEN) exists

---
*Phase: 07-controller-gamescope-integration*
*Completed: 2026-02-25*
