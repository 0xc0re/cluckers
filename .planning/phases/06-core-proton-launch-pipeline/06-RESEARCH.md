# Phase 6: Core Proton Launch Pipeline - Research

**Researched:** 2026-02-24
**Domain:** Proton-GE game launching, Wine prefix management, Linux game compatibility layer
**Confidence:** MEDIUM-HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Detection order: Bundled (CLUCKERS_BUNDLED_PROTON env) > Config override > System scan of known directories
- When no Proton-GE found: Error with per-distro install instructions, launcher exits (no fallback to system Wine)
- Version display: Always show detected version (e.g., "Detected Proton-GE 9-27"), source path shown only in verbose mode
- Old Proton-GE versions: Warn but allow (e.g., "Proton-GE 7 detected, version 9+ recommended"), non-blocking
- Dedicated spinner step for prefix creation: "Preparing Proton environment (first launch only)..." -- sets expectations about one-time wait
- Quick prefix health check every launch: verify compatdata directory exists and pfx/drive_c is present
- Corrupted or missing prefix: Auto-recreate with warning ("Proton environment damaged, recreating..."), delete old compatdata and rebuild
- Setup completion: Success checkmark only, verbose mode shows compatdata path
- Launch failure: Show Proton log path + 2-3 common fixes (delete compatdata and relaunch, update Proton-GE, verify game files)
- Proton stderr/stdout: Capture output, show last ~10 lines on crash for immediate context, full output in verbose mode
- PROTON_LOG=1: Only enabled when user runs with -v flag, keeps compatdata tidy on normal launches
- SHM bridge failures: Distinct error message separate from general Proton failures -- detect shm_launcher exit codes/patterns and show specific guidance
- Follow existing UserError pattern (Message + Detail + Suggestion) for all Proton-related errors
- Proton detection should extend existing FindProtonGE() scan approach, adding bundled priority on top
- Spinner step names should feel consistent with existing pipeline steps
- Per-distro error messages for missing Proton-GE

### Claude's Discretion
- Exact Proton environment variable set (beyond those specified in requirements)
- Proton version parsing implementation
- Prefix health check implementation details
- stderr/stdout capture mechanism
- shm_launcher exit code detection approach

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

## Summary

Phase 6 replaces the current direct `wine64` invocation with Proton-GE's `python3 proton run` command for all Linux launches. The Proton script is a Python orchestrator that handles prefix creation, library path setup, DLL configuration (DXVK, VKD3D), and Wine invocation. When called with `proton run <exe> <args>`, Proton auto-creates a complete Wine prefix at the path specified by `STEAM_COMPAT_DATA_PATH`, then invokes wine64 with the correct environment -- eliminating the need for manual prefix template copying, winetricks, and DLL verification that the current code performs.

The current codebase already has strong Proton-GE detection (`FindProtonGE()`, `IsProtonGE()`, version parsing) and the shared memory bridge (`shm_launcher.exe` via `CreateFileMappingW`/`OpenFileMapping`) works within Wine -- it will work identically under Proton since Proton is Wine with additional setup. The main implementation work is: (1) changing the invocation from `wine64 shm_launcher.exe <args>` to `python3 /path/to/proton run shm_launcher.exe <args>` with correct environment variables, (2) changing prefix management from the current copy-template-and-verify approach to Proton's own prefix creation, and (3) updating the pipeline steps from Wine-centric to Proton-centric naming.

