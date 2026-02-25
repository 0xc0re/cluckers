# Roadmap: Project Crown Linux Launcher

## Overview

Take a working Python POC and build a production Go launcher that takes Linux users from zero to playing Realm Royale on Project Crown with a single command. Three phases: first get the core auth-to-launch pipeline working as a distributable Go binary, then add Wine prefix and game file management so the launcher is self-sufficient, then wire up the zero-to-playing guided setup with Steam Deck support.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Foundation and Core Launch** - Go scaffold, CLI, auth, launch pipeline, config, distribution, error handling (completed 2004-02-22)
- [x] **Phase 2: Wine and Game Management** - Wine detection/prefix automation, game download/versioning, update and status commands (completed 2026-02-22)
- [ ] **Phase 3: First-Run Setup and Steam Deck** - Guided setup wizard, prerequisite detection, Steam Deck Gaming Mode compatibility

## Phase Details

### Phase 1: Foundation and Core Launch
**Goal**: User can authenticate and launch Realm Royale from a single static Go binary on a machine where Wine and game files are already present
**Depends on**: Nothing (first phase)
**Requirements**: AUTH-01, AUTH-02, AUTH-03, AUTH-04, CLI-01, LAUN-01, LAUN-02, LAUN-03, LAUN-04, CONF-01, CONF-02, CONF-03, DIST-01, DIST-02, DIST-03, DIST-04, ERR-01, ERR-02, ERR-03
**Success Criteria** (what must be TRUE):
  1. User can run `cluckers launch`, enter credentials, and the game starts under Wine with correct arguments (auth, OIDC, bootstrap, SHM all working)
  2. User's credentials are saved encrypted after first login and auto-used on subsequent launches without re-prompting
  3. User sees clear, actionable error messages when login fails, gateway is down, or Wine is not found
  4. A single static binary can be downloaded from GitHub releases via curl one-liner and run with zero Go/Python dependencies
  5. Temp files (OIDC token, bootstrap data, extracted shm_launcher.exe) are cleaned up after the game exits
**Plans**: 3 plans

Plans:
- [x] 01-01-PLAN.md -- Go scaffold, CLI skeleton, gateway client, config system, UI helpers
- [x] 01-02-PLAN.md -- NaCl encryption, credential persistence, gateway auth (login, OIDC, bootstrap)
- [x] 01-03-PLAN.md -- Launch pipeline, Wine detection, SHM extraction, process management, distribution config

### Phase 2: Wine and Game Management
**Goal**: Launcher manages its own Wine prefix and game files -- detects Wine, creates prefix with dependencies, downloads/updates game binaries, and reports status
**Depends on**: Phase 1
**Requirements**: WINE-01, WINE-02, WINE-03, WINE-04, GAME-01, GAME-02, GAME-03, GAME-04, CLI-03, CLI-04
**Success Criteria** (what must be TRUE):
  1. Launcher auto-detects Wine/Proton-GE across standard Linux paths without user configuration
  2. Launcher creates Wine prefix at ~/.cluckers/prefix/ and installs vcrun2022, d3dx11_43, DXVK via winetricks, verifying DLL presence after install
  3. User can run `cluckers update` to download game files with a progress bar showing speed and ETA
  4. Launcher auto-checks game version before launch and downloads updates when the local version is stale
  5. User can run `cluckers status` to see local game version, server version, Wine path, and prefix health
**Plans**: 3 plans

Plans:
- [ ] 02-01-PLAN.md -- Wine/Proton-GE detection, two-tier prefix creation, DLL verification
- [ ] 02-02-PLAN.md -- Game version checking, resumable download with progress, BLAKE3 verification, zip extraction
- [ ] 02-03-PLAN.md -- Pipeline integration (auto-download on launch), update command, status command

