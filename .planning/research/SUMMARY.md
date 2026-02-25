# Project Research Summary

**Project:** Cluckers v1.1 -- Wine to Proton Launch Pipeline Migration
**Domain:** Native game launcher -- Linux Wine/Proton execution pipeline
**Researched:** 2026-02-24
**Confidence:** HIGH

## Executive Summary

Cluckers v1.1 is a targeted engineering effort to replace direct Wine binary execution (`wine64 shm_launcher.exe ...`) with Proton's managed launch pipeline (`proton run shm_launcher.exe ...`). The primary motivation is the Steam Deck controller input failure during UE3 ServerTravel: Gamescope loses track of the game window when the D3D window is destroyed and recreated during the lobby-to-match transition, causing Steam Input to revert the controller firmware to desktop mode. Proton's launch pipeline registers the game session with Gamescope via the `STEAM_GAME` X11 property, which is set internally by Proton from the `SteamGameId` environment variable. This is the only known fix path that does not require launching through Steam itself or using an external USB controller.

The recommended approach is direct Proton invocation -- calling the `proton` Python script that ships with every Proton-GE installation with the verb `waitforexitandrun`. No new Go dependencies are needed. The changes are surgical: `process_linux.go` replaces `exec.Command(wine64, ...)` with `exec.Command("python3", protonScript, "run", ...)`, `detect.go` is extended to find the `proton` script alongside `wine64`, and `prefix.go` is simplified because Proton's `setup_prefix()` replaces all manual prefix management (150+ lines of code removed). The `shm_launcher.exe` binary requires zero changes -- it is a standard Win32 executable that runs identically under Proton's embedded Wine. The AppImage already bundles Proton-GE including the `proton` script; it just was not invoked until now.

The most important risk is that `proton run` invoked outside Steam may not provide Gamescope integration equivalent to launching through Steam. This is the critical assumption underlying the entire v1.1 milestone -- if Proton's `STEAM_GAME` X11 property is not set correctly when running standalone, the controller fix will not work and the strategy must pivot to requiring launch through Steam as a non-Steam shortcut. Secondary risks include the `STEAM_COMPAT_CLIENT_INSTALL_PATH` requirement (must be set or Proton's Python script crashes with `KeyError`), the `WINEDLLOVERRIDES` override mechanism changing from direct env var to `PROTON_WINEDLLOVERRIDES`, and the prefix location migrating from `~/.cluckers/prefix/` to `~/.cluckers/compatdata/pfx/`.

## Key Findings

### Recommended Stack

The Proton migration requires zero new Go dependencies. The launch mechanism shifts from `exec.Command(wine64Path, args...)` to `exec.Command("python3", protonScript, "run", args...)` using only stdlib packages already in use (`os/exec`, `os`, `path/filepath`, `strings`). The `proton` Python script is already bundled in the AppImage at `$CLUCKERS_BUNDLED_PROTON/proton` -- it was simply not invoked before. Python 3 is required by the `proton` script and is pre-installed on all target platforms (SteamOS, Arch, Ubuntu, Fedora). For system dependencies, winetricks is eliminated for Proton-GE users because Proton bundles DXVK, vcruntime140, msvcp140, and d3dx11 in its `default_pfx` template.

**Core technologies (changes only):**
- `python3 proton waitforexitandrun` -- replaces direct `wine64` invocation; `waitforexitandrun` verb blocks until game exits, matching current `cmd.Run()` behavior
- `STEAM_COMPAT_DATA_PATH=~/.cluckers/compatdata` -- replaces manual `WINEPREFIX`; Proton creates `pfx/` subdirectory automatically on first run
- `PROTON_WINEDLLOVERRIDES=dxgi=n` -- replaces direct `WINEDLLOVERRIDES`; Proton merges this with its own DXVK overrides rather than clobbering the variable
- `UMU_ID=0` + `SteamGameId=0` -- signals non-Steam game to Proton; enables Unix-path handling via `start.exe /unix` and populates `STEAM_GAME` X11 property
- `STEAM_COMPAT_CLIENT_INSTALL_PATH` -- required by Proton Python script; auto-detect from common Steam paths, point to Proton dir as fallback (non-Steam-DLL features fail silently)

**Explicitly rejected alternatives:**
- `umu-launcher` -- adds Python/system dependency, downloads Steam Runtime container; unnecessary since AppImage already bundles Proton-GE
- `pressure-vessel` containerization -- adds complexity without benefit; AppImage provides its own library isolation
- Proton-managed version download -- ProtonUp-Qt and the AppImage already handle this