**Primary recommendation:** Refactor `LaunchGame()` in `process_linux.go` to invoke `python3 <proton_dir>/proton run` instead of direct `wine64`, set `STEAM_COMPAT_DATA_PATH` to `~/.cluckers/compatdata/`, and let Proton handle prefix creation. Replace the current 3-step Wine pipeline (detect/ensure prefix/verify prefix) with 2 Proton steps (detect Proton-GE/ensure compatdata).

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| PROTON-01 | Launcher detects Proton-GE installation (bundled via CLUCKERS_BUNDLED_PROTON, config override, or system Proton-GE scan paths) | Existing `FindProtonGE()` and `FindWine()` provide 90% of the detection logic. Refactor to return `ProtonGEInstall` directly (not wine64 path), add bundled priority, remove system Wine fallback. |
| PROTON-02 | Launcher invokes game via `proton run` with correct environment (STEAM_COMPAT_DATA_PATH, PROTON_WINEDLLOVERRIDES=dxgi=n, UMU_ID=0, SteamGameId, SteamAppId=0) | Proton script requires STEAM_COMPAT_DATA_PATH (prefix location), STEAM_COMPAT_CLIENT_INSTALL_PATH (can be empty or dummy for non-Steam), SteamGameId/SteamAppId (set to 0 for non-Steam). Invocation: `python3 <proton>/proton run <exe> <args>`. |
| PROTON-03 | Proton automatically creates and manages Wine prefix at ~/.cluckers/compatdata/pfx/ on first launch without manual winetricks or DLL management | Proton's `setup_prefix()` auto-creates prefix when STEAM_COMPAT_DATA_PATH points to a non-existent or empty directory. Copies default_pfx template, sets up DLLs (DXVK, VKD3D), creates dosdevices, runs wineboot -- all automatically. |
| PROTON-04 | shm_launcher.exe shared memory bridge works correctly under Proton (CreateFileMappingW/OpenFileMapping) | Wine/Proton's CreateFileMappingW with INVALID_HANDLE_VALUE creates shared memory backed by the Wine server. shm_launcher.exe creates the mapping, then CreateProcessW launches game as child -- both share the same Wine server instance. No changes needed to shm_launcher.c. |
| PROTON-05 | All game arguments pass through correctly via proton run (-user, -token, -eac_oidc_token_file, -hostx) | Proton's `run` verb passes `sys.argv[2:]` directly to Wine. So `proton run shm_launcher.exe <bootstrap_file> <shm_name> <game.exe> -user=X -token=Y ...` flows through correctly. Path arguments must still use Wine Z: drive notation since Wine is the ultimate executor. |
</phase_requirements>

## Standard Stack

### Core

This phase uses no new external libraries. The implementation is entirely within the existing Go codebase, using `os/exec` to invoke the Proton Python script.

| Component | Version | Purpose | Why Standard |
|-----------|---------|---------|--------------|
| Proton-GE | 9+ (10-32 latest) | Wine compatibility layer with auto-prefix management | Required by project -- replaces direct Wine invocation |
| Python 3 | 3.x (system) | Proton's `proton` script is Python | Proton requires Python 3 runtime; present on all modern Linux distros |
| Go `os/exec` | stdlib | Process execution for `python3 proton run` | Already used for Wine invocation, same pattern |
| Go `os` | stdlib | Environment variable management, file operations | Already used throughout codebase |

### Supporting

| Tool | Purpose | When to Use |
|------|---------|-------------|
| `filepath.EvalSymlinks` | Resolve Proton-GE directory symlinks | Already used in detection, continues to be needed |
| `regexp` | Parse Proton-GE version from directory names | Already used via `protonVersionRe` |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Direct `proton run` | umu-launcher | umu-launcher is officially recommended for non-Steam games, adds Steam Runtime container. But: adds Python pip dependency, downloads Steam Runtime (~300MB), and project REQUIREMENTS.md explicitly lists umu-launcher as out of scope ("Direct proton run is simpler and works inside AppImage"). |
| Direct `proton run` | Direct `wine64` (current) | Current approach works but requires manual prefix management, winetricks, DLL verification -- all eliminated by Proton's orchestration. |

## Architecture Patterns

### Current vs New Invocation Chain

**Current (direct Wine):**
```
cluckers -> wine64 shm_launcher.exe <bootstrap_file> <shm_name> <game.exe> <game_args>
  env: WINEPREFIX=~/.cluckers/prefix, WINEFSYNC=1, WINEDLLOVERRIDES=dxgi=n
```

**New (Proton):**
```
cluckers -> python3 <proton_dir>/proton run shm_launcher.exe <bootstrap_file> <shm_name> <game.exe> <game_args>
  env: STEAM_COMPAT_DATA_PATH=~/.cluckers/compatdata, SteamGameId=0, SteamAppId=0, WINEDLLOVERRIDES=dxgi=n
  (Proton internally: sets WINEPREFIX, LD_LIBRARY_PATH, WINEDLLPATH, launches wine64)
```

### Directory Structure Change

**Current:**
```
~/.cluckers/
  prefix/           # Direct Wine prefix (managed by cluckers)
    drive_c/
    dosdevices/
```

**New:**
```
~/.cluckers/
  compatdata/        # Proton-managed compatdata
    pfx/             # The actual Wine prefix (managed by Proton)
      drive_c/
      dosdevices/
    version           # Proton version tracking file
    tracked_files     # Proton's file tracking
```

### Pattern 1: Proton Detection (Refactored FindProtonGE)

**What:** Refactor the existing detection to return the Proton root directory (not the wine64 binary path), since `proton run` needs the `proton` script path, not `wine64`.

**Current data model:**
```go
type ProtonGEInstall struct {
    WinePath  string // /path/to/GE-Proton10-1/files/bin/wine64
    ProtonDir string // /path/to/GE-Proton10-1
}
```

