# Feature Landscape: Proton Launch Pipeline (v1.1)

**Domain:** Proton-based game launch pipeline for non-Steam game launcher (Steam Deck controller fix)
**Researched:** 2026-02-24
**Confidence:** HIGH -- based on Valve Proton source analysis, umu-launcher docs, Gamescope issue tracker, GE-Proton source, and existing codebase analysis

## Context

This research is for v1.1 only -- the transition from direct Wine binary execution (`wine64 shm_launcher.exe ...`) to Proton launch pipeline (`proton run shm_launcher.exe ...`). The core motivation is fixing Steam Deck controller input loss during UE3 ServerTravel (window recreation). All existing v1.0 features (auth, download, GUI, AppImage, etc.) are already shipped and working.

## Table Stakes

Features that MUST work for v1.1 to ship. Missing any = regression from v1.0.

| Feature | Why Expected | Complexity | Dependencies |
|---------|--------------|------------|--------------|
| Proton runtime detection | Users already have Proton-GE installed (v1.0 detects it). v1.1 must find the `proton` Python script in the same Proton-GE install, not just `wine64`. | Low | Existing `wine.FindProtonGE()` -- extend to return `proton` script path alongside `wine64` path |
| `proton run` invocation | Replace `exec.Command(wine64, shm_launcher.exe, ...)` with `exec.Command(proton, "run", shm_launcher.exe, ...)`. The Proton script handles Wine environment setup internally. | Med | Proton runtime detection, environment variables setup |
| Required environment variables | `proton run` requires `STEAM_COMPAT_DATA_PATH` (prefix location) and `STEAM_COMPAT_CLIENT_INSTALL_PATH` (Steam install dir). Without these, the Proton Python script raises `KeyError` and crashes. | Low | Steam install detection, prefix path configuration |
| Proton-managed prefix creation | `proton run` auto-creates the prefix at `$STEAM_COMPAT_DATA_PATH/pfx/` on first launch using its own `default_pfx` template. This replaces the current manual `copyProtonTemplate()` + `wineboot --init` + `ensureDosdevices()` flow. Proton handles DLL symlinks, registry setup, dosdevices, font symlinks, and version tracking -- all things the current code does manually. | Med | STEAM_COMPAT_DATA_PATH set correctly, proton script accessible |
| Backward compatibility with system Wine | Users without Proton-GE (using system Wine) must still be able to launch. The Proton pipeline is Proton-GE only; system Wine users keep the existing `wine64` direct execution path. | Low | Existing Wine detection already distinguishes Proton-GE vs system Wine via `IsProtonGE()` |
| All existing game arguments preserved | `-user`, `-token`, `-eac_oidc_token_file`, `-hostx`, `-content_bootstrap_shm`, `-Language=INT`, `-dx11`, etc. must all pass through to the game exe unchanged. `proton run` passes all trailing arguments after the executable to Wine. | Low | None -- `proton run exe arg1 arg2` works identically to `wine64 exe arg1 arg2` for argument forwarding |
| Temp file cleanup | OIDC token file, bootstrap file, extracted shm_launcher.exe must still be cleaned up after game exits. `proton run` blocks until the game exits (verb `waitforexitandrun` is default), so the existing defer-based cleanup works. | Low | None -- existing pattern works unchanged |
| Wine log capture | Current code tees Wine stderr to `/tmp/cluckers_wine.log`. Proton redirects differently -- must verify stderr still flows through or adjust log capture. | Low | Test with actual Proton execution |
| Non-Deck Linux still works | Desktop Linux users (Arch, Ubuntu, Fedora) must not see regressions. Proton launch should work identically on desktop as on Deck -- the controller fix is a bonus on Deck, but the launch pipeline change must not break desktop. | Low | Testing across platforms |

## Differentiators

Features that make v1.1 more than just "a different way to launch the same game."

