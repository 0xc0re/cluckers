# Technology Stack: Proton Launch Pipeline

**Project:** Cluckers v1.1 -- Switch from direct Wine to `proton run`
**Researched:** 2026-02-24
**Overall confidence:** HIGH (Proton source code reviewed, multiple sources corroborated)

## Executive Summary

Switching from direct Wine execution (`wine64 shm_launcher.exe ...`) to `proton run` is primarily an **invocation change**, not a dependency change. No new Go libraries are needed. The changes are in `internal/launch/process_linux.go` (how the game process is spawned), `internal/wine/detect.go` (locating the `proton` script instead of `wine64`), and `internal/wine/prefix.go` (letting Proton manage its own prefix instead of manually copying `default_pfx`).

The `proton` binary is a **Python script** that ships with every Proton-GE install at `<proton_dir>/proton`. It accepts the verb `waitforexitandrun` followed by a Windows executable path. It reads environment variables to find the prefix, sets up Wine with DXVK/WINEFSYNC/DLL overrides automatically, and launches the executable via its bundled `wine64`. This eliminates the need for manual DLL management, winetricks, and most environment variable setup that the current code does.

---

## What Changes (and What Does Not)

### Changes Required

| Area | Current (v1.0) | New (v1.1) | Impact |
|------|---------------|------------|--------|
| **Process invocation** | `exec.Command(wine64Path, shmLauncher, ...)` | `exec.Command(protonScript, "waitforexitandrun", shmLauncher, ...)` | Modify `process_linux.go` |
| **Environment variables** | `WINEPREFIX`, `WINEFSYNC=1`, `WINEDLLOVERRIDES=dxgi=n` | `STEAM_COMPAT_DATA_PATH`, `STEAM_COMPAT_CLIENT_INSTALL_PATH`, `SteamGameId`, `SteamAppId` | Modify `process_linux.go` |
| **Prefix management** | Manual: copy `default_pfx`, create dosdevices, `wineboot --init`, verify 4 DLLs | Automatic: Proton creates `$STEAM_COMPAT_DATA_PATH/pfx/` on first run from its own `default_pfx` | Simplify `prefix.go`, remove `verify.go` DLL checks |
| **Wine detection** | Find `wine64` binary at `<proton_dir>/files/bin/wine64` | Find `proton` script at `<proton_dir>/proton` | Modify `detect.go` |
| **Path conversion** | `LinuxToWinePath()` converts `/path` to `Z:\path` | Same -- Proton still uses Wine underneath, Z: drive still exists | No change |
| **DLL override** | `WINEDLLOVERRIDES=dxgi=n` set manually | Proton sets DXVK overrides automatically; may still need game-specific `dxgi=n` | Test, possibly simplify |
| **Prefix location** | `~/.cluckers/prefix/` (WINEPREFIX) | `~/.cluckers/proton/` (STEAM_COMPAT_DATA_PATH, actual prefix at `pfx/` subdir) | New config path |

### No Changes Needed

| Area | Why |
|------|-----|
| **Go dependencies** | `exec.Command` is stdlib. No new packages. |
| **shm_launcher.exe** | Works identically under Proton (same Wine/CreateFileMappingW). Proton runs Win32 executables the same way Wine does -- it IS Wine underneath. |
| **Game arguments** | All `-user=`, `-token=`, `-hostx=`, `-eac_oidc_token_file=` args pass through unchanged. |
| **Credential encryption** | NaCl secretbox is unrelated to the launch mechanism. |
| **Download/update pipeline** | Game file management is unchanged. |
| **GUI (Fyne)** | Pipeline reporter interface is unchanged; only step names and implementations change. |
| **AppImage packaging** | Still bundles Proton-GE; the `proton` script is already inside the bundle at `<proton_dir>/proton`. |
| **Windows build** | `process_windows.go` is completely unaffected (no Wine/Proton on Windows). |

---

## The `proton` Script: How It Works

### What It Is

The `proton` binary is a **Python 3 script** (~1500 lines) located at the root of every Proton/GE-Proton installation directory. It is the official entry point that Valve's Steam client uses to launch games. GE-Proton includes the same script with additional game-specific compatibility patches.

**Confidence:** HIGH (reviewed Proton source at `github.com/ValveSoftware/Proton` branch `proton_9.0` and GE-Proton `master`)