**New fields needed:** The existing `ProtonDir` field already points to the right place. The `proton` script lives at `<ProtonDir>/proton`. Add a method to get the script path:

```go
// ProtonScript returns the path to the proton Python script.
func (p ProtonGEInstall) ProtonScript() string {
    return filepath.Join(p.ProtonDir, "proton")
}

// DisplayVersion returns a human-readable version string like "GE-Proton10-1".
func (p ProtonGEInstall) DisplayVersion() string {
    return filepath.Base(p.ProtonDir)
}
```

### Pattern 2: New FindProton Function

**What:** A new `FindProton()` function that replaces `FindWine()` for the Proton-only path. Returns a `ProtonGEInstall` or error.

```go
func FindProton(configOverride string) (*ProtonGEInstall, error) {
    // 1. Bundled (CLUCKERS_BUNDLED_PROTON env var)
    if bundled := os.Getenv("CLUCKERS_BUNDLED_PROTON"); bundled != "" {
        protonScript := filepath.Join(bundled, "proton")
        if _, err := os.Stat(protonScript); err == nil {
            return &ProtonGEInstall{
                WinePath:  filepath.Join(bundled, "files", "bin", "wine64"),
                ProtonDir: bundled,
            }, nil
        }
        ui.Warn("Bundled Proton-GE not found at " + bundled + ", searching system...")
    }

    // 2. Config override (wine_path or new proton_path setting)
    if configOverride != "" {
        // Resolve to ProtonDir from wine64 path or proton script path
        // ...
    }

    // 3. System scan (existing FindProtonGE)
    home := userHome()
    installs := FindProtonGE(home)
    if len(installs) > 0 {
        return &installs[0], nil
    }

    // 4. Error with per-distro instructions
    distro := DetectDistro()
    return nil, &ui.UserError{
        Message:    "Proton-GE not found. Proton-GE is required to run Realm Royale.",
        Suggestion: ProtonInstallInstructions(distro),
    }
}
```

### Pattern 3: Proton Invocation in LaunchGame

**What:** Replace direct wine64 exec with python3 proton run.

```go
// Build the command: python3 <proton_script> run <shm_launcher.exe> <args...>
protonScript := cfg.ProtonInstall.ProtonScript()
args := []string{protonScript, "run"}
args = append(args, shmPath)  // shm_launcher.exe (Linux path -- Proton converts)
args = append(args, wine.LinuxToWinePath(bootstrapPath))
args = append(args, shmName)
args = append(args, wine.LinuxToWinePath(gameExe))
args = append(args, gameArgs...)

cmd := exec.CommandContext(ctx, "python3", args...)

// Set Proton environment
cmd.Env = buildProtonEnv(cfg)
```

**Key difference from current:** The current code runs `wine64` directly with `WINEPREFIX`. The new code runs `python3 proton run` with `STEAM_COMPAT_DATA_PATH`. Proton internally sets `WINEPREFIX` to `<STEAM_COMPAT_DATA_PATH>/pfx/`.

### Pattern 4: Proton Environment Variables

**What:** Build the correct environment for `proton run`.

```go
func buildProtonEnv(cfg *LaunchConfig) []string {
    env := os.Environ()

    // Filter conflicting vars
    filtered := filterEnv(env, "LD_LIBRARY_PATH", "WINEPREFIX", "WINE",
                          "WINEDLLOVERRIDES", "WINEFSYNC", "WINEESYNC")

    // Required by Proton
    filtered = append(filtered,
        "STEAM_COMPAT_DATA_PATH="+cfg.CompatDataPath,
        "STEAM_COMPAT_CLIENT_INSTALL_PATH=",  // Empty -- we're not inside Steam
        "SteamGameId=0",
        "SteamAppId=0",
    )

    // DLL override for DXVK (dxgi native)
    filtered = append(filtered, "WINEDLLOVERRIDES=dxgi=n")

    // Conditional: debug logging
    if cfg.Verbose {
        filtered = append(filtered, "PROTON_LOG=1")
    }

    return filtered
}
```

### Pattern 5: Compatdata Health Check (Replaces DLL Verification)

**What:** Simple directory existence check instead of individual DLL verification.

```go
// CompatdataHealthy checks if the Proton compatdata directory looks valid.
// Quick check: compatdata exists and pfx/drive_c is present.
func CompatdataHealthy(compatdataPath string) bool {
    driveC := filepath.Join(compatdataPath, "pfx", "drive_c")
    info, err := os.Stat(driveC)
    return err == nil && info.IsDir()
}
```

### Pattern 6: Proton Output Capture

