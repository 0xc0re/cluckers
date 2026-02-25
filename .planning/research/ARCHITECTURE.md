# Architecture: Proton Launch Pipeline Integration

**Domain:** Game launcher -- Wine-to-Proton migration for Linux game execution
**Researched:** 2026-02-24
**Confidence:** HIGH (verified against actual proton script source in build/appimage/, existing codebase, and official Proton/umu documentation)

## Executive Summary

Switching from direct Wine execution (`wine64 shm_launcher.exe ...`) to the Proton launch pipeline (`proton run shm_launcher.exe ...`) fundamentally changes how the launcher invokes the game on Linux. The core change is small in code surface area but deep in architectural implications: prefix management moves from the launcher to Proton, environment setup becomes Proton's responsibility, and the invocation model shifts from `exec(wine64, args)` to `exec(python3, proton, run, args)`.

The shm_launcher.exe helper requires **no changes** -- it is a Win32 binary that runs identically whether Wine executes it directly or Proton's embedded Wine executes it. The critical integration point is providing the correct environment variables that Proton expects (`STEAM_COMPAT_DATA_PATH` is mandatory) while preserving the launcher's existing temp file and shared memory naming patterns.

Two viable approaches exist: **Direct Proton invocation** (simpler, what the codebase already partially supports since it bundles Proton-GE in the AppImage) or **umu-run** (adds a dependency but provides Steam Runtime container isolation and automatic protonfixes). The recommendation is **direct Proton invocation** because the launcher already bundles Proton-GE and controls the entire environment.

## Current Architecture (Direct Wine)

### Data Flow: v1.0 (Current)

```
Go launcher
  |
  +--> FindWine() --> wine64 path (Proton-GE/files/bin/wine64 or system wine)
  |
  +--> CreatePrefix() --> ~/.cluckers/prefix/
  |      |-- Proton-GE: copy default_pfx template + wineboot --init
  |      +-- System Wine: wineboot --init + winetricks vcrun2022 d3dx11_43 dxvk
  |
  +--> VerifyPrefix() --> check 4 DLLs (vcruntime140, msvcp140, d3dx11_43, d3d11)
  |
  +--> ExtractSHMLauncher() --> /tmp/shm_launcher_*.exe
  +--> WriteBootstrapFile() --> /tmp/realm_bootstrap_*.bin
  |
  +--> exec.Command(
  |      wine64,                              # direct Wine binary
  |      /tmp/shm_launcher_*.exe,             # Win32 SHM helper
  |      Z:\tmp\realm_bootstrap_*.bin,         # bootstrap path (Wine Z: drive)
  |      Local\realm_content_bootstrap_<pid>,  # SHM name
  |      Z:\path\to\game.exe,                 # game binary (Wine Z: drive)
  |      -user=X, -token=X, ...               # game args
  |    )
  |    env: WINEPREFIX, WINEFSYNC=1, WINEDLLOVERRIDES=dxgi=n
  |
  +--> Blocks until game exits, tees stderr to /tmp/cluckers_wine.log
```

### Key Files