### Directory Layout

```
GE-Proton10-1/              # ProtonDir (already detected by FindProtonGE)
  proton                     # <-- Python script, the new launch target
  files/
    bin/
      wine64                 # <-- Current launch target (v1.0)
      wine
      wineserver
    lib/
      wine/
        i386-windows/        # 32-bit Windows DLLs (DXVK, vcruntime, etc.)
        x86_64-windows/      # 64-bit Windows DLLs
        i386-unix/           # 32-bit Unix DLLs
        x86_64-unix/         # 64-bit Unix DLLs
    share/
      default_pfx/           # Template prefix (used by proton's setup_prefix())
      fonts/                 # Fonts symlinked into prefix
  default_pfx/               # Alternative template location (ProtonUp-Qt installs)
```

### Invocation Interface

```bash
# Minimal invocation:
STEAM_COMPAT_DATA_PATH=/path/to/compatdata \
STEAM_COMPAT_CLIENT_INSTALL_PATH=/path/to/steam \
  /path/to/GE-Proton10-1/proton waitforexitandrun /path/to/game.exe [args...]
```

### Accepted Verbs

| Verb | Behavior | Use Case |
|------|----------|----------|
| `run` | Launch executable, return immediately (wineserver may still run) | Multiple processes in same prefix |
| `waitforexitandrun` | Wait for any existing wineserver to exit, then launch. Blocks until launched process exits. | **Use this.** Clean launch, blocks like current `cmd.Run()`. |
| `runinprefix` | Run command inside existing prefix without setup | Utility commands in existing prefix |
| `getcompatpath` | Convert Linux path to Wine path | Path conversion (not needed, we have `LinuxToWinePath`) |
| `getnativepath` | Convert Wine path to Linux path | Reverse path conversion |
| `destroyprefix` | Remove tracked files from prefix | Cleanup |

**Use `waitforexitandrun`** because:
1. It blocks until the game exits, matching current `cmd.Run()` behavior
2. It ensures a clean wineserver state before launch
3. It is the verb Steam itself uses for game launches

### Required Environment Variables

| Variable | Required | Value | Purpose |
|----------|----------|-------|---------|
| `STEAM_COMPAT_DATA_PATH` | **YES** | `~/.cluckers/proton/` | Root of compatdata directory. Proton creates `pfx/` subdirectory here for the actual Wine prefix. Script exits with error if missing. |
| `STEAM_COMPAT_CLIENT_INSTALL_PATH` | **YES** | Path to Steam install, or a fake directory | Used to locate `steamclient.so` and legacy Steam DLLs. Script raises `KeyError` if missing. Can point to Steam install (`~/.local/share/Steam/`) or a dummy path if Steam is not installed. |
| `SteamGameId` | Recommended | Any numeric string (e.g., `"0"`) | Identifies game to Proton. When unset, some protonfixes and Gamescope integration may not work correctly. Set to `"0"` for generic non-Steam game. |
| `SteamAppId` | Optional | Same as SteamGameId | Used for game-specific compatibility configs. |

### Variables Proton Sets Automatically (DO NOT set manually)

| Variable | What Proton Does |
|----------|-----------------|
| `WINEPREFIX` | Set to `$STEAM_COMPAT_DATA_PATH/pfx/` |
| `WINEFSYNC` | Set to `"1"` (unless `nofsync` in config) |
| `WINEESYNC` | Set to `"1"` (unless `noesync` in config) |
| `WINENTSYNC` | Set to `"1"` (GE-Proton addition) |
| `WINEDLLPATH` | Set to Proton's DLL directories |
| `WINEDLLOVERRIDES` | Set with DXVK, steam.exe, and other builtin overrides |
| `LD_LIBRARY_PATH` | Set to Proton's lib directories |

**Critical insight:** The current `process_linux.go` manually sets `WINEPREFIX`, `WINEFSYNC=1`, and `WINEDLLOVERRIDES=dxgi=n`. With `proton run`, NONE of these should be set -- Proton manages them all. Setting them manually can conflict with Proton's own configuration.

### Prefix Auto-Creation

When `STEAM_COMPAT_DATA_PATH` points to an empty or non-existent directory, Proton's `setup_prefix()` automatically:

1. Creates the directory structure
2. Copies `default_pfx/` template (same source the current `createFromProtonTemplate` uses)
3. Creates `dosdevices/` symlinks (c:, z:, optional s:, t:)
4. Filters registry paths
5. Creates font symlinks
6. Writes `version` file for upgrade tracking
7. Writes `tracked_files` for clean upgrades
8. Calls `os.sync()` for durability

This eliminates 150+ lines of code in `prefix.go` (`createFromProtonTemplate`, `ensureDosdevices`, `copyProtonTemplate`, `isWineLibPath`, `copyFile`) and 30+ lines in `verify.go` (DLL verification).

### Prefix Directory Structure (Proton-managed)

```
~/.cluckers/proton/                    # STEAM_COMPAT_DATA_PATH
  pfx/                                 # Actual Wine prefix (WINEPREFIX)
    drive_c/
      windows/
        system32/                      # 64-bit DLLs (DXVK d3d11.dll, vcruntime140.dll, etc.)
        syswow64/                      # 32-bit DLLs
      users/steamuser/
    dosdevices/
      c: -> ../drive_c
      z: -> /
    system.reg
    user.reg
  version                              # Proton version string (e.g., "GE-Proton10-32")
  config_info                          # Configuration hash
  tracked_files                        # Files managed by Proton
  pfx.lock                             # Concurrent access protection
```

---

## Implementation Plan: Code Changes

### 1. `internal/wine/detect.go` -- Find `proton` Script

**Current:** `FindProtonGE()` returns `ProtonGEInstall{WinePath: ".../files/bin/wine64", ProtonDir: "..."}`.

**Change:** Add `ProtonScript` field to `ProtonGEInstall`:

```go
type ProtonGEInstall struct {
    WinePath     string // Full path to wine64 binary (kept for fallback/system wine)
    ProtonDir    string // Root of Proton-GE installation
    ProtonScript string // Full path to the proton Python script
}
```

Populate it in `FindProtonGE()`:
```go
protonScript := filepath.Join(protonDir, "proton")
if _, err := os.Stat(protonScript); err == nil {
    install.ProtonScript = protonScript
}
```

Update `FindWine()` to also return the proton script path (or add a new `FindProton()` function that the pipeline calls).

### 2. `internal/launch/process.go` -- Add Proton Config Fields

```go
type LaunchConfig struct {
    // Existing fields (keep for Windows and system Wine fallback):
    WinePath         string
    WinePrefix       string
    GameDir          string
    Username         string
    AccessToken      string
    OIDCTokenPath    string
    ContentBootstrap []byte
    HostX            string
    Verbose          bool

    // New fields for Proton launch:
    ProtonScript     string // Path to proton Python script (empty = use direct Wine)
    ProtonCompatData string // STEAM_COMPAT_DATA_PATH
    SteamInstallPath string // STEAM_COMPAT_CLIENT_INSTALL_PATH
}
```

### 3. `internal/launch/process_linux.go` -- Switch Launch Method

```go
func LaunchGame(ctx context.Context, cfg *LaunchConfig) error {
    if cfg.ProtonScript != "" {
        return launchViaProton(ctx, cfg)
    }
    return launchViaWine(ctx, cfg) // Current code, kept as fallback for system Wine
}

func launchViaProton(ctx context.Context, cfg *LaunchConfig) error {
    gameExe := game.GameExePath(cfg.GameDir)
    // ... validate gameExe exists ...

    // Build game args (same as current)
    gameArgs := []string{
        fmt.Sprintf("-user=%s", cfg.Username),
        // ... etc ...
    }

    var args []string
    if cfg.ContentBootstrap != nil && len(cfg.ContentBootstrap) > 0 {
        shmPath, shmCleanup, err := ExtractSHMLauncher()
        // ... same temp file setup ...

        // Key difference: proton waitforexitandrun <exe> [args]
        // Proton's wine handles the path conversion internally
        args = append(args,
            cfg.ProtonScript,
            "waitforexitandrun",
            shmPath,                              // shm_launcher.exe (Linux path -- proton converts)
            wine.LinuxToWinePath(bootstrapPath),   // Bootstrap file (Wine path for shm_launcher args)
            shmName,                               // SHM name
            wine.LinuxToWinePath(gameExe),         // Game exe (Wine path for shm_launcher args)
        )
        args = append(args, gameArgs...)
    } else {
        args = append(args,
            cfg.ProtonScript,
            "waitforexitandrun",
            gameExe,
        )
        args = append(args, gameArgs...)
    }

    // Environment: ONLY set what Proton needs. Do NOT set WINEPREFIX/WINEFSYNC/WINEDLLOVERRIDES.
    env := cleanAppImageEnv()  // Strip LD_LIBRARY_PATH from AppImage
    env = append(env,
        "STEAM_COMPAT_DATA_PATH="+cfg.ProtonCompatData,
        "STEAM_COMPAT_CLIENT_INSTALL_PATH="+cfg.SteamInstallPath,
        "SteamGameId=0",
        "SteamAppId=0",
    )

    // Optional: game-specific WINEDLLOVERRIDES if needed
    // Test first without any overrides -- Proton's defaults may be sufficient
    // env = append(env, "WINEDLLOVERRIDES=dxgi=n")

    cmd := exec.CommandContext(ctx, args[0], args[1:]...)
    cmd.Env = env
    cmd.Dir = cfg.GameDir
    // ... same stdout/stderr handling ...

    return cmd.Run()
}
```

