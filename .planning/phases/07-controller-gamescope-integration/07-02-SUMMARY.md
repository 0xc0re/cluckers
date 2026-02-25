---
phase: 07-controller-gamescope-integration
plan: 02
subsystem: launch
tags: [steam, proton, pipeline, gamescope, env-vars, linux]

# Dependency graph
requires:
  - phase: 07-controller-gamescope-integration
    provides: FindSteamInstall() for Steam root directory detection (07-01)
  - phase: 06-core-proton-launch-pipeline
    provides: buildProtonEnvFrom, LaunchState, LaunchConfig, pipeline infrastructure
provides:
  - Updated buildProtonEnvFrom with steamInstallPath and steamGameId parameters
  - stepResolveSteamIntegration pipeline step for Steam path and app ID resolution
  - Exported FindCluckersAppID for pipeline use
  - SteamInstallPath and SteamGameId fields on LaunchState and LaunchConfig
affects: [07-03, gamescope, controller-input, steam-deck]

# Tech tracking
tech-stack:
  added: []
  patterns: [non-fatal-detection pipeline pattern, env-var parameterization]

key-files:
  created:
    - internal/launch/deckconfig_test.go
  modified:
    - internal/launch/proton_env.go
    - internal/launch/proton_env_test.go
    - internal/launch/deckconfig.go
    - internal/launch/pipeline.go
    - internal/launch/pipeline_linux.go
    - internal/launch/process.go
    - internal/launch/process_linux.go

key-decisions:
  - "SteamAppId set to match SteamGameId (not hardcoded 0) for Proton Wine X11 class hints"
  - "All Steam detection failures non-fatal -- game launches with fallback '0' values"
  - "stepResolveSteamIntegration placed after stepEnsureCompatdata in pipeline ordering"

patterns-established:
  - "Non-fatal detection step: stepResolveSteamIntegration returns nil on all failures, sets defaults"
  - "Env var parameterization: buildProtonEnvFrom accepts resolved values rather than detecting internally"

requirements-completed: [CTRL-01, CTRL-02]

# Metrics
duration: 3min
completed: 2026-02-25
---

# Phase 07 Plan 02: Steam Integration Pipeline Wiring Summary

**Parameterized Proton env vars (SteamGameId, SteamAppId, STEAM_COMPAT_CLIENT_INSTALL_PATH) from pipeline-resolved Steam path and shortcut app ID**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-25T09:47:42Z
- **Completed:** 2026-02-25T09:50:39Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments
- buildProtonEnvFrom accepts steamInstallPath and steamGameId, setting SteamGameId/SteamAppId/STEAM_COMPAT_CLIENT_INSTALL_PATH from resolved values
- New pipeline step "Resolving Steam integration" detects Steam root and resolves non-Steam shortcut app ID via FindCluckersAppID
- FindCluckersAppID exported from deckconfig.go for pipeline use, with 3 binary VDF fixture tests
- 4 new tests for parameterized Steam integration env vars, all existing tests updated and passing

## Task Commits

Each task was committed atomically:

1. **Task 1: Update buildProtonEnvFrom signature and export FindCluckersAppID** - `d92c79c` (feat)
2. **Task 2: Add Steam integration pipeline step and wire through LaunchState/LaunchConfig** - `a348473` (feat)

## Files Created/Modified
- `internal/launch/proton_env.go` - Updated buildProtonEnvFrom with steamInstallPath/steamGameId params
- `internal/launch/proton_env_test.go` - Updated existing tests to new signature, added 4 new Steam integration tests
- `internal/launch/deckconfig.go` - Exported FindCluckersAppID wrapper
- `internal/launch/deckconfig_test.go` - 3 tests for FindCluckersAppID with binary VDF fixtures
- `internal/launch/pipeline.go` - SteamInstallPath/SteamGameId on LaunchState, wired to LaunchConfig in stepLaunchGame
- `internal/launch/pipeline_linux.go` - stepResolveSteamIntegration step, added to platformSteps
- `internal/launch/process.go` - SteamInstallPath/SteamGameId fields on LaunchConfig
- `internal/launch/process_linux.go` - Updated buildProtonEnv call with new parameters

## Decisions Made
- SteamAppId set to match SteamGameId (not hardcoded "0") because Proton Wine reads SteamAppId for X11 class hints (steam_app_{id}), which Gamescope uses for window tracking
- All Steam detection failures are non-fatal -- the game still launches with fallback values ("0" for game ID, empty for install path)
- stepResolveSteamIntegration placed after stepEnsureCompatdata in pipeline ordering (logically separate from Proton detection)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added LaunchConfig fields in Task 1 instead of Task 2**
- **Found during:** Task 1 (process_linux.go compilation)
- **Issue:** Plan Task 1 updated process_linux.go to pass cfg.SteamInstallPath/cfg.SteamGameId but these fields were scheduled for Task 2 in process.go
- **Fix:** Added SteamInstallPath/SteamGameId to LaunchConfig in Task 1 to maintain compilation
- **Files modified:** internal/launch/process.go
- **Verification:** go build ./internal/launch/ succeeds
- **Committed in:** d92c79c (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Necessary to keep package compiling between tasks. No scope creep -- the field additions were planned work moved earlier.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Steam integration fully wired into pipeline: SteamGameId and STEAM_COMPAT_CLIENT_INSTALL_PATH flow from detection through to Proton env vars
- Ready for Plan 07-03 (Gamescope launch args and final integration testing)
- SteamGameId enables Gamescope window tracking across D3D recreation (ServerTravel)

## Self-Check: PASSED

- [x] internal/launch/proton_env.go modified
- [x] internal/launch/proton_env_test.go modified
- [x] internal/launch/deckconfig.go modified (FindCluckersAppID exported)
- [x] internal/launch/deckconfig_test.go created
- [x] internal/launch/pipeline.go modified
- [x] internal/launch/pipeline_linux.go modified
- [x] internal/launch/process.go modified
- [x] internal/launch/process_linux.go modified
- [x] Commit d92c79c (Task 1) exists
- [x] Commit a348473 (Task 2) exists

---
*Phase: 07-controller-gamescope-integration*
*Completed: 2026-02-25*
