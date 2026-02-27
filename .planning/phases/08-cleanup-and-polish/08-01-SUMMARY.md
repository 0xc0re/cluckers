---
phase: 08-cleanup-and-polish
plan: 01
subsystem: launch
tags: [proton, wine-removal, dead-code, cleanup]

# Dependency graph
requires:
  - phase: 07.1-input-proxy
    provides: input proxy code that was abandoned and is now deleted
provides:
  - "Clean codebase with no dead Wine/proxy code paths"
  - "Removed ~1900 lines of dead code across 12 deleted files"
  - "Removed github.com/kenshaw/evdev dependency"
  - "Pipeline steps using Proton-only terminology"
affects: [08-02-status-rewrite, CLAUDE.md]

# Tech tracking
tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified:
    - internal/wine/detect.go
    - internal/launch/pipeline.go
    - internal/launch/pipeline_linux.go
    - internal/launch/process.go
    - internal/launch/process_linux.go
    - internal/cli/status_linux.go
    - go.mod
    - go.sum

key-decisions:
  - "Updated status_linux.go to use FindProton/CompatdataHealthy as Rule 3 fix to maintain compilability"

patterns-established: []

requirements-completed: [POLISH-01, POLISH-03]

# Metrics
duration: 4min
completed: 2026-02-27
---

# Phase 08 Plan 01: Dead Wine/Proxy Code Removal Summary

**Surgical deletion of ~1900 lines of dead code: Wine prefix management, DLL verification, input proxy package, XInput DLL proxy, and legacy struct fields/pipeline steps**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-27T04:59:43Z
- **Completed:** 2026-02-27T05:03:49Z
- **Tasks:** 2
- **Files modified:** 21

## Accomplishments
- Deleted 12 dead files: wine/prefix.go, wine/verify.go, 8 inputproxy files, xinput_remap.c, xinput1_3.def
- Removed FindWine(), WineInstallInstructions() functions from detect.go
- Removed stepPatchWinebus, stepStartInputProxy pipeline steps and InputProxyCleanup struct field
- Removed WinePath/WinePrefix legacy fields from LaunchState and LaunchConfig
- Removed github.com/kenshaw/evdev dependency via go mod tidy
- Codebase compiles cleanly on both Linux (go build) and Windows (go vet)

## Task Commits

Each task was committed atomically:

1. **Task 1: Delete dead files and the inputproxy package** - `662ede2` (refactor)
2. **Task 2: Remove dead functions, struct fields, pipeline steps, and fix imports** - `7683408` (refactor)

## Files Created/Modified
- `internal/wine/prefix.go` - DELETED (entire prefix management: CreatePrefix, copyProtonTemplate, etc.)
- `internal/wine/verify.go` - DELETED (entire DLL verification: VerifyPrefix, RepairInstructions, RequiredDLLs)
- `internal/launch/inputproxy/` - DELETED (entire package: 10 files, evdev proxy)
- `tools/xinput_remap.c` - DELETED (abandoned XInput DLL proxy source)
- `tools/xinput1_3.def` - DELETED (abandoned XInput DLL exports)
- `internal/wine/detect.go` - Removed FindWine(), WineInstallInstructions(), removed exec import
- `internal/wine/proton.go` - Removed stale comment referencing WineInstallInstructions
- `internal/launch/pipeline.go` - Removed InputProxyCleanup/WinePath/PrefixPath from LaunchState, removed from stepLaunchGame config
- `internal/launch/pipeline_linux.go` - Removed stepPatchWinebus, stepStartInputProxy, inputproxy import, dead WinePath assignment
- `internal/launch/process.go` - Removed WinePath, WinePrefix, InputProxyCleanup from LaunchConfig
- `internal/launch/process_linux.go` - Removed input proxy cleanup block
- `internal/cli/status_linux.go` - Rewritten to use FindProton/CompatdataHealthy (Rule 3 fix)
- `go.mod` - Removed kenshaw/evdev dependency
- `go.sum` - Updated (evdev removed)

## Decisions Made
- Updated status_linux.go to use FindProton and CompatdataHealthy instead of deleted FindWine/VerifyPrefix/PrefixPath. This was necessary for compilation (Rule 3 auto-fix) and bridges to Plan 02 which does a full status command rewrite.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated status_linux.go to compile after function deletions**
- **Found during:** Task 2 (removing dead functions from detect.go)
- **Issue:** status_linux.go called wine.FindWine(), wine.PrefixPath(), wine.VerifyPrefix(), and wine.RequiredDLLs -- all from files being deleted. The plan's must-have requires `go build ./...` to succeed.
- **Fix:** Rewrote checkWineStatus() to call wine.FindProton() and checkPrefixStatus() to call wine.CompatdataPath()/wine.CompatdataHealthy(). Kept existing wineStatusResult/prefixStatusResult struct types for compatibility with status.go display code (full rewrite is Plan 02).
- **Files modified:** internal/cli/status_linux.go
- **Verification:** `go build ./...` succeeds
- **Committed in:** 7683408 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Auto-fix was necessary for the must-have compilation requirement. Plan 02 already accounts for status_linux.go rewrite, so this is compatible.

## Issues Encountered
None -- plan executed cleanly aside from the expected status_linux.go compilation dependency.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Plan 02 (status rewrite + CLAUDE.md update) can proceed immediately
- status_linux.go is now using Proton-era functions but still wrapped in legacy struct types (Plan 02 rewrites these)
- All deleted code is fully excised with no remaining references

## Self-Check: PASSED

All deleted files confirmed absent. All modified files confirmed present. Both commit hashes (662ede2, 7683408) verified in git log.

---
*Phase: 08-cleanup-and-polish*
*Completed: 2026-02-27*