| File | Role | Changes Needed |
|------|------|----------------|
| `internal/wine/detect.go` | Finds Proton-GE/system Wine | **MODIFY**: Add `FindProton()` that returns the Proton root (not just wine64 binary) |
| `internal/wine/prefix.go` | Creates Wine prefix from template | **REPLACE**: Proton manages its own prefix via `setup_prefix()` |
| `internal/wine/verify.go` | Checks 4 DLLs exist | **REMOVE/SIMPLIFY**: Proton handles all DLL deployment |
| `internal/launch/pipeline_linux.go` | Platform steps: detect Wine, ensure prefix, verify prefix | **REWRITE**: New steps for Proton detection, compat data setup |
| `internal/launch/process_linux.go` | Builds `exec.Command(wine64, ...)` | **REWRITE**: Builds `exec.Command(python3, proton, run, ...)` |
| `internal/launch/pipeline.go` | Shared pipeline + `LaunchState` | **MODIFY**: Add `ProtonDir` and `CompatDataPath` to `LaunchState` |
| `internal/launch/process.go` | `LaunchConfig` struct | **MODIFY**: Add `ProtonDir` field, remove `WinePrefix` (Proton manages it) |
| `internal/config/config.go` | `Config` struct | **MODIFY**: Add `ProtonDir` field, deprecate `WinePrefix` |
| `deploy/AppRun` | Sets `CLUCKERS_BUNDLED_PROTON` env var | **KEEP AS-IS**: Already points to Proton root |
| `internal/launch/shm.go` | Extracts shm_launcher.exe to temp | **NO CHANGE** |
| `tools/shm_launcher.c` | Win32 SHM helper source | **NO CHANGE** |
| `assets/embed.go` | Embeds shm_launcher.exe | **NO CHANGE** |
| `scripts/build-appimage.sh` | Downloads Proton-GE, builds AppImage | **NO CHANGE** (already bundles full Proton-GE) |

## Target Architecture (Proton Launch Pipeline)

### Data Flow: v1.1 (Proton)

```
Go launcher
  |
  +--> FindProton() --> Proton root directory (contains proton script + files/)
  |      Priority: CLUCKERS_BUNDLED_PROTON > config override > system Proton-GE
  |
  +--> EnsureCompatData() --> ~/.cluckers/compatdata/
  |      Just mkdir. Proton's setup_prefix() handles everything else on first run.
  |
  +--> ExtractSHMLauncher() --> /tmp/shm_launcher_*.exe  [UNCHANGED]
  +--> WriteBootstrapFile() --> /tmp/realm_bootstrap_*.bin [UNCHANGED]
  |
  +--> exec.Command(
  |      python3,                                    # Proton script is Python
  |      /path/to/proton/proton,                     # The proton script
  |      run,                                        # Verb: "run"
  |      /tmp/shm_launcher_*.exe,                    # Win32 SHM helper (Unix path)
  |      Z:\tmp\realm_bootstrap_*.bin,               # bootstrap (Wine Z: drive path)
  |      Local\realm_content_bootstrap_<pid>,        # SHM name
  |      Z:\path\to\game.exe,                        # game binary (Wine Z: drive path)
  |      -user=X, -token=X, ...                      # game args
  |    )
  |    env:
  |      STEAM_COMPAT_DATA_PATH=~/.cluckers/compatdata   # REQUIRED by proton script
  |      STEAM_COMPAT_CLIENT_INSTALL_PATH=<steam path>   # Optional, for Steam DLLs
  |      UMU_ID=0                                         # Tells proton: non-Steam game
  |      SteamGameId=0                                    # Tells proton: non-Steam game
  |      SteamAppId=0                                     # Avoids game-specific hacks
  |      WINEDLLOVERRIDES=dxgi=n                          # Keep existing DX override
  |
  +--> Blocks until proton script exits (which blocks until game exits)
  +--> Tees stderr to /tmp/cluckers_wine.log
```

### Critical Detail: UMU_ID Path in Proton Script

Reading the actual Proton script (GE-Proton10-32, line 2309), when `UMU_ID` is set in the environment, Proton's `run()` method takes a special code path:

```python
elif "UMU_ID" in os.environ or os.environ.get("SteamGameId", 0) in ["3347400"]:
    log(sys.argv[2])
    if os.environ.get("UMU_USE_STEAM", "0") == "1":
        # ... Steam DLL path
    elif len(sys.argv) >= 3 and sys.argv[2].startswith('/'):
        log("Executable a unix path, launching with /unix option.")
        argv = [g_proton.wine64_bin, "c:\\windows\\system32\\start.exe", "/unix"]
    else:
        log("Executable is inside wine prefix, launching normally.")
        argv = [g_proton.wine64_bin]
```