**What:** Capture Proton/Wine stderr for error reporting without cluttering normal output.

```go
// Capture stderr to buffer for error reporting
var stderrBuf bytes.Buffer
if cfg.Verbose {
    cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
} else {
    cmd.Stderr = &stderrBuf
}

// On failure, show last N lines of stderr
if err := cmd.Run(); err != nil {
    if ctx.Err() != nil {
        return nil // Ctrl+C
    }
    lastLines := lastNLines(stderrBuf.String(), 10)
    return &ui.UserError{
        Message:    "Game exited with an error.",
        Detail:     lastLines,
        Suggestion: "Common fixes:\n  1. Delete ~/.cluckers/compatdata/ and relaunch\n  2. Update Proton-GE to latest version\n  3. Run `cluckers update` to verify game files",
    }
}
```

### Anti-Patterns to Avoid

- **Setting WINEPREFIX directly:** Let Proton manage WINEPREFIX via STEAM_COMPAT_DATA_PATH. Setting both causes conflicts.
- **Running winetricks on Proton prefix:** This is already documented in CLAUDE.md as critical. With the Proton path, winetricks is never needed -- Proton bundles everything.
- **Checking individual DLLs in Proton prefix:** Proton manages its own DLLs. The current `VerifyPrefix()` checking for vcruntime140.dll, d3d11.dll, etc. is unnecessary and fragile with Proton (DLLs may be in different locations or use different names).
- **Passing Linux paths to proton run for the exe argument:** The exe path passed to `proton run` should be a Linux path. Proton's script converts it internally. But arguments within the game args that represent file paths (like -eac_oidc_token_file) need Wine Z: drive paths since they're consumed by the Windows game process.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Wine prefix creation | Template copying, dosdevices symlinks, wineboot | `proton run` (auto-creates prefix) | Proton handles 50+ initialization steps including DLL deployment, registry setup, font symlinks, library paths. The current `createFromProtonTemplate()` replicates maybe 10% of this. |
| DLL management | DLL verification, winetricks | Proton's built-in DLL management | Proton copies DXVK, VKD3D, vcruntime from its own bundled files based on config flags. No external dependencies. |
| Wine library path setup | Manual LD_LIBRARY_PATH | Proton's init_wine() | Proton configures x86_64-unix, i386-unix, x86_64-windows, i386-windows library paths correctly for its own bundled Wine. |
| DXVK/VKD3D deployment | DLL overrides + file copies | Proton's setup_prefix() | Proton deploys the correct DXVK/VKD3D version matched to its Wine build. |

**Key insight:** Proton is not just Wine -- it's an orchestrator that handles ~500 lines of prefix initialization, library path configuration, and DLL deployment. By using `proton run`, we delegate all of this complexity to the Proton script and only need to set 4-5 environment variables.

## Common Pitfalls

### Pitfall 1: Python3 Not Found
**What goes wrong:** `python3` not in PATH on minimal Linux installs.
**Why it happens:** Some distros ship Python as `python` not `python3`. Proton-GE's shebang is `#!/usr/bin/env python3`.
**How to avoid:** Check for `python3` at Proton detection time (not at launch time). Proton-GE directory also contains a `proton` script that is executable -- try running it directly first (`exec.Command(protonScript, "run", ...)`), falling back to `exec.Command("python3", protonScript, "run", ...)` if the direct execution fails.
**Warning signs:** "python3: command not found" error at launch time.

### Pitfall 2: STEAM_COMPAT_CLIENT_INSTALL_PATH
**What goes wrong:** Proton may warn or fail if STEAM_COMPAT_CLIENT_INSTALL_PATH is not set.
**Why it happens:** Proton's script checks for this variable to locate Steam client DLLs (steamclient.so, etc.).
**How to avoid:** Set it to an empty string or a dummy path. For Phase 6, this is acceptable since we're not running through Steam. Phase 7 will properly detect Steam installation. The Proton script handles missing Steam files gracefully -- it skips copying steamclient if the path doesn't exist.
**Warning signs:** Warnings about missing steamclient.so in Proton log output.

### Pitfall 3: Path Argument Confusion
**What goes wrong:** Game fails to find OIDC token file or game executable.
**Why it happens:** Two different path contexts: (1) paths passed TO proton run (the exe argument) should be Linux paths -- Proton converts them, and (2) paths passed AS game arguments (-eac_oidc_token_file=Z:\tmp\...) must be Wine Z: drive paths because the game process reads them.
**How to avoid:** Continue using `wine.LinuxToWinePath()` for all paths that appear as game argument values. The first positional argument to `proton run` (the exe to launch) should be a Linux path.
**Warning signs:** "File not found" errors for OIDC token or bootstrap file.

