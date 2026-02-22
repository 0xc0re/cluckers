# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-22)

**Core value:** A Linux user can run one command and go from zero to playing Realm Royale on Project Crown
**Current focus:** Phase 2 - Wine and Game Management

## Current Position

Phase: 2 of 2 (Wine and Game Management)
Plan: 3 of 3 in current phase (COMPLETE)
Status: Phase Complete
Last activity: 2026-02-22 - Completed quick task 9: restore DXVK override and Steam Deck controller env vars

Progress: [██████████] 100%

## Performance Metrics

**Velocity:**
- Total plans completed: 6
- Average duration: 7min
- Total execution time: 0.75 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01-foundation-and-core-launch | 3 | 30min | 10min |
| 02-wine-and-game-management | 3 | 10min | 3min |

**Recent Trend:**
- Last 5 plans: 01-03 (25min), 02-02 (3min), 02-01 (4min), 02-03 (3min)
- Trend: stable

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Go over Python for single binary distribution (curl + chmod for gamers)
- NaCl secretbox for credential encryption (no D-Bus/keyring on Deck)
- Embed shm_launcher.exe via go:embed (users don't need mingw)
- Single ~/.cluckers/ directory (simpler than XDG split)
- Go 1.25 module minimum (matches local Go 1.26 toolchain)
- Viper BindPFlag in init(), read values in RunE to avoid flag binding race
- UserError pattern: message + detail + suggestion for layered error display
- CLUCKERS_HOME env var overrides ~/.cluckers base directory
- NaCl secretbox with scrypt key derivation from machine-id (no D-Bus/keyring dependency)
- Missing credentials file returns nil,nil (first-time user path, not an error)
- Missing bootstrap returns nil,nil (game can launch without it)
- Base64 padding fix for bootstrap data (servers may omit trailing =)
- FlexBool type for API responses returning 1/0 instead of true/false
- Removed PORTAL_INFO_1 from LoginResponse (cosmetics array, not bootstrap)
- wine_prefix config option instead of hardcoded path
- Force-exit goroutine for Ctrl+C during blocking stdin reads
- Embedded shm_launcher.exe via assets/embed.go top-level package
- stdlib net/http for updater API (no auth, unlike gateway retryablehttp)
- Disk space pre-check at 2x zip size for download + extraction headroom
- Non-fatal zip cleanup after extraction (warn, don't fail)
- Streaming BLAKE3 via io.Copy for large file verification
- IsProtonGE matches both proton-ge-custom and GE-Proton* paths
- Symlink fixup uses os.Lstat (not DirEntry.Info) for correct symlink detection
- Prefix path always pre-resolved in pipeline, stored in LaunchState.PrefixPath
- WINEPREFIX set unconditionally when prefix resolved; WINEFSYNC only for Proton-GE
- Download step stops spinner before progress bar to avoid visual conflict
- Status checks use 5-second timeouts to avoid hanging on network issues
- Status command is online by default (checks server version and gateway health)
- Game exe path resolved via game.GameExePath for consistency across pipeline and process
- [Phase quick]: CLAUDE.md and README.md created as project documentation foundation
- [Phase quick]: Fixed .gitignore from 'cluckers' to '/cluckers' to avoid matching cmd/cluckers/ directory
- [Phase quick]: CGO_ENABLED=0 enforced at job level in CI workflow
- [Phase quick]: Goreleaser changelog with 5 commit-type groups, v0.1.0 released on GitHub
- [Phase quick]: SECURITY.md with vulnerability reporting via GitHub private reporting, NaCl secretbox security model
- [Phase quick]: POSIX sh curl-pipe installer with GitHub Releases download, SHA-256 checksum, Steam Deck detection
- [Phase quick]: 55-minute OIDC TTL (under 1 hour) to avoid edge-case expiry mid-launch
- [Phase quick]: JSON file cache (no encryption) for session tokens since they are short-lived
- [Phase quick]: .desktop file approach for Steam integration (reliable, no binary VDF manipulation)
- [Phase quick]: Symlink resolution via EvalSymlinks for ~/.steam/root and ~/.steam/steam
- [Phase quick]: STEAM_INPUT_DISABLE=1 to bypass Steam Input virtual gamepad on Steam Deck
- [Phase quick]: WINEDLLOVERRIDES=dxgi=n set unconditionally (was in POC but missing from Go launcher)
- [Phase quick]: COM vtable patching for dinput8 proxy DLL diagnostics (not full interface wrapping)
- [Phase quick]: Makefile for mingw-w64 cross-compiled Windows tools
- [Phase quick-9]: WINEDLLOVERRIDES=dxgi=n restored unconditionally, isSteamDeck() controller env vars added

### Pending Todos

None yet.

### Roadmap Evolution

- Phase 4 added: automatic downloads

### Blockers/Concerns

- Game file download URL source not documented in POC -- must clarify with Project Crown team before Phase 2
- winetricks reliability on SteamOS needs real-world testing in Phase 2

### Quick Tasks Completed

| # | Description | Date | Commit | Status | Directory |
|---|-------------|------|--------|--------|-----------|
| 1 | create CLAUDE.md and README.md | 2026-02-22 | f775088 | | [1-create-claude-md-and-readme-md](./quick/1-create-claude-md-and-readme-md/) |
| 2 | create GitHub repo and update module path | 2026-02-22 | 88356b6 | | [2-create-a-git-repo-on-github-and-update-p](./quick/2-create-a-git-repo-on-github-and-update-p/) |
| 3 | implement build workflow | 2026-02-22 | 6b53305 | | [3-implement-build-workflow-preferably-in-g](./quick/3-implement-build-workflow-preferably-in-g/) |
| 4 | implement full GitHub release pipeline | 2026-02-22 | 76846f5 | Verified | [4-implement-full-github-release-pipeline-c](./quick/4-implement-full-github-release-pipeline-c/) |
| 5 | create SECURITY.md | 2026-02-22 | 043db6f | Complete | [5-create-security-md](./quick/5-create-security-md/) |
| 6 | create install.sh curl-pipe installer | 2026-02-22 | 7358bcb | Complete | [6-create-sh-script-for-install-with-curl-p](./quick/6-create-sh-script-for-install-with-curl-p/) |
| 7 | fix Steam Deck Proton-GE detection, add Steam integration, token caching | 2026-02-22 | fda65a0 | Complete | [7-fix-steam-deck-proton-ge-detection-add-s](./quick/7-fix-steam-deck-proton-ge-detection-add-s/) |
| 8 | resolve issues with inputs on Steam Deck | 2026-02-22 | a0754b3 | Complete | [8-resolve-issues-with-inputs-on-steam-deck](./quick/8-resolve-issues-with-inputs-on-steam-deck/) |
| 9 | restore DXVK override and Steam Deck controller env vars | 2026-02-22 | 30c217c | Complete | [9-resolve-issues-with-steamdeck-controller](./quick/9-resolve-issues-with-steamdeck-controller/) |

## Session Continuity

Last session: 2026-02-22
Stopped at: Completed quick task 9: restore DXVK override and Steam Deck controller env vars
Resume file: None
