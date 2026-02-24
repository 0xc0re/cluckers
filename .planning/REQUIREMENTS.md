# Requirements: Project Crown Linux Launcher

**Defined:** 2026-02-21
**Core Value:** A Linux user can run one command and go from zero to playing Realm Royale on Project Crown

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Authentication

- [x] **AUTH-01**: User can log in with username and password via gateway API
- [x] **AUTH-02**: User's credentials are encrypted and saved to ~/.cluckers/config after first login
- [x] **AUTH-03**: User is auto-logged-in on subsequent launches using saved credentials
- [x] **AUTH-04**: User sees clear error message on login failure (wrong password, server down, account issues)

### CLI Interface

- [x] **CLI-01**: User can run `cluckers launch` to authenticate and start the game
- [ ] **CLI-02**: User can run `cluckers setup` to perform guided first-run setup
- [x] **CLI-03**: User can run `cluckers update` to force re-download game files and refresh config
- [x] **CLI-04**: User can run `cluckers status` to see local version, server version, Wine info, and prefix health

### Wine Management

- [x] **WINE-01**: Launcher auto-detects Wine/Proton-GE across standard Linux and Steam Deck paths
- [x] **WINE-02**: Launcher auto-creates Wine prefix at ~/.cluckers/prefix/ on first run
- [x] **WINE-03**: Launcher installs required prefix dependencies (vcrun2022, d3dx11_43, DXVK) via winetricks
- [x] **WINE-04**: Launcher verifies required DLLs exist in prefix before launch, offers repair if missing

### Game Files

- [x] **GAME-01**: Launcher downloads game files from API-provided URL to ~/.cluckers/game/
- [x] **GAME-02**: Launcher tracks local game version and compares against API version
- [x] **GAME-03**: Launcher auto-checks version before launch and downloads updates if stale
- [x] **GAME-04**: User sees download progress bar with speed and ETA for game file downloads

### Launch Pipeline

- [x] **LAUN-01**: Launcher executes full auth → OIDC → bootstrap → SHM → Wine launch pipeline
- [x] **LAUN-02**: Launcher extracts embedded shm_launcher.exe and creates named shared memory for content bootstrap
- [x] **LAUN-03**: Launcher converts Linux paths to Wine Z: drive notation for game arguments
- [x] **LAUN-04**: Launcher cleans up temp files (OIDC token, bootstrap, extracted shm_launcher.exe) after game exits

### Steam Deck

- [ ] **DECK-01**: Launcher works in non-interactive mode using saved credentials (no prompts during launch)
- [ ] **DECK-02**: Launcher discovers Proton-GE in SteamOS-specific paths (Flatpak Steam, native Steam)
- [ ] **DECK-03**: Launcher provides helper to generate .desktop shortcut for adding as non-Steam game

### First-Run Setup

- [ ] **SETUP-01**: `cluckers setup` detects missing prerequisites and guides user through setup
- [ ] **SETUP-02**: Setup orchestrates Wine prefix creation, dependency install, and game file download
- [ ] **SETUP-03**: Setup prompts for credentials and saves them for future launches
- [ ] **SETUP-04**: Setup is idempotent — safe to re-run to fix incomplete setups

### Configuration

- [x] **CONF-01**: Launcher stores all config in TOML format at ~/.cluckers/config/settings.toml
- [x] **CONF-02**: Launcher stores all data (game, prefix, config, logs) under ~/.cluckers/
- [x] **CONF-03**: User can override Wine path, game directory, and gateway URL via config file or CLI flags

### Distribution

- [x] **DIST-01**: Launcher is a single static Go binary with zero runtime dependencies (except Wine and winetricks)
- [x] **DIST-02**: User can install via `curl` one-liner downloading from GitHub releases
- [x] **DIST-03**: shm_launcher.exe is pre-compiled and embedded in the Go binary via `//go:embed`
- [x] **DIST-04**: CI pipeline (GitHub Actions + goreleaser) produces release binaries on tag

### Error Handling

- [x] **ERR-01**: User sees human-readable error messages with actionable suggestions at every failure point
- [x] **ERR-02**: Launcher detects gateway down and tells user immediately (not hang or crash)
- [x] **ERR-03**: Launcher detects missing Wine and prints per-distro install instructions

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Authentication

- **AUTH-05**: User can complete PIN flow for linked accounts
- **AUTH-06**: Dynamic game server IP (hostx) retrieved from API instead of hardcoded

### Maintenance

- **MAINT-01**: User can run `cluckers repair` to rebuild Wine prefix without losing game files or credentials
- **MAINT-02**: Structured logging with verbosity levels (-v, -vv, --log-file)
- **MAINT-03**: Launcher prints "new version available" when a newer launcher release exists

