# Project State

Last activity: 2026-02-25 - Completed quick task 30: make sure appimage builds locally and in CI

## Current Phase Execution

- **Phase:** 05-containers-appimage (COMPLETE - 3/3 plans)
- **Current Plan:** Not started
- **Last Completed:** 05-03-PLAN.md (CI/CD integration and AppImage-aware self-update)
- **Last Session:** 2026-02-25T01:12:15.894Z
- **Stopped At:** Completed quick task 30: AppImage build verification

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
| 21 | Fix Windows launch issues: zip removal, Steam add, display config | 2026-02-24 | dbf09a4 | [21-fix-windows-launch-issues-zip-removal-af](./quick/21-fix-windows-launch-issues-zip-removal-af/) |
| 22 | Merge GUI and CLI into single release archive | 2026-02-24 | 29878b7 | [22-gui-and-cli-were-supposed-to-be-a-single](./quick/22-gui-and-cli-were-supposed-to-be-a-single/) |
| 23 | Single binary with GUI merge (no separate cluckers-gui) | 2026-02-24 | f2d4904 | [23-single-windows-binary-with-gui-merge-clu](./quick/23-single-windows-binary-with-gui-merge-clu/) |
| 24 | Fix self-update on Windows (platform-specific binary replacement) | 2026-02-24 | 888303b | [24-cluckers-self-update-does-not-work-on-wi](./quick/24-cluckers-self-update-does-not-work-on-wi/) |
| 25 | Update GUI URLs, icon, settings cleanup, two bot name fields | 2026-02-24 | 2635841 | [25-update-gui-support-url-discord-link-remo](./quick/25-update-gui-support-url-discord-link-remo/) |
| 26 | Review and update README documentation with Steam integration instructions | 2026-02-24 | f760143 | [26-review-then-update-documentation-add-ins](./quick/26-review-then-update-documentation-add-ins/) |
| 27 | Fix bot name setting: inline auth fallback and improved error display | 2026-02-24 | 7bfc93a | [27-setting-bot-names-fails](./quick/27-setting-bot-names-fails/) |
| 28 | Dependabot and repo maintenance automation | 2026-02-24 | 06fae6c | [28-implement-github-dependabot-version-upda](./quick/28-implement-github-dependabot-version-upda/) |
| 29 | Build-time endpoint configuration via ldflags | 2026-02-24 | db390d9 | [29-implement-changes-that-will-make-it-easi](./quick/29-implement-changes-that-will-make-it-easi/) |
| 30 | Verify AppImage builds locally and in CI, fix strip/FUSE/zsync issues | 2026-02-25 | cea7ee9 | [30-make-sure-appimage-builds-locally-and-in](./quick/30-make-sure-appimage-builds-locally-and-in/) |

## Accumulated Context

### Roadmap Evolution
- Phase 4 added: cross platform gui
- Phase 5 added: Containers / AppImage

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
- [Quick 22]: Single goreleaser archive per platform replacing separate CLI and GUI archives
- [Quick 22]: Install script asset regex anchored to ^cluckers_ to avoid ambiguous matching
- [Quick 22]: Fixed irm -> iwr in install.ps1 usage comment (irm auto-parses, breaks pipe to iex)
- [Phase quick-23]: Windows Fyne build requires CGO_ENABLED=1 + mingw (go-gl/gl needs CGO on all platforms)
- [Quick 24]: Windows binary replacement uses rename-swap strategy (rename running exe, place new, cleanup old on next run)
- [Quick 24]: CleanupOldBinary called at start of self-update command only, not on every app startup
- [Quick 25]: Bot name API uses cached access token via auth.LoadTokenCache() rather than re-authenticating
- [Quick 25]: Game Server (hostx) removed from settings UI but default retained in config.go
- [Quick 25]: Settings form wrapped in 440px GridWrap for wider input fields
- [Phase quick-26]: Restructured Steam integration into dedicated 'Adding to Steam' section with platform subsections
- [Phase quick-27]: Bot name handler authenticates inline via auth.Login() fallback when no cached token exists
- [Quick 28]: Grouped Dependabot updates for golang.org/* and fyne.io/* to reduce PR noise
- [Quick 28]: Stale thresholds: 60/30 days (issue/PR) with 14/7 day close grace period
- [Quick 29]: Build-time ldflags inject gateway URL and hostx IP; SetBuildDefaults pattern with InitFlags() for CLI help text
- [Quick 29]: GitHub repo variables (vars.*) used for endpoint config, not secrets (URLs are not sensitive)
- [Quick 29]: Fallback defaults via || syntax in release workflow so builds work without repo variables
- [Phase 05-02]: curl used over wget for Proton-GE download (more universally available)
- [Phase 05-02]: Proton-GE tarball deleted after extraction to save CI disk space
- [Phase 05-02]: resolve_cmd helper handles both standalone and AppImage-suffixed tool binaries
- [Phase 05-01]: AppImage detection via env vars (APPIMAGE, APPDIR, CLUCKERS_BUNDLED_PROTON) is cross-platform safe -- no build tags needed
- [Phase 05-01]: LD_LIBRARY_PATH stripped entirely from Wine env rather than selectively filtered -- Wine manages its own library paths
- [Phase 05-01]: Bundled Proton-GE is priority 2 in FindWine chain (between user config override and system Proton-GE scan)
- [Phase 05-03]: Self-update reads APPIMAGE env var to determine download artifact type at runtime
- [Phase 05-03]: replaceAppImage uses os.ReadFile/os.WriteFile for in-place overwrite (no archive extraction)
- [Phase 05-03]: CI uploads AppImage via gh release upload with goreleaser extra_files as backup
- [Phase 05-03]: Install script prefers AppImage URL; falls back to tar.gz for older releases
- [Phase 05-03]: Build script RUNTIME_PATH made configurable via env var for CI override
- [Phase quick-30]: NO_STRIP=true for linuxdeploy: bundled strip too old for .relr.dyn ELF sections on modern distros
- [Phase quick-30]: APPIMAGE_EXTRACT_AND_RUN=1 for CI: ubuntu-22.04 runners lack FUSE for AppImage tools