This is exactly the path we want. When `UMU_ID` is set and the executable path starts with `/` (Unix path), Proton uses `start.exe /unix` to resolve the path. For temp file paths like `/tmp/shm_launcher_*.exe`, Proton will correctly translate them to Wine-accessible paths.

**IMPORTANT NUANCE**: The shm_launcher.exe path passed to `proton run` is a **Unix path** (e.g., `/tmp/shm_launcher_12345.exe`). Proton's `start.exe /unix` handles the path translation. But the **arguments** passed to shm_launcher.exe are received by the Win32 process, which means:
- The bootstrap file path should be a Wine path (`Z:\tmp\...`) because shm_launcher.exe reads it with `CreateFileW`
- The game exe path should be a Wine path (`Z:\path\to\...`) because shm_launcher.exe passes it to `CreateProcessW`
- The SHM name is a pure Win32 name (`Local\realm_content_bootstrap_<pid>`) -- no path translation needed

This matches the current behavior in `process_linux.go` where `wine.LinuxToWinePath()` converts paths for arguments.

### Key Difference: What the Go Launcher No Longer Does

| Responsibility | v1.0 (Direct Wine) | v1.1 (Proton) |
|----------------|---------------------|---------------|
| Find Wine binary | Go: `FindWine()` | Go: `FindProton()` returns Proton dir |
| Create prefix | Go: `CreatePrefix()` (copy template, wineboot, winetricks) | Proton: `setup_prefix()` does everything |
| Install vcrun2022/d3dx11/DXVK | Go: winetricks or template copy | Proton: automatic via `update_builtin_libs()` |
| Verify DLLs present | Go: `VerifyPrefix()` checks 4 DLLs | Proton: handles automatically |
| Set WINEPREFIX | Go: passes env var | Proton: sets from STEAM_COMPAT_DATA_PATH + `/pfx/` |
| Set WINEFSYNC/WINEESYNC | Go: checks `IsProtonGE()` | Proton: always enables both by default |
| Set WINEDLLOVERRIDES | Go: `dxgi=n` | Go: still passes `dxgi=n` (Proton appends its own) |
| LD_LIBRARY_PATH management | Go: strips AppImage paths | Proton: prepends its own lib paths |
| Launch process | Go: `exec(wine64, args)` | Go: `exec(python3, proton, run, args)` |

## Component Changes (Detailed)

### 1. `internal/launch/pipeline.go` -- LaunchState Extension

```go
type LaunchState struct {
    // ... existing fields ...
    WinePath      string  // DEPRECATE: No longer used for Proton path
    PrefixPath    string  // DEPRECATE: Proton manages prefix

    // NEW fields
    ProtonDir      string  // Root of Proton installation (contains proton script + files/)
    CompatDataPath string  // Path to STEAM_COMPAT_DATA_PATH dir
}
```

### 2. `internal/launch/process.go` -- LaunchConfig Extension

```go
type LaunchConfig struct {
    // REMOVE or keep for fallback
    WinePath         string  // Only used for system Wine fallback
    WinePrefix       string  // Only used for system Wine fallback

    // NEW
    ProtonDir        string  // Proton root directory
    CompatDataPath   string  // STEAM_COMPAT_DATA_PATH

    // UNCHANGED
    GameDir          string
    Username         string
    AccessToken      string
    OIDCTokenPath    string
    ContentBootstrap []byte
    HostX            string
    Verbose          bool
}
```

### 3. `internal/launch/pipeline_linux.go` -- Simplified Steps

Current steps:
1. "Detecting Wine" -- `stepDetectWine`
2. "Ensuring Wine prefix" -- `stepEnsurePrefix`
3. "Verifying Wine prefix" -- `stepVerifyPrefix`

New steps:
1. "Detecting Proton" -- `stepDetectProton` (finds Proton dir via bundled/system)
2. "Preparing compatibility data" -- `stepPrepareCompatData` (mkdir only; Proton creates prefix on first `proton run`)

