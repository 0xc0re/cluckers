# Phase 7: Controller and Gamescope Integration - Research

**Researched:** 2026-02-25
**Domain:** Steam Deck controller input persistence, Gamescope window tracking, Proton environment configuration
**Confidence:** MEDIUM

## Summary

Phase 7 addresses the Steam Deck controller input loss that occurs during UE3 ServerTravel (lobby-to-match transition). The root cause is well-understood from extensive prior debugging (documented in `controller-debugging.md`): when UE3 destroys and recreates its D3D window during ServerTravel, Gamescope loses track of the game window. Steam Input then reconfigures the controller firmware to "desktop mode," zeroing all button data (HID report bytes 8-13) at the hardware level. Joystick axes continue to work because they operate through a different HID path.

The fix has two concrete implementation tasks: (1) set `SteamGameId` to a meaningful value (the non-Steam game shortcut's app ID) instead of the current `0`, so Gamescope can associate newly created windows with the same game identity across window recreation, and (2) auto-detect the Steam installation path and set `STEAM_COMPAT_CLIENT_INSTALL_PATH` so Proton can locate Steam client libraries (`steamclient.so`) needed for proper Steam integration. Currently, `SteamGameId=0` means "unknown game" to Gamescope, and `STEAM_COMPAT_CLIENT_INSTALL_PATH=` (empty) means Proton cannot find Steam's native libraries. Both are set in `buildProtonEnvFrom()` in `internal/launch/proton_env.go`.

The third requirement (CTRL-03) is a hardware validation gate -- it cannot be tested in automated tests and requires running on actual Steam Deck hardware to confirm that controller buttons persist through the ServerTravel transition. This research identifies the implementation changes needed and the testing strategy.

**Primary recommendation:** Implement `FindSteamInstall()` in the `wine` package (parallel to existing `FindProtonGE()`) that scans known Steam paths for native/Flatpak/Snap installations. Set `SteamGameId` to the non-Steam shortcut app ID (already computed in `deckconfig.go` via `findCluckersAppID()`). Update `buildProtonEnvFrom()` to accept both values. Hardware validation on Steam Deck is required for CTRL-03 sign-off.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| CTRL-01 | Launcher injects SteamGameId env var for Gamescope window tracking across UE3 ServerTravel | Current code sets `SteamGameId=0` in `proton_env.go:62`. Change to use the non-Steam game shortcut app ID, which is already extractable from Steam's `shortcuts.vdf` via `findCluckersAppID()` in `deckconfig.go`. The `SteamGameId` is read by Proton's Wine patches to set X11 window class hints as `steam_app_{id}` (confirmed in Proton Wine source `dlls/winex11.drv/window.c`), which Gamescope uses for window-to-game association. |
| CTRL-02 | Launcher auto-detects Steam installation path for STEAM_COMPAT_CLIENT_INSTALL_PATH (native, Flatpak, Snap) | Current code sets `STEAM_COMPAT_CLIENT_INSTALL_PATH=` (empty) in `proton_env.go:61`. Need to detect Steam's root directory across 3 installation types. The existing `protonSearchDirs()` in `detect.go` already has the directory patterns for all 3 types. New `FindSteamInstall()` function checks for Steam's root marker files (`steam.sh` or `ubuntu12_32/steamclient.so`). |
| CTRL-03 | Controller buttons persist through lobby-to-match transition on Steam Deck (validated on hardware) | This is a hardware validation requirement. Cannot be automated. Depends on CTRL-01 and CTRL-02 being correctly implemented. Prior debugging proved the input loss is at the Steam Input firmware level -- the fix depends on Gamescope maintaining window tracking through D3D recreation, which requires a valid SteamGameId. |
</phase_requirements>

## Standard Stack

### Core

This phase uses no new external libraries. All implementation is within the existing Go codebase, modifying the environment variable construction and adding Steam installation detection.

| Component | Version | Purpose | Why Standard |
|-----------|---------|---------|--------------|
| Go `os` | stdlib | Environment variable construction, file/directory existence checks | Already used throughout codebase |
| Go `path/filepath` | stdlib | Path construction for Steam installation detection | Already used in `detect.go` |
| Go `encoding/binary` | stdlib | Binary VDF parsing for app ID extraction | Already used in `deckconfig.go` |

### Supporting

| Tool | Purpose | When to Use |
|------|---------|-------------|
| `filepath.EvalSymlinks` | Resolve `~/.steam/root` and `~/.steam/steam` symlinks to find actual Steam dir | Already used in `symlinkResolvedDirs()` |
| `os.ReadFile` | Read `shortcuts.vdf` for app ID extraction | Already used in `deployDeckControllerLayout()` |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Direct `SteamGameId` from shortcuts.vdf | Hardcoded app ID | Hardcoding is fragile -- the app ID is a hash of shortcut name + exe path, so it changes if the user renames the shortcut or moves the binary. Dynamic detection is more robust. |
| File-based Steam path detection | `steam` process inspection (`/proc/*/exe`) | Process inspection is fragile (Steam may not be running), requires elevated permissions, and does not work when cluckers launches before Steam. File-based detection is simpler and more reliable. |
| Scanning for `steam.sh` marker | Using `~/.steam/steam` symlink only | The symlink approach misses Flatpak and Snap installations. Scanning multiple known paths covers all 3 installation types. |

## Architecture Patterns

### Current State of Relevant Code

The environment variables are constructed in `internal/launch/proton_env.go`:
```go
// proton_env.go lines 58-65 (current)
env = append(env,
    "STEAM_COMPAT_DATA_PATH="+compatDataPath,
    "STEAM_COMPAT_CLIENT_INSTALL_PATH=",    // <-- CTRL-02: currently empty
    "SteamGameId=0",                         // <-- CTRL-01: currently 0
    "SteamAppId=0",
    "WINEDLLOVERRIDES=dxgi=n",
)
```

The non-Steam game app ID extraction already exists in `internal/launch/deckconfig.go`:
```go
// deckconfig.go -- already extracts app ID from shortcuts.vdf
func findCluckersAppID(data []byte) uint32 { ... }
```

Steam directory paths are already enumerated in `internal/wine/detect.go`:
```go
// detect.go -- paths that overlap with Steam install detection
func protonSearchDirs(home string) []string {
    return []string{
        "/usr/share/steam/compatibilitytools.d",
        filepath.Join(home, ".steam", "root", "compatibilitytools.d"),
        filepath.Join(home, ".local", "share", "Steam", "compatibilitytools.d"),
        filepath.Join(home, ".var", "app", "com.valvesoftware.Steam",
            "data", "Steam", "compatibilitytools.d"),
        filepath.Join(home, "snap", "steam", "common", ".steam",
            "steam", "compatibilitytools.d"),
        // ...
    }
}
```

### Pattern 1: Steam Installation Detection

**What:** New `FindSteamInstall()` function in `internal/wine/` that returns the Steam root directory path.

**When to use:** Called during Proton environment construction to set `STEAM_COMPAT_CLIENT_INSTALL_PATH`.

**Design:**
```go
// steamdir.go (new file in internal/wine/)

// steamInstallDirs returns directories to check for Steam installations.
func steamInstallDirs(home string) []string {
    return []string{
        // Native Steam (most common)
        filepath.Join(home, ".local", "share", "Steam"),
        // Native Steam via symlink
        filepath.Join(home, ".steam", "steam"),
        filepath.Join(home, ".steam", "root"),
        // Flatpak Steam
        filepath.Join(home, ".var", "app", "com.valvesoftware.Steam",
            "data", "Steam"),
        // Snap Steam
        filepath.Join(home, "snap", "steam", "common", ".local",
            "share", "Steam"),
    }
}

// FindSteamInstall returns the Steam root directory, or "" if not found.
// Checks for the presence of steam.sh or ubuntu12_32/steamclient.so
// as markers of a valid Steam installation.
func FindSteamInstall() string {
    return findSteamInstall(userHome())
}

// findSteamInstall is the internal implementation for testability.
func findSteamInstall(home string) string {
    seen := make(map[string]bool)
    for _, dir := range steamInstallDirs(home) {
        resolved := resolveReal(dir)
        if seen[resolved] {
            continue
        }
        seen[resolved] = true
        if isSteamDir(resolved) {
            return resolved
        }
    }
    return ""
}

// isSteamDir checks for Steam marker files.
func isSteamDir(dir string) bool {
    markers := []string{
        filepath.Join(dir, "steam.sh"),
        filepath.Join(dir, "ubuntu12_32", "steamclient.so"),
    }
    for _, m := range markers {
        if _, err := os.Stat(m); err == nil {
            return true
        }
    }
    return false
}
```

### Pattern 2: Non-Steam Game App ID Resolution

**What:** Extract the Steam shortcut app ID for cluckers from `shortcuts.vdf`, to use as `SteamGameId`.

**When to use:** Called during pipeline setup (a new pipeline step or incorporated into an existing step).

**Design:** The `findCluckersAppID()` function already exists in `deckconfig.go`. It needs to be:
1. Exported (or a new wrapper function exported)
2. Made accessible from the environment construction code
3. Called with the Steam userdata path (derived from Steam install path)

```go
// The app ID flows through LaunchState:
// pipeline step -> state.SteamGameId -> LaunchConfig.SteamGameId -> buildProtonEnvFrom()

// In pipeline_linux.go, a new step or addition to stepDetectProton:
func resolveSteamGameId(state *LaunchState) {
    steamDir := wine.FindSteamInstall()
    if steamDir == "" {
        return // Non-fatal: SteamGameId stays "0"
    }
    // Scan userdata directories for shortcuts.vdf containing cluckers
    pattern := filepath.Join(steamDir, "userdata", "*", "config", "shortcuts.vdf")
    matches, _ := filepath.Glob(pattern)
    for _, shortcutsPath := range matches {
        data, _ := os.ReadFile(shortcutsPath)
        if appID := FindCluckersAppID(data); appID != 0 {
            state.SteamGameId = fmt.Sprintf("%d", appID)
            return
        }
    }
}
```

### Pattern 3: Updated Proton Environment Construction

**What:** Update `buildProtonEnvFrom()` to accept `steamInstallPath` and `steamGameId` parameters.

**Design:**
```go
// Updated signature (breaking change to internal function)
func buildProtonEnvFrom(baseEnv []string, compatDataPath, steamInstallPath, steamGameId string, verbose bool) []string {
    env := filterEnv(baseEnv, strippedEnvKeys...)
    env = append(env,
        "STEAM_COMPAT_DATA_PATH="+compatDataPath,
        "STEAM_COMPAT_CLIENT_INSTALL_PATH="+steamInstallPath,
        "SteamGameId="+steamGameId,
        "SteamAppId=0",
        "WINEDLLOVERRIDES=dxgi=n",
    )
    if verbose {
        env = append(env, "PROTON_LOG=1")
    }
    return env
}
```

### Pattern 4: Data Flow Through Pipeline

**What:** New fields on LaunchState and LaunchConfig to carry Steam installation path and game ID.

```
LaunchState additions:
  SteamInstallPath string  // Detected Steam root directory (or "")
  SteamGameId      string  // Non-Steam shortcut app ID (or "0")

LaunchConfig additions:
  SteamInstallPath string
  SteamGameId      string
```

### Anti-Patterns to Avoid

- **Hardcoding the app ID:** The non-Steam game app ID is a CRC32-based hash of the shortcut name + exe path. It changes if the user renames the Steam shortcut or moves the cluckers binary. Always detect dynamically.
- **Making Steam detection a hard requirement:** Not all users will have Steam installed (they may have added Proton-GE manually). Steam detection failure should be non-fatal -- fall back to `SteamGameId=0` and `STEAM_COMPAT_CLIENT_INSTALL_PATH=""` with a verbose warning.
- **Reading from running Steam process:** Do not inspect `/proc` or similar. File-based detection is deterministic and does not require Steam to be running.
- **Modifying shortcuts.vdf:** This phase only reads shortcuts.vdf. Never write to Steam's data files.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Binary VDF parsing | Full VDF parser | Existing `findCluckersAppID()` byte-scanning approach | Binary VDF format is complex. The existing targeted byte-scan in `deckconfig.go` finds exactly what's needed (exe field containing "cluckers" + corresponding appid field). No general parser needed. |
| Steam app ID calculation | CRC32 hash of shortcut properties | Read from `shortcuts.vdf` directly | Steam calculates the app ID using a specific algorithm. Rather than reimplementing it, read the authoritative value from Steam's own data file. |
| Gamescope X11 property setting | Custom X11 property manipulation | Proton's built-in window class hints | Proton's Wine patches in `dlls/winex11.drv/window.c` already read `SteamAppId` env var and set X11 class hints as `steam_app_{id}`. Setting the env var is sufficient -- no need to manipulate X11 properties directly from Go. |

**Key insight:** The entire Gamescope integration mechanism works through environment variables that Proton reads and translates into X11 window properties. The launcher's job is to set the right env vars -- Proton/Wine handles the rest.

## Common Pitfalls

### Pitfall 1: SteamGameId=0 Means "Unknown" to Gamescope
**What goes wrong:** With `SteamGameId=0`, Gamescope has no way to associate a newly created window (after ServerTravel) with the same game. It treats the new window as an unrelated application.
**Why it happens:** Proton's Wine reads `SteamAppId` and sets X11 window class to `steam_app_0`, which Gamescope cannot match to any known game. When the old window is destroyed and a new one created, Gamescope sees `steam_app_0` but has no state to carry over.
**How to avoid:** Set `SteamGameId` to the actual non-Steam shortcut app ID from `shortcuts.vdf`. Gamescope then recognizes all windows with the same game ID as belonging to the same application.
**Warning signs:** Controller works in lobby but stops in match (the exact symptom we're fixing).

### Pitfall 2: Steam Not Installed or shortcuts.vdf Missing
**What goes wrong:** `FindSteamInstall()` returns empty, `findCluckersAppID()` has no file to read, falls back to `SteamGameId=0`.
**Why it happens:** User installed Proton-GE manually without Steam, or has not yet added cluckers as a non-Steam game.
**How to avoid:** Graceful degradation. All detection failures are non-fatal. Log verbose warnings. The launcher still works -- controller tracking on Steam Deck will not persist through ServerTravel, but the game will launch and work otherwise.
**Warning signs:** Verbose output shows "Steam installation not found" or "Cluckers shortcut not found in Steam."

### Pitfall 3: Multiple Steam Userdata Directories
**What goes wrong:** User has multiple Steam accounts. Each has its own `userdata/<id>/` directory with potentially different shortcuts.
**Why it happens:** Steam supports multiple user accounts, each with separate shortcuts.vdf.
**How to avoid:** Scan all userdata directories and use the first one that contains a cluckers shortcut. This matches the pattern already used in `deployDeckControllerLayout()`.

### Pitfall 4: STEAM_COMPAT_CLIENT_INSTALL_PATH Pointing to Wrong Location
**What goes wrong:** Proton tries to load `steamclient.so` from the detected path but it's the wrong Steam installation (e.g., Flatpak vs native).
**Why it happens:** Flatpak Steam's libraries are inside the Flatpak sandbox and may not be accessible from the host.
**How to avoid:** Verify that `ubuntu12_32/steamclient.so` exists at the detected path. If not accessible, fall back to empty string (current behavior). Proton handles missing `steamclient.so` gracefully -- it just skips Steam client integration.
**Warning signs:** Proton log warnings about missing steamclient.so.

### Pitfall 5: Symlink Resolution for Steam Directories
**What goes wrong:** `~/.steam/steam` is a symlink to `~/.local/share/Steam`. Without resolving symlinks, we might detect the same installation twice or miss it entirely.
**Why it happens:** Steam Deck and many Linux distros use symlinks for the Steam directory structure.
**How to avoid:** Use `filepath.EvalSymlinks()` and deduplicate by resolved path. This pattern already exists in `symlinkResolvedDirs()` in `detect.go`.

### Pitfall 6: Proton Wine Does Not Set STEAM_GAME X11 Property from SteamGameId
**What goes wrong:** Setting `SteamGameId` alone may not cause the STEAM_GAME X11 atom to be set on game windows.
**Why it happens:** The controller debugging notes state "Wine does NOT set STEAM_GAME from SteamGameId env var (contrary to expectations)." Proton Wine reads `SteamAppId` for window class hints (`steam_app_{id}`), which is a different mechanism than the `STEAM_GAME` X11 atom. The `STEAM_GAME` atom may only be set by Steam itself when it manages the game process.
**How to avoid:** This is the highest-risk pitfall. Two mitigations: (1) Set both `SteamGameId` AND `SteamAppId` to the non-Steam app ID (currently `SteamAppId=0`). The Wine source confirms `SteamAppId` is used for X11 class hints. (2) Hardware validation (CTRL-03) is required to confirm whether this approach works. If it does not, the fallback is that the game must be launched as a non-Steam game through Steam itself (which is the established workaround).
**Warning signs:** Despite correct env vars, controller still drops during ServerTravel on Steam Deck.

## Code Examples

### Example 1: Steam Installation Detection

```go
// internal/wine/steamdir.go

//go:build linux

package wine

import (
    "os"
    "path/filepath"
)

// steamInstallDirs returns directories to check for Steam installations,
// ordered by likelihood (most common first).
func steamInstallDirs(home string) []string {
    return []string{
        filepath.Join(home, ".local", "share", "Steam"),
        filepath.Join(home, ".steam", "steam"),
        filepath.Join(home, ".steam", "root"),
        filepath.Join(home, ".var", "app", "com.valvesoftware.Steam",
            "data", "Steam"),
        filepath.Join(home, "snap", "steam", "common", ".local",
            "share", "Steam"),
    }
}

// FindSteamInstall returns the Steam root directory, or "" if not found.
func FindSteamInstall() string {
    return findSteamInstall(userHome())
}

func findSteamInstall(home string) string {
    seen := make(map[string]bool)
    for _, dir := range steamInstallDirs(home) {
        resolved := resolveReal(dir)
        if seen[resolved] {
            continue
        }
        seen[resolved] = true
        if isSteamDir(resolved) {
            return resolved
        }
    }
    return ""
}

func isSteamDir(dir string) bool {
    markers := []string{
        filepath.Join(dir, "steam.sh"),
        filepath.Join(dir, "ubuntu12_32", "steamclient.so"),
    }
    for _, m := range markers {
        if _, err := os.Stat(m); err == nil {
            return true
        }
    }
    return false
}
```

### Example 2: App ID Resolution (Exporting Existing Logic)

```go
// internal/launch/deckconfig.go -- export the existing function

// FindCluckersAppID searches Steam's shortcuts.vdf files for a shortcut
// whose exe field contains "cluckers" and returns its app ID.
// Returns 0 if not found. Exported for use by pipeline env construction.
func FindCluckersAppID(data []byte) uint32 {
    return findCluckersAppID(data) // delegate to existing private function
}
```

### Example 3: Updated Pipeline Step

```go
// In pipeline_linux.go -- extend stepDetectProton or add new step

func stepResolveSteamIntegration(_ context.Context, state *LaunchState) error {
    // Detect Steam installation path.
    steamDir := wine.FindSteamInstall()
    if steamDir == "" {
        ui.Verbose("Steam installation not found, controller tracking may be limited", state.Config.Verbose)
        return nil // Non-fatal
    }
    state.SteamInstallPath = steamDir
    ui.Verbose(fmt.Sprintf("Steam: %s", steamDir), state.Config.Verbose)

    // Resolve non-Steam game app ID from shortcuts.vdf.
    pattern := filepath.Join(steamDir, "userdata", "*", "config", "shortcuts.vdf")
    matches, _ := filepath.Glob(pattern)
    for _, shortcutsPath := range matches {
        data, err := os.ReadFile(shortcutsPath)
        if err != nil {
            continue
        }
        if appID := FindCluckersAppID(data); appID != 0 {
            state.SteamGameId = fmt.Sprintf("%d", appID)
            ui.Verbose(fmt.Sprintf("Steam shortcut app ID: %s", state.SteamGameId), state.Config.Verbose)
            return nil
        }
    }

    ui.Verbose("Cluckers shortcut not found in Steam, using default game ID", state.Config.Verbose)
    return nil
}
```

### Example 4: Updated Environment Construction

```go
// In proton_env.go -- updated buildProtonEnvFrom

func buildProtonEnvFrom(baseEnv []string, compatDataPath, steamInstallPath, steamGameId string, verbose bool) []string {
    env := filterEnv(baseEnv, strippedEnvKeys...)

    // Default to "0" if not resolved.
    if steamGameId == "" {
        steamGameId = "0"
    }

    env = append(env,
        "STEAM_COMPAT_DATA_PATH="+compatDataPath,
        "STEAM_COMPAT_CLIENT_INSTALL_PATH="+steamInstallPath,
        "SteamGameId="+steamGameId,
        "SteamAppId="+steamGameId, // Set BOTH to same value for X11 class hints
        "WINEDLLOVERRIDES=dxgi=n",
    )

    if verbose {
        env = append(env, "PROTON_LOG=1")
    }

    return env
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| SteamGameId=0 (unknown game) | SteamGameId=<actual app ID> | Phase 7 | Enables Gamescope to track game windows across D3D recreation |
| STEAM_COMPAT_CLIENT_INSTALL_PATH="" (empty) | STEAM_COMPAT_CLIENT_INSTALL_PATH=<detected path> | Phase 7 | Allows Proton to load steamclient.so for full Steam integration |
| XInput proxy DLL approach | Environment variable approach via Proton | Phase 7 (replaces feature/deck-controller branch) | Proxy DLL failed because input loss is firmware-level. Env vars work at the right layer (Gamescope window tracking). |

**Deprecated/outdated:**
- **XInput proxy DLL (xinput_remap.c):** Tried in `feature/deck-controller` branch. Proved that the problem is firmware-level Steam Input reconfiguration, not application-level input filtering. Proxy remains as historical reference in `tools/`.
- **Direct Gamescope X11 property manipulation:** Tried via `xprop` commands. Does not work because Wine/Proton manages its own X11 properties. The correct approach is to set environment variables that Proton reads.
- **HID raw reader approach:** Proved that button bytes are zero at the hardware level during match. Not a viable fix path.

## Open Questions

1. **Does SteamGameId (or SteamAppId) actually cause Gamescope to track windows across recreation?**
   - What we know: Proton Wine reads `SteamAppId` and sets X11 window class to `steam_app_{id}` (confirmed in source). Previous debugging found "Wine does NOT set STEAM_GAME from SteamGameId env var."
   - What's unclear: Whether Gamescope uses the `steam_app_*` X11 class (set by Proton Wine) or the `STEAM_GAME` X11 atom (set by Steam client only) for window-to-game association during window recreation. These are two different mechanisms.
   - Recommendation: Implement the env var approach (it is the correct thing to do regardless), then validate on hardware. If it does not work, the fallback is requiring the game to be launched as a non-Steam game through Steam itself (which gives Steam full control over the game process lifecycle and sets all necessary atoms). **MEDIUM confidence** that env vars alone solve the problem. **HIGH confidence** that the implementation is correct regardless of whether it fully resolves CTRL-03.

2. **Should SteamAppId match SteamGameId or remain 0?**
   - What we know: Proton Wine uses `SteamAppId` for X11 class hints. The Proton Python script uses `SteamGameId` for per-game compatibility fixes and logging. Currently both are 0.
   - What's unclear: Whether setting `SteamAppId` to the non-Steam shortcut app ID could trigger unintended per-game Proton fixes (the proton script has game-specific workarounds keyed by SteamGameId).
   - Recommendation: Set `SteamAppId` to the resolved app ID (same as `SteamGameId`). The app ID for a non-Steam shortcut is a large 32-bit unsigned integer (e.g., 3928144816) that is extremely unlikely to collide with any real Steam app ID. **HIGH confidence** this is safe.

3. **What if the user has not added cluckers as a non-Steam game?**
   - What we know: `findCluckersAppID()` searches `shortcuts.vdf` for an exe containing "cluckers". If the user has not added cluckers to Steam, this returns 0.
   - What's unclear: Whether we should provide a `cluckers steam add` prerequisite step or just gracefully degrade.
   - Recommendation: Gracefully degrade to `SteamGameId=0`. The `cluckers steam add` command already exists and guides users through adding the shortcut. Add a user-facing hint: "For best Steam Deck controller support, add cluckers to Steam as a non-Steam game." **HIGH confidence** on this approach.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | None -- `go test ./...` |
| Quick run command | `go test ./internal/wine/ ./internal/launch/ -count=1` |
| Full suite command | `go test ./... -count=1` |
| Estimated runtime | ~5 seconds |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CTRL-01 | buildProtonEnvFrom sets SteamGameId to resolved app ID | unit | `go test ./internal/launch/ -run TestBuildProtonEnv_SetsSteamGameId -count=1` | Yes (needs update) |
| CTRL-01 | buildProtonEnvFrom sets SteamAppId to match SteamGameId | unit | `go test ./internal/launch/ -run TestBuildProtonEnv_SetsSteamAppId -count=1` | Yes (needs update) |
| CTRL-01 | buildProtonEnvFrom defaults SteamGameId to "0" when empty | unit | `go test ./internal/launch/ -run TestBuildProtonEnv_DefaultsSteamGameId -count=1` | No -- Wave 0 gap |
| CTRL-02 | FindSteamInstall finds native Steam | unit | `go test ./internal/wine/ -run TestFindSteamInstall_Native -count=1` | No -- Wave 0 gap |
| CTRL-02 | FindSteamInstall finds Flatpak Steam | unit | `go test ./internal/wine/ -run TestFindSteamInstall_Flatpak -count=1` | No -- Wave 0 gap |
| CTRL-02 | FindSteamInstall finds Snap Steam | unit | `go test ./internal/wine/ -run TestFindSteamInstall_Snap -count=1` | No -- Wave 0 gap |
| CTRL-02 | FindSteamInstall returns empty when no Steam | unit | `go test ./internal/wine/ -run TestFindSteamInstall_NotFound -count=1` | No -- Wave 0 gap |
| CTRL-02 | FindSteamInstall deduplicates symlinked paths | unit | `go test ./internal/wine/ -run TestFindSteamInstall_Dedup -count=1` | No -- Wave 0 gap |
| CTRL-02 | buildProtonEnvFrom sets STEAM_COMPAT_CLIENT_INSTALL_PATH | unit | `go test ./internal/launch/ -run TestBuildProtonEnv_SetsSTEAM_COMPAT_CLIENT_INSTALL_PATH -count=1` | Yes (needs update) |
| CTRL-01 | FindCluckersAppID extracts app ID from shortcuts.vdf | unit | `go test ./internal/launch/ -run TestFindCluckersAppID -count=1` | No -- Wave 0 gap (function exists but unexported, tests exist implicitly via deckconfig tests) |
| CTRL-03 | Controller buttons persist through lobby-to-match on Steam Deck | manual-only | Test on Steam Deck hardware | N/A -- requires hardware |

### Nyquist Sampling Rate
- **Minimum sample interval:** After every committed task, run: `go test ./internal/wine/ ./internal/launch/ -count=1`
- **Full suite trigger:** Before merging final task of any plan wave
- **Phase-complete gate:** Full suite green (`go test ./... -count=1`) + hardware validation on Steam Deck before verification
- **Estimated feedback latency per task:** ~3-5 seconds

### Wave 0 Gaps (must be created before implementation)
- [ ] `internal/wine/steamdir_test.go` -- covers CTRL-02 (FindSteamInstall for native, Flatpak, Snap, not found, symlink dedup)
- [ ] Update `internal/launch/proton_env_test.go` -- update existing tests for new function signature, add default SteamGameId test
- [ ] `internal/launch/deckconfig_test.go` -- covers CTRL-01 (FindCluckersAppID with test shortcuts.vdf data)

*(Note: CTRL-03 is hardware validation only. Cannot be automated.)*

## Sources

### Primary (HIGH confidence)
- Proton Wine source (`dlls/winex11.drv/window.c`, proton_9.0 branch) -- confirmed `SteamAppId` is read for X11 class hints `steam_app_{id}`, and `SteamGameId` is read for WM detection workarounds
- Existing cluckers codebase -- `internal/launch/proton_env.go`, `internal/launch/deckconfig.go`, `internal/wine/detect.go` -- current implementation and patterns
- Controller debugging record (`controller-debugging.md`) -- definitive root cause analysis of firmware-level input loss, HID raw dump proof, failed approaches documented

### Secondary (MEDIUM confidence)
- [Gamescope issue #416](https://github.com/ValveSoftware/gamescope/issues/416) -- GAMESCOPE_FOCUSED_APP behavior in Steam vs non-Steam mode, resolved by restricting property to Steam mode
- [Proton issue #9068](https://github.com/ValveSoftware/Proton/issues/9068) -- STEAM_COMPAT_CLIENT_INSTALL_PATH usage and limitations for standalone Proton
- [umu-launcher FAQ](https://github.com/Open-Wine-Components/umu-launcher/wiki/Frequently-asked-questions-(FAQ)) -- how umu-launcher handles SteamGameId and STEAM_GAME property for non-Steam games
- [Gamescope issue #1707](https://github.com/ValveSoftware/gamescope/issues/1707) -- window visibility and tracking issues

### Tertiary (LOW confidence)
- Whether `SteamAppId` X11 class hints are sufficient for Gamescope window tracking (vs `STEAM_GAME` atom set by Steam client) -- not definitively confirmed by any source. The debugging notes state Wine does NOT set STEAM_GAME from env vars. Hardware validation required.
- Whether setting `SteamAppId` to a non-Steam shortcut ID triggers any unwanted Proton per-game fixes -- theoretically safe (IDs don't overlap) but not explicitly confirmed.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new libraries, pure Go implementation using existing patterns
- Architecture: MEDIUM-HIGH -- detection patterns mirror existing `FindProtonGE()` / `protonSearchDirs()`, env var plumbing is straightforward
- Pitfalls: MEDIUM -- the core uncertainty is whether env vars alone cause Gamescope to track windows correctly (Pitfall 6). Everything else is well-understood.
- CTRL-03 resolution: LOW-MEDIUM -- high confidence the implementation is correct, but LOW confidence it fully resolves the controller issue without hardware validation. The STEAM_GAME atom vs steam_app_* class distinction is the key uncertainty.

**Research date:** 2026-02-25
**Valid until:** 2026-03-25 (Proton/Gamescope release cycle is rapid, but env var interface is stable)
