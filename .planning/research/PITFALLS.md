# Pitfalls Research

**Domain:** Switching from direct Wine to Proton launch pipeline in a Go game launcher
**Researched:** 2026-02-24
**Confidence:** HIGH (Proton source code verified, existing codebase analyzed, community issues documented)

## Critical Pitfalls

### Pitfall 1: Proton Overwrites Your WINEDLLOVERRIDES (Double-Override Trap)

**What goes wrong:**
The current `process_linux.go` sets `WINEDLLOVERRIDES=dxgi=n` before launching Wine. When switching to `proton run`, the Proton Python script builds its own `WINEDLLOVERRIDES` string internally -- setting overrides for DXVK DLLs (d3d9, d3d10, d3d11, dxgi), vkd3d-proton, and other components. Proton **replaces** any `WINEDLLOVERRIDES` you set in the environment. Your `dxgi=n` override silently vanishes. The game may crash or render incorrectly.

**Why it happens:**
Proton is designed to fully manage the DLL override chain. The `proton` script constructs `WINEDLLOVERRIDES` from its internal configuration and only merges user overrides if they are passed via a separate `PROTON_WINEDLLOVERRIDES` variable -- not via `WINEDLLOVERRIDES` directly. This is documented in a [Proton PR](https://github.com/ValveSoftware/Proton/pull/1705/files) but is not obvious from the Proton documentation.

**How to avoid:**
1. Never set `WINEDLLOVERRIDES` when invoking `proton run` -- Proton will clobber it
2. Use `PROTON_WINEDLLOVERRIDES="dxgi=n"` instead, which Proton prepends to its own overrides
3. Audit every environment variable the current `LaunchGame()` sets -- categorize each as "Proton manages this" vs. "we still need to set this"
4. For `WINEFSYNC=1`: Proton enables fsync/esync by default, so explicitly setting it is unnecessary and potentially conflicting

**Warning signs:**
- Game renders with wrong graphics backend after switching to Proton
- Wine debug logs show different DLL override chain than expected
- `WINEDEBUG=+dll` output shows Proton's overrides but not yours

**Phase to address:**
Phase 1 (core Proton launch pipeline). This is the first thing to change: replace `WINEDLLOVERRIDES` with `PROTON_WINEDLLOVERRIDES` and remove `WINEFSYNC` (Proton handles it).

---

### Pitfall 2: WINEPREFIX vs STEAM_COMPAT_DATA_PATH Confusion (Prefix Structure Mismatch)

**What goes wrong:**
The current code sets `WINEPREFIX=~/.cluckers/prefix` and launches Wine directly. When switching to `proton run`, you must set `STEAM_COMPAT_DATA_PATH` -- but the prefix lives at `$STEAM_COMPAT_DATA_PATH/pfx/`, NOT at `$STEAM_COMPAT_DATA_PATH` itself. If you point `STEAM_COMPAT_DATA_PATH` at the old `~/.cluckers/prefix/`, Proton will create `~/.cluckers/prefix/pfx/` as a NEW prefix inside your existing one. Two prefixes coexist, the game uses the wrong one, and DLLs are missing.

**Why it happens:**
Proton's `CompatData` class derives `prefix_dir` as `os.path.join(compat_data_path, "pfx")`. The actual Wine prefix is always one level deeper than `STEAM_COMPAT_DATA_PATH`. This is an internal Proton convention that differs from how raw Wine uses `WINEPREFIX`. The current codebase in `prefix.go` creates the prefix directly at `~/.cluckers/prefix/` -- there is no `pfx/` subdirectory.

**How to avoid:**
1. Set `STEAM_COMPAT_DATA_PATH=~/.cluckers/proton-data/` (a NEW path, not the old prefix path)
2. Let Proton create its own prefix at `~/.cluckers/proton-data/pfx/`
3. Do NOT set `WINEPREFIX` at all when using `proton run` -- Proton derives it from `STEAM_COMPAT_DATA_PATH`
4. If you set both `WINEPREFIX` and `STEAM_COMPAT_DATA_PATH`, they may conflict. Some Proton versions use `WINEPREFIX` from the environment, others derive it. Do not gamble -- set only `STEAM_COMPAT_DATA_PATH`.
5. Deprecate the old `~/.cluckers/prefix/` directory. Do not attempt to migrate it -- Proton prefixes have different internal structure (version tracking, tracked_files, DLL symlinks vs copies).

**Warning signs:**
- Directory structure shows `~/.cluckers/prefix/pfx/drive_c/` (double nesting)
- Proton logs show "creating new prefix" on every launch despite prefix existing
- DLL verification finds files in the old prefix but game fails because Proton uses the new one

**Phase to address:**
Phase 1. The new compat data path and prefix structure must be the first change, before any launch logic changes.

---

### Pitfall 3: Missing Required Environment Variables Crash Proton's Python Script

**What goes wrong:**
The `proton` script (Python 3) requires specific environment variables that Steam normally provides. Without them, the script crashes with `KeyError`. The minimum required set is:
- `STEAM_COMPAT_DATA_PATH` -- where to store the prefix (required, crashes without it)
- `STEAM_COMPAT_CLIENT_INSTALL_PATH` -- where Steam is installed (required, crashes without it)
- `SteamGameId` -- game identifier (needed for Proton's internal game tracking)

Currently `process_linux.go` sets none of these because it calls `wine64` directly.

**Why it happens:**
Proton was designed to be invoked by Steam, which sets dozens of environment variables before calling the `proton` script. When you call `proton run` outside of Steam, you must replicate the critical subset. The Proton script does no graceful error handling for missing variables -- it simply raises `KeyError`.

**How to avoid:**
1. Set all three required variables before invoking `proton run`:
   ```
   STEAM_COMPAT_DATA_PATH=~/.cluckers/proton-data
   STEAM_COMPAT_CLIENT_INSTALL_PATH=~/.local/share/Steam  (or detect actual Steam install path)
   SteamGameId=0                                           (dummy value for non-Steam games)
   ```
2. For `STEAM_COMPAT_CLIENT_INSTALL_PATH`: detect the Steam installation by checking common paths (`~/.local/share/Steam`, `~/.steam/steam`, Flatpak path). If Steam is not installed, this variable still must be set -- point it at the Proton-GE directory itself as a workaround, or use umu-launcher which handles this.
3. Also set `STEAM_COMPAT_INSTALL_PATH` (the game installation directory) for drive mappings
4. Test with `PROTON_LOG=1` to see Proton's initialization -- it will show what variables it reads and what defaults it uses

**Warning signs:**
- `KeyError: 'STEAM_COMPAT_CLIENT_INSTALL_PATH'` traceback on first test
- `KeyError: 'STEAM_COMPAT_DATA_PATH'` traceback
- Proton exits immediately with no visible error (traceback goes to stderr, which may be hidden)

**Phase to address:**
Phase 1. This is the first thing you will hit when replacing `wine64` with `proton run`. Build the environment variable setup as the very first step.

---

### Pitfall 4: Python 3 Dependency for proton Script (AppImage Bundling Headache)

**What goes wrong:**
The `proton` entry point is a Python 3 script (`#!/usr/bin/env python3`). The current AppImage bundles Proton-GE but only uses its `files/bin/wine64` binary directly -- no Python needed. Switching to `proton run` means invoking the Python script, which requires a Python 3 interpreter. On minimal systems (some container setups, certain SteamOS configurations, very stripped-down Arch installs), Python 3 may not be available. More critically, the AppImage must now bundle or depend on Python 3.

**Why it happens:**
Proton is not a compiled binary. It is a Python orchestration script that sets up the environment and then calls Wine internally. When you called `wine64` directly, you bypassed all of Proton's Python infrastructure. Switching to `proton run` means you now depend on Python 3 being available.

**How to avoid:**
1. **Preferred: Use umu-launcher** -- umu-launcher is a purpose-built tool for running Proton outside of Steam. It handles all the Steam Runtime container setup, environment variables, and Python dependencies. It is the community-endorsed way to run Proton for non-Steam games. Dependencies: Python 3 (which it manages via its own setup).
2. **Alternative: Bundle a minimal Python 3** in the AppImage (adds ~15-25 MB to AppImage size). Use `python-appimage` to create a relocatable Python 3.
3. **Alternative: Detect Python 3** at launch and give a clear error with install instructions if missing.
4. **Do NOT rewrite the proton script in Go** -- the script is complex (2400+ lines), tightly coupled to Proton internals, and changes with every Proton version. Maintaining a Go port would be a maintenance nightmare.
5. In AppRun, verify `python3` exists before attempting `proton run`, with a clear fallback error.

**Warning signs:**
- AppImage that worked with direct Wine fails on systems without Python 3
- Users report "proton: command not found" or "/usr/bin/env: 'python3': No such file or directory"
- AppImage size increases unexpectedly (Python bundled)

**Phase to address:**
Phase 2 (AppImage integration). Must be solved before distributing the Proton-based launcher to users. Consider umu-launcher as the path of least resistance.

---

### Pitfall 5: shm_launcher.exe Process Hierarchy Change Under Proton

**What goes wrong:**
The current launch chain is: `Go launcher -> wine64 shm_launcher.exe -> CreateProcessW(game.exe)`. Under Proton, the chain becomes: `Go launcher -> python3 proton -> wine64 shm_launcher.exe -> CreateProcessW(game.exe)`. Proton's script does additional setup before calling Wine (steam.exe stub, drive mappings, DLL setup). This changes the process tree. The Go launcher currently waits on the Wine process -- but with Proton, it must wait on the Python process, which waits on Wine, which waits on shm_launcher, which waits on the game. If any layer in this chain handles signals differently, the shared memory mapping may be cleaned up prematurely.

Additionally, Proton spawns a `steam.exe` stub process as part of its initialization. This stub may interfere with process tracking or change the wineserver state.

**Why it happens:**
Direct Wine execution is a simple parent-child relationship. Proton adds an orchestration layer (Python) and additional Wine processes (steam.exe stub). The process model becomes more complex, and signal propagation, exit code forwarding, and process lifetime guarantees are different.

**How to avoid:**
1. The shm_launcher.exe approach itself should still work -- `CreateFileMapping` and `OpenFileMapping` operate within the same wineserver instance regardless of whether Proton or direct Wine is used. The shared memory mechanism does not care about the Python wrapper.
2. Verify that the Go launcher waits on the correct process (the `proton run` Python process, not a background Wine process)
3. Test signal handling: send SIGINT to the Go process and verify the entire chain terminates cleanly
4. Proton's steam.exe stub runs briefly during initialization and exits before the game -- it should not interfere with shm_launcher. But verify this by checking `PROTON_LOG=1` output.
5. Add explicit process tree tracking or use `cmd.Process.Wait()` rather than relying on child process inheritance

**Warning signs:**
- Game launches but Go process returns immediately (orphaned game process)
- shm_launcher exits with code 0 but game has not started yet
- `ps aux | grep wine` shows unexpected process tree
- `PROTON_LOG=1` shows steam.exe stub errors

**Phase to address:**
Phase 1. Test shm_launcher.exe under Proton early -- this is the highest-risk integration point. If it works (likely), great. If not, you need a different SHM delivery mechanism.

---

### Pitfall 6: LD_LIBRARY_PATH Triple Collision (AppImage + Proton + Wine)

**What goes wrong:**
The current `process_linux.go` already strips `LD_LIBRARY_PATH` before launching Wine to prevent AppImage-bundled libraries from conflicting. But Proton sets its OWN `LD_LIBRARY_PATH` pointing to its `dist/lib64:dist/lib` directories. If the AppImage's LD_LIBRARY_PATH is not fully cleaned, Proton loads wrong libraries. If you strip LD_LIBRARY_PATH too aggressively, Proton cannot find its own libraries. There are now three competing library path sources: AppImage, Proton, and system.

**Why it happens:**
AppImage sets LD_LIBRARY_PATH in AppRun for the Go binary. The Go binary strips it before Wine. But with Proton, the Python script sets its own LD_LIBRARY_PATH before calling Wine. If the Python process inherits the AppImage's LD_LIBRARY_PATH (even partially), Proton's library resolution breaks because it finds AppImage libs instead of its own Wine/Proton libs.

**How to avoid:**
1. Continue stripping `LD_LIBRARY_PATH` entirely before invoking `proton run` (the current approach in process_linux.go is correct for this)
2. Proton's Python script will set its own LD_LIBRARY_PATH from scratch -- this is fine as long as it does not inherit AppImage paths
3. Also strip `LD_PRELOAD` if the AppImage sets it
4. Consider saving and restoring `LD_LIBRARY_PATH` as `ORIG_LD_LIBRARY_PATH` (Proton checks for this variable and may use it)
5. Test the full chain: AppImage -> Go launcher -> proton run -> Wine -> game, verifying library resolution at each stage with `ldd` or `LD_DEBUG=libs`

**Warning signs:**
- Wine crashes with "symbol lookup error" or "version mismatch" for core libraries
- Proton's Wine fails to start but direct Wine works
- `PROTON_LOG=1` shows library loading errors
- Game runs from tarball but not from AppImage

**Phase to address:**
Phase 2 (AppImage integration). The library path sanitization must be tested specifically in the AppImage context.

---

### Pitfall 7: Prefix Corruption During Migration (Old Prefix + New Proton Prefix Coexistence)

**What goes wrong:**
Users upgrading from v1.0 (direct Wine) to v1.1 (Proton) still have `~/.cluckers/prefix/` from the old approach. The new Proton approach creates `~/.cluckers/proton-data/pfx/`. If the migration is not explicit, users end up with both directories, confusion about which one is in use, and potentially wasted disk space (~1-2 GB per prefix). Worse, if the old code path is accidentally triggered (fallback to direct Wine), it uses the old prefix while Proton uses the new one -- registry settings, DLL overrides, and save data diverge.

**Why it happens:**
Proton prefixes have different internal structure from manually-created Wine prefixes:
- Proton uses `version` and `tracked_files` files for upgrade management
- Proton creates symlinks to its own DLL distribution rather than copying DLLs
- Proton runs `steam.exe` stub which writes Steam-specific registry keys
- Proton's DLL override chain (DXVK, vkd3d-proton) is managed differently from manual winetricks DXVK installation

You cannot simply rename the old prefix to `pfx/` under a compatdata directory and expect it to work with Proton.

**How to avoid:**
1. On first Proton launch, check for the old `~/.cluckers/prefix/` directory
2. If it exists, warn the user: "Migrating to Proton launch pipeline. Your old Wine prefix at ~/.cluckers/prefix/ is no longer used. It can be safely deleted to reclaim disk space."
3. Let Proton create a fresh prefix from its template -- do NOT attempt to migrate the old prefix
4. Set a marker file (e.g., `~/.cluckers/.proton-migrated`) to track migration state
5. Remove the old `prefix.go` code paths for manual winetricks setup once Proton is the only launch method

**Warning signs:**
- Both `~/.cluckers/prefix/` and `~/.cluckers/proton-data/` exist
- `verify.go`'s DLL check passes (checking old prefix) but game fails (Proton uses new prefix)
- Users report "I already set up the prefix, why is it downloading again?"

**Phase to address:**
Phase 1. The migration path must be designed alongside the new prefix structure.

---

### Pitfall 8: Gamescope Window Tracking Requires Steam Integration for Controller Lifecycle

**What goes wrong:**
The whole reason for switching to Proton is to fix Steam Deck controller input loss during UE3 ServerTravel (window recreation). But simply using `proton run` outside of Steam does NOT automatically give you Gamescope integration. Gamescope tracks game windows through Steam's process management. If the game is not launched through Steam (or at minimum registered with Gamescope as a game process), Gamescope may not track the window recreation during ServerTravel, and the controller input loss persists.

**Why it happens:**
Gamescope on Steam Deck uses Steam's process tree to determine which windows belong to "the game." When Steam launches a game, it tells Gamescope the process ID. When the game recreates its window (UE3 ServerTravel), Gamescope knows to track the new window because it belongs to a known game process tree. If the game is launched outside Steam (even with Proton), Gamescope may lose track of it during window recreation, reverting the controller to desktop mode.

Non-Steam games added to Steam get this integration automatically. Games launched entirely outside Steam do not.

**How to avoid:**
1. **Primary path:** Launch through Steam as a non-Steam game shortcut. The `cluckers steam add` command already creates this. Proton launch through Steam gives full Gamescope integration for free.
2. **Alternative:** Use `gamescope --steam` flags when launching outside Steam to register the process. This requires Gamescope to be available (it is on Steam Deck).
3. **Alternative:** Use umu-launcher, which replicates Steam's runtime container and may provide the necessary Gamescope integration signals.
4. **Do NOT assume** that `proton run` alone fixes the controller problem. The fix requires Gamescope integration, which requires either Steam or explicit Gamescope configuration.
5. Test on actual Steam Deck hardware with the exact ServerTravel scenario (lobby to match transition).

**Warning signs:**
- Controller works in lobby (same as before) but loses input on match start (same as before)
- `proton run` works on desktop Linux but controller still breaks on Steam Deck
- Gamescope logs show "unknown window" for the recreated game window

**Phase to address:**
Phase 1 must validate this assumption. If `proton run` alone does not fix controllers, the entire milestone strategy must pivot to "launch through Steam" as the primary path.

---

### Pitfall 9: Proton's steam.exe Stub Attempts Steam API Initialization

**What goes wrong:**
When Proton launches, it runs a `steam.exe` stub process that attempts to initialize the Steamworks API and connect to the native Steam client via `lsteamclient.dll`. For non-Steam games (like Realm Royale on Project Crown), this initialization either fails silently (acceptable) or produces errors/warnings that confuse users. In some cases, the stub may hang briefly while trying to connect to a Steam client that is not running.

**Why it happens:**
Proton's steam.exe stub is part of its initialization sequence. It writes Steam-related registry keys, sets up drive mappings, and configures the prefix for Steam integration. For games launched through Steam, this is transparent. For non-Steam games launched via `proton run`, the stub still runs but the Steam client may not be available (especially in the AppImage scenario where Steam is not installed).

**How to avoid:**
1. Set `SteamGameId=0` (or any non-zero dummy value) to satisfy the stub's requirements
2. Ensure the steam.exe stub timeout is handled gracefully -- if it hangs, it should not block the game launch indefinitely
3. Check if `PROTON_NO_STEAM_INTEGRATION` or similar flags exist in the Proton version you are bundling to suppress the stub
4. Test with Steam not running and verify the game still launches (it should -- the stub failure is usually non-fatal)
5. Suppress or redirect the stub's stderr output so users do not see confusing Steam API errors

**Warning signs:**
- "Steam API initialization failed" errors in Wine output
- 5-10 second delay at launch before the game window appears (stub timeout)
- Game works when Steam is running but fails when Steam is closed

**Phase to address:**
Phase 1. Test early with Steam not running on the target system.

---

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Setting `WINEPREFIX` alongside `STEAM_COMPAT_DATA_PATH` | "Safety net" for prefix location | Proton versions disagree on which takes precedence; creates subtle bugs where prefix changes between Proton updates | Never -- use only `STEAM_COMPAT_DATA_PATH` |
| Hardcoding `STEAM_COMPAT_CLIENT_INSTALL_PATH` to a single path | Works on developer's machine | Fails on Flatpak Steam, Snap Steam, or systems with non-standard Steam installs | Only if detection code is in the backlog for Phase 2 |
| Keeping the old direct-Wine fallback code | Users without Proton can still launch | Two code paths to maintain, test, and debug; env var conflicts between the two paths; DLL verification logic differs between them | Phase 1 (transition period), remove by Phase 3 |
| Calling `proton run` directly instead of using umu-launcher | No additional dependency | Missing Steam Runtime container isolation, no game-specific fixes database, no automatic environment setup | MVP only -- evaluate umu-launcher for production |
| Setting `SteamGameId=0` | Quick workaround for missing ID | May trigger unexpected behavior in Proton's game-specific fix logic or logging | Acceptable permanently for a non-Steam game, but log the value |
| Not stripping `ORIG_LD_LIBRARY_PATH` | Proton "might need it" | AppImage library paths leak into Proton through this variable | Never -- strip it alongside `LD_LIBRARY_PATH` |

## Integration Gotchas

Common mistakes when connecting Proton's systems to the existing launcher.

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Proton environment variables | Setting `WINEDLLOVERRIDES` directly | Use `PROTON_WINEDLLOVERRIDES` -- Proton replaces the direct variable |
| Proton environment variables | Setting `WINEFSYNC=1` explicitly | Remove it -- Proton enables fsync/esync/ntsync automatically based on kernel support |
| Proton prefix path | Setting `STEAM_COMPAT_DATA_PATH` to the Wine prefix directory | Set it to the PARENT of where you want the prefix -- Proton adds `pfx/` subdirectory |
| Proton + Wine binary | Calling Proton's `wine64` binary directly AND using `proton run` | Choose one. `proton run` invokes wine64 internally. Calling wine64 directly bypasses all Proton setup. |
| shm_launcher.exe path conversion | Using `wine.LinuxToWinePath()` for paths passed to Proton | Proton may handle path conversion differently. Test that `Z:\tmp\...` paths still work under Proton's Wine. If Proton remaps drives, paths may break. |
| Process waiting | Waiting for the `wine64` process PID | Wait for the `proton run` process (Python). It internally waits for Wine, which waits for shm_launcher, which waits for the game. |
| Prefix verification | Running `verify.go`'s DLL check on the old prefix path | Update DLL verification to check `$STEAM_COMPAT_DATA_PATH/pfx/drive_c/...` -- or better, skip manual DLL verification entirely since Proton manages its own DLLs |

## Performance Traps

Patterns that work at small scale but fail as usage grows.

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Proton prefix creation on first launch | 15-60 second delay as Proton copies template, runs wineboot, installs DLLs | Warn user that first launch takes longer. Show progress. Cache the prefix. | First launch only, but users may think the app is frozen |
| Proton steam.exe stub on every launch | 2-5 second extra delay compared to direct Wine | This is inherent to Proton's design. Cannot be eliminated. Warn users or show a spinner. | Every launch |
| Python 3 startup overhead | ~200-500ms added to launch time (Python interpreter startup + proton script parsing) | Negligible for game launch. Do not optimize. | Never a real problem |
| Prefix upgrade on Proton-GE update | Proton detects version mismatch and runs migration logic, adding 10-30 seconds to first launch after update | Inform users: "Updating Proton compatibility data..." Do not kill the process during this phase. | After every Proton-GE version update |

## Security Mistakes

Domain-specific security issues for the Proton migration.

| Mistake | Risk | Prevention |
|---------|------|------------|
| Setting `STEAM_COMPAT_CLIENT_INSTALL_PATH` to a writable temp directory | Proton may read or write files to this path; a malicious actor could plant files there | Point to the actual Steam install or a read-only fallback directory |
| Bundling an outdated Proton-GE in the AppImage | Known Wine/Proton CVEs accumulate; game files run under a vulnerable Wine | Pin to a specific Proton-GE version and update it in AppImage releases. Document the Proton-GE version in release notes. |
| Proton prefix in a world-readable directory | Other users on the system can read game credentials or session data from the prefix's registry | Ensure `~/.cluckers/proton-data/` and all contents are 0700 |
| Running Proton's Python script with elevated privileges | Proton downloads and executes files as part of prefix setup | Never run as root. Check early and abort. |

## UX Pitfalls

Common user experience mistakes in the Proton migration.

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Silent prefix migration (old disappears, new appears) | User has no idea what happened to their "setup" | Inform user: "Switching to Proton launch mode. Creating new compatibility data..." |
| No explanation of first-launch delay | User kills the process during Proton prefix creation | Show explicit message: "Setting up Proton (one-time setup, may take 30-60 seconds)..." |
| "Python not found" error with no context | User has no idea why a game launcher needs Python | Error message: "Proton requires Python 3. Install with: [distro-specific command]" |
| Proton steam.exe errors in console output | User sees "Steam API failed" and thinks something is broken | Suppress or filter steam.exe stub warnings from user-visible output |
| Both old and new prefix directories exist | User confusion about which is active, disk space waste | On migration, inform user the old prefix can be deleted. Provide a `cluckers cleanup` command. |

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

- [ ] **Proton launch works:** Test with Steam NOT running -- steam.exe stub must not hang or prevent game launch
- [ ] **Proton launch works:** Test from AppImage (not just development build) -- LD_LIBRARY_PATH must be clean
- [ ] **Proton launch works:** Test on Steam Deck in Gaming Mode -- Python 3 must be available, no interactive prompts
- [ ] **Controller fix validated:** Test the actual ServerTravel scenario (lobby to match) on Steam Deck -- do not assume `proton run` alone fixes it
- [ ] **Prefix isolation verified:** Verify that `~/.cluckers/proton-data/pfx/` is created, NOT `~/.cluckers/prefix/pfx/`
- [ ] **DLL verification updated:** `verify.go` must check the new prefix path, or be removed if Proton manages DLLs
- [ ] **Old prefix handling:** `stepEnsurePrefix` and `stepVerifyPrefix` must be updated or replaced, not silently bypassed
- [ ] **LinuxToWinePath still works:** Verify that `Z:\tmp\...` paths work under Proton's Wine (Z: drive mapping must exist)
- [ ] **WINEDLLOVERRIDES migrated:** Confirm `dxgi=n` is applied via `PROTON_WINEDLLOVERRIDES`, not the direct variable
- [ ] **winetricks removed from flow:** Proton manages its own DXVK, vcruntime, etc. -- winetricks must NOT run on a Proton-managed prefix
- [ ] **Fallback to direct Wine:** If Proton launch fails, does the launcher fall back gracefully or crash? Decide on strategy.

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Proton prefix corrupted | LOW | Delete `~/.cluckers/proton-data/`, Proton recreates on next launch |
| Wrong STEAM_COMPAT_DATA_PATH set | LOW | Fix the path in code, delete the misplaced prefix directory |
| LD_LIBRARY_PATH leak from AppImage | MEDIUM | Add LD_LIBRARY_PATH stripping, rebuild AppImage, test again |
| Old and new prefix both exist | LOW | `cluckers cleanup` or manual deletion of `~/.cluckers/prefix/` |
| Python 3 not found in AppImage | HIGH | Must rebuild AppImage with bundled Python, OR switch to umu-launcher, OR fallback to direct Wine |
| Controller still broken under Proton | HIGH | Pivot strategy: must launch through Steam as non-Steam game, not standalone `proton run` |
| WINEDLLOVERRIDES being clobbered | LOW | Switch to `PROTON_WINEDLLOVERRIDES`, redeploy |
| Proton steam.exe stub hangs | MEDIUM | Set timeout on proton process, investigate STEAM_COMPAT_CLIENT_INSTALL_PATH, test without Steam |

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls.

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| WINEDLLOVERRIDES clobbered | Phase 1 (core pipeline) | Run with `WINEDEBUG=+dll`, verify dxgi override is applied |
| WINEPREFIX vs STEAM_COMPAT_DATA_PATH | Phase 1 (core pipeline) | Check directory structure: `proton-data/pfx/drive_c/` exists, NOT `prefix/pfx/` |
| Missing env vars crash Proton | Phase 1 (core pipeline) | `proton run` succeeds without Steam running, no KeyError tracebacks |
| Python 3 dependency | Phase 2 (AppImage) | AppImage launch on minimal system without Python 3 -- either works or gives clear error |
| shm_launcher process hierarchy | Phase 1 (core pipeline) | Game launches, shm_launcher provides bootstrap, game reads it successfully |
| LD_LIBRARY_PATH triple collision | Phase 2 (AppImage) | AppImage -> Proton -> game works. `ldd` on Wine shows Proton's libs, not AppImage's |
| Prefix corruption during migration | Phase 1 (core pipeline) | Upgrade from v1.0 to v1.1, verify old prefix not modified, new prefix created |
| Gamescope window tracking | Phase 1 (validation) | Controller works through ServerTravel on Steam Deck. If not, pivot to Steam-launched path |
| steam.exe stub issues | Phase 1 (core pipeline) | Launch with Steam closed, game starts within 15 seconds, no hang |

## Sources

- [Proton source code](https://github.com/ValveSoftware/Proton) -- Python entry point, CompatData class, prefix management
- [Proton Wine Prefix Management (DeepWiki)](https://deepwiki.com/ValveSoftware/Proton/2.2-wine-prefix-management) -- Detailed prefix lifecycle, version tracking, upgrade logic
- [Proton WINEDLLOVERRIDES PR #1705](https://github.com/ValveSoftware/Proton/pull/1705/files) -- How Proton merges user DLL overrides via PROTON_WINEDLLOVERRIDES
- [Running Proton outside Steam (gist)](https://gist.github.com/michaelbutler/f364276f4030c5f449252f2c4d960bd2) -- Required environment variables and prefix structure
- [STEAM_COMPAT_CLIENT_INSTALL_PATH issue #9068](https://github.com/ValveSoftware/Proton/issues/9068) -- steam.exe stub hardcoded path problem
- [Non-Steam game focus issue #8513](https://github.com/ValveSoftware/steam-for-linux/issues/8513) -- Gamescope focus tracking for non-Steam games
- [umu-launcher](https://github.com/Open-Wine-Components/umu-launcher) -- Unified launcher for Proton without Steam dependency
- [WINEESYNC/WINEFSYNC override issue #3761](https://github.com/ValveSoftware/Proton/issues/3761) -- Sync variable conflicts between system and Proton
- [Non-Steam Proton compat paths issue #8103](https://github.com/ValveSoftware/steam-for-linux/issues/8103) -- Missing compat paths for non-Steam games
- [Proton FAQ](https://github.com/ValveSoftware/Proton/wiki/Proton-FAQ) -- Official FAQ on prefix management and environment
- [Wine shared memory forum](https://forum.winehq.org/viewtopic.php?t=36968) -- CreateFileMapping behavior under Wine
- [simshmbridge](https://github.com/Spacefreak18/simshmbridge) -- Reference implementation for shared memory in Wine/Proton
- [Python3 requirement for Steam Play](https://steamcommunity.com/app/221410/discussions/8/3276824275008597014/) -- Python 3 as Proton dependency
- [Proton launch outside Steam (Steam community)](https://steamcommunity.com/discussions/forum/11/4031347072444352282/) -- Community discussion on using Proton standalone
- [Non-Steam game API initialization #10256](https://github.com/ValveSoftware/steam-for-linux/issues/10256) -- steam.exe stub Steam API failures
- [Gamescope ArchWiki](https://wiki.archlinux.org/title/Gamescope) -- Gamescope configuration and limitations
- [LD_LIBRARY_PATH AppImage issues](https://github.com/AppImage/AppImageKit/issues/126) -- Library path conflicts in AppImage
- [Proton LD_LIBRARY_PATH issue #6475](https://github.com/ValveSoftware/steam-for-linux/issues/6475) -- Non-Steam games not inheriting runtime LD_LIBRARY_PATH
- Project codebase: `internal/launch/process_linux.go`, `internal/wine/prefix.go`, `internal/wine/detect.go`, `internal/wine/verify.go`, `deploy/AppRun`

---
*Pitfalls research for: Switching from direct Wine to Proton launch pipeline*
*Researched: 2026-02-24*