**Two steps removed.** The Proton script handles prefix creation and DLL verification internally during `proton run`.

```go
func platformSteps(_ *LaunchState) []Step {
    return []Step{
        {Name: "Detecting Proton", Fn: stepDetectProton},
        {Name: "Preparing compatibility data", Fn: stepPrepareCompatData},
    }
}
```

### 4. `internal/launch/process_linux.go` -- New Launch Logic

The core change: instead of building the command with wine64 as the binary, build it with python3 invoking the proton script.

```go
func LaunchGame(ctx context.Context, cfg *LaunchConfig) error {
    // ... validate game exe (unchanged) ...
    // ... extract shm_launcher + bootstrap (unchanged) ...

    protonScript := filepath.Join(cfg.ProtonDir, "proton")

    // Build args: python3 <proton> run <shm_launcher.exe> <bootstrap_path> <shm_name> <game.exe> <game_args...>
    // Note: shm_launcher path is Unix (proton's start.exe /unix translates it)
    // But args TO shm_launcher are Wine paths (it uses CreateFileW/CreateProcessW)
    args := []string{
        protonScript,
        "run",
        shmPath,                                   // Unix path -- proton handles it
        wine.LinuxToWinePath(bootstrapPath),        // Wine path -- shm_launcher uses CreateFileW
        shmName,                                    // Pure Win32 name
        wine.LinuxToWinePath(gameExe),              // Wine path -- shm_launcher uses CreateProcessW
    }
    args = append(args, gameArgs...)

    env := buildProtonEnv(cfg)

    cmd := exec.CommandContext(ctx, "python3", args...)
    cmd.Env = env
    cmd.Dir = cfg.GameDir
    // ... stdout/stderr handling (unchanged) ...
}
```

### 5. `internal/wine/detect.go` -- New FindProton Function

The existing `FindProtonGE()` returns `ProtonGEInstall{WinePath, ProtonDir}`. The `ProtonDir` field already contains exactly what we need. Add a new top-level function:

```go
// FindProton locates a Proton installation suitable for proton run.
// Priority: CLUCKERS_BUNDLED_PROTON > config override > system Proton-GE
// Returns the Proton root directory (containing the proton script and files/).
func FindProton(configOverride string) (string, error) {
    // 1. Config override (direct Proton dir path)
    if configOverride != "" { ... }

    // 2. Bundled Proton (AppImage)
    if bundled := os.Getenv("CLUCKERS_BUNDLED_PROTON"); bundled != "" {
        if protonScriptExists(bundled) {
            return bundled, nil
        }
    }

    // 3. System Proton-GE installations
    installs := FindProtonGE(userHome())
    if len(installs) > 0 {
        return installs[0].ProtonDir, nil
    }

    // 4. Error with distro-specific instructions
    return "", &ui.UserError{...}
}
```

The existing `FindWine()` function and all its Wine-specific logic remains for backward compatibility (system Wine fallback path, if we decide to keep it).

### 6. `internal/wine/prefix.go` -- Simplified or Removed

The entire `CreatePrefix()`, `createFromProtonTemplate()`, `createWithWinetricks()`, `runWineboot()`, `copyProtonTemplate()`, and `ensureDosdevices()` functions become **dead code** for Proton launches. Proton's `setup_prefix()` method handles:

- Copying `default_pfx` template
- Creating `dosdevices/c:` and `dosdevices/z:` symlinks
- Running wineboot
- Installing/updating DLLs (vcruntime, DXVK, d3dx, etc.)
- Version tracking and upgrades

**Recommendation**: Keep the existing prefix code guarded behind a `UseProton` flag for system Wine fallback, but the primary path should skip all of it.

### 7. `internal/wine/verify.go` -- Simplified

Prefix verification becomes unnecessary with Proton. Proton's `setup_prefix()` runs on every launch and ensures all DLLs are correct. The `VerifyPrefix()` function can remain for the system Wine fallback path.