### Expected Features

**Must have (table stakes -- regression from v1.0 if missing):**
- Proton runtime detection: find `proton` script alongside `wine64` in `ProtonGEInstall` struct
- `proton run` invocation with correct environment: `STEAM_COMPAT_DATA_PATH`, `STEAM_COMPAT_CLIENT_INSTALL_PATH`, `SteamGameId`, `UMU_ID`
- Proton-managed prefix creation: remove manual `stepEnsurePrefix` + `stepVerifyPrefix`; Proton's `setup_prefix()` handles everything on first run
- System Wine fallback: users without Proton-GE must still launch via existing direct-Wine code path
- All game arguments preserved: `-user`, `-token`, `-eac_oidc_token_file`, `-hostx`, etc. pass through unchanged
- Temp file cleanup: defer-based cleanup works unchanged since `waitforexitandrun` blocks until game exits
- Old prefix migration warning: detect `~/.cluckers/prefix/`, warn user it is no longer used

**Should have (the actual value of v1.1):**
- Steam Deck controller persistence through ServerTravel: `SteamGameId` injection + Gamescope `STEAM_GAME` property
- Steam install detection: auto-find `STEAM_COMPAT_CLIENT_INSTALL_PATH` from `~/.local/share/Steam`, Flatpak, Snap paths
- Non-Steam shortcut app ID detection: read from `shortcuts.vdf` via existing `findCluckersAppID()` code
- Elimination of DLL verification step for Proton path (`verify.go` DLL checks are dead code under Proton)
- `cluckers status` updated to show Proton version instead of Wine prefix DLL status
- Clear first-launch messaging: "Setting up Proton (one-time, 30-60 seconds)"

**Defer to v1.2+:**
- umu-launcher integration (not needed; direct `proton run` is simpler)
- Gamescope wrapper for desktop Linux users
- `cluckers cleanup` command for removing old prefix
- Proton log management / `PROTON_LOG` rotation

### Architecture Approach

The architecture change is shallow in code surface area but requires careful sequencing. Proton takes over responsibilities that Cluckers currently owns: prefix creation, DLL installation, fsync/esync configuration, and `WINEPREFIX` management. The Go launcher's role narrows to: find the Proton root, construct the required environment variables, ensure `STEAM_COMPAT_DATA_PATH` directory exists (empty, just `mkdir`), and invoke `python3 proton run`. The `shm_launcher.exe` integration is unchanged -- Proton's `start.exe /unix` translates the Unix temp path into the Wine address space identically to how direct Wine handles it.

**Major components and their changes:**

1. `internal/wine/detect.go` -- Add `FindProton()` returning Proton root dir; add `ProtonScript string` field to `ProtonGEInstall`; keep `FindWine()` for system Wine fallback
2. `internal/launch/pipeline_linux.go` -- Replace 3-step Wine pipeline (detect, ensure prefix, verify prefix) with 2-step Proton pipeline (detect Proton, prepare compat data dir via `mkdir`)
3. `internal/launch/process_linux.go` -- Replace `exec.Command(wine64, args)` with `exec.Command("python3", protonScript, "run", args)`; centralize env construction in `buildProtonEnv()`
4. `internal/launch/process.go` + `pipeline.go` -- Add `ProtonDir`, `CompatDataPath` to `LaunchConfig` and `LaunchState`; keep `WinePath`/`WinePrefix` for system Wine fallback
5. `internal/wine/prefix.go` -- Remove Proton-specific manual prefix code (`createFromProtonTemplate`, `ensureDosdevices`, etc.); keep `createWithWinetricks` for system Wine path
6. `internal/wine/verify.go` -- Remove or gate behind system Wine fallback; Proton manages DLLs automatically
7. `internal/config/config.go` -- Add `ProtonDir string` field; deprecate (but keep) `WinePrefix`

