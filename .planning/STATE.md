# Project State

Last activity: 2026-02-25 - Phase 07.1 plan 03 complete (proxy pipeline integration)

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-24)

**Core value:** A user can download one file and go from zero to playing Realm Royale on Project Crown
**Current focus:** v1.1 Full Controller Functionality on Steam Deck

## Current Position

Phase: 7.1 of 8 (Steam Deck Controller Input Proxy)
Plan: 3 of 4 in current phase
Status: Executing Phase 7.1 -- plan 03 complete
Last activity: 2026-02-25 -- Completed 07.1-03 proxy pipeline integration

Progress: [###################░] 97% (v1.0 complete, v1.1 phase 7.1: 3/4 plans, phase 8 remaining)

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
- 07-03: CTRL-03 hardware test FAILED -- Gamescope window tracking hypothesis disproven; controller loss is Steam Input firmware behavior independent of window class/focus
- 07-03: Env var changes (SteamGameId, STEAM_COMPAT_CLIENT_INSTALL_PATH) retained for correct Proton integration despite not fixing controller drop
- 07-03: Controller persistence deferred to v1.2+ (fallback: launch through Steam, external USB controller, or Valve fix)
- 07.1-01: Raw uinput ioctls via unix.Syscall instead of kenshaw/evdev UserInput (WithAbsoluteTypes is empty/no-op)
- 07.1-01: kenshaw/evdev for device detection (OpenFile + ID()), not for uinput creation
- 07.1-01: Button/axis constants as named Go constants in uinput.go for package-wide reuse
- 07.1-02: hadTrigActivity flag prevents false positive dead reckoning on button-only releases
- 07.1-02: Button constants (btnA, absZ, etc.) live in uinput.go alongside kernel ABI types
- 07.1-02: invertY implemented in uinput.go as shared utility
- 07.1-03: kenshaw/evdev Poll() for event reading (context-aware channel, typed event envelopes)
- 07.1-03: Proxy non-fatal on all systems; IsSteamDeck gate prevents startup on desktop Linux
- 07.1-03: 100ms sleep after virtual device creation for udev registration
- 07.1-03: Proxy cleanup prepended to defer chain for prompt goroutine shutdown

### Roadmap Evolution

- Phase 7.1 inserted after Phase 7: Steam Deck controller input proxy (URGENT)

### Blockers/Concerns

- ~~MEDIUM confidence: Standalone `proton run` may not set STEAM_GAME X11 property for Gamescope.~~ **RESOLVED (07-03):** Hardware test confirmed STEAM_GAME absent and controller drops persist. Gamescope window tracking (WM_CLASS, GAMESCOPE_FOCUSED_APP) works correctly but does not prevent Steam Input firmware reconfiguration. Controller fix deferred to v1.2+ -- standalone `proton run` with env vars is correct for Proton integration; controller persistence requires a fundamentally different approach (launch through Steam runtime, external controller, or Valve fix).

## Session Continuity

Last session: 2026-02-25
Stopped at: Completed 07.1-03-PLAN.md (proxy pipeline integration)
Resume file: None
