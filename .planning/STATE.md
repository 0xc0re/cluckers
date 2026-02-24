# Project State

Last activity: 2026-02-24 - Completed Phase 04 Plan 05: CI/CD for GUI and CLI-only Dual-Build (Phase 04 COMPLETE)

## Current Phase Execution

- **Phase:** 04-cross-platform-gui (COMPLETE - 5/5 plans)
- **Current Plan:** 5 of 5 (all complete)
- **Last Completed:** 04-05-PLAN.md (CI/CD updates: goreleaser dual-build, workflow changes, human verification)
- **Last Session:** 2026-02-24T16:35:54.680Z
- **Stopped At:** Completed 04-05-PLAN.md (Phase 04 complete)

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 10 | Resolve Steam Deck controller input not recognized by Realm Royale | 2026-02-23 | 5409539 | [10-resolve-steam-deck-controller-input-not-](./quick/10-resolve-steam-deck-controller-input-not-/) |
| 11 | Create CLAUDE.md and code map | 2026-02-23 | n/a (no git commit) | [11-create-claude-md-and-code-map](./quick/11-create-claude-md-and-code-map/) |
| 13 | Resolve controller input loss in match (XInputEnable + WINEDLLOVERRIDES fix) | 2026-02-23 | e95ea1b | [13-resolve-controller-input-loss-in-match](./quick/13-resolve-controller-input-loss-in-match/) |
| 14 | Launch-readiness code review: remove committed binary, fix docs, deduplicate isSteamDeck | 2026-02-24 | 7db50fd | [14-perform-a-thorough-review-of-the-code-an](./quick/14-perform-a-thorough-review-of-the-code-an/) |
| 15 | Add Windows build target with cross-platform code split | 2026-02-24 | 8cdd503 | [15-make-a-windows-version-no-wine-needed-ju](./quick/15-make-a-windows-version-no-wine-needed-ju/) |
| 16 | Implement self-update command for launcher binary replacement via GitHub releases | 2026-02-24 | a534a0c | [16-implement-self-update-we-have-update-for](./quick/16-implement-self-update-we-have-update-for/) |
| 17 | Split token cache into independent per-token TTL timestamps | 2026-02-24 | e3f7dec | [17-login-and-launch-should-refresh-oidc-tok](./quick/17-login-and-launch-should-refresh-oidc-tok/) |
| 19 | Update docs and install script for Windows support | 2026-02-24 | 6f943ae | [19-implement-windows-builds-and-releases-up](./quick/19-implement-windows-builds-and-releases-up/) |
| 20 | Resolve code scanning alert (add CI workflow permissions) | 2026-02-24 | e5ec06c | [20-resolve-code-scanning-results-from-githu](./quick/20-resolve-code-scanning-results-from-githu/) |

## Accumulated Context

### Roadmap Evolution
- Phase 4 added: cross platform gui

### Decisions
- All GUI package files use //go:build gui tag to keep CLI-only build path clean (CGO_ENABLED=0)
- Steam Deck detection in GUI package is independent of wine package (uses DMI board vendor)
- GUI binary: CGO_ENABLED=1 go build -tags gui; CLI-only binary: CGO_ENABLED=0 go build (unchanged)
- Fyne v2.7.3 selected as GUI framework (most mature Go GUI, cross-platform, works on SteamOS)
- ProgressReporter interface stored on LaunchState; Step.Fn simplified to (ctx, state) signature
- Login screen uses fyne.Do() for goroutine-to-UI updates per Fyne v2.6+ threading model
- Screen navigation via w.SetContent() swapping between login and main view
- Saved credentials checked at startup to skip login for returning users
- [Phase 04]: ProgressReporter interface stored on LaunchState; Step.Fn simplified to (ctx, state) signature
- [Phase 04]: Login screen uses fyne.Do() for goroutine-to-UI updates per Fyne v2.6+ threading model
- [Phase 04]: Settings uses widget.NewForm with runtime.GOOS for platform-conditional Wine fields
- [Phase 04]: Config persistence via viper.Set + viper.WriteConfigAs to TOML file
- [Phase 04]: Bot name field is placeholder until gateway endpoint documented
- [Phase 04]: Main view extracted from app.go to screens/main.go following screens package pattern
- [Phase 04]: StepListWidget uses container-based composition, not widget.BaseWidget, exposing layout via GetContainer()
- [Phase 04]: RunWithReporterAndCreds uses context cancellation (no os.Signal), stepAuthenticate detects pre-populated credentials
- [Phase 04]: buildSteps extracted as shared helper for DRY step construction across CLI and GUI pipelines
- [Phase 04]: Per-step CGO_ENABLED in CI (no job-level env) for mixed GUI+CLI builds
- [Phase 04]: Windows GUI uses CGO_ENABLED=0 (Fyne does not need CGO on Windows)
- [Phase 04]: goreleaser uses 3 build IDs (cluckers-cli, cluckers-gui-linux, cluckers-gui-windows) for per-target CGO
- [Phase 04]: Settings screen deferred to future release per user decision