### 8. `internal/config/config.go` -- New Fields

```go
type Config struct {
    Gateway    string `mapstructure:"gateway"`
    WinePath   string `mapstructure:"wine_path"`      // DEPRECATED for Proton
    WinePrefix string `mapstructure:"wine_prefix"`     // DEPRECATED for Proton
    ProtonDir  string `mapstructure:"proton_dir"`      // NEW: override Proton location
    GameDir    string `mapstructure:"game_dir"`
    HostX      string `mapstructure:"hostx"`
    Verbose    bool   `mapstructure:"verbose"`
}
```

### 9. `deploy/AppRun` -- No Change

The current AppRun already sets `CLUCKERS_BUNDLED_PROTON` to the Proton root directory. The new `FindProton()` function reads this env var. No AppRun changes needed.

### 10. `scripts/build-appimage.sh` -- No Change

Already downloads full Proton-GE, copies it to `$APPDIR/proton/`, and the AppRun exposes it. The `proton` Python script is already bundled.

## The shm_launcher.exe Question

**Does shm_launcher.exe work under Proton?** YES, with zero changes.

The shm_launcher.exe is a Win32 program that:
1. Reads a file with `CreateFileW()` -- works in any Wine/Proton environment
2. Creates named shared memory with `CreateFileMappingW(INVALID_HANDLE_VALUE, ...)` -- standard Win32 API, works in Wine/Proton
3. Launches the game with `CreateProcessW()` -- works in any Wine/Proton environment
4. Waits for the game with `WaitForSingleObject()` -- works in any Wine/Proton environment

All of these are standard Win32 APIs that Wine has implemented for decades. Proton uses Wine internally, so shm_launcher.exe sees the exact same Windows API surface whether launched via direct Wine or via Proton.

**The only difference**: Proton's `run()` method wraps the invocation through `start.exe /unix` when the executable path is a Unix path and `UMU_ID` is set. This means:
- `proton run /tmp/shm_launcher_12345.exe Z:\tmp\bootstrap.bin ...`
- Proton translates to: `wine64 c:\windows\system32\start.exe /unix /tmp/shm_launcher_12345.exe Z:\tmp\bootstrap.bin ...`
- `start.exe` resolves the Unix path and launches shm_launcher.exe in the Wine environment
- shm_launcher.exe receives its arguments as-is (Wine-format paths for bootstrap and game exe)

## Environment Variable Matrix

### Required for Proton Script

| Variable | Value | Why | Set By |
|----------|-------|-----|--------|
| `STEAM_COMPAT_DATA_PATH` | `~/.cluckers/compatdata` | Proton stores prefix at `$STEAM_COMPAT_DATA_PATH/pfx/` | Go launcher |
| `UMU_ID` | `0` | Signals non-Steam game to Proton; enables Unix path handling | Go launcher |
| `SteamGameId` | `0` | Prevents game-specific workarounds from triggering | Go launcher |
| `SteamAppId` | `0` | Prevents game-specific workarounds from triggering | Go launcher |

### Optional / Inherited

| Variable | Value | Why | Set By |
|----------|-------|-----|--------|
| `STEAM_COMPAT_CLIENT_INSTALL_PATH` | `~/.steam/steam` or empty | Proton copies Steam client DLLs if present; not needed for our game | Go launcher (best-effort) |
| `WINEDLLOVERRIDES` | `dxgi=n` | Game-specific DX override (Proton appends its own overrides) | Go launcher |
| `PROTON_LOG` | `1` (verbose mode) | Enables Wine debug logging | Go launcher (optional) |

### Set Automatically by Proton (DO NOT set)