| Feature | Value Proposition | Complexity | Dependencies |
|---------|-------------------|------------|--------------|
| Steam Deck controller persistence through ServerTravel | THE primary reason for v1.1. When launching through `proton run`, the Proton session registers properly with Gamescope via `STEAM_GAME` X11 property and `SteamGameId` env var. This means Gamescope tracks the game window across UE3 ServerTravel (destroy + recreate D3D window), keeping the controller in game mode instead of reverting to desktop mode. Fixes the button zeroing issue (HID bytes 8-13) that makes the game unplayable on Steam Deck in matches. | High | Proton launch pipeline, correct SteamGameId/app ID setup, game added as non-Steam shortcut with Proton compatibility forced |
| Elimination of manual DLL verification | Current `wine.VerifyPrefix()` checks for 4 specific DLLs (vcruntime140, msvcp140, d3dx11_43, d3d11). Proton-managed prefixes include all of these automatically because Proton bundles its own DXVK, VKD3D, and VC runtime. The DLL verification step becomes unnecessary for Proton prefixes. Simplifies codebase and removes a class of user-facing errors. | Low | Proton-managed prefix |
| Elimination of winetricks dependency | Current system Wine path requires `winetricks -q vcrun2022 d3dx11_43 dxvk`. Proton bundles all of these. For users on Proton (the vast majority), winetricks is no longer needed at all. Removes an external dependency and a 10-minute install step that frequently fails. | Low | Proton-managed prefix |
| Simpler prefix repair | Current repair instructions differ by Wine type and list specific winetricks commands. With Proton: "delete the prefix directory, re-launch, Proton recreates it." One instruction for everyone. | Low | Proton-managed prefix |
| Gamescope-aware environment setup | Set `SteamGameId` and configure the Proton environment so Gamescope properly tracks the game window. This is what makes the controller fix work -- Gamescope uses `GAMESCOPE_FOCUSED_APP` (set only in Steam mode) and `STEAM_GAME` X11 properties to decide which window gets controller input. Proton sets these from `SteamGameId`. | Med | Non-Steam shortcut app ID detection, Proton launch |
| Clean prefix migration | Detect existing `~/.cluckers/prefix/` (old Wine prefix), warn user it is no longer used, offer to remove it. New prefix lives at `~/.cluckers/proton/pfx/` (or wherever `STEAM_COMPAT_DATA_PATH` points). Prevents confusion from having two prefix directories. | Low | None |

## Anti-Features

Features to explicitly NOT build for v1.1.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| umu-launcher integration | umu-launcher is the officially recommended way to run Proton outside Steam (endorsed by GE-Proton). However, it adds a Python dependency, downloads its own Steam Runtime (hundreds of MB), and requires either system install or Flatpak. Cluckers already bundles Proton-GE in the AppImage and can invoke `proton run` directly. umu-launcher solves problems we do not have (Steam Runtime containerization, protonfixes database, multi-store support). | Invoke the `proton` Python script directly with the required environment variables. This is simpler, avoids the umu dependency, and works inside the AppImage where we already bundle Proton-GE. |
| Steam Runtime (pressure-vessel) containerization | Proton launched through Steam runs inside a pressure-vessel container for library isolation. Invoking `proton run` directly bypasses this container. This is intentional -- the AppImage already provides library isolation, and pressure-vessel adds complexity (requires Steam Runtime sniper tarball, D-Bus access, namespace setup). Heroic and Lutris both support non-containerized Proton execution. | Run `proton run` directly without pressure-vessel. The Proton script still sets up Wine environment, DLLs, DXVK, etc. correctly. If a game needs container isolation, users can launch through Steam directly. |
| Automatic Proton version management | umu-launcher can auto-download GE-Proton and UMU-Proton. ProtonUp-Qt manages Proton versions. Building another Proton version manager adds complexity with no unique value. | Detect installed Proton-GE versions (already done). Use newest. Error with install instructions if none found. AppImage bundles one. |
| PROTON_VERB customization | Proton supports verbs: `run`, `waitforexitandrun`, `runinprefix`, `getcompatpath`, etc. Only `waitforexitandrun` (default) and `run` are relevant. Exposing this as a user config adds confusion. | Always use the default verb (`waitforexitandrun` via `proton run`), which blocks until the game exits. This matches the current behavior where `cmd.Run()` blocks. |
| Custom SteamGameId assignment | The non-Steam game app ID is generated by Steam when the shortcut is created. Allowing users to override it creates broken Gamescope tracking. | Auto-detect the app ID from `shortcuts.vdf` (existing `findCluckersAppID()` code) or use a sensible default. |
| Proton log management | Proton can write detailed logs via `PROTON_LOG=1`. Building log rotation, viewing, or analysis into Cluckers is scope creep. | Mention `PROTON_LOG=1` in verbose output and troubleshooting docs. Let users enable it manually for debugging. |
| Wine-GE support | Wine-GE (standalone GE Wine builds, not Proton-GE) exists but is a different distribution than Proton-GE. It does not include the `proton` script. Supporting Wine-GE as a Proton runtime adds complexity for an edge case. | Treat Wine-GE as system Wine (direct `wine64` execution, existing path). Only Proton-GE gets the `proton run` pipeline. |

