---
phase: 07-controller-gamescope-integration
plan: 03
subsystem: launch
tags: [steam-deck, controller, gamescope, hardware-validation, x11, steam-input]

# Dependency graph
requires:
  - phase: 07-controller-gamescope-integration
    provides: SteamGameId and STEAM_COMPAT_CLIENT_INSTALL_PATH wiring (07-01, 07-02)
  - phase: 06-core-proton-launch-pipeline
    provides: Proton launch pipeline infrastructure
provides:
  - Hardware validation evidence that SteamGameId/Gamescope window tracking does NOT fix controller loss
  - X11 diagnostic evidence (WM_CLASS, GAMESCOPE_FOCUSED_APP, STEAM_GAME) for fallback strategy
  - Confirmed root cause: Steam Input firmware reconfiguration during UE3 ServerTravel is independent of window tracking
affects: [08-cleanup-polish, controller-debugging, steam-deck]

# Tech tracking
tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified: []

key-decisions:
  - "Gamescope window tracking hypothesis disproven -- controller loss is not caused by focus/window class issues"
  - "CTRL-03 remains unsatisfied -- controller input does not persist through ServerTravel on Steam Deck"
  - "Env var changes (SteamGameId, SteamAppId, STEAM_COMPAT_CLIENT_INSTALL_PATH) still valuable for correct Proton integration even though they do not fix controller drop"

patterns-established: []

requirements-completed: []

# Metrics
duration: 0min
completed: 2026-02-25
---

# Phase 07 Plan 03: Hardware Validation of Controller Persistence Summary

**Steam Deck hardware test FAILED: controller input drops at ServerTravel despite correct Gamescope window tracking (SteamGameId, WM_CLASS, GAMESCOPE_FOCUSED_APP all correct)**

## Performance

- **Duration:** 0 min (documentation of hardware test results; no code tasks)
- **Started:** 2026-02-25T14:17:45Z
- **Completed:** 2026-02-25T14:17:45Z
- **Tasks:** 1 (checkpoint:human-verify)
- **Files modified:** 0

## Hardware Test Results

**Verdict: FAILED**

Controller buttons do NOT persist through the lobby-to-match transition (UE3 ServerTravel) on Steam Deck, even with correct SteamGameId and Gamescope window tracking.

### Environment Verification (PASSED)

The env var changes from Plans 07-01 and 07-02 are working correctly:

| Variable | Value | Status |
|----------|-------|--------|
| SteamGameId | 3928144816 | Correctly set from shortcuts.vdf |
| SteamAppId | 3928144816 | Matches SteamGameId |
| STEAM_COMPAT_CLIENT_INSTALL_PATH | (detected path) | Set correctly |

### X11 Property Investigation

Diagnostic results from xprop inspection on Steam Deck:

| Property | Display | Value | Status |
|----------|---------|-------|--------|
| WM_CLASS on game window | :1 (nested) | `steam_app_3928144816` | CORRECT -- Proton Wine set class hint from SteamGameId |
| GAMESCOPE_FOCUSED_APP | :0 (root) | `3928144816` | CORRECT -- Gamescope is tracking the right window |
| STEAM_GAME | :0 (root) | Not found | ABSENT -- standalone proton run does not set this |

### Controller Behavior

- **Main menu:** Controller works (A/B/X/Y, bumpers, triggers, joysticks all functional)
- **First ServerTravel (menu to lobby):** Controller input DROPS completely
- **Lobby:** No controller input
- **In-match:** No controller input

### Root Cause Analysis

The Gamescope window tracking hypothesis has been **disproven**. The evidence shows:

1. **SteamGameId is correctly propagated** -- Proton Wine reads the env var and sets WM_CLASS to `steam_app_3928144816` on the game window.

2. **Gamescope tracks the window correctly** -- GAMESCOPE_FOCUSED_APP on Display :0 shows `3928144816`, confirming Gamescope associates the game window with the correct app ID even after D3D window recreation.