### Downloads

- **DL-01**: Game downloads can resume after interruption (Range header support)
- **DL-02**: Game downloads are hash-verified for integrity

### GUI

- [x] **GUI-01**: Running `cluckers` with no subcommand opens a GUI window when a display is available
- [x] **GUI-02**: Login-first screen with username/password fields
- [x] **GUI-03**: Main view with launch button, Discord link, donate/support, verify/repair, bot name
- [x] **GUI-04**: Step-by-step launch progress with checkmarks
- [x] **GUI-05**: Fullscreen on Steam Deck, windowed on desktop
- [x] **GUI-06**: Headless environments automatically fall back to CLI mode
- [x] **GUI-07**: CLI subcommands continue working unchanged
- [x] **GUI-08**: Launcher closes when game launches
- [ ] **GUI-09**: Cross-platform build (Linux + Windows amd64)
- [x] **GUI-10**: Settings accessible from GUI

## Out of Scope

| Feature | Reason |
|---------|--------|
| ~~GUI / graphical interface~~ | ~~CLI is the product~~ -- **Now in scope (Phase 4)** |
| Bundled Wine/Proton | Distro-specific, legally complex, enormous (~500MB+). User manages their own. |
| Custom skin/cosmetic management | Windows launcher handles this. Cosmetics are server-side. |
| Multi-game support | This is a Realm Royale launcher. YAGNI. |
| Windows support | Windows launcher already exists. This is Linux-only. |
| Auto-install Wine/Proton-GE | Distro-specific package management. Recommend ProtonUp-Qt instead. |
| Launcher self-update | Premature for v1. Distribute via GitHub releases. |
| EAC bypass/modification | Anti-cheat disabled server-side. Not our concern. |
| XDG directory splitting | Single ~/.cluckers/ is simpler for users to find/backup/delete. |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| AUTH-01 | Phase 1 | Complete |
| AUTH-02 | Phase 1 | Complete |
| AUTH-03 | Phase 1 | Complete |
| AUTH-04 | Phase 1 | Complete |
| CLI-01 | Phase 1 | Complete |
| CLI-02 | Phase 3 | Pending |
| CLI-03 | Phase 2 | Complete |
| CLI-04 | Phase 2 | Complete |
| WINE-01 | Phase 2 | Complete |
| WINE-02 | Phase 2 | Complete |
| WINE-03 | Phase 2 | Complete |
| WINE-04 | Phase 2 | Complete |
| GAME-01 | Phase 2 | Complete |
| GAME-02 | Phase 2 | Complete |
| GAME-03 | Phase 2 | Complete |
| GAME-04 | Phase 2 | Complete |
| LAUN-01 | Phase 1 | Complete |
| LAUN-02 | Phase 1 | Complete |
| LAUN-03 | Phase 1 | Complete |
| LAUN-04 | Phase 1 | Complete |
| DECK-01 | Phase 3 | Pending |
| DECK-02 | Phase 3 | Pending |
| DECK-03 | Phase 3 | Pending |
| SETUP-01 | Phase 3 | Pending |
| SETUP-02 | Phase 3 | Pending |
| SETUP-03 | Phase 3 | Pending |
| SETUP-04 | Phase 3 | Pending |
| CONF-01 | Phase 1 | Complete |
| CONF-02 | Phase 1 | Complete |
| CONF-03 | Phase 1 | Complete |
| DIST-01 | Phase 1 | Complete |
| DIST-02 | Phase 1 | Complete |
| DIST-03 | Phase 1 | Complete |
| DIST-04 | Phase 1 | Complete |
| ERR-01 | Phase 1 | Complete |
| ERR-02 | Phase 1 | Complete |
| ERR-03 | Phase 1 | Complete |
| GUI-01 | Phase 4 | Complete |
| GUI-02 | Phase 4 | Complete |
| GUI-03 | Phase 4 | Complete |
| GUI-04 | Phase 4 | Complete |
| GUI-05 | Phase 4 | Complete |
| GUI-06 | Phase 4 | Complete |
| GUI-07 | Phase 4 | Complete |
| GUI-08 | Phase 4 | Complete |
| GUI-09 | Phase 4 | Pending |
| GUI-10 | Phase 4 | Complete |

**Coverage:**
- v1 requirements: 37 total
- GUI requirements: 10 total (2 complete, 8 pending)
- Mapped to phases: 47
- Unmapped: 0

---
*Requirements defined: 2026-02-21*
*Last updated: 2026-02-24 -- Added GUI requirements (Phase 4), marked GUI-01 and GUI-06 complete*
