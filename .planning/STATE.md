# Project State

Last activity: 2026-02-25 - Phase 07 (Controller & Gamescope Integration) in progress - 07-02 complete

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-24)

**Core value:** A user can download one file and go from zero to playing Realm Royale on Project Crown
**Current focus:** v1.1 Full Controller Functionality on Steam Deck

## Current Position

Phase: 7 of 8 (Controller & Gamescope Integration)
Plan: 2 of 3 in current phase
Status: In Progress
Last activity: 2026-02-25 -- Completed 07-02 Steam integration pipeline wiring

Progress: [##################░░] 90% (v1.0 complete, v1.1 phase 7: 2/3 plans)

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
- 06-03: WinePath/WinePrefix kept in structs for Windows compat (Phase 8 cleanup candidate)
- 07-01: Reuse resolveReal() and userHome() from detect.go for FindSteamInstall consistency
- 07-01: Priority order native > symlink > Flatpak > Snap; two marker files (steam.sh, steamclient.so)
- 07-02: SteamAppId set to match SteamGameId for Proton Wine X11 class hints (Gamescope window tracking)
- 07-02: All Steam detection failures non-fatal -- game launches with fallback values
- 07-02: stepResolveSteamIntegration placed after stepEnsureCompatdata in pipeline

### Blockers/Concerns

- MEDIUM confidence: Standalone `proton run` may not set STEAM_GAME X11 property for Gamescope. Phase 6 must include hardware validation on Steam Deck. If it fails, pivot to requiring launch through Steam as non-Steam shortcut.

## Session Continuity

Last session: 2026-02-25
Stopped at: Completed 07-02-PLAN.md (Steam integration pipeline wiring)
Resume file: None
