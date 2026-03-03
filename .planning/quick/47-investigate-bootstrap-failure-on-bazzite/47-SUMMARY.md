---
phase: quick-47
plan: 01
subsystem: launch
tags: [wine, proton, bazzite, selinux, temp-files, immutable-distro]

# Dependency graph
requires:
  - phase: quick-45
    provides: base64 decoding fix for content bootstrap
  - phase: quick-46
    provides: dynamic bootstrap size instead of hardcoded 136
provides:
  - config.TmpDir() helper for Wine-accessible temp directory
  - All temp files (shm_launcher, bootstrap, OIDC token) written to ~/.cluckers/tmp/
  - Compatibility with SELinux, noexec /tmp, and container namespaced /tmp
affects: [launch, process_linux, process_windows]

# Tech tracking
tech-stack:
  added: []
  patterns: [user-data-dir temp files for Wine accessibility]

key-files:
  created:
    - internal/launch/shm_test.go
  modified:
    - internal/config/paths.go
    - internal/launch/shm.go
    - internal/launch/pipeline.go

key-decisions:
  - "TmpDir uses DataDir()/tmp following existing ConfigDir/CacheDir/BinDir pattern"
  - "EnsureDir called at each temp file creation site for idempotent auto-creation"

patterns-established:
  - "config.TmpDir(): Use for all Wine-consumed temp files instead of os.TempDir()"

requirements-completed: [BAZZITE-BOOTSTRAP-FIX]

# Metrics
duration: 2min
completed: 2026-03-03
---

# Quick Task 47: Fix Bootstrap Failure on Bazzite Summary

**Move all Wine-consumed temp files from /tmp/ to ~/.cluckers/tmp/ to fix Bazzite SELinux/container namespace access restrictions**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-03T20:56:41Z
- **Completed:** 2026-03-03T20:58:19Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Added `config.TmpDir()` helper returning `DataDir()/tmp` (follows existing ConfigDir/CacheDir/BinDir pattern)
- Migrated all three temp file creation functions to use `~/.cluckers/tmp/` instead of system `/tmp/`
- Added 4 tests verifying temp file location and auto-creation of tmp directory
- All existing tests pass, both platform builds succeed

## Task Commits

Each task was committed atomically:

1. **Task 1: Add TmpDir helper and migrate temp file creation** - `8b0bafb` (fix)
2. **Task 2: Add tests for temp file location** - `8305b0c` (test)

## Files Created/Modified
- `internal/config/paths.go` - Added TmpDir() helper function
- `internal/launch/shm.go` - ExtractSHMLauncher() and WriteBootstrapFile() use config.TmpDir()
- `internal/launch/pipeline.go` - writeOIDCTokenFile() uses config.TmpDir()
- `internal/launch/shm_test.go` - 4 tests verifying temp file location under CLUCKERS_HOME/tmp/

## Decisions Made
- TmpDir follows existing ConfigDir/CacheDir/BinDir pattern (filepath.Join(DataDir(), "tmp"))
- EnsureDir called at each creation site rather than once at startup, matching the idempotent pattern used throughout the codebase

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Bazzite users should be unblocked -- temp files now live in user-writable ~/.cluckers/tmp/
- Wine Z: drive can always access files under ~/.cluckers/ regardless of /tmp mount restrictions

## Self-Check: PASSED

All files exist, all commits verified, all content patterns confirmed.

---
*Phase: quick-47*
*Completed: 2026-03-03*