| Variable | Proton Sets It To | Notes |
|----------|-------------------|-------|
| `WINEPREFIX` | `$STEAM_COMPAT_DATA_PATH/pfx/` | Proton computes this from STEAM_COMPAT_DATA_PATH |
| `WINEFSYNC` | `1` | Enabled by default |
| `WINEESYNC` | `1` | Enabled by default |
| `WINENTSYNC` | `1` | Enabled by default (new in GE-Proton10+) |
| `WINEDEBUG` | `-all` | Performance default |
| `LD_LIBRARY_PATH` | Proton's lib dirs | Proton prepends its own paths |
| `WINEDLLPATH` | Proton's DLL dirs | For DXVK/vkd3d-proton loading |

## Prefix Location Change

### v1.0: `~/.cluckers/prefix/` (Wine prefix directly)

```
~/.cluckers/prefix/
  drive_c/
    windows/
      system32/
  dosdevices/
    c: -> ../drive_c
    z: -> /
  system.reg
  user.reg
```

### v1.1: `~/.cluckers/compatdata/` (Proton compat data)

```
~/.cluckers/compatdata/
  pfx/                    # The actual Wine prefix (Proton creates this)
    drive_c/
      windows/
        system32/
    dosdevices/
    system.reg
    user.reg
  version                 # Proton version tracking file
  config_info             # Proton configuration state
  tracked_files           # Files Proton installed (for clean upgrades)
  pfx.lock                # File lock during prefix operations
```

### Migration Strategy

On first Proton launch:
1. Check if `~/.cluckers/prefix/` exists (old v1.0 prefix)
2. If yes, warn user: "Switching to Proton. Old prefix at ~/.cluckers/prefix/ is no longer used. You can delete it."
3. Create `~/.cluckers/compatdata/` directory (empty)
4. Let Proton's `setup_prefix()` create the new prefix on `proton run`
5. Old prefix is not migrated -- Proton creates a clean one with its own DLL management

This is safe because the game stores no user data in the Wine prefix (it uses the game server for everything).

## Steam Deck Controller: Why Proton Fixes It

The root cause documented in PROJECT.md: "Steam Input reconfigures controller firmware to desktop mode when Gamescope loses track of the game window."

With direct Wine, Gamescope has no awareness that the Wine window belongs to a "game." When UE3 does ServerTravel (lobby-to-match), it destroys and recreates the window. Gamescope sees the window disappear and switches the controller to desktop mode.

With `proton run`:
1. Proton sets `steam-runtime-launcher-interface-0 proton` as the adverb (line 2283-2286)
2. This registers the process with the Steam Runtime, which Gamescope monitors
3. Gamescope maintains the "game mode" controller configuration across window recreation because it tracks the Proton process, not individual windows
4. Even without Steam running, the Proton environment variables (`SteamGameId`, etc.) signal to Gamescope that this is a game session

**Confidence: MEDIUM** -- This is the hypothesis from the PROJECT.md investigation. Actual testing required.

## Patterns to Follow

### Pattern 1: Proton Environment Builder

Centralize Proton environment construction in a single function:

```go
func buildProtonEnv(cfg *LaunchConfig) []string {
    env := os.Environ()

    // Strip LD_LIBRARY_PATH (AppImage contamination)
    env = filterEnv(env, "LD_LIBRARY_PATH")

    // Required Proton variables
    env = append(env,
        "STEAM_COMPAT_DATA_PATH="+cfg.CompatDataPath,
        "UMU_ID=0",
        "SteamGameId=0",
        "SteamAppId=0",
    )

    // Optional: Steam client path for overlay DLLs
    if steamDir := findSteamDir(); steamDir != "" {
        env = append(env, "STEAM_COMPAT_CLIENT_INSTALL_PATH="+steamDir)
    }

    // Game-specific override
    env = append(env, "WINEDLLOVERRIDES=dxgi=n")

    // Verbose logging
    if cfg.Verbose {
        env = append(env, "PROTON_LOG=1")
    }

    return env
}
```

### Pattern 2: Fallback Detection Chain

Keep system Wine as a fallback for users without Proton:

