---
phase: 06-core-proton-launch-pipeline
plan: 02
subsystem: launch
tags: [proton, wine, environment, command-building, shm, tdd]

# Dependency graph
requires:
  - phase: 06-core-proton-launch-pipeline
    provides: "ProtonGEInstall detection and compatdata health check (plan 01)"
provides:
  - "buildProtonEnv() for filtered Proton environment construction"
  - "buildProtonCommand() for python3 proton run argument ordering"
  - "protonErrorSuggestion() for actionable Proton error recovery text"
  - "shmBridgeError() for distinct SHM bridge failure detection"
  - "filterEnv() for environment variable key-based filtering"
  - "lastNLines() helper for stderr truncation"
affects: [06-03-PLAN, pipeline-integration]

# Tech tracking
tech-stack:
  added: []
  patterns: ["testable env construction via buildProtonEnvFrom with injectable base env", "case-insensitive stderr pattern matching for error detection"]

key-files:
  created:
    - internal/launch/proton_env.go
    - internal/launch/proton_env_test.go
  modified: []

key-decisions:
  - "Used buildProtonEnvFrom with injectable base env for deterministic testing instead of mocking os.Environ"
  - "SHM bridge error detection uses case-insensitive substring matching on 4 patterns: createfilemapping, openfilemapping, shm_launcher, shared memory"
  - "proton run shmPath uses Linux path (proton converts), but bootstrapPath and gameExe use Wine Z: paths (consumed by Windows processes)"

patterns-established:
  - "Injectable environment pattern: public func wraps internal func that accepts base env slice"
  - "Stderr pattern matching for process error classification"

requirements-completed: [PROTON-02, PROTON-04, PROTON-05]

# Metrics
duration: 3min
completed: 2026-02-25
---

# Phase 6 Plan 2: Proton Environment and Command Construction Summary

**TDD-driven Proton env construction (filterEnv, buildProtonEnv, buildProtonCommand) with SHM bridge error detection and 32 unit tests**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-25T05:36:43Z
- **Completed:** 2026-02-25T05:39:14Z
- **Tasks:** 1 TDD feature (6 sub-features, RED-GREEN cycle)
- **Files modified:** 2

## Accomplishments
- buildProtonEnv strips 6 conflicting env vars (LD_LIBRARY_PATH, WINEPREFIX, WINE, WINEDLLOVERRIDES, WINEFSYNC, WINEESYNC) and sets 5 required Proton vars
- buildProtonCommand produces correct python3 proton run argument ordering for both SHM (bootstrap present) and direct launch modes
- shmBridgeError detects shm_launcher failures via case-insensitive stderr pattern matching, returning distinct UserError separate from general Proton crashes
- protonErrorSuggestion provides 3-step actionable recovery (delete compatdata, update Proton-GE, run cluckers update)
- All 32 unit tests passing with go vet clean

## Task Commits

Each task was committed atomically (TDD RED-GREEN cycle):

1. **RED: Failing tests** - `68280db` (test) - 32 tests covering all 6 sub-features
2. **GREEN: Implementation** - `a934afe` (feat) - All functions implemented, all tests passing

**No REFACTOR commit** - implementation was clean on first pass.

## Files Created/Modified
- `internal/launch/proton_env.go` - Proton environment construction, command building, error formatting (Linux-only)
- `internal/launch/proton_env_test.go` - 32 unit tests covering all behavior cases (Linux-only)

## Decisions Made
- Used injectable `buildProtonEnvFrom` pattern for deterministic testing rather than mocking `os.Environ()`
- SHM bridge error detection uses 4 case-insensitive stderr patterns rather than exit code analysis (more reliable since Wine exit codes are not well-defined)
- In SHM mode, `shmPath` stays as Linux path (proton converts) while `bootstrapPath` and `gameExe` use Wine Z: paths (consumed by Windows processes under Wine)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Removed unused ui import from test file**
- **Found during:** GREEN phase
- **Issue:** Test file imported `github.com/0xc0re/cluckers/internal/ui` but never directly referenced the `ui` package (accesses `*ui.UserError` fields through returned struct, not the type name)
- **Fix:** Removed the unused import
- **Files modified:** internal/launch/proton_env_test.go
- **Verification:** Build succeeded, all tests pass
- **Committed in:** a934afe (GREEN commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Trivial import fix. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All Proton env/command functions ready for pipeline integration in plan 06-03
- buildProtonEnv and buildProtonCommand will be called from the new Proton launch path in process_linux.go
- shmBridgeError and protonErrorSuggestion will be used in the error handling path after proton run exits

## Self-Check: PASSED

- FOUND: internal/launch/proton_env.go
- FOUND: internal/launch/proton_env_test.go
- FOUND: 06-02-SUMMARY.md
- FOUND: 68280db (RED commit)
- FOUND: a934afe (GREEN commit)

---
*Phase: 06-core-proton-launch-pipeline*
*Completed: 2026-02-25*
