---
phase: 04-cross-platform-gui
plan: 05
subsystem: infra
tags: [ci, goreleaser, github-actions, dual-build, gui-release, cgo, fyne]

# Dependency graph
requires:
  - phase: 04-cross-platform-gui
    plan: 03
    provides: "GUI screens, pipeline integration, GUIReporter"
  - phase: 04-cross-platform-gui
    plan: 04
    provides: "Settings screen, bot name, CLI coexistence verification"
provides:
  - "CI workflow building and vetting both GUI (-tags gui) and CLI-only variants"
  - "goreleaser config producing 4 binaries: cluckers (linux/windows) + cluckers-gui (linux/windows)"
  - "Release workflow with Fyne build dependencies for CGO_ENABLED=1 GUI builds"
  - "Separate cli and gui archive groups with distinct naming"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns: [per-step-cgo-control, dual-archive-goreleaser, combined-apt-install]

key-files:
  created: []
  modified:
    - .github/workflows/ci.yaml
    - .github/workflows/release.yaml
    - .goreleaser.yaml

key-decisions:
  - "Per-step CGO_ENABLED instead of job-level env (GUI needs CGO=1, CLI needs CGO=0)"
  - "Windows GUI uses CGO_ENABLED=0 because Fyne does not need CGO on Windows"
  - "Separate goreleaser build IDs (cluckers-cli, cluckers-gui-linux, cluckers-gui-windows) for mixed CGO"
  - "Combined mingw + Fyne deps into single apt-get step in release workflow for efficiency"

patterns-established:
  - "Dual-build CI pattern: CLI (CGO=0) and GUI (CGO=1) builds with shared checkout/setup"
  - "goreleaser multi-archive: separate cli and gui archive groups with distinct name templates"

requirements-completed: [GUI-09]

# Metrics
duration: 23min
completed: 2026-02-24
---

# Phase 04 Plan 05: CI/CD for GUI and CLI-only Dual-Build Summary

**CI/CD pipeline producing 4 binary variants (CLI+GUI x Linux+Windows) via goreleaser with per-build CGO control and Fyne system dependencies**

## Performance

- **Duration:** 23 min
- **Started:** 2026-02-24T16:11:11Z
- **Completed:** 2026-02-24T16:33:52Z
- **Tasks:** 3 (2 auto + 1 human-verify)
- **Files modified:** 3

## Accomplishments
- Updated CI workflow to build and vet 4 variants: CLI Linux, CLI Windows, GUI Linux (CGO=1), GUI Windows (CGO=0)
- Added Fyne system library dependencies (libgl1-mesa-dev, xorg-dev, libxkbcommon-dev) to CI and release workflows
- Configured goreleaser with 3 build targets (cluckers-cli, cluckers-gui-linux, cluckers-gui-windows) and 2 archive groups
- Removed job-level CGO_ENABLED=0 from both CI and release workflows to allow per-build CGO control
- User verified full GUI functionality: login, main view, launch progress, dark theme, headless fallback, CLI coexistence

## Task Commits

Each task was committed atomically:

1. **Task 1: Update CI workflow for dual-build (GUI + CLI-only)** - `043c842` (ci)
2. **Task 2: Update goreleaser and release workflow for GUI binaries** - `1dd0df2` (ci)
3. **Task 3: Verify GUI builds and full functionality** - Human-verified (approved)

## Files Created/Modified
- `.github/workflows/ci.yaml` - Added GUI build/vet steps, Fyne deps install, per-step CGO_ENABLED
- `.github/workflows/release.yaml` - Combined build deps, removed job-level CGO_ENABLED=0
- `.goreleaser.yaml` - Three build targets (cli, gui-linux, gui-windows), two archive groups (cli, gui)

## Decisions Made
- **Per-step CGO_ENABLED**: Job-level env removed so each build step controls its own CGO setting. GUI Linux needs CGO=1 for Fyne's OpenGL bindings; CLI and Windows GUI use CGO=0.
- **Windows GUI with CGO_ENABLED=0**: Fyne compiles to pure Go on Windows (no C compiler needed). The `gui` build tag just includes the GUI Go source files.
- **Combined apt-get in release**: Merged mingw-w64 and Fyne deps (gcc, libgl1-mesa-dev, xorg-dev, libxkbcommon-dev) into a single install step to reduce workflow time.
- **Separate goreleaser build IDs**: Three distinct builds allow per-target CGO, tags, and binary naming without goreleaser conflicts.

## Deviations from Plan

None - plan executed exactly as written.

## User Verification Notes

User approved with note: "settings will be available at a future release. everything else worked." The Settings screen (implemented in plan 04-04) is deferred to a future release per user decision. All other GUI functionality confirmed working: login screen, main view with launch/Discord/support/verify/update/repair, launch progress with step checkmarks, dark theme, headless CLI fallback, and CLI subcommand coexistence.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 04 (Cross-Platform GUI) is complete
- All 5 plans executed: Fyne foundation, login screen, main view + launch progress, settings screen, CI/CD
- GUI binary: `CGO_ENABLED=1 go build -tags gui -o cluckers-gui ./cmd/cluckers`
- CLI binary: `CGO_ENABLED=0 go build -o cluckers ./cmd/cluckers`
- Settings screen deferred to future release per user decision
- Ready for tag push to produce first GUI release

## Self-Check: PASSED

All 3 claimed files verified present. Task commits 043c842 and 1dd0df2 verified in git log. Content checks confirmed GUI build steps, vet steps, goreleaser build targets, and Fyne dependencies all present.

---
*Phase: 04-cross-platform-gui*
*Completed: 2026-02-24*