```go
func FindProton(configOverride string) (protonDir string, isProton bool, err error) {
    // Try Proton first
    dir, err := findProtonDir(configOverride)
    if err == nil {
        return dir, true, nil
    }

    // Fall back to system Wine (existing FindWine logic)
    winePath, wineErr := FindWine(configOverride)
    if wineErr == nil {
        return winePath, false, nil  // Signals: use direct Wine path
    }

    return "", false, err  // Both failed
}
```

### Pattern 3: Feature Flag for Gradual Migration

```go
type Config struct {
    // ...
    UseProton bool `mapstructure:"use_proton"`  // Default: true for v1.1
}
```

This allows users to opt out of Proton if something breaks, falling back to direct Wine.

## Anti-Patterns to Avoid

### Anti-Pattern 1: Setting WINEPREFIX Explicitly

**What:** Setting `WINEPREFIX` in the Go launcher when using Proton.
**Why bad:** Proton computes `WINEPREFIX` from `STEAM_COMPAT_DATA_PATH` + `/pfx/`. If the launcher also sets `WINEPREFIX`, it may conflict with Proton's internal state tracking (version file, config_info, tracked_files are all relative to `STEAM_COMPAT_DATA_PATH`, not `WINEPREFIX`).
**Instead:** Only set `STEAM_COMPAT_DATA_PATH` and let Proton derive `WINEPREFIX`.

### Anti-Pattern 2: Running wineboot Separately from Proton

**What:** Running `wineboot --init` before `proton run` to "pre-create" the prefix.
**Why bad:** Proton's `setup_prefix()` does versioned prefix management. If a prefix is created by wineboot outside of Proton, the version file won't exist, and Proton may try to upgrade a "version-less" prefix unpredictably.
**Instead:** Let the first `proton run` create the prefix. Just ensure the `STEAM_COMPAT_DATA_PATH` directory exists (empty).

### Anti-Pattern 3: Mixing Proton and Direct Wine on the Same Prefix

**What:** Using `proton run` with a prefix that was created by direct Wine/winetricks.
**Why bad:** Proton tracks files it installs in `tracked_files`. A prefix created outside Proton has no tracking, so Proton can't clean up or upgrade it properly.
**Instead:** Use separate prefix directories -- `~/.cluckers/prefix/` for direct Wine (legacy), `~/.cluckers/compatdata/` for Proton (new).

### Anti-Pattern 4: Stripping LD_LIBRARY_PATH Before Proton

**What:** Removing LD_LIBRARY_PATH before invoking `proton run` (as current code does before Wine).
**Why bad:** Proton's `init_wine()` prepends its own library paths to LD_LIBRARY_PATH. However, if LD_LIBRARY_PATH is completely empty, Proton still works fine. The risk is that AppImage's LD_LIBRARY_PATH includes libraries that conflict with Proton's Wine.
**Instead:** Strip only AppImage-specific paths, or let Proton override. The current approach (strip everything) is actually safe because Proton rebuilds it.

## Build Order (Implementation Sequence)

### Step 1: Add `FindProton()` to `internal/wine/detect.go`

**Depends on:** Nothing new
**Changes:** Add `FindProton()` function. Keep existing `FindWine()` for fallback.
**Testable:** Unit test with mock directories.

### Step 2: Add Proton fields to `Config` and `LaunchState`

**Depends on:** Step 1
**Changes:** Add `ProtonDir`, `CompatDataPath` to config and state structs. Add `proton_dir` viper default.
**Testable:** Config loading tests.

### Step 3: Create `buildProtonEnv()` in `internal/launch/process_linux.go`

**Depends on:** Step 2
**Changes:** New function that constructs the environment for `proton run`.
**Testable:** Unit test env var construction.

### Step 4: Rewrite `LaunchGame()` in `internal/launch/process_linux.go`

**Depends on:** Steps 1-3
**Changes:** Replace the command construction to use python3 + proton script instead of wine64 directly. Keep shm_launcher extraction and bootstrap file writing unchanged.
**Testable:** Integration test (requires actual Proton install).