### Phase 3: First-Run Setup and Steam Deck
**Goal**: A new user can go from zero to playing with `cluckers setup`, and the launcher works in Steam Deck Gaming Mode as a non-Steam game
**Depends on**: Phase 2
**Requirements**: SETUP-01, SETUP-02, SETUP-03, SETUP-04, DECK-01, DECK-02, DECK-03, CLI-02
**Success Criteria** (what must be TRUE):
  1. User can run `cluckers setup` which detects missing prerequisites, guides through credential entry, creates Wine prefix, installs dependencies, and downloads game files in one flow
  2. Running `cluckers setup` a second time skips already-completed steps and fixes incomplete ones (idempotent)
  3. Launcher works on Steam Deck in Gaming Mode as a non-Steam game using saved credentials with zero interactive prompts during launch
  4. Launcher discovers Proton-GE in SteamOS-specific paths (Flatpak Steam, native Steam)
  5. User can generate a .desktop shortcut for adding the launcher as a non-Steam game
**Plans**: TBD

Plans:
- [ ] 03-01: TBD
- [ ] 03-02: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation and Core Launch | 3/3 | Complete    | 2004-02-22 |
| 2. Wine and Game Management | 0/3 | Complete    | 2026-02-22 |
| 3. First-Run Setup and Steam Deck | 0/2 | Not started | - |
| 4. Cross-Platform GUI | 5/5 | Complete    | 2026-02-24 |
| 5. Containers / AppImage | 0/3 | Complete    | 2026-02-25 |

### Phase 4: Cross-Platform GUI

**Goal:** Cross-platform GUI launcher for Cluckers that works on Linux, Windows, and Steam Deck -- GUI opens by default, all CLI subcommands preserved, login-first flow, step-by-step launch progress, fullscreen on Steam Deck
**Depends on:** Phase 3
**Requirements:** GUI-01, GUI-02, GUI-03, GUI-04, GUI-05, GUI-06, GUI-07, GUI-08, GUI-09, GUI-10
**Success Criteria** (what must be TRUE):
  1. Running `cluckers` with no subcommand opens a GUI window with login screen or main view
  2. GUI has full feature parity with CLI: login, launch, update, status, settings, logout
  3. Launch pipeline shows step-by-step progress with checkmarks in the GUI
  4. Launcher closes when the game launches
  5. Headless environments automatically fall back to CLI mode
  6. Steam Deck runs fullscreen, desktop runs windowed
  7. CI/CD produces both GUI and CLI-only binary variants
**Plans:** 5/5 plans complete

Plans:
- [x] 04-01-PLAN.md -- Fyne foundation: dependency, build tags, headless detection, theme, GUI skeleton
- [x] 04-02-PLAN.md -- Login screen, ProgressReporter pipeline refactor
- [x] 04-03-PLAN.md -- Main view, launch progress with step checkmarks, GUIReporter
- [x] 04-04-PLAN.md -- Settings screen, bot name, Steam Deck fullscreen, CLI verification
- [x] 04-05-PLAN.md -- CI/CD updates: goreleaser dual-build, workflow changes, human verification

### Phase 5: Containers / AppImage

**Goal:** Package Cluckers as a self-contained AppImage that bundles Proton-GE and all dependencies, so Linux users can go from download to playing with zero system-level dependency management
**Depends on:** Phase 4
**Requirements:** APIMG-01, APIMG-02, APIMG-03, APIMG-04, APIMG-05, APIMG-06, APIMG-07, APIMG-08
**Success Criteria** (what must be TRUE):
  1. User can download a single AppImage, chmod +x, and run it with zero Wine/Proton/library management
  2. AppImage bundles Proton-GE (GE-Proton10-32) and all Fyne/GL shared libraries
  3. Self-update correctly downloads new AppImage (not tar.gz) when running in AppImage mode
  4. GitHub release includes AppImage + zsync file alongside existing tar.gz and zip archives
  5. Install script defaults to AppImage download for new Linux installations
  6. Wine prefix stays external at ~/.cluckers/prefix/ (not inside AppImage)
**Plans:** 3/3 plans complete

Plans:
- [ ] 05-01-PLAN.md -- AppImage deploy assets, detection helpers, bundled Proton-GE detection, LD_LIBRARY_PATH isolation
- [ ] 05-02-PLAN.md -- Build script (scripts/build-appimage.sh), Proton-GE bundling, license compliance
- [ ] 05-03-PLAN.md -- CI/CD integration, self-update for AppImage mode, goreleaser extra_files, install script update