## Feature Dependencies

```
Proton Runtime Detection --> proton run Invocation (need proton script path)
Proton Runtime Detection --> STEAM_COMPAT_DATA_PATH Setup (need Proton base dir)
Steam Install Detection --> STEAM_COMPAT_CLIENT_INSTALL_PATH (need Steam root path)
STEAM_COMPAT_DATA_PATH Setup --> Proton Prefix Auto-Creation (Proton creates pfx/ here)
proton run Invocation --> Controller Persistence (Proton sets STEAM_GAME property)
Non-Steam App ID Detection --> SteamGameId Env Var (for Gamescope tracking)
SteamGameId Env Var --> Controller Persistence (Gamescope uses this for focus)
Old Prefix Detection --> Migration Warning (check if ~/.cluckers/prefix/ exists)
```

Launch pipeline (v1.1, Proton-GE path):
```
Health Check -> Auth -> OIDC -> Bootstrap
  -> Detect Proton Runtime (find proton script + wine64)
  -> Setup Proton Environment (STEAM_COMPAT_DATA_PATH, STEAM_COMPAT_CLIENT_INSTALL_PATH, SteamGameId)
  -> Prefix Ready Check (does pfx/ exist? proton run creates it on first launch if not)
  -> Check Version -> Download Game -> Deck Config
  -> Launch via proton run (shm_launcher.exe + game args)
```

Launch pipeline (v1.1, system Wine fallback):
```
Health Check -> Auth -> OIDC -> Bootstrap
  -> Detect Wine (system wine, no proton script)
  -> Ensure Wine Prefix (existing code path: wineboot + winetricks)
  -> Verify Wine Prefix DLLs (existing code path)
  -> Check Version -> Download Game -> Deck Config
  -> Launch via wine64 (existing code path, unchanged)
```

## Detailed Feature Specifications

### 1. Proton Runtime Detection

**Current state:** `wine.FindProtonGE()` scans ~10 directories for `GE-Proton*/files/bin/wine64`. Returns `ProtonGEInstall{WinePath, ProtonDir}`.

**What changes:** Also locate the `proton` script. For GE-Proton, it lives at `{ProtonDir}/proton`. This is a Python script that sets up the Wine environment and invokes Wine. Need to verify the script exists and is executable.

**Key detail:** The `proton` script requires Python 3. On SteamOS, Python 3 is available. On most Linux distros, Python 3 is available. If Python 3 is missing, fall back to direct Wine execution with a warning.

**Existing code to modify:** `wine.ProtonGEInstall` struct -- add `ProtonScript string` field. `wine.FindProtonGE()` -- check for `proton` script alongside `wine64`. `wine.FindWine()` -- return enough info to determine launch method (proton vs direct wine).

### 2. Environment Variable Setup

**Required variables for `proton run`:**

