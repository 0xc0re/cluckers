---
phase: quick-46
plan: 01
subsystem: launch
tags: [game-launch, bootstrap, shm, proton]

requires:
  - phase: quick-45
    provides: "Robust base64 decoding of content bootstrap (variable-length output)"
provides:
  - "Dynamic content_bootstrap_size game arg matching actual decoded data length"
  - "Bootstrap size arg only present when bootstrap data exists"
affects: [launch, prep, steam-managed-launch]

tech-stack:
  added: []
  patterns: ["dynamic-length-over-hardcoded-constant"]

key-files:
  created: []
  modified:
    - internal/launch/process_linux.go
    - internal/launch/process_windows.go
    - internal/launch/prep.go
    - internal/launch/prep_test.go

key-decisions:
  - "Bootstrap size arg moved inside conditional block so it only appears when data exists"

patterns-established:
  - "Content bootstrap size: always use len() of actual data, never hardcode"

requirements-completed: [QUICK-46]

duration: 1min
completed: 2026-03-03
---

# Quick Task 46: Fix Hardcoded Content Bootstrap Size Summary

**Replace hardcoded `-content_bootstrap_size=136` with dynamic `len()` in all 4 launch arg locations to match actual decoded bootstrap data length**

## Performance

- **Duration:** 1 min
- **Started:** 2026-03-03T19:30:04Z
- **Completed:** 2026-03-03T19:31:24Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Removed hardcoded 136-byte assumption from all 3 production files (process_linux.go, process_windows.go, prep.go)
- Bootstrap size arg now only included when bootstrap data is present (process_linux.go, process_windows.go)
- Added test proving non-136-byte bootstrap gets correct dynamic size in launch config
- Zero occurrences of hardcoded `-content_bootstrap_size=136` remain in production code

## Task Commits

Each task was committed atomically:

1. **Task 1: Move bootstrap size arg into conditional block with dynamic length** - `cc85927` (fix)
2. **Task 2: Update test to validate dynamic bootstrap size** - `88f1be8` (test)

## Files Created/Modified
- `internal/launch/process_linux.go` - Moved size arg into bootstrap conditional, uses `len(cfg.ContentBootstrap)`
- `internal/launch/process_windows.go` - Same change as Linux variant
- `internal/launch/prep.go` - Changed hardcoded 136 to `len(state.Bootstrap)`
- `internal/launch/prep_test.go` - Updated existing test to use dynamic check, added new test with 21-byte bootstrap

## Decisions Made
- Bootstrap size arg moved inside the `if cfg.ContentBootstrap != nil && len(cfg.ContentBootstrap) > 0` conditional block in both process files, so the arg is completely absent when no bootstrap data exists (consistent with how `-content_bootstrap_shm` was already handled)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Fix is complete. Users whose API-returned bootstrap data decodes to a length other than 136 bytes will no longer hit "Encrypted package bootstrap failed (EngineInitStartup)" errors.
- This complements quick-45's multi-strategy base64 decoding fix (variable-length decoded output now correctly propagated to game args).

## Self-Check: PASSED

- All 4 modified files exist
- Both task commits verified (cc85927, 88f1be8)
- Dynamic `len()` present in all 3 production files
- No hardcoded `content_bootstrap_size=136` in production code
- All 50 tests in launch package pass

---
*Phase: quick-46*
*Completed: 2026-03-03*
