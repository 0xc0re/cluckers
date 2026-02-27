# Requirements: Project Crown Launcher (Cluckers)

**Defined:** 2026-02-24
**Core Value:** A user can download one file and go from zero to playing Realm Royale on Project Crown

## v1.1 Requirements

Requirements for Proton launch pipeline migration. Each maps to roadmap phases.

### Proton Launch Pipeline

- [x] **PROTON-01**: Launcher detects Proton-GE installation (bundled via CLUCKERS_BUNDLED_PROTON, config override, or system Proton-GE scan paths)
- [x] **PROTON-02**: Launcher invokes game via `proton run` with correct environment (STEAM_COMPAT_DATA_PATH, PROTON_WINEDLLOVERRIDES=dxgi=n, UMU_ID=0, SteamGameId, SteamAppId=0)
- [x] **PROTON-03**: Proton automatically creates and manages Wine prefix at ~/.cluckers/compatdata/pfx/ on first launch
- [x] **PROTON-04**: shm_launcher.exe shared memory bridge works correctly under Proton (CreateFileMappingW/OpenFileMapping)
- [x] **PROTON-05**: All game arguments pass through correctly via proton run (-user, -token, -eac_oidc_token_file, -hostx)

### Controller / Gamescope

- [x] **CTRL-01**: Launcher injects SteamGameId env var for Gamescope window tracking across UE3 ServerTravel
- [x] **CTRL-02**: Launcher auto-detects Steam installation path for STEAM_COMPAT_CLIENT_INSTALL_PATH (native, Flatpak, Snap)
- [x] **CTRL-03**: Controller buttons persist through lobby-to-match transition on Steam Deck (validated on hardware)

### Cleanup / Polish

- [x] **POLISH-01**: Manual prefix management code removed for Proton path (createFromProtonTemplate, DLL verification, winetricks dependency)
- [x] **POLISH-02**: `cluckers status` shows Proton version and compatdata path instead of Wine prefix DLL status
- [x] **POLISH-03**: GUI launch progress displays Proton-specific step names ("Detecting Proton", "Preparing compatibility data")

## v1.2+ Requirements

Deferred to future release. Tracked but not in current roadmap.

### Infrastructure

- **INFRA-01**: Dynamic server IP from API rather than build-time ldflags
- **INFRA-02**: PIN flow for linked accounts

### Proton Enhancements

- **PROTON-06**: umu-launcher integration as alternative to direct proton run
- **PROTON-07**: Gamescope wrapper for desktop Linux users
- **PROTON-08**: `cluckers cleanup` command for removing old Wine prefix

## Out of Scope

| Feature | Reason |
|---------|--------|
| System Wine fallback | Proton-GE required for all Linux -- simplifies codebase, eliminates two parallel code paths |
| Old prefix migration | Proton prefixes are structurally different; no data migration possible. Old prefix silently ignored. |
| umu-launcher | Direct `proton run` is simpler and works inside AppImage; umu adds Python dependency and downloads Steam Runtime |
| Proton log rotation | Low value; defer to future if users need debug logging management |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| PROTON-01 | Phase 6 | Complete |
| PROTON-02 | Phase 6 | Complete |
| PROTON-03 | Phase 6 | Complete |
| PROTON-04 | Phase 6 | Complete |
| PROTON-05 | Phase 6 | Complete |
| CTRL-01 | Phase 7 | Complete |
| CTRL-02 | Phase 7 | Complete |
| CTRL-03 | Phase 7 | FAILED (deferred to v1.2+) |
| POLISH-01 | Phase 8 | Complete |
| POLISH-02 | Phase 8 | Complete |
| POLISH-03 | Phase 8 | Complete |

**Coverage:**
- v1.1 requirements: 11 total
- Mapped to phases: 11
- Unmapped: 0

---
*Requirements defined: 2026-02-24*
*Last updated: 2026-02-24 after roadmap creation*
