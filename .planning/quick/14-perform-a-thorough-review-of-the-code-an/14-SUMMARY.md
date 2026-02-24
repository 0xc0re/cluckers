---
phase: quick-14
plan: 01
subsystem: infra
tags: [ci, release, documentation, cleanup, goreleaser, mingw-w64]

# Dependency graph
requires: []
provides:
  - Clean repo with no committed binaries (shm_launcher.exe built from source)
  - Accurate CLAUDE.md, README.md, SECURITY.md documentation
  - Release workflow that builds shm_launcher.exe from source before goreleaser
  - Deduplicated wine.IsSteamDeck() function
affects: [release, documentation, steam-deck]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Build embedded binaries from source in CI/release (not committed to git)"
    - "Shared platform detection functions in wine package"

key-files:
  created: []
  modified:
    - .github/workflows/release.yaml
    - .gitignore
    - CLAUDE.md
    - README.md
    - SECURITY.md
    - internal/wine/detect.go
    - internal/launch/deckconfig.go
    - internal/cli/steam.go

key-decisions:
  - "Removed shm_launcher.exe from git tracking -- binary built from source in CI and release workflows"
  - "Moved IsSteamDeck() to wine package as single source of truth for platform detection"
  - "Updated tools/ section to mark xinput source files as historical (source retained for reference)"

patterns-established:
  - "Embedded binaries are not committed to git; built from source in CI/release"

requirements-completed: []

# Metrics
duration: 4min
completed: 2026-02-24
---

# Quick Task 14: Launch-Readiness Code Review Summary

**Removed committed binary from git, fixed stale XInput proxy references across all docs, deduplicated Steam Deck detection into wine.IsSteamDeck()**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-24T03:11:06Z
- **Completed:** 2026-02-24T03:15:30Z
- **Tasks:** 3
- **Files modified:** 8

## Accomplishments
- Removed `assets/shm_launcher.exe` from git tracking; release workflow now builds it from source via mingw-w64 before goreleaser runs
- Fixed all stale XInput proxy references across CLAUDE.md, README.md, and SECURITY.md; added missing `login` command documentation
- Deduplicated Steam Deck detection: removed private `isSteamDeck()` and `detectSteamDeck()` in favor of exported `wine.IsSteamDeck()`

## Task Commits

Each task was committed atomically:

1. **Task 1: Remove shm_launcher.exe from git and build from source in release workflow** - `2acc0ae` (chore)
2. **Task 2: Fix stale documentation (CLAUDE.md, README.md, SECURITY.md)** - `54da573` (docs)
3. **Task 3: Deduplicate isSteamDeck into wine.IsSteamDeck()** - `7db50fd` (refactor)

## Files Created/Modified
- `.github/workflows/release.yaml` - Added mingw-w64 install and shm_launcher.exe build steps before goreleaser; added CGO_ENABLED=0 env
- `.gitignore` - Added shm_launcher.exe and xinput1_3_remap.dll to ignore list with build instructions
- `CLAUDE.md` - Removed XInput proxy references, updated code map (added login.go, fixed deckconfig.go/process.go descriptions, updated assets/tools sections, build instructions, domain knowledge)
- `README.md` - Added login command, replaced XInput proxy section with INI patching description, moved Go to build-from-source prerequisites
- `SECURITY.md` - Updated embedded binary section to document both assets (shm_launcher.exe and controller_neptune_config.vdf) with source audit note
- `internal/wine/detect.go` - Added exported `IsSteamDeck()` function
- `internal/launch/deckconfig.go` - Replaced private `isSteamDeck()` with `wine.IsSteamDeck()`
- `internal/cli/steam.go` - Replaced private `detectSteamDeck()` with `wine.IsSteamDeck()`

## Decisions Made
- Removed shm_launcher.exe from git rather than keeping it as a convenience binary. CI and release both build from source, and the build command is documented for local dev.
- Placed `IsSteamDeck()` in `internal/wine/detect.go` rather than `internal/launch/` because Steam Deck detection is a platform concern that aligns with Wine/distro detection, and avoids circular import issues (cli imports launch, launch imports wine).
- Marked xinput source files in tools/ as "historical" rather than deleting them, preserving the source for reference while making clear they are not part of the active build pipeline.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Updated tools/ section to remove stale xinput1_3_remap.dll build reference**
- **Found during:** Task 2 (documentation fixes)
- **Issue:** Plan said "keep tools/ as-is" but verification required no `xinput1_3_remap` references in CLAUDE.md. The tools/ section contained the full build command including output DLL name.
- **Fix:** Kept tools/ entries but marked xinput_remap.c and xinput1_3.def as historical, removing the build command that referenced xinput1_3_remap.dll
- **Files modified:** CLAUDE.md
- **Verification:** `grep 'xinput1_3_remap' CLAUDE.md` returns no matches
- **Committed in:** 54da573 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** Minor scope expansion to satisfy verification criteria. No scope creep.

## Issues Encountered
- `git add assets/shm_launcher.exe` failed after the file was added to .gitignore (it was already staged for deletion by `git rm --cached`). Resolved by adding only the other modified files to staging.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Codebase is clean: no committed binaries, accurate documentation, release pipeline builds from source
- All builds, tests, and vet pass
- Ready for tag push and release

---
*Quick Task: 14*
*Completed: 2026-02-24*
