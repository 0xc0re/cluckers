---
phase: 09-steam-deck-input-again
plan: 02
subsystem: launch
tags: [vdf, shortcuts, steam-deck, steam-managed-launch, proton, rungameid]

# Dependency graph
requires:
  - phase: 09-steam-deck-input-again
    provides: "Binary VDF shortcut writer (AddShortcutToVDF, CalculateBPID) from Plan 01"
provides:
  - "Automated shortcuts.vdf writing on Steam Deck via cluckers steam add"
  - "Steam-managed launch mode (steam://rungameid/) for Steam Deck"
  - "platformLaunchStep() dispatch pattern for platform-specific launch behavior"
affects: [09-03, steam-deck-deployment]

# Tech tracking
tech-stack:
  added: []
  patterns: ["Platform-dispatched launch step via platformLaunchStep()", "Steam Deck shortcut automation with backup/idempotency"]

key-files:
  created: []
  modified:
    - internal/cli/steam_linux.go
    - internal/launch/pipeline.go
    - internal/launch/pipeline_linux.go
    - internal/launch/pipeline_windows.go

key-decisions:
  - "Steam-managed launch writes prep config then triggers steam://rungameid/ instead of direct proton run"
  - "Deck without shortcut falls back to direct proton run with warning suggesting cluckers steam add"
  - "shortcuts.vdf backed up to .cluckers-backup before modification"
  - "platformLaunchStep() replaces hardcoded stepLaunchGame for cross-platform dispatch"

patterns-established:
  - "platformLaunchStep() pattern: platform-specific launch dispatch without modifying shared pipeline.go build logic"
  - "Deck shortcut automation: find/create shortcuts.vdf, check existing, backup, write, print setup instructions"

requirements-completed: [CTRL-03]

# Metrics
duration: 2min
completed: 2026-02-28
---

# Phase 9 Plan 02: Steam-Managed Launch and Shortcut Automation Summary

**Automated shortcuts.vdf writing on Steam Deck and steam://rungameid/ launch dispatch for Steam-managed Proton lifecycle**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-28T21:23:33Z
- **Completed:** 2026-02-28T21:25:56Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- `cluckers steam add` on Steam Deck auto-writes shortcuts.vdf with shm_launcher.exe target and `cluckers prep && %command%` launch options
- `cluckers launch` on Steam Deck with configured shortcut runs prep pipeline then launches via `steam://rungameid/` for Steam-managed Proton
- Platform-dispatched launch step (`platformLaunchStep()`) enables Deck vs desktop vs Windows without modifying shared pipeline logic
- Desktop Linux unchanged: .desktop file creation with manual Steam instructions

## Task Commits

Each task was committed atomically:

1. **Task 1: Rewrite `cluckers steam add` to automate shortcuts.vdf on Steam Deck** - `8365d12` (feat)
2. **Task 2: Add Steam-managed launch mode for Steam Deck** - `915e15e` (feat)

## Files Created/Modified
- `internal/cli/steam_linux.go` - Rewritten: Deck path auto-writes shortcuts.vdf, desktop path creates .desktop file. No WINEDLLOVERRIDES.
- `internal/launch/pipeline.go` - Added SteamShortcutAppID to LaunchState, replaced hardcoded launch step with platformLaunchStep()
- `internal/launch/pipeline_linux.go` - Added stepLaunchGameLinux (Deck dispatch), launchViaSteam (steam://rungameid/), platformLaunchStep(), stores appID from shortcuts.vdf
- `internal/launch/pipeline_windows.go` - Added platformLaunchStep() returning standard stepLaunchGame

## Decisions Made
- Steam-managed launch writes prep config (bootstrap, OIDC, launch-config.txt) then triggers `steam://rungameid/<BPID>` instead of direct `proton run`. This lets Steam manage the Proton lifecycle so Gamescope tracks the game window through ServerTravel.
- On Deck without a configured shortcut, fall back to direct proton run with a warning. This avoids breaking existing workflows while encouraging the optimal path.
- shortcuts.vdf is backed up to `shortcuts.vdf.cluckers-backup` before any modification. If shortcut already exists (FindCluckersAppID returns non-zero), skip writing and just print setup instructions.
- `platformLaunchStep()` replaces the hardcoded `Step{Name: "Launching game", Fn: stepLaunchGame}` in buildSteps(). Each platform provides its own launch step function. Windows returns the standard stepLaunchGame. Linux returns stepLaunchGameLinux which dispatches based on IsSteamDeck() and SteamShortcutAppID.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Steam-managed launch path complete: shortcuts.vdf writing + steam://rungameid/ dispatch
- Plan 03 can now focus on hardware deployment testing on Steam Deck
- All verification passes: go build, go vet (both platforms), go test ./..., no WINEDLLOVERRIDES

## Self-Check: PASSED

- All modified files exist (steam_linux.go, pipeline.go, pipeline_linux.go, pipeline_windows.go)
- Both commits found (8365d12, 915e15e)
- No WINEDLLOVERRIDES=dxgi in production Go source
- All tests pass, go vet clean on both platforms

---
*Phase: 09-steam-deck-input-again*
*Completed: 2026-02-28*