### Pitfall 4: LD_LIBRARY_PATH Contamination
**What goes wrong:** Wine crashes or loads wrong libraries.
**Why it happens:** The current code already strips LD_LIBRARY_PATH for AppImage compatibility. Proton sets its own LD_LIBRARY_PATH internally. If the AppImage's LD_LIBRARY_PATH leaks through, Proton's Wine may load incompatible libraries.
**How to avoid:** Continue stripping LD_LIBRARY_PATH before launching Proton (current pattern in process_linux.go already does this). Proton will set its own paths.
**Warning signs:** Wine crashes with "wrong ELF class" or segfaults in library loading.

### Pitfall 5: WINEDLLOVERRIDES Conflict
**What goes wrong:** Proton sets its own WINEDLLOVERRIDES internally (for DLL override chain). If we set WINEDLLOVERRIDES externally, it may override Proton's settings entirely.
**Why it happens:** Proton constructs WINEDLLOVERRIDES based on its configuration (DXVK, VKD3D, etc.). External WINEDLLOVERRIDES replaces, not appends.
**How to avoid:** Check if Proton-GE supports `PROTON_WINEDLLOVERRIDES` (an additive variable). The requirements specify `PROTON_WINEDLLOVERRIDES=dxgi=n`. If this variable is not supported by Proton-GE, set `WINEDLLOVERRIDES=dxgi=n` and accept that it replaces Proton's defaults. Since Proton already sets `dxgi=n` for DXVK games, this should be idempotent. Validate on actual hardware.
**Warning signs:** DXVK not loading, D3D11 errors, or black screen at game launch.

### Pitfall 6: Proton Script Not Executable
**What goes wrong:** `exec.Command("python3", protonScript, ...)` works but `exec.Command(protonScript, ...)` fails with permission denied.
**Why it happens:** Some Proton-GE installations (especially system packages) may not have the proton script marked as executable.
**How to avoid:** Always use `python3 <proton_script>` as the invocation pattern. This is consistent with how other launchers (Lutris, Heroic) invoke Proton.
**Warning signs:** "permission denied" when trying to run proton script directly.

### Pitfall 7: First-Launch Timeout
**What goes wrong:** Proton prefix creation takes 30-60 seconds on first launch, and the user thinks the launcher is hung.
**Why it happens:** Proton's `setup_prefix()` copies hundreds of files, runs wineboot, configures DLLs.
**How to avoid:** Use the dedicated spinner step ("Preparing Proton environment (first launch only)...") as specified in the user constraints. The prefix creation happens inside the `proton run` command itself (before it launches the exe), so the spinner should be shown before launching proton. However, since proton does prefix creation at the start of `proton run`, we need to either: (a) run `proton run` for the game and show a spinner around the entire command (imprecise), or (b) run a quick dummy command first (`proton run cmd /c exit`) to trigger prefix creation, then run the actual game launch.
**Warning signs:** Long pause with no visible output on first launch.

## Code Examples

### Example 1: Proton Invocation (Complete)

