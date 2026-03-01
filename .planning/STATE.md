# Project State

Last activity: 2026-03-01 - Completed quick task 38: GUI background launch, system tray, download progress

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-24)

**Core value:** A user can download one file and go from zero to playing Realm Royale on Project Crown
**Current focus:** v1.1 Full Controller Functionality on Steam Deck

## Current Position

Phase: 9 (Steam Deck Input Again)
Plan: 2 of 3 in current phase — COMPLETE
Status: Executing Phase 9
Last activity: 2026-02-28 -- Steam-managed launch + shortcut automation

Progress: [#############       ] 67% (Phase 9: 2/3 plans complete)

## Performance Metrics

**Velocity (v1.0):**
- Total plans completed: 14
- Average duration: ~25 min
- Total execution time: ~6 hours

**By Phase (v1.0):**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1. Foundation | 3 | ~1h | ~20 min |
| 2. Wine/Game | 3 | ~1h | ~20 min |
| 4. GUI | 5 | ~2.5h | ~30 min |
| 5. AppImage | 3 | ~1.5h | ~30 min |

*v1.1 metrics will be tracked from Phase 6 onward*

**By Phase (v1.1):**

| Phase | Plan | Duration | Tasks | Files |
|-------|------|----------|-------|-------|
| 6. Core Proton | 06-01 | 5min | 3 (TDD) | 4 |
| 6. Core Proton | 06-02 | 3min | 1 (TDD) | 2 |
| 6. Core Proton | 06-03 | 3min | 2 | 4 |
| 7. Controller/Gamescope | 07-01 | 1min | 2 (TDD) | 2 |
| 7. Controller/Gamescope | 07-02 | 3min | 2 | 8 |
| 7. Controller/Gamescope | 07-03 | 0min | 1 (checkpoint) | 0 |
| 7.1 Input Proxy | 07.1-01 | 6min | 2 (TDD) | 6 |
| 7.1 Input Proxy | 07.1-02 | 7min | 2 (TDD) | 6 |
| 7.1 Input Proxy | 07.1-03 | 3min | 2 | 5 |
| 7.1 Input Proxy | 07.1-04 | ~6h | 14 deploys | 4 |
| 8. Cleanup | 08-01 | 4min | 2 | 21 |
| 8. Cleanup | 08-02 | 6min | 2 | 6 |
| 9. Steam Deck Input | 09-01 | 2min | 2 | 6 |
| 9. Steam Deck Input | 09-02 | 2min | 2 | 4 |

## Accumulated Context

### Decisions

- See .planning/PROJECT.md Key Decisions table for full list
- v1.1: Switch from direct Wine to Proton launch pipeline for all Linux
- v1.1: Fresh Proton prefix at ~/.cluckers/compatdata/ (no migration of old prefix)
- v1.1: Proton-GE required for all Linux (system Wine fallback out of scope per REQUIREMENTS.md)
- v1.1: Direct `proton run` invocation (not umu-launcher)
- 06-01: Internal findProton(configOverride, home) pattern for testability
- 06-01: Tests skip "not found" scenarios when real Proton-GE exists at system paths
- 06-02: Injectable env pattern (buildProtonEnvFrom) for deterministic testing
- 06-02: SHM bridge error detection via 4 case-insensitive stderr patterns
- 06-02: proton run shmPath uses Linux path, bootstrapPath/gameExe use Wine Z: paths
- 06-03: Proton info stored as simple strings in cross-platform LaunchState (no wine package import)
- 06-03: ~~WinePath/WinePrefix kept in structs for Windows compat (Phase 8 cleanup candidate)~~ **RESOLVED (08-01):** Fields removed from LaunchState and LaunchConfig
- 07-01: Reuse resolveReal() and userHome() from detect.go for FindSteamInstall consistency
- 07-01: Priority order native > symlink > Flatpak > Snap; two marker files (steam.sh, steamclient.so)
- 07-02: SteamAppId set to match SteamGameId for Proton Wine X11 class hints (Gamescope window tracking)
- 07-02: All Steam detection failures non-fatal -- game launches with fallback values
- 07-02: stepResolveSteamIntegration placed after stepEnsureCompatdata in pipeline
- 07-03: CTRL-03 hardware test FAILED -- Gamescope window tracking hypothesis disproven; controller loss is Steam Input firmware behavior independent of window class/focus
- 07-03: Env var changes (SteamGameId, STEAM_COMPAT_CLIENT_INSTALL_PATH) retained for correct Proton integration despite not fixing controller drop
- 07-03: Controller persistence deferred to v1.2+ (fallback: launch through Steam, external USB controller, or Valve fix)
- 07.1-04: evdev proxy ABANDONED — uinput gamepad creation kills Steam Input virtual pads (14 deploys)
- 07.1-04: XInput DLL proxy ABANDONED — bypasses Proton's Steam Input IPC, breaks button input
- 07.1-04: Clean baseline CONFIRMED FAIL — Steam-managed Proton launch alone does not fix ServerTravel drop
- 07.1-04: Controller fix deferred to v1.2+ — firmware-level issue beyond software fix
- 08-02: Removed WinePrefix from Config and GUI settings; status command now shows Proton version and compatdata health
- 09-01: VDF writer uses appid=0 placeholder; Steam assigns real appid on restart
- 09-01: WINEDLLOVERRIDES kept in strippedEnvKeys but never re-added to env (dxgi=n causes instant crash)
- 09-02: Steam-managed launch writes prep config then triggers steam://rungameid/ instead of direct proton run
- 09-02: Deck without shortcut falls back to direct proton run with warning
- 09-02: platformLaunchStep() replaces hardcoded stepLaunchGame for cross-platform dispatch

### Roadmap Evolution

- Phase 7.1 inserted after Phase 7: Steam Deck controller input proxy (URGENT)
- Phase 7.1 outcome: FAIL — proxy cannot fix firmware-level issue, deferred to v1.2+
- Phase 9 added: Steam Deck Input (again)

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 34 | Fix Windows update verification and remove auto-update from launch | 2026-02-27 | ebd89b4 | [34-fix-windows-update-verification-and-remo](./quick/34-fix-windows-update-verification-and-remo/) |
| 35 | Fix permission denied error during game extraction on Linux | 2026-02-27 | cd25d21 | [35-fix-permission-denied-error-during-game-](./quick/35-fix-permission-denied-error-during-game-/) |
| 36 | UI/CLI review: fix double errors, dead code, stale descriptions | 2026-03-01 | cfd1760 | [36-complete-ui-and-cli-review-look-for-issu](./quick/36-complete-ui-and-cli-review-look-for-issu/) |
| 37 | Disable conflicting file-operation buttons during Verify/Update/Repair | 2026-03-01 | c690cc2 | [37-update-ui-button-states-dim-disable-butt](./quick/37-update-ui-button-states-dim-disable-butt/) |
| 38 | GUI background launch, system tray, download progress | 2026-03-01 | 5ae3793 | [38-update-gui-to-launch-in-background-minim](./quick/38-update-gui-to-launch-in-background-minim/) |

### Blockers/Concerns

- ~~MEDIUM confidence: Standalone `proton run` may not set STEAM_GAME X11 property for Gamescope.~~ **RESOLVED (07-03):** Hardware test confirmed STEAM_GAME absent and controller drops persist. Gamescope window tracking (WM_CLASS, GAMESCOPE_FOCUSED_APP) works correctly but does not prevent Steam Input firmware reconfiguration. Controller fix deferred to v1.2+ -- standalone `proton run` with env vars is correct for Proton integration; controller persistence requires a fundamentally different approach (launch through Steam runtime, external controller, or Valve fix).
- ~~Phase 7.1 proxy code should be removed or disabled in Phase 8 cleanup~~ **RESOLVED (08-01):** Entire inputproxy package deleted, all proxy references removed

## Session Continuity

Last session: 2026-03-01
Stopped at: Completed quick task 38 (GUI background launch, system tray, download progress)
Resume file: none