### 4. `internal/wine/prefix.go` -- Simplify for Proton

**Remove** (Proton handles these):
- `createFromProtonTemplate()`
- `ensureDosdevices()`
- `copyProtonTemplate()`
- `isWineLibPath()`
- `copyFile()`
- `runWineboot()` (for Proton path only)

**Keep** (for system Wine fallback):
- `createWithWinetricks()` -- still needed if user has only system Wine

**Add:**
```go
// ProtonCompatDataPath returns the default Proton compatdata path.
func ProtonCompatDataPath() string {
    return filepath.Join(config.DataDir(), "proton")
}

// EnsureProtonCompatData ensures the compatdata directory exists.
// Proton will auto-populate it on first run.
func EnsureProtonCompatData() (string, error) {
    path := ProtonCompatDataPath()
    if err := os.MkdirAll(path, 0755); err != nil {
        return "", fmt.Errorf("create proton compatdata dir: %w", err)
    }
    return path, nil
}
```

### 5. `internal/launch/pipeline_linux.go` -- Simplify Steps

```go
func platformSteps(_ *LaunchState) []Step {
    return []Step{
        {Name: "Detecting Proton", Fn: stepDetectProton},
        {Name: "Preparing prefix", Fn: stepPreparePrefix},
        // Remove: "Verifying Wine prefix" -- Proton manages DLLs
    }
}

func stepDetectProton(_ context.Context, state *LaunchState) error {
    // Try to find proton script first
    protonScript, protonDir, err := wine.FindProton(state.Config.WinePath)
    if err != nil {
        // Fall back to direct Wine (system wine without proton script)
        winePath, wineErr := wine.FindWine(state.Config.WinePath)
        if wineErr != nil {
            return wineErr
        }
        state.WinePath = winePath
        return nil
    }
    state.ProtonScript = protonScript
    state.ProtonDir = protonDir
    return nil
}

func stepPreparePrefix(_ context.Context, state *LaunchState) error {
    if state.ProtonScript != "" {
        // Proton manages its own prefix -- just ensure the directory exists
        compatData, err := wine.EnsureProtonCompatData()
        if err != nil {
            return err
        }
        state.ProtonCompatData = compatData
        return nil
    }
    // System Wine fallback: use existing prefix logic
    // ... current stepEnsurePrefix + stepVerifyPrefix logic ...
    return nil
}
```

---

## STEAM_COMPAT_CLIENT_INSTALL_PATH: The Steam Dependency Problem

### The Problem

The `proton` Python script requires `STEAM_COMPAT_CLIENT_INSTALL_PATH` and raises a `KeyError` if it is missing. This variable should point to the Steam client installation directory, which is used to locate:
- `steamclient.so` (native Steam API library)
- Legacy compatibility DLLs
- Steam overlay components

### Impact on Cluckers

Cluckers targets three scenarios:

| Scenario | Steam Installed? | Solution |
|----------|-----------------|----------|
| Steam Deck (Game Mode or Desktop) | **YES** (always) | Use real path: `~/.local/share/Steam/` |
| Desktop Linux with Steam | **YES** | Use real path: auto-detect Steam install |
| AppImage / tarball without Steam | **MAYBE NOT** | Use dummy path (see below) |

### Solution: Auto-detect with Dummy Fallback

```go
func FindSteamInstallPath() string {
    home := userHome()
    candidates := []string{
        filepath.Join(home, ".local", "share", "Steam"),
        filepath.Join(home, ".steam", "steam"),
        filepath.Join(home, ".steam", "root"),
        filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", "data", "Steam"), // Flatpak
        filepath.Join(home, "snap", "steam", "common", ".steam", "steam"),              // Snap
    }
    for _, path := range candidates {
        if info, err := os.Stat(path); err == nil && info.IsDir() {
            return path
        }
    }
    // Steam not found -- return the compatdata path itself as a dummy.
    // Proton uses this for steamclient.so which we don't need (no Steam DRM).
    // The proton script will not crash; it will just skip Steam overlay integration.
    return ProtonCompatDataPath()
}
```

**Why a dummy path works:** Cluckers' game (Realm Royale on Project Crown) does not use Steam DRM, Steam overlay, or Steam API. The `steamclient.so` that Proton tries to load is only needed for Steamworks integration. Without it, Proton logs a warning but continues to launch the game. The game itself does not call any Steam API functions.

**Confidence:** MEDIUM -- This needs testing. The proton script may fail hard if `steamclient.so` is not found at the path. If so, the fallback is to create a minimal dummy directory structure or use `umu-launcher` as a thin wrapper.

---

## Gamescope Integration (Steam Deck Controller Fix)

### Why This Matters

The v1.1 milestone exists because of the Steam Deck controller bug: during UE3 ServerTravel (lobby-to-match transition), Gamescope loses track of the game window, and Steam Input switches the controller to desktop mode. The fix is to launch through Proton so Gamescope properly tracks the game as a "Steam game" with the `STEAM_GAME` X window property.

### What Proton Does for Gamescope

When running under Gamescope (Steam Deck Game Mode), Proton's internal `steam.exe` stub:
1. Sets the `STEAM_GAME` X window property on the game window
2. Communicates with Gamescope via `GAMESCOPECTRL_BASELAYER_APPID`
3. Ensures the game window gets proper focus priority
4. Keeps Steam Input in game mode (not desktop mode) during window transitions

### What Cluckers Needs to Do