```go
// In process_linux.go -- new LaunchGame implementation

func LaunchGame(ctx context.Context, cfg *LaunchConfig) error {
    gameExe := game.GameExePath(cfg.GameDir)
    if _, err := os.Stat(gameExe); err != nil {
        return &ui.UserError{
            Message:    "Game executable not found: " + gameExe,
            Detail:     err.Error(),
            Suggestion: "Run `cluckers update` to download game files.",
        }
    }

    var cleanups []func()
    defer func() {
        for _, fn := range cleanups {
            fn()
        }
    }()

    // Game arguments (same as before -- these are consumed by the Windows game process)
    gameArgs := []string{
        fmt.Sprintf("-user=%s", cfg.Username),
        fmt.Sprintf("-token=%s", cfg.AccessToken),
        fmt.Sprintf("-eac_oidc_token_file=%s", wine.LinuxToWinePath(cfg.OIDCTokenPath)),
        fmt.Sprintf("-hostx=%s", cfg.HostX),
        "-Language=INT",
        "-dx11",
        "-content_bootstrap_size=136",
        "-seekfreeloadingpcconsole",
        "-nohomedir",
    }

    // Build proton command args
    protonScript := cfg.ProtonInstall.ProtonScript()
    var protonArgs []string

    if cfg.ContentBootstrap != nil && len(cfg.ContentBootstrap) > 0 {
        shmPath, shmCleanup, err := ExtractSHMLauncher()
        if err != nil {
            return fmt.Errorf("extract shm_launcher: %w", err)
        }
        cleanups = append(cleanups, shmCleanup)

        bootstrapPath, bootstrapCleanup, err := WriteBootstrapFile(cfg.ContentBootstrap)
        if err != nil {
            return fmt.Errorf("write bootstrap file: %w", err)
        }
        cleanups = append(cleanups, bootstrapCleanup)

        shmName := fmt.Sprintf(`Local\realm_content_bootstrap_%d`, os.Getpid())
        gameArgs = append(gameArgs, fmt.Sprintf("-content_bootstrap_shm=%s", shmName))

        // proton run <shm_launcher.exe> <bootstrap_file(wine path)> <shm_name> <game_exe(wine path)> <game_args>
        protonArgs = []string{protonScript, "run", shmPath,
            wine.LinuxToWinePath(bootstrapPath),
            shmName,
            wine.LinuxToWinePath(gameExe),
        }
        protonArgs = append(protonArgs, gameArgs...)
    } else {
        protonArgs = []string{protonScript, "run", gameExe}
        protonArgs = append(protonArgs, gameArgs...)
    }

    // Build environment
    env := buildProtonEnv(cfg)

    cmd := exec.CommandContext(ctx, "python3", protonArgs...)
    cmd.Env = env
    cmd.Dir = cfg.GameDir

    // stderr capture for error reporting
    var stderrBuf bytes.Buffer
    if cfg.Verbose {
        cmd.Stdout = os.Stdout
        cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
    } else {
        cmd.Stdout = nil
        cmd.Stderr = &stderrBuf
    }

    if err := cmd.Run(); err != nil {
        if ctx.Err() != nil {
            return nil
        }
        detail := lastNLines(stderrBuf.String(), 10)
        return &ui.UserError{
            Message:    "Game exited with an error.",
            Detail:     detail,
            Suggestion: protonErrorSuggestion(cfg.CompatDataPath),
        }
    }

    return nil
}
```

### Example 2: Updated Pipeline Steps (Linux)

```go
// In pipeline_linux.go -- new platform steps

func platformSteps(_ *LaunchState) []Step {
    return []Step{
        {Name: "Detecting Proton", Fn: stepDetectProton},
        {Name: "Preparing Proton environment", Fn: stepEnsureCompatdata},
    }
}

func stepDetectProton(_ context.Context, state *LaunchState) error {
    install, err := wine.FindProton(state.Config.WinePath)
    if err != nil {
        return err
    }
    state.ProtonInstall = install

    // Version warning for old Proton-GE
    version := install.DisplayVersion()
    ui.Verbose(fmt.Sprintf("Proton: %s (%s)", version, install.ProtonDir), state.Config.Verbose)

    m := wine.ProtonVersionRe.FindStringSubmatch(filepath.Base(install.ProtonDir))
    if m != nil {
        major, _ := strconv.Atoi(m[1])
        if major < 9 {
            ui.Warn(fmt.Sprintf("%s detected, version 9+ recommended", version))
        }
    }

    return nil
}

func stepEnsureCompatdata(_ context.Context, state *LaunchState) error {
    compatdata := filepath.Join(config.DataDir(), "compatdata")

    if wine.CompatdataHealthy(compatdata) {
        ui.Verbose(fmt.Sprintf("Proton environment: %s", compatdata), state.Config.Verbose)
        state.CompatDataPath = compatdata
        return nil
    }

    // Missing or corrupted -- (re)create
    if _, err := os.Stat(compatdata); err == nil {
        ui.Warn("Proton environment damaged, recreating...")
        os.RemoveAll(compatdata)
    }

    // Create the compatdata directory. Proton's setup_prefix() will populate it on first run.
    if err := os.MkdirAll(compatdata, 0755); err != nil {
        return fmt.Errorf("create compatdata directory: %w", err)
    }

    state.CompatDataPath = compatdata
    return nil
}
```

### Example 3: LaunchConfig Changes

```go
// Updated LaunchConfig to hold Proton-specific fields
type LaunchConfig struct {
    ProtonInstall    *wine.ProtonGEInstall  // Replaces WinePath for Linux
    CompatDataPath   string                 // Replaces WinePrefix for Linux
    GameDir          string
    Username         string
    AccessToken      string
    OIDCTokenPath    string
    ContentBootstrap []byte
    HostX            string
    Verbose          bool
}
```

### Example 4: Proton Version Warning