3. **Controller still drops** -- Despite correct window class hints and Gamescope focus tracking, Steam Input still reconfigures the controller firmware during the UE3 ServerTravel transition.

**Conclusion:** The controller input loss is caused by Steam Input's firmware-level behavior during UE3 ServerTravel (D3D window destruction/recreation), and this behavior is **independent** of Gamescope window class/focus state. The STEAM_GAME X11 property (which is absent because we use standalone `proton run` rather than launching through Steam) may be a factor, but setting it would require launching through Steam's own runtime -- a fundamentally different launch approach.

## Accomplishments

- Hardware validation completed on actual Steam Deck hardware
- Gamescope window tracking hypothesis definitively disproven with X11 property evidence
- Root cause narrowed: Steam Input firmware reconfiguration is independent of Gamescope focus/window association
- Env var changes from 07-01/07-02 confirmed working (correct WM_CLASS, correct GAMESCOPE_FOCUSED_APP)

## Task Commits

No code commits -- this plan was a hardware validation checkpoint only.

## Files Created/Modified

None -- hardware validation and documentation only.

## Decisions Made

- **Gamescope hypothesis disproven:** The env var approach (SteamGameId/SteamAppId/STEAM_COMPAT_CLIENT_INSTALL_PATH) correctly establishes window class hints and Gamescope tracking, but Steam Input firmware-level reconfiguration still occurs during D3D window recreation regardless. This is not a focus/tracking issue.
- **CTRL-03 remains unsatisfied:** Controller input does not persist through ServerTravel on Steam Deck. This requirement cannot be met with the current env-var-only approach.
- **Env var changes retained:** Despite not fixing the controller drop, the SteamGameId and STEAM_COMPAT_CLIENT_INSTALL_PATH changes from 07-01/07-02 are still valuable for correct Proton integration (proper window class hints, Steam client library access).

## Deviations from Plan

None -- the plan was a checkpoint with defined pass/fail criteria. The FAIL path was documented as specified.

## Issues Encountered

None -- the hardware test executed as planned. The result was a failure of the hypothesis, not a failure of the test process.

## User Setup Required

None - no external service configuration required.

## Fallback Strategies for Controller Persistence

Based on the diagnostic evidence, the following approaches may address the controller drop in future work:

1. **Launch through Steam as non-Steam game** -- Instead of standalone `proton run`, launch via Steam's own runtime. This would set the STEAM_GAME X11 property and give Steam Input full session context. Requires `steam -applaunch {appid}` or Steam URL handler. Trade-off: loses direct process control, adds Steam as a runtime dependency.

2. **External USB/Bluetooth controller** -- Bypass Steam Input entirely by using a non-Neptune controller. External Xbox/PlayStation controllers are not subject to Steam Input firmware reconfiguration because they use standard HID rather than the Neptune firmware interface. Trade-off: requires additional hardware.

3. **Steam community/Valve fix** -- The ServerTravel D3D recreation pattern is a known issue with UE3 games under Steam Input. Valve may address this in future Gamescope/Steam Input updates. Trade-off: no ETA, not actionable.

4. **STEAM_INPUT_DISABLE=1 with external input** -- Disable Steam Input entirely and rely on a different input method. Trade-off: loses all Steam Deck built-in controller support (touchpads, gyro, back buttons).

## Next Phase Readiness

- Phase 7 implementation work (07-01, 07-02) is complete and correct -- env vars are properly wired
- CTRL-03 is NOT satisfied -- controller persistence through ServerTravel remains an open problem
- Phase 8 (Cleanup and Polish) can proceed independently as it focuses on code cleanup, not controller fixes
- Controller fix should be tracked as a v1.2+ item (see fallback strategies above)

## Self-Check: PASSED

- [x] 07-03-SUMMARY.md created with hardware test results
- [x] X11 diagnostic evidence documented (WM_CLASS, GAMESCOPE_FOCUSED_APP, STEAM_GAME)
- [x] Fallback strategies documented
- [x] No code commits expected (hardware validation checkpoint only)

---
*Phase: 07-controller-gamescope-integration*
*Completed: 2026-02-25*