**For Steam Deck (launched from Steam as non-Steam game):**
- Set `SteamGameId` to a consistent numeric value (the non-Steam game's app ID from Steam)
- Gamescope is already running (Steam Deck Game Mode provides it)
- Proton handles the rest

**For Desktop Linux (not in Gamescope):**
- No special Gamescope handling needed
- `SteamGameId=0` is sufficient

**Optional future enhancement:** Wrap the launch in `gamescope` for desktop Linux users:
```bash
gamescope -W 1920 -H 1080 -r 60 -- proton waitforexitandrun game.exe
```
This is NOT needed for v1.1. Focus on the Proton switch first.

---

## shm_launcher.exe Under Proton: Compatibility

### Current Flow (Wine)
```
wine64 shm_launcher.exe Z:\tmp\bootstrap.bin "Local\realm..." Z:\path\to\game.exe [game_args]
```

### New Flow (Proton)
```
proton waitforexitandrun shm_launcher.exe Z:\tmp\bootstrap.bin "Local\realm..." Z:\path\to\game.exe [game_args]
```

**Why this works:** `proton waitforexitandrun` passes its arguments to `wine64` internally. The `shm_launcher.exe` is a standard Win32 executable using `CreateFileMappingW` and `CreateProcessW`. These are core Win32 APIs that Wine has supported for decades. Proton is Wine with additional components (DXVK, vkd3d-proton, lsteamclient); the core Win32 API layer is identical.

The `simshmbridge` project (GitHub: Spacefreak18/simshmbridge) explicitly creates shared memory mapped files for use between Linux and Wine/Proton, confirming that `CreateFileMappingW`/`OpenFileMapping` works correctly under Proton.

**Path handling note:** The `proton` script's first argument after the verb is a Linux path to the executable. Proton converts it internally. However, the ARGUMENTS to that executable are passed through as-is to the Windows process. So `Z:\path\to\bootstrap.bin` (Wine path) is correct for the shm_launcher arguments, while the shm_launcher.exe path itself should be a Linux path.

**Testing needed:** Verify whether `proton waitforexitandrun` accepts a Linux path for the executable (it should -- Proton converts via `getcompatpath` internally) or requires a Wine path. If it requires a Wine path, use `wine.LinuxToWinePath(shmPath)`.

**Confidence:** HIGH for core compatibility. MEDIUM for exact path handling (needs testing).

---

## AppImage Bundling Considerations

### Current AppImage Structure
```
Cluckers.AppImage
  AppRun
  cluckers (Go binary)
  proton-ge/                          # Bundled GE-Proton
    files/bin/wine64                   # Current launch target
    default_pfx/
    proton                             # <-- Already present, just not used
```

### Changes for v1.1
The `proton` script is already bundled inside the AppImage's Proton-GE directory. The current code uses `CLUCKERS_BUNDLED_PROTON` env var (set by AppRun) to locate the bundled Proton-GE:

```go
if bundled := os.Getenv("CLUCKERS_BUNDLED_PROTON"); bundled != "" {
    winePath := filepath.Join(bundled, "files", "bin", "wine64")
```

Change to also check for the proton script:
```go
if bundled := os.Getenv("CLUCKERS_BUNDLED_PROTON"); bundled != "" {
    protonScript := filepath.Join(bundled, "proton")
    if _, err := os.Stat(protonScript); err == nil {
        return protonScript, nil
    }
}
```

### Python Dependency in AppImage

The `proton` script requires Python 3. On Steam Deck and most desktop Linux, Python 3 is pre-installed. In the AppImage context:

| Scenario | Python Available? | Action |
|----------|------------------|--------|
| Steam Deck | YES (system Python) | Works |
| Desktop Linux (Arch, Ubuntu, Fedora) | YES (system Python) | Works |
| Minimal server/container | MAYBE NOT | Detect and warn |

**Risk:** If the AppImage strips `LD_LIBRARY_PATH` (which it currently does for Wine), ensure this does not break Python. Python is a native Linux executable, not a Wine process, so the AppImage `LD_LIBRARY_PATH` cleanup should not affect it.

**Mitigation:** Test the AppImage launch on a clean Steam Deck and desktop Linux. If Python is not found, fall back to direct Wine launch (current behavior).

---

## Recommended Stack (Changes Only)

### No New Go Dependencies

The entire Proton launch pipeline change requires zero new Go imports. It uses:
- `os/exec` (stdlib) -- already used
- `os` (stdlib) -- already used
- `path/filepath` (stdlib) -- already used
- `strings` (stdlib) -- already used

### System Dependencies

| Dependency | Version | Required By | Notes |
|------------|---------|-------------|-------|
| Python 3 | 3.8+ | `proton` script | Pre-installed on all target platforms (SteamOS, Arch, Ubuntu, Fedora). NOT a new dependency -- it was always required by Proton-GE but was not invoked by the launcher. |
| Proton-GE | GE-Proton9+ | Game launch | Already detected and used. Now using the `proton` script instead of `wine64` directly. |

### Removed System Dependencies (for Proton path)

| Dependency | Why Removed |
|------------|-------------|
| winetricks | Proton bundles all DLLs (DXVK, vcruntime140, msvcp140, d3dx11_43). No winetricks needed. |
| Manual DXVK setup | Proton bundles and configures DXVK automatically. |
| Manual vcrun2022 | Proton includes Visual C++ runtime DLLs in its `default_pfx` template. |

---

## Alternatives Considered

| Approach | Recommended | Why/Why Not |
|----------|-------------|-------------|
| `proton waitforexitandrun` | **YES** | Official entry point. Handles prefix setup, DLL management, WINEFSYNC, DXVK -- everything. Matches what Steam does. |
| `proton run` | No | Does not wait for game exit. Current `cmd.Run()` blocks until exit. `waitforexitandrun` matches this behavior. |
| Direct `wine64` with Proton env vars | No | Bypasses Proton's prefix setup, DLL management, and compatibility patches. Would need to replicate all of Proton's environment setup manually. |
| `umu-launcher` / `umu-run` | No (for now) | Adds a system dependency (umu-launcher package). Runs Proton inside Steam Runtime container. Overkill for our use case -- we already bundle Proton-GE and manage our own prefix. **Reconsider if STEAM_COMPAT_CLIENT_INSTALL_PATH causes problems without Steam.** |
| Keep direct Wine, add Gamescope wrapper | No | Does not fix the controller issue. Gamescope needs Proton's `steam.exe` stub to set `STEAM_GAME` window property. Direct Wine does not set this. |

---

## Risk Assessment

| Risk | Severity | Likelihood | Mitigation |
|------|----------|------------|------------|
| `STEAM_COMPAT_CLIENT_INSTALL_PATH` fails without Steam | Medium | Medium | Auto-detect Steam, fall back to dummy path. Test on systems without Steam. |
| `proton` Python script not found (corrupt install) | Low | Low | Already handle corrupt Proton installs in current code. Add specific check for `proton` file. |
| Python not available in AppImage context | Low | Very Low | Python is pre-installed on all target platforms. Add detection and fallback to direct Wine. |
| shm_launcher.exe path handling differs | Low | Medium | Test both Linux path and Wine Z: path. Document which works. |
| Proton sets conflicting WINEDLLOVERRIDES | Medium | Low | Do not set `WINEDLLOVERRIDES` when using Proton. If game-specific override needed, use `PROTON_*` env vars. |
| Old prefix at `~/.cluckers/prefix/` confuses users | Low | Medium | Log deprecation warning. Do not auto-delete. Document migration in release notes. |

---

## Sources

- [Proton source (proton_9.0 branch)](https://github.com/ValveSoftware/Proton/blob/proton_9.0/proton) -- Python script, env vars, prefix setup (HIGH confidence)
- [GE-Proton source (master)](https://raw.githubusercontent.com/GloriousEggroll/proton-ge-custom/master/proton) -- GE-specific additions, same interface (HIGH confidence)
- [Proton Wine Prefix Management (DeepWiki)](https://deepwiki.com/ValveSoftware/Proton/2.2-wine-prefix-management) -- CompatData class, prefix structure, auto-creation (HIGH confidence)
- [Proton Architecture Overview (DeepWiki)](https://deepwiki.com/ValveSoftware/Proton) -- Component architecture, env var requirements (HIGH confidence)
- [Running Proton Outside Steam](https://megadarken.github.io/software/2024/08/05/proton-outside-steam.html) -- CLI invocation, env vars (MEDIUM confidence)
- [Launch Games via Proton CLI (GitHub Gist)](https://gist.github.com/sxiii/6b5cd2e7d2321df876730f8cafa12b2e) -- Practical examples (MEDIUM confidence)
- [Run .exe in Proton Prefix (GitHub Gist)](https://gist.github.com/michaelbutler/f364276f4030c5f449252f2c4d960bd2) -- Path handling, WINEPREFIX mapping (MEDIUM confidence)
- [Non-Steam Game Focus Issue (#8513)](https://github.com/ValveSoftware/steam-for-linux/issues/8513) -- SteamGameId, STEAM_GAME property, Gamescope focus (HIGH confidence)
- [umu-launcher](https://github.com/Open-Wine-Components/umu-launcher) -- Alternative approach, Steam-free Proton usage (HIGH confidence)
- [Gamescope (ArchWiki)](https://wiki.archlinux.org/title/Gamescope) -- Compositor options, Steam Deck integration (HIGH confidence)
- [simshmbridge](https://github.com/Spacefreak18/simshmbridge) -- CreateFileMappingW works under Wine/Proton (MEDIUM confidence)
- [STEAM_COMPAT_CLIENT_INSTALL_PATH KeyError](https://github.com/ValveSoftware/Proton/issues/9068) -- Required var, fails if missing (HIGH confidence)
- [Proton DXVK Bundling](https://dxvk.org/why-use-dxvk-with-proton/) -- DXVK included in Proton automatically (HIGH confidence)
- [GE-Proton vs Valve Proton](https://www.howtogeek.com/proton-vs-proton-ge-whats-the-difference-and-which-one-should-you-use/) -- Same interface, different patches (MEDIUM confidence)

---
*Stack research for: Proton launch pipeline (v1.1 milestone)*
*Researched: 2026-02-24*
