# Roadmap: Project Crown Launcher (Cluckers)

## Milestones

- ✅ **v1.0 MVP** -- Phases 1-5 (shipped 2026-02-25)
- **v1.1 Full Controller Functionality on Steam Deck** -- Phases 6-8 (in progress)

## Phases

<details>
<summary>v1.0 MVP (Phases 1-5) -- SHIPPED 2026-02-25</summary>

- [x] Phase 1: Foundation and Core Launch (3/3 plans) -- completed 2026-02-22
- [x] Phase 2: Wine and Game Management (3/3 plans) -- completed 2026-02-22
- [x] Phase 3: First-Run Setup and Steam Deck (absorbed into pipeline) -- completed 2026-02-24
- [x] Phase 4: Cross-Platform GUI (5/5 plans) -- completed 2026-02-24
- [x] Phase 5: Containers / AppImage (3/3 plans) -- completed 2026-02-25

See `.planning/milestones/v1.0-ROADMAP.md` for full details.

</details>

### v1.1 Full Controller Functionality on Steam Deck

- [ ] **Phase 6: Core Proton Launch Pipeline** - Replace direct Wine execution with `proton run` for all Linux launches
- [x] **Phase 7: Controller and Gamescope Integration** - Fix Steam Deck controller input loss through Gamescope session tracking (completed 2026-02-25)
- [ ] **Phase 8: Cleanup and Polish** - Remove dead Wine code paths and update UI for Proton terminology

## Phase Details

### Phase 6: Core Proton Launch Pipeline
**Goal**: Users can launch the game through Proton instead of direct Wine, with automatic prefix management and correct environment setup
**Depends on**: Phase 5 (v1.0 complete)
**Requirements**: PROTON-01, PROTON-02, PROTON-03, PROTON-04, PROTON-05
**Success Criteria** (what must be TRUE):
  1. Running `cluckers launch` on Linux invokes the game via `python3 proton run` instead of direct `wine64`
  2. Proton auto-creates a working Wine prefix at `~/.cluckers/compatdata/pfx/` on first launch without manual winetricks or DLL management
  3. The shared memory bridge (shm_launcher.exe) correctly passes content bootstrap data to the game under Proton
  4. All game arguments (-user, -token, -eac_oidc_token_file, -hostx) reach the game process through the Proton invocation chain
  5. The launcher detects Proton-GE from bundled path, config override, or system scan and reports the version found
**Plans**: 3 plans
Plans:
- [ ] 06-01-PLAN.md — Proton-GE detection with priority ordering and compatdata health check (TDD)
- [ ] 06-02-PLAN.md — Proton environment variable construction and command building (TDD)
- [ ] 06-03-PLAN.md — Pipeline integration: wire Proton detection and invocation into launch pipeline

### Phase 7: Controller and Gamescope Integration
**Goal**: Steam Deck controller buttons persist through the lobby-to-match transition (UE3 ServerTravel) without input loss
**Depends on**: Phase 6
**Requirements**: CTRL-01, CTRL-02, CTRL-03
**Success Criteria** (what must be TRUE):
  1. The launcher injects `SteamGameId` environment variable so Gamescope tracks the game window across D3D window recreation
  2. The launcher auto-detects the Steam installation path (native, Flatpak, or Snap) for `STEAM_COMPAT_CLIENT_INSTALL_PATH`
  3. On Steam Deck hardware, controller buttons (A/B/X/Y, bumpers, triggers) remain functional after transitioning from lobby to in-match gameplay
**Plans**: 3 plans
Plans:
- [x] 07-01-PLAN.md — Steam installation detection with TDD (FindSteamInstall for native, Flatpak, Snap)
- [x] 07-02-PLAN.md — Wire Steam detection and app ID into Proton env vars and launch pipeline
- [x] 07-03-PLAN.md — Hardware validation of controller persistence on Steam Deck (FAILED -- CTRL-03 unsatisfied, deferred to v1.2+)

### Phase 07.1: Steam Deck controller input proxy (INSERTED)

**Goal**: A pure Go uinput proxy creates a persistent virtual Xbox 360 gamepad that forwards Steam Input events and holds last-known button state during ServerTravel transitions, preventing controller disconnect on Steam Deck
**Depends on:** Phase 7
**Requirements**: CTRL-03
**Success Criteria** (what must be TRUE):
  1. The launcher creates a virtual Xbox 360 gamepad via `/dev/uinput` before launching the game, and the game receives input exclusively from this virtual device
  2. The proxy reads evdev events from Steam Input's virtual pad and forwards them to the virtual gamepad with sub-frame latency
  3. When Steam Input zeros button data during ServerTravel, the proxy holds the last known button state for a configurable timeout instead of forwarding zeros
  4. Wine prefix is configured with `DisableHidraw=1` and `Enable SDL=1` so the game only sees the proxy's virtual pad
**Plans:** 3/4 plans executed

Plans:
- [ ] 07.1-01-PLAN.md — Uinput virtual Xbox 360 gamepad creation and Steam Input device detection (TDD)
- [ ] 07.1-02-PLAN.md — Dead reckoning state machine and Wine registry patching (TDD)
- [ ] 07.1-03-PLAN.md — Proxy Run loop and launch pipeline integration
- [ ] 07.1-04-PLAN.md — Hardware validation of controller persistence on Steam Deck

### Phase 8: Cleanup and Polish
**Goal**: The codebase reflects the Proton-only reality for Proton-GE users, with UI terminology and status output updated to match
**Depends on**: Phase 7
**Requirements**: POLISH-01, POLISH-02, POLISH-03
**Success Criteria** (what must be TRUE):
  1. Manual Proton prefix management code is removed (createFromProtonTemplate, DLL verification, winetricks dependency detection) -- no dead code paths for Proton-GE users
  2. `cluckers status` displays the detected Proton version and compatdata path instead of Wine prefix DLL status
  3. The GUI launch progress shows Proton-specific step names ("Detecting Proton", "Preparing compatibility data") instead of Wine terminology
**Plans**: 2 plans
Plans:
- [ ] 08-01-PLAN.md — Remove dead Wine/proxy code: delete prefix.go, verify.go, inputproxy/, xinput tools, dead functions and struct fields
- [ ] 08-02-PLAN.md — Rewrite status command for Proton terminology and update CLAUDE.md

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Foundation and Core Launch | v1.0 | 3/3 | Complete | 2026-02-22 |
| 2. Wine and Game Management | v1.0 | 3/3 | Complete | 2026-02-22 |
| 3. First-Run Setup and Steam Deck | v1.0 | 0/0 (absorbed) | Complete | 2026-02-24 |
| 4. Cross-Platform GUI | v1.0 | 5/5 | Complete | 2026-02-24 |
| 5. Containers / AppImage | v1.0 | 3/3 | Complete | 2026-02-25 |
| 6. Core Proton Launch Pipeline | v1.1 | 3/3 | Complete | 2026-02-25 |
| 7. Controller and Gamescope Integration | v1.1 | Complete    | 2026-02-25 | 2026-02-25 |
| 7.1 Steam Deck Controller Input Proxy | v1.1 | 1/4 | In Progress | - |
| 8. Cleanup and Polish | v1.1 | 0/2 | Not started | - |