| Variable | Value | Purpose |
|----------|-------|---------|
| `STEAM_COMPAT_DATA_PATH` | `~/.cluckers/proton/` | Where Proton creates `pfx/` subdirectory with the Wine prefix |
| `STEAM_COMPAT_CLIENT_INSTALL_PATH` | `~/.local/share/Steam/` or detected Steam root | Proton reads Steam client files from here; can point to a minimal stub if Steam is not installed |
| `SteamGameId` | App ID from `shortcuts.vdf` or `0` | Proton uses this to set `STEAM_GAME` X11 property for Gamescope focus tracking |
| `SteamAppId` | Same as SteamGameId | Some Proton internals read this variant |

**Variables to NOT set (Proton handles these internally):**
- `WINEPREFIX` -- Proton sets this to `$STEAM_COMPAT_DATA_PATH/pfx/`
- `WINEDLLPATH` -- Proton sets this based on its own lib directories
- `WINEFSYNC` / `WINEESYNC` -- Proton enables fsync/esync based on kernel support
- `WINEDLLOVERRIDES` -- need to verify if `dxgi=n` override is still needed or if Proton's DXVK handles this

**Key risk:** `STEAM_COMPAT_CLIENT_INSTALL_PATH` is required but may not exist if Steam is not installed (AppImage users, some desktop users). The Proton script does `os.environ["STEAM_COMPAT_CLIENT_INSTALL_PATH"]` which raises `KeyError` if unset. Options: (a) detect Steam install, (b) create a minimal stub directory, (c) set to the Proton directory itself (some community scripts do this).

### 3. Proton Prefix Lifecycle

**Current state:** Cluckers manually manages a Wine prefix at `~/.cluckers/prefix/`:
- Copy `default_pfx` template from Proton-GE
- Create `dosdevices/` symlinks (c: and z:)
- Run `wineboot --init`
- Verify 4 specific DLLs

**New state (Proton-managed):** Proton's `setup_prefix()` does ALL of the above automatically when `proton run` is invoked and the prefix does not exist at `$STEAM_COMPAT_DATA_PATH/pfx/`. It also:
- Sets up `tracked_files` for clean upgrades
- Creates a `version` file for prefix versioning
- Handles `update-timestamp` to prevent Wine auto-updates
- Creates font symlinks
- Sets machine GUID in registry
- Handles prefix upgrades when Proton version changes

**What Cluckers needs to do:**
1. Set `STEAM_COMPAT_DATA_PATH` to `~/.cluckers/proton/`
2. On first launch, `proton run` creates `~/.cluckers/proton/pfx/` automatically (takes 5-30 seconds)
3. On subsequent launches, `proton run` verifies prefix version matches and upgrades if needed
4. Remove the manual `stepEnsurePrefix` and `stepVerifyPrefix` pipeline steps for Proton path
5. Replace with a simpler "Preparing Proton environment" step that sets env vars and checks if `proton` script exists

### 4. proton run Invocation

**Current command (direct Wine):**
```
wine64 shm_launcher.exe Z:\tmp\bootstrap.bin shmname Z:\path\to\game.exe -user=X -token=Y ...
```

**New command (Proton):**
```
STEAM_COMPAT_DATA_PATH=~/.cluckers/proton \
STEAM_COMPAT_CLIENT_INSTALL_PATH=~/.local/share/Steam \
SteamGameId=3928144816 \
/path/to/GE-Proton10-1/proton run \
  /tmp/shm_launcher.exe \
  Z:\tmp\bootstrap.bin \
  shmname \
  Z:\path\to\game.exe \
  -user=X -token=Y ...
```

**Key differences:**
1. Binary changes from `wine64` to `proton` (the Python script)
2. First arg to proton is `run` (the verb)
3. Path format for shm_launcher.exe: Proton accepts Linux paths (it converts internally). May not need `LinuxToWinePath()` for the exe path -- but arguments to the game that reference Wine paths (like `-eac_oidc_token_file=Z:\...`) still need Wine path format.
4. `WINEPREFIX` is NOT set externally -- Proton derives it from `STEAM_COMPAT_DATA_PATH/pfx/`
5. `WINEFSYNC`, `WINEDLLOVERRIDES` may not need to be set -- Proton configures these