**Key architectural invariants:**
- Path handling: `wine.LinuxToWinePath()` still required for arguments TO `shm_launcher.exe` (it uses `CreateFileW`/`CreateProcessW` which need Wine paths); the executable path passed to `proton run` is a Unix path (Proton's `start.exe /unix` translates it)
- Environment: NEVER set `WINEPREFIX`, `WINEFSYNC`, `WINEESYNC` when using Proton -- Proton derives/sets these internally; setting them conflicts with Proton's internal state tracking
- Process waiting: Go launcher waits on the `python3 proton` process; this process internally waits on Wine, which waits on `shm_launcher.exe`, which waits on the game; entire chain terminates on game exit

### Critical Pitfalls

1. **WINEDLLOVERRIDES clobbered by Proton** -- Proton replaces `WINEDLLOVERRIDES` entirely with its own DXVK chain; the existing `dxgi=n` override silently vanishes. Fix: use `PROTON_WINEDLLOVERRIDES=dxgi=n` instead -- Proton prepends this to its own overrides. Must be the first env var change made.

2. **STEAM_COMPAT_DATA_PATH structure mismatch** -- Proton creates the Wine prefix at `$STEAM_COMPAT_DATA_PATH/pfx/`, not at `$STEAM_COMPAT_DATA_PATH` itself. If you point `STEAM_COMPAT_DATA_PATH` at the old `~/.cluckers/prefix/`, Proton creates `~/.cluckers/prefix/pfx/` as a nested second prefix. Use a new path `~/.cluckers/compatdata/`. Never set `WINEPREFIX` alongside `STEAM_COMPAT_DATA_PATH`.

3. **Missing env vars crash the Python script** -- The `proton` script raises `KeyError` (no graceful handling) if `STEAM_COMPAT_DATA_PATH` or `STEAM_COMPAT_CLIENT_INSTALL_PATH` are missing. For `STEAM_COMPAT_CLIENT_INSTALL_PATH`, auto-detect Steam install locations; if not found, set to the Proton dir as a dummy (Steam DLL copies fail silently, which is fine for a non-Steam game).

4. **Gamescope controller fix not guaranteed with standalone `proton run`** -- The entire v1.1 premise assumes that invoking `proton run` outside Steam is sufficient to register the game session with Gamescope. Gamescope tracks game processes via Steam's process tree management. If standalone `proton run` does not set the `STEAM_GAME` X11 property on game windows, the controller fix will not work and the milestone strategy must pivot to requiring launch through Steam as a non-Steam shortcut (which `cluckers steam add` already creates).

5. **LD_LIBRARY_PATH triple collision (AppImage context)** -- AppImage sets `LD_LIBRARY_PATH` for the Go binary. Current code strips it before Wine. With Proton, the Python script sets its own `LD_LIBRARY_PATH` from Proton's lib dirs. If AppImage paths leak into Python's environment, Proton loads wrong libraries. Continue stripping `LD_LIBRARY_PATH` (and `ORIG_LD_LIBRARY_PATH`) before invoking `proton run` -- Proton rebuilds it from scratch internally.

## Implications for Roadmap

Based on research, a 3-phase structure is recommended with a validation gate at the end of Phase 1.

### Phase 1: Core Proton Launch Pipeline

**Rationale:** All other work depends on the basic `proton run` invocation working. The three critical pitfalls (WINEDLLOVERRIDES, prefix path mismatch, missing env vars) and the highest-risk integration point (shm_launcher under Proton) must be resolved before any polish work begins. Phase 1 ends with a binary validation: does the controller fix work on Steam Deck? If not, the strategy pivots.

**Delivers:** Working Proton-based launch pipeline on development builds (non-AppImage). System Wine fallback preserved. Old prefix migration warning in place.

**Addresses:** All table stakes features -- Proton detection, `proton run` invocation, env var setup, Proton-managed prefix, system Wine fallback, game arg preservation, temp file cleanup.

**Build order (sequential, each step testable):**
1. Add `FindProton()` to `internal/wine/detect.go` (unit testable with mock dirs)
2. Add `ProtonDir`, `CompatDataPath` to `LaunchConfig`, `LaunchState`, `Config` structs
3. Create `buildProtonEnv()` in `process_linux.go` (unit testable)
4. Rewrite `LaunchGame()` in `process_linux.go` to invoke `python3 proton run`
5. Replace 3-step platform pipeline with 2-step in `pipeline_linux.go`
6. Add old prefix migration warning to launch flow
7. **VALIDATE:** Test on Steam Deck -- controller through ServerTravel; gate Phase 2 on this result

**Avoids:** WINEDLLOVERRIDES clobber (use `PROTON_WINEDLLOVERRIDES`), prefix structure mismatch (new path `~/.cluckers/compatdata/`), env var `KeyError` crash, WINEPREFIX/WINEFSYNC conflicts, mixing old and new prefixes.

### Phase 2: AppImage Integration and Gamescope Tuning

**Rationale:** AppImage-specific issues (LD_LIBRARY_PATH, Python 3 in AppImage context, bundled Proton script invocation) cannot be tested until Phase 1 is working on development builds. Gamescope/SteamGameId tuning depends on Phase 1 validation results.

**Delivers:** Working Proton launch from the distributed AppImage. Gamescope `SteamGameId` injection for improved window tracking. Steam install auto-detection for `STEAM_COMPAT_CLIENT_INSTALL_PATH`.

**Addresses:** Python 3 dependency in AppImage, LD_LIBRARY_PATH triple collision, Steam install detection, non-Steam app ID detection from `shortcuts.vdf`, `SteamGameId` env var injection.

**Uses:** `CLUCKERS_BUNDLED_PROTON` env var (already set by AppRun), existing `findCluckersAppID()` code, common Steam install path detection.

**Avoids:** LD_LIBRARY_PATH AppImage leak, Python-not-found silent failure (add explicit check with distro-specific install instructions), `STEAM_COMPAT_CLIENT_INSTALL_PATH` pointing to writable temp dir (security issue).

### Phase 3: Cleanup and Polish

**Rationale:** Dead code removal and UX improvements are low-risk and done after Phase 2 confirms the new pipeline is stable. Removing old code paths eliminates the maintenance burden of two parallel Wine/Proton code paths.

**Delivers:** Simplified codebase with Proton as the only launch path for Proton-GE users. Updated status command, error messages, and repair instructions. Code deletion of 150+ lines of manual prefix management.

**Addresses:** Dead code removal (`createFromProtonTemplate`, `ensureDosdevices`, `copyProtonTemplate`, DLL verification for Proton path), `cluckers status` showing Proton version, updated error messages and repair instructions, `cluckers steam add` instructions mentioning forced Proton compatibility.

**Avoids:** Two-prefix confusion (old prefix cleanup messaging), steam.exe stub warning leakage to user output (filter from visible output), no-explanation first-launch delay (explicit "one-time setup" message).

### Phase Ordering Rationale

- Phase 1 before Phase 2: AppImage-specific issues cannot be debugged until the base invocation works on development builds. The controller validation gate at the end of Phase 1 is load-bearing -- if it fails, Phase 2 scope changes fundamentally.
- Phase 3 after Phase 2: Dead code removal is safe only after the new path is proven stable in production (AppImage). Removing system Wine fallback code too early would strand users who encounter Proton regressions.
- shm_launcher.exe is the highest-risk integration: test it in Phase 1 step 4 before building other Phase 1 steps on top. If shm_launcher requires changes under Proton (unlikely but possible), knowing early bounds rework.

### Research Flags

Phases needing deeper research during planning:

- **Phase 1 (Gamescope validation):** The theory that standalone `proton run` sets the `STEAM_GAME` X11 property correctly is MEDIUM confidence. No way to verify this without Steam Deck hardware testing. Phase 1 planning should include an explicit spike task: "Run game with `PROTON_LOG=1`, confirm `STEAM_GAME` property appears on game windows via `xprop`."
- **Phase 2 (Python 3 in AppImage):** GE-Proton bundles its own Python at `files/bin/python3`. Whether this Python is correctly invoked when running from inside an AppImage (with modified `LD_LIBRARY_PATH` and `PATH`) is MEDIUM confidence. Needs a specific AppImage build test before committing to this approach.

Phases with standard patterns (skip additional research):

- **Phase 1 (env var setup):** The required Proton environment variables are fully documented from Proton source code. Implementation is mechanical.
- **Phase 1 (shm_launcher compatibility):** Win32 `CreateFileMappingW`/`OpenFileMapping` APIs are core Wine functionality that Proton does not change. HIGH confidence shm_launcher works unchanged. Confirmed by simshmbridge reference implementation.
- **Phase 3 (code deletion):** Removing dead code paths follows standard Go patterns. No research needed.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Proton source reviewed at source level (GE-Proton10-32 in build/appimage/). Zero new Go deps confirmed. Python 3 requirement confirmed. |
| Features | HIGH | Feature set derived from existing codebase analysis and Proton/Gamescope source and issue tracker. Table stakes are clear regression tests against v1.0. |
| Architecture | HIGH | Component boundaries verified against actual Proton script (2396 lines, reviewed). shm_launcher compatibility analysis is authoritative. |
| Pitfalls | HIGH | WINEDLLOVERRIDES issue sourced from Proton PR #1705. Prefix structure from Proton CompatData class. Env var KeyError from issue #9068. |

**Overall confidence:** HIGH for implementation plan. MEDIUM for the controller fix outcome (Gamescope integration with standalone `proton run`).

### Gaps to Address

- **Gamescope `STEAM_GAME` property with standalone `proton run`:** Whether this works without Steam running is unconfirmed. Must be validated in Phase 1 on Steam Deck hardware. If `STEAM_GAME` is not set, the controller fix requires the user to launch through Steam as a non-Steam shortcut (which `cluckers steam add` already enables). Plan B: document that the controller fix only works when launched through Steam.

- **`STEAM_COMPAT_CLIENT_INSTALL_PATH` fallback behavior:** Confirmed that `try_get_steam_dir()` in the Proton script returns `None` gracefully when Steam is not installed (Steam DLL copies are silently skipped). However, this behavior may differ between Proton-GE versions. Test on a system without Steam installed.

- **Proton steam.exe stub timing:** The stub runs during Proton initialization and should exit before the game starts. If it hangs (e.g., attempting to connect to a non-running Steam client), game launch stalls. Needs validation with `PROTON_LOG=1` and Steam not running.

- **Bundled Python 3 path in AppImage:** GE-Proton's bundled `files/bin/python3` may or may not be on `PATH` when AppRun invokes the launcher. The `proton` script shebang (`#!/usr/bin/env python3`) finds whatever is first on `PATH`. Test AppImage on a system without system Python 3.

## Sources

### Primary (HIGH confidence)
- GE-Proton10-32 `proton` script -- `/home/cstory/cluckers/build/appimage/Cluckers.AppDir/proton/proton` (2396 lines, reviewed in full for env var requirements, prefix lifecycle, UMU_ID path handling)
- [Proton Wine Prefix Management (DeepWiki)](https://deepwiki.com/ValveSoftware/Proton/2.2-wine-prefix-management) -- CompatData class, `setup_prefix()` behavior, version tracking, directory structure
- [Proton WINEDLLOVERRIDES PR #1705](https://github.com/ValveSoftware/Proton/pull/1705/files) -- `PROTON_WINEDLLOVERRIDES` as the correct mechanism for user overrides
- [STEAM_COMPAT_CLIENT_INSTALL_PATH KeyError issue #9068](https://github.com/ValveSoftware/Proton/issues/9068) -- confirmed required variable, crash behavior
- [Non-Steam game focus issue #8513](https://github.com/ValveSoftware/steam-for-linux/issues/8513) -- `SteamGameId` env var, `STEAM_GAME` X11 property, Gamescope focus tracking
- [Gamescope issue #416](https://github.com/ValveSoftware/gamescope/issues/416) -- `GAMESCOPE_FOCUSED_APP`, `STEAM_GAME` property behavior in Steam vs non-Steam mode
- Existing codebase: `internal/wine/`, `internal/launch/`, `internal/config/` -- definitive source of what changes are needed
- [simshmbridge](https://github.com/Spacefreak18/simshmbridge) -- confirms `CreateFileMappingW`/`OpenFileMapping` works under Wine/Proton

### Secondary (MEDIUM confidence)
- [Running Proton outside Steam](https://megadarken.github.io/software/2024/08/05/proton-outside-steam.html) -- CLI invocation, env var requirements
- [Run .exe in Proton prefix (GitHub Gist)](https://gist.github.com/michaelbutler/f364276f4030c5f449252f2c4d960bd2) -- path handling, WINEPREFIX mapping
- [Launch games via Proton CLI (GitHub Gist)](https://gist.github.com/sxiii/6b5cd2e7d2321df876730f8cafa12b2e) -- practical invocation examples
- [umu-launcher](https://github.com/Open-Wine-Components/umu-launcher) -- alternative approach; confirms direct `proton run` is viable for non-Steam use
- [Arch Wiki - Gamescope](https://wiki.archlinux.org/title/Gamescope) -- Gamescope configuration, Steam Deck integration

### Tertiary (context)
- [GE-Proton vs Valve Proton (HowToGeek)](https://www.howtogeek.com/proton-vs-proton-ge-whats-the-difference-and-which-one-should-you-use/) -- same interface, different patches
- [LD_LIBRARY_PATH AppImage issues](https://github.com/AppImage/AppImageKit/issues/126) -- library path conflicts in AppImage context
- [Python3 requirement for Steam Play](https://steamcommunity.com/app/221410/discussions/8/3276824275008597014/) -- Python 3 as Proton dependency

---
*Research completed: 2026-02-24*
*Ready for roadmap: yes*
