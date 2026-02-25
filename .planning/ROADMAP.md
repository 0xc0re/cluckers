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
- [ ] **Phase 7: Controller and Gamescope Integration** - Fix Steam Deck controller input loss through Gamescope session tracking
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
**Plans**: TBD

### Phase 8: Cleanup and Polish
**Goal**: The codebase reflects the Proton-only reality for Proton-GE users, with UI terminology and status output updated to match
**Depends on**: Phase 7
**Requirements**: POLISH-01, POLISH-02, POLISH-03
**Success Criteria** (what must be TRUE):
  1. Manual Proton prefix management code is removed (createFromProtonTemplate, DLL verification, winetricks dependency detection) -- no dead code paths for Proton-GE users
  2. `cluckers status` displays the detected Proton version and compatdata path instead of Wine prefix DLL status
  3. The GUI launch progress shows Proton-specific step names ("Detecting Proton", "Preparing compatibility data") instead of Wine terminology
**Plans**: TBD

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Foundation and Core Launch | v1.0 | 3/3 | Complete | 2026-02-22 |
| 2. Wine and Game Management | v1.0 | 3/3 | Complete | 2026-02-22 |
| 3. First-Run Setup and Steam Deck | v1.0 | 0/0 (absorbed) | Complete | 2026-02-24 |
| 4. Cross-Platform GUI | v1.0 | 5/5 | Complete | 2026-02-24 |
| 5. Containers / AppImage | v1.0 | 3/3 | Complete | 2026-02-25 |
| 6. Core Proton Launch Pipeline | v1.1 | 1/3 | In progress | - |
| 7. Controller and Gamescope Integration | v1.1 | 0/? | Not started | - |
| 8. Cleanup and Polish | v1.1 | 0/? | Not started | - |