**Critical detail:** The `proton` script is Python. On the AppImage, Python must be available. Current AppImage bundles Proton-GE which includes a Python interpreter at `files/bin/python3` (Proton bundles its own). Verify this works inside AppImage context.

### 5. Gamescope / Controller Integration

**The problem (proven):** When UE3 does ServerTravel, it destroys and recreates the D3D window. Gamescope loses track of the game window. Steam reconfigures the controller firmware to desktop mode. Buttons (HID bytes 8-13) are zeroed at hardware level.

**Why Proton helps:** When Steam launches a game (including non-Steam shortcuts with forced Proton), it:
1. Sets `SteamGameId` env var for the process
2. Proton's Wine sets `STEAM_GAME` X11 property on game windows
3. Gamescope uses `STEAM_GAME` to track which app owns which window
4. When the D3D window is recreated, the new window gets the same `STEAM_GAME` property
5. Gamescope recognizes it as the same app and keeps controller in game mode

**What we need:**
- Set `SteamGameId` to the app ID Steam assigned to our non-Steam shortcut
- Detect this app ID from `shortcuts.vdf` (existing `findCluckersAppID()` code works)
- If app ID is unknown (game not added to Steam), warn user and set a fallback value
- The `proton` script reads `SteamGameId` and passes it to Wine, which sets the X11 property

**Confidence level:** MEDIUM. The theory is sound based on how Steam+Proton+Gamescope interact. However, invoking `proton run` directly (outside Steam) may not set X11 properties the same way as when Steam invokes Proton. The `STEAM_GAME` property is set by Wine's window management code, which reads from `SteamGameId`. This should work regardless of whether Steam or a third-party launcher invokes Proton. Testing on actual hardware is required to confirm.

**Fallback:** If direct `proton run` does not properly set `STEAM_GAME` on the game windows, the alternative is launching Cluckers as a non-Steam shortcut that Steam launches with Proton forced on. In that model, Steam itself manages the entire lifecycle and Gamescope tracking works natively. The downside is the user must launch through Steam rather than from a terminal or .desktop file.

### 6. Steam Install Detection

**Purpose:** Find `STEAM_COMPAT_CLIENT_INSTALL_PATH` value.

**Search locations (priority order):**
1. `~/.local/share/Steam/` (native Steam, most common)
2. `~/.steam/steam/` (symlink, common alternative)
3. `~/.steam/root/` (symlink, some setups)
4. `~/.var/app/com.valvesoftware.Steam/data/Steam/` (Flatpak Steam)
5. `~/snap/steam/common/.steam/steam/` (Snap Steam)

**Validation:** Check for `steam.sh` or `ubuntu12_32/steam-runtime/` subdirectory to confirm it is a real Steam installation.

**If not found:** Not a blocker for launch. Set to the Proton directory itself or a stub path. The `proton` script primarily uses this for Steam Runtime container setup (which we skip) and some font/library paths. Test what happens when this path does not point to a real Steam install.

### 7. Old Prefix Migration

**Scenario:** User has `~/.cluckers/prefix/` from v1.0 (direct Wine execution). v1.1 creates a new prefix at `~/.cluckers/proton/pfx/`.

**Migration plan:**
1. On launch, check if `~/.cluckers/prefix/` exists
2. If it does, inform user: "Migrating from Wine prefix to Proton prefix. Your old prefix at ~/.cluckers/prefix/ is no longer used."
3. Do not delete automatically -- user may have save files or configs in the prefix
4. After successful Proton launch, suggest: "You can safely remove the old prefix: rm -rf ~/.cluckers/prefix/"
5. After 2-3 releases, consider auto-removal with backup prompt

### 8. AppImage Proton Script Access

**Current AppImage:** Bundles Proton-GE at a path like `$APPDIR/proton-ge/`. Sets `CLUCKERS_BUNDLED_PROTON` env var pointing to this directory. Current code finds `wine64` at `$CLUCKERS_BUNDLED_PROTON/files/bin/wine64`.

**What changes:** Also need the `proton` script at `$CLUCKERS_BUNDLED_PROTON/proton`. This Python script needs Python 3 to run. Proton-GE bundles its own Python at `files/bin/python3`.