```go
// Parse and warn on old versions
func warnOldProton(protonDir string) {
    name := filepath.Base(protonDir)
    m := protonVersionRe.FindStringSubmatch(name)
    if m == nil {
        return // Can't parse version, skip warning
    }
    major, _ := strconv.Atoi(m[1])
    if major < 9 {
        ui.Warn(fmt.Sprintf("%s detected, version 9+ recommended", name))
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Direct wine64 + manual prefix | proton run (auto-prefix) | Proton 5.0+ (2020) | Eliminates need for winetricks, DLL management, prefix template copying |
| WINEPREFIX env var | STEAM_COMPAT_DATA_PATH | Proton 3.0+ (2018) | Proton creates prefix at `<compat_data>/pfx/`, manages version tracking |
| Proton standalone invocation | umu-launcher for non-Steam | 2024 (GE-Proton 9-2) | umu-launcher recommended by GE for full Steam Runtime container. Direct `proton run` still works but without container. |
| Manual DXVK installation | Proton-bundled DXVK | Proton 3.7+ (2018) | DXVK automatically deployed by Proton during prefix creation |

**Deprecated/outdated:**
- **createFromProtonTemplate()**: Current code manually copies default_pfx and creates dosdevices. With `proton run`, this is entirely handled by Proton's setup_prefix(). Will be removed in Phase 8.
- **VerifyPrefix() DLL checks**: Current code verifies vcruntime140.dll, msvcp140.dll, d3dx11_43.dll, d3d11.dll. These are unnecessary with Proton since Proton manages its own DLLs. Will be removed in Phase 8.
- **System Wine fallback**: Per REQUIREMENTS.md, system Wine is out of scope for v1.1. The current `FindWine()` fallback to `exec.LookPath("wine")` will not be used.

## Open Questions

1. **proton run exe path format**
   - What we know: Proton's `run` verb takes an exe path and passes it to wine64 via `sys.argv[2:]`. Users report both Linux paths and Wine paths working.
   - What's unclear: Whether `proton run /tmp/shm_launcher_12345.exe` (Linux path) is automatically converted to a Wine path, or if we need to pass `Z:\tmp\shm_launcher_12345.exe`. The Proton script may do path conversion internally.
   - Recommendation: Test both approaches on actual hardware. Start with Linux path (simpler); if it fails, switch to Wine Z: drive path. LOW confidence on which is correct without testing.

2. **First-launch prefix creation timing**
   - What we know: Proton creates the prefix at the start of `proton run`. There's no separate "create prefix" command.
   - What's unclear: Whether the spinner step "Preparing Proton environment" should run a dummy `proton run cmd /c exit` to trigger prefix creation separately from the game launch, or if we show the spinner around the game launch itself (which blocks for prefix creation + game execution).
   - Recommendation: Use the approach of checking `CompatdataHealthy()` before launch. If not healthy, show the "Preparing Proton environment (first launch only)..." message, then let the actual `proton run` for the game handle both prefix creation and launch. The Proton script writes progress to stderr which can be used to detect when prefix setup is complete. MEDIUM confidence.

3. **SteamGameId=0 behavior**
   - What we know: The Proton script uses SteamGameId for logging and some per-game fixes. Setting it to 0 means "unknown game."
   - What's unclear: Whether SteamGameId=0 causes any Proton warnings or unexpected behavior. Phase 7 will use the actual Steam non-Steam-game ID for Gamescope tracking.
   - Recommendation: Set to 0 for Phase 6. If Proton complains, try omitting it entirely. MEDIUM confidence.

4. **WINEDLLOVERRIDES vs PROTON_WINEDLLOVERRIDES**
   - What we know: Requirements specify `PROTON_WINEDLLOVERRIDES=dxgi=n`. Research found that WINEDLLOVERRIDES is the standard variable, and PROTON_WINEDLLOVERRIDES may or may not exist in Proton-GE.
   - What's unclear: Whether Proton-GE supports PROTON_WINEDLLOVERRIDES as an additive variable. If not, setting WINEDLLOVERRIDES externally may clobber Proton's own overrides.
   - Recommendation: Test with WINEDLLOVERRIDES=dxgi=n first (since Proton already sets dxgi=n for DXVK, it should be idempotent). If issues arise, investigate PROTON_WINEDLLOVERRIDES. MEDIUM confidence.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | None -- `go test ./...` |
| Quick run command | `go test ./internal/wine/ ./internal/launch/ ./internal/config/ -count=1` |
| Full suite command | `go test ./... -count=1` |
| Estimated runtime | ~5 seconds |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PROTON-01 | FindProton() returns correct install with bundled > config > scan priority | unit | `go test ./internal/wine/ -run TestFindProton -count=1` | No -- Wave 0 gap |
| PROTON-01 | FindProton() returns error with per-distro instructions when not found | unit | `go test ./internal/wine/ -run TestFindProtonNotFound -count=1` | No -- Wave 0 gap |
| PROTON-01 | Version warning for Proton-GE < 9 | unit | `go test ./internal/wine/ -run TestProtonVersionWarning -count=1` | No -- Wave 0 gap |
| PROTON-02 | buildProtonEnv() sets correct environment variables | unit | `go test ./internal/launch/ -run TestBuildProtonEnv -count=1` | No -- Wave 0 gap |
| PROTON-02 | LaunchGame() constructs correct python3 proton run command | unit | `go test ./internal/launch/ -run TestProtonCommand -count=1` | No -- Wave 0 gap |
| PROTON-03 | CompatdataHealthy() returns true when pfx/drive_c exists | unit | `go test ./internal/wine/ -run TestCompatdataHealthy -count=1` | No -- Wave 0 gap |
| PROTON-03 | CompatdataHealthy() returns false for missing/corrupt prefix | unit | `go test ./internal/wine/ -run TestCompatdataUnhealthy -count=1` | No -- Wave 0 gap |
| PROTON-03 | Proton creates prefix automatically on first run | manual-only | Test on Linux with Proton-GE installed | N/A -- requires Proton runtime |
| PROTON-04 | shm_launcher.exe works under Proton | manual-only | Test on Linux with Proton-GE + game | N/A -- requires game + Proton |
| PROTON-05 | Game arguments pass through proton run correctly | unit (args construction) | `go test ./internal/launch/ -run TestGameArgs -count=1` | No -- Wave 0 gap |
| PROTON-05 | Wine Z: drive path conversion still works for game args | unit | `go test ./internal/wine/ -run TestLinuxToWinePath -count=1` | No -- Wave 0 gap (test exists in concept but not as file) |

### Nyquist Sampling Rate
- **Minimum sample interval:** After every committed task, run: `go test ./internal/wine/ ./internal/launch/ ./internal/config/ -count=1`
- **Full suite trigger:** Before merging final task of any plan wave
- **Phase-complete gate:** Full suite green (`go test ./... -count=1`) before verification
- **Estimated feedback latency per task:** ~3-5 seconds

### Wave 0 Gaps (must be created before implementation)
- [ ] `internal/wine/detect_test.go` -- covers PROTON-01 (FindProton, version parsing, priority order)
- [ ] `internal/wine/compatdata_test.go` -- covers PROTON-03 (CompatdataHealthy)
- [ ] `internal/launch/proton_env_test.go` -- covers PROTON-02 (buildProtonEnv, env variable correctness)
- [ ] `internal/launch/proton_args_test.go` -- covers PROTON-05 (argument construction, path handling)

*(Note: Tests for PROTON-03 prefix creation and PROTON-04 SHM bridge require Proton runtime and are manual-only.)*

## Sources

### Primary (HIGH confidence)
- Valve Proton source code (proton_9 branch) -- proton script entry point, setup_prefix, run verb behavior
- GloriousEggroll/proton-ge-custom master branch -- proton script, GE-Proton version scheme
- Existing cluckers codebase -- `internal/wine/detect.go`, `internal/launch/process_linux.go`, `internal/launch/pipeline_linux.go`

### Secondary (MEDIUM confidence)
- [How to launch games via Proton from CLI](https://gist.github.com/sxiii/6b5cd2e7d2321df876730f8cafa12b2e) -- CLI invocation pattern, env vars
- [How to run another .exe in an existing proton wine prefix](https://gist.github.com/michaelbutler/f364276f4030c5f449252f2c4d960bd2) -- proton run with custom exe, arg passing
- [umu-launcher FAQ](https://github.com/Open-Wine-Components/umu-launcher/wiki/Frequently-asked-questions-(FAQ)) -- context on why direct proton run vs umu-launcher
- [GloriousEggroll/proton-ge-custom README](https://github.com/GloriousEggroll/proton-ge-custom) -- official stance on standalone usage

### Tertiary (LOW confidence)
- WINEDLLOVERRIDES vs PROTON_WINEDLLOVERRIDES behavior -- could not verify from official source, needs hardware testing
- proton run exe path format (Linux path vs Wine path) -- conflicting reports, needs hardware testing
- SteamGameId=0 behavior -- no official documentation found, needs hardware testing

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new libraries, using existing patterns with os/exec
- Architecture: MEDIUM-HIGH -- Proton invocation pattern is well-documented, but path handling and env var interaction need hardware validation
- Pitfalls: MEDIUM -- common pitfalls identified from multiple sources, but WINEDLLOVERRIDES behavior and first-launch timing are uncertain

**Research date:** 2026-02-24
**Valid until:** 2026-03-24 (Proton-GE releases frequently but the core invocation API is stable)
