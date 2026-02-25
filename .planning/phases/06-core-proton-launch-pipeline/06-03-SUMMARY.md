---
phase: 06-core-proton-launch-pipeline
plan: 03
subsystem: launch
tags: [proton, pipeline, launch, linux, wine-replacement]

# Dependency graph
requires:
  - phase: 06-core-proton-launch-pipeline
    provides: "FindProton() with 3-tier detection and CompatdataHealthy() (plan 01)"
  - phase: 06-core-proton-launch-pipeline
    provides: "buildProtonEnv, buildProtonCommand, shmBridgeError, protonErrorSuggestion (plan 02)"
provides:
  - "Full Proton launch pipeline replacing Wine on Linux"
  - "stepDetectProton and stepEnsureCompatdata pipeline steps"
  - "LaunchGame using python3 proton run with stderr capture"
  - "LaunchState and LaunchConfig with ProtonScript, ProtonDir, CompatDataPath fields"
affects: [07-controller-gamescope-integration, 08-cleanup-polish]

# Tech tracking
tech-stack:
  added: []
  patterns: ["Proton pipeline integration wiring Plans 01+02 into live launch path"]

key-files:
  created: []
  modified:
    - internal/launch/pipeline.go
    - internal/launch/process.go
    - internal/launch/pipeline_linux.go
    - internal/launch/process_linux.go

key-decisions:
  - "Store Proton info as simple strings (ProtonScript, ProtonDir, ProtonDisplayVersion, CompatDataPath) in LaunchState to avoid importing linux-only wine package from cross-platform code"
  - "Keep WinePath and WinePrefix in LaunchState/LaunchConfig for Windows pipeline compatibility"
  - "Local regex for version warning in pipeline step instead of exporting wine.protonVersionRe"

patterns-established:
  - "Cross-platform struct fields as simple strings: avoid importing platform-specific packages by flattening struct data into primitive types"

requirements-completed: [PROTON-01, PROTON-02, PROTON-03, PROTON-04, PROTON-05]

# Metrics
duration: 3min
completed: 2026-02-25
---

# Phase 6 Plan 03: Proton Pipeline Integration Summary

**Full Proton launch pipeline replacing Wine: stepDetectProton + stepEnsureCompatdata + python3 proton run with stderr capture and SHM bridge error detection**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-25T05:45:05Z
- **Completed:** 2026-02-25T05:48:00Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Replaced 3 Wine pipeline steps (Detecting Wine, Ensuring Wine prefix, Verifying Wine prefix) with 2 Proton steps (Detecting Proton, Preparing Proton environment)
- Rewrote LaunchGame to invoke python3 proton run instead of wine64, with stderr capture and SHM bridge error detection
- Added ProtonScript, ProtonDir, ProtonDisplayVersion, CompatDataPath fields to LaunchState and LaunchConfig without breaking Windows builds
- Corrupted compatdata auto-detected and recreated with warning; first launch shows informational message
- Verbose mode shows Proton version, compatdata path, and PROTON_LOG=1 environment variable

## Task Commits

Each task was committed atomically:

1. **Task 1: Update LaunchState and LaunchConfig for Proton** - `694ce27` (feat)
2. **Task 2: Rewrite Linux pipeline steps and LaunchGame for Proton** - `6f1ad72` (feat)

## Files Created/Modified
- `internal/launch/pipeline.go` - Added ProtonScript, ProtonDir, ProtonDisplayVersion, CompatDataPath to LaunchState; updated stepLaunchGame to pass Proton fields
- `internal/launch/process.go` - Added ProtonScript, ProtonDir, CompatDataPath to LaunchConfig
- `internal/launch/pipeline_linux.go` - Replaced Wine steps with stepDetectProton and stepEnsureCompatdata; version warning for Proton-GE < 9
- `internal/launch/process_linux.go` - Rewrote LaunchGame to use python3 proton run with buildProtonEnv/buildProtonCommand, stderr capture, SHM bridge error detection

## Decisions Made
- Stored Proton info as simple strings (ProtonScript, ProtonDir, etc.) in the cross-platform LaunchState struct rather than importing the linux-only wine package. This avoids build tag complications and keeps pipeline.go compilable on both platforms.
- Kept WinePath and WinePrefix fields in LaunchState/LaunchConfig for Windows compatibility (process_windows.go still uses them). Phase 8 cleanup can remove if desired.
- Used a local regex (`protonMajorVersionRe`) in pipeline_linux.go for the version warning instead of exporting the unexported `protonVersionRe` from detect.go, since detect.go is not in the plan's files_modified list.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 6 (Core Proton Launch Pipeline) is fully complete: detection, environment, and pipeline integration
- Ready for Phase 7 (Controller/Gamescope Integration) which builds on the Proton launch path
- Hardware validation on Steam Deck recommended to verify standalone proton run sets correct X11 properties for Gamescope

## Self-Check: PASSED

All files verified present:
- internal/launch/pipeline.go (modified)
- internal/launch/process.go (modified)
- internal/launch/pipeline_linux.go (modified)
- internal/launch/process_linux.go (modified)
- .planning/phases/06-core-proton-launch-pipeline/06-03-SUMMARY.md (created)

All commits verified:
- 694ce27 (Task 1: LaunchState and LaunchConfig)
- 6f1ad72 (Task 2: pipeline_linux.go and process_linux.go)

---
*Phase: 06-core-proton-launch-pipeline*
*Completed: 2026-02-25*