**Concern:** The AppImage cleans `LD_LIBRARY_PATH` before launching Wine (to prevent AppImage-bundled libs from conflicting with Wine). Need to verify that the bundled Python 3 works correctly in this environment.

## MVP Recommendation

**Phase 1: Core Proton Pipeline (must ship)**
1. Proton runtime detection (find `proton` script alongside `wine64`)
2. Environment variable setup (`STEAM_COMPAT_DATA_PATH`, `STEAM_COMPAT_CLIENT_INSTALL_PATH`)
3. `proton run` invocation replacing `wine64` direct execution
4. Proton-managed prefix (remove manual prefix creation for Proton path)
5. System Wine fallback (preserve existing code path)
6. Old prefix migration warning

**Phase 2: Gamescope Controller Fix (core value)**
1. Steam install path detection
2. Non-Steam app ID detection from `shortcuts.vdf`
3. `SteamGameId` environment variable injection
4. Verification that `STEAM_GAME` X11 property is set on game windows
5. Testing controller persistence through ServerTravel on actual Steam Deck hardware

**Phase 3: Cleanup (polish)**
1. Remove dead code (manual prefix template copy, DLL verification for Proton path, winetricks path if fully deprecated)
2. Update `cluckers status` to show Proton runtime info instead of Wine prefix DLL status
3. Update error messages and repair instructions for Proton prefix
4. Update `cluckers steam add` instructions to mention forcing Proton compatibility

**Defer:**
- umu-launcher integration: Not needed; direct `proton run` is simpler
- Steam Runtime containerization: Not needed; adds complexity without benefit
- Proton log management: Users can set `PROTON_LOG=1` manually

## Sources

- [Valve Proton source - Wine prefix management](https://deepwiki.com/ValveSoftware/Proton/2.2-wine-prefix-management) -- prefix creation, versioning, tracked_files system, directory structure (HIGH confidence)
- [GE-Proton source - proton script](https://raw.githubusercontent.com/GloriousEggroll/proton-ge-custom/master/proton) -- environment variables, setup_prefix(), run verb implementation (HIGH confidence)
- [umu-launcher documentation](https://github.com/Open-Wine-Components/umu-launcher) -- GAMEID, PROTONPATH, WINEPREFIX env vars, Steam Runtime container approach (HIGH confidence)
- [Gamescope issue #416 - GAMESCOPE_FOCUSED_APP behavior](https://github.com/ValveSoftware/gamescope/issues/416) -- STEAM_GAME property only set in Steam mode (HIGH confidence)
- [Steam for Linux issue #8513 - non-Steam game focus](https://github.com/ValveSoftware/steam-for-linux/issues/8513) -- SteamGameId env var, STEAM_GAME X11 property, Gamescope focus tracking (HIGH confidence)
- [How to launch games via Proton from CLI](https://gist.github.com/sxiii/6b5cd2e7d2321df876730f8cafa12b2e) -- STEAM_COMPAT_DATA_PATH, STEAM_COMPAT_CLIENT_INSTALL_PATH required vars (MEDIUM confidence)
- [Using Proton outside Steam](https://megadarken.github.io/software/2024/08/05/proton-outside-steam.html) -- direct proton run invocation without Steam client (MEDIUM confidence)
- [umu-launcher man page](https://man.archlinux.org/man/umu.1.en) -- PROTON_VERB, environment variable reference (HIGH confidence)
- [umu-launcher FAQ - proton run outside Steam](https://github.com/Open-Wine-Components/umu-launcher/wiki/Frequently-asked-questions-(FAQ)) -- umu-launcher is officially recommended method for Proton outside Steam (HIGH confidence)
- [Arch Wiki - Gamescope](https://wiki.archlinux.org/title/Gamescope) -- Gamescope configuration, controller input, Steam Deck role (HIGH confidence)
- Existing Cluckers codebase -- `internal/wine/`, `internal/launch/`, controller debugging memory file (HIGH confidence, primary source)