### Step 5: Rewrite `platformSteps()` in `internal/launch/pipeline_linux.go`

**Depends on:** Steps 1-4
**Changes:** Replace 3 steps (detect Wine, ensure prefix, verify prefix) with 2 steps (detect Proton, prepare compat data). The "prepare compat data" step is just an `os.MkdirAll`.
**Testable:** Pipeline test with mock state.

### Step 6: Update CLI status command

**Depends on:** Step 1
**Changes:** `status_linux.go` should show Proton version instead of Wine path. Check `proton --version` or read `version` file from Proton dir.
**Testable:** Manual.

### Step 7: Add old prefix deprecation warning

**Depends on:** Step 5
**Changes:** If `~/.cluckers/prefix/` exists, warn user on launch that it's no longer used.
**Testable:** Manual.

### Step 8: Update GUI step names

**Depends on:** Step 5
**Changes:** `StepNames()` returns new step names. GUI step list widget shows "Detecting Proton" instead of "Detecting Wine", etc.
**Testable:** GUI visual check.

## STEAM_COMPAT_CLIENT_INSTALL_PATH Handling

The Proton script's `setup_prefix()` accesses `STEAM_COMPAT_CLIENT_INSTALL_PATH` at line 1081 to copy Steam client DLLs into the prefix. If this variable is not set or points to a non-existent directory, the `try_copy` calls will fail silently (they use try/except). The game does NOT require these Steam client DLLs -- they are for Steam Overlay which is irrelevant for a non-Steam game.

**Recommendation**: Set `STEAM_COMPAT_CLIENT_INSTALL_PATH` best-effort by checking common Steam locations:
```go
func findSteamDir() string {
    home := userHome()
    candidates := []string{
        filepath.Join(home, ".steam", "steam"),
        filepath.Join(home, ".local", "share", "Steam"),
        filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", "data", "Steam"),
    }
    for _, dir := range candidates {
        if _, err := os.Stat(filepath.Join(dir, "steam.sh")); err == nil {
            return dir
        }
    }
    return ""
}
```

If Steam is not installed, leave the variable unset. The `try_get_steam_dir()` function in the Proton script returns `None`, and the Steam DLL copies are silently skipped.

## Scalability Considerations

| Concern | Impact | Approach |
|---------|--------|----------|
| First launch time | Proton's `setup_prefix()` takes 5-15 seconds on first run (copies default_pfx, installs DLLs) | Show spinner with "Setting up Proton prefix (first time only)..." |
| Prefix size | Proton prefix is ~1.5GB vs ~500MB for direct Wine (includes more DLLs, Steam client files) | Document in release notes |
| Python dependency | `proton run` requires Python 3 | Python 3 is present on all target distros (Arch, Ubuntu, Fedora, SteamOS) |
| Proton upgrades | When bundled Proton version changes, prefix auto-upgrades | Proton handles this via version tracking |

## Sources

- Proton script source: `/home/cstory/cluckers/build/appimage/Cluckers.AppDir/proton/proton` (GE-Proton10-32, 2396 lines) -- **PRIMARY SOURCE, HIGH confidence**
- Existing codebase: `internal/wine/`, `internal/launch/`, `internal/config/` -- **PRIMARY SOURCE, HIGH confidence**
- [umu-launcher](https://github.com/Open-Wine-Components/umu-launcher) -- UMU launcher project for non-Steam Proton usage
- [umu man page](https://man.archlinux.org/man/umu.1.en) -- Environment variable reference
- [How to run .exe in existing Proton prefix](https://gist.github.com/michaelbutler/f364276f4030c5f449252f2c4d960bd2) -- Community guide on direct Proton invocation
- [Gamescope ArchWiki](https://wiki.archlinux.org/title/Gamescope) -- Gamescope compositor docs
- [Non-Steam games controller focus issue](https://github.com/ValveSoftware/steam-for-linux/issues/8513) -- Gamescope + non-Steam game window tracking
