# Phase 9: Steam Deck Input (again) - Research

**Researched:** 2026-02-27
**Domain:** Steam-managed Proton launch, Steam shortcuts.vdf binary format, Gamescope window tracking, Steam Input controller persistence
**Confidence:** MEDIUM

## Summary

Phase 9 addresses CTRL-03 (controller buttons persist through lobby-to-match transition on Steam Deck) using a fundamentally different approach from Phase 7/7.1. Instead of trying to work around Steam Input at the evdev/XInput layer (which Phase 7.1 proved impossible across 14 hardware deploys), this phase leverages Steam's own game management: the game is added as a non-Steam shortcut pointing to `shm_launcher.exe`, Proton is forced as the compatibility tool, and Steam fully manages the Proton lifecycle including Gamescope window tracking and Steam Input state.

The thesis is that when Steam manages the game like a "real" Steam game (proper shortcut with appid, Proton forced, launch options with `%command%`), Gamescope will track the window through UE3 ServerTravel's D3D recreation, and Steam Input will maintain game-mode controller configuration. This is untested on hardware -- Phase 7.1's Deploy 14b tested "Steam-managed Proton launch" but with `cluckers prep && %command%` (not a proper shortcut with forced Proton). The key difference is that Deploy 14b may not have had Proton forced as compatibility tool via Steam's UI, or the shortcut/appid may not have been properly configured for Gamescope tracking.

The existing `cluckers prep` command and `shm_launcher.exe` infrastructure are fully implemented. The main work involves: (1) removing the known-crash `WINEDLLOVERRIDES=dxgi=n` from multiple locations, (2) automating or improving the Steam shortcut creation (`cluckers steam add`), (3) optionally automating `steam://rungameid/` launch from CLI, and (4) hardware validation.

**Primary recommendation:** Clean up WINEDLLOVERRIDES references, improve `cluckers steam add` to automate shortcuts.vdf writing (or provide clear step-by-step instructions if automation proves too fragile), test on hardware. The approach is sound in principle but hardware validation is the only way to confirm.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **Primary approach: Steam-managed Proton launch** -- `cluckers prep` writes auth/token/bootstrap files, then Steam launches `shm_launcher.exe` via Proton as a non-Steam game shortcut. Steam/Gamescope fully manage the game lifecycle, which is how real Steam games avoid the ServerTravel controller disconnect.
- **Secondary: Gamescope window tracking** -- If Steam-managed launch alone doesn't fix controller, investigate X11 property manipulation or Gamescope configuration to ensure the new D3D window is recognized after ServerTravel.
- **Last resort: HID feature report injection** -- Reverse-engineer Steam's HID commands to controller firmware to keep game mode active during ServerTravel. Only attempted if first two approaches fail.
- Multiple approaches can be tried within this phase -- the goal is "do whatever it takes" to get controller working.
- `cluckers prep` already exists and is fully implemented.
- `shm_launcher.c` already supports reading launch-config.txt -- no C code changes expected.
- Steam shortcut needs to be created pointing to `~/.cluckers/bin/shm_launcher.exe` with Proton-GE forced as compatibility tool.
- The prep command's help text references `WINEDLLOVERRIDES=dxgi=n` which is KNOWN TO CRASH -- must be removed. Launch options should be: `/path/to/cluckers prep && %command%` with NO WINEDLLOVERRIDES.
- **Hard requirement**: Controller buttons (A/B/X/Y, bumpers, triggers) persist through ServerTravel on Steam Deck. CTRL-03 must be fully satisfied.
- **Hardware validation required**: Phase 9 is NOT complete until tested on actual Steam Deck hardware and controller works through a lobby-to-match transition.
- **One-time manual setup acceptable**: If automating shortcuts.vdf proves too fragile, falling back to printed instructions for initial Steam shortcut setup is acceptable.

### Claude's Discretion
- **Launch mode selection**: Whether `cluckers launch` auto-detects Steam Deck and uses Steam-managed mode, or keeps both direct-proton and Steam-managed as separate paths. Recommendation from user: auto-detect Deck hardware, use Steam-managed on Deck, direct proton on desktop Linux.
- **CLI triggers Steam vs prep-only**: Whether `cluckers launch` programmatically triggers the Steam shortcut (via `steam://rungameid/`) or the user launches from Steam. Recommendation from user: CLI triggers Steam launch after prep, for a seamless single-command experience.
- **Shortcut automation vs instructions**: Whether `cluckers steam add` modifies `shortcuts.vdf` directly or prints instructions. Recommendation from user: automate shortcuts.vdf modification.
- **Fallback behavior**: Whether direct `proton run` remains as a fallback. Recommendation from user: keep both modes -- direct proton run continues working for desktop Linux users who don't need Steam-managed launch.

### Deferred Ideas (OUT OF SCOPE)
- Windows Steam integration improvements -- separate concern, not related to Deck controller
- External USB/BT controller documentation -- only if all approaches fail and we need a workaround doc
- Steam community bug report for ServerTravel -- worth doing regardless but not a code deliverable
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| CTRL-03 | Controller buttons persist through lobby-to-match transition on Steam Deck (validated on hardware) | Steam-managed Proton launch via non-Steam shortcut to shm_launcher.exe. Steam manages Gamescope window tracking and Steam Input state. Must have proper shortcut appid and forced Proton compatibility tool. Hardware-only validation gate. |
</phase_requirements>

## Standard Stack

### Core

No new external dependencies required. All implementation uses existing Go stdlib and project infrastructure.

| Component | Version | Purpose | Why Standard |
|-----------|---------|---------|--------------|
| Go `encoding/binary` | stdlib | Binary VDF shortcuts.vdf reading/writing | Already used in `deckconfig.go` for `findCluckersAppID()`. Extending to full VDF read/write avoids external dependency. |
| Go `os/exec` | stdlib | Invoking `steam steam://rungameid/` for CLI-triggered launch | Already used throughout codebase for process launching |
| Go `hash/crc32` | stdlib | Computing non-Steam game shortcut appid for new entries | Standard CRC32 used by Steam's shortcut ID algorithm |

### Supporting

| Component | Version | Purpose | When to Use |
|-----------|---------|---------|-------------|
| `wine.IsSteamDeck()` | existing | Auto-detect Steam Deck for launch mode selection | Already in `internal/wine/detect.go` |
| `wine.FindSteamInstall()` | existing | Locate Steam root for userdata/shortcuts.vdf | Already in `internal/wine/steamdir.go` |
| `findCluckersAppID()` | existing | Parse shortcuts.vdf to find existing Cluckers shortcut | Already in `internal/launch/deckconfig.go` |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled binary VDF writer | `github.com/Wakeful-Cloud/vdf` Go library | External dep, adds to binary. Pure Go, no CGO. Key limitation: "order of key-value's are not preserved" which may corrupt shortcuts.vdf if Steam expects ordered fields. **Recommendation: hand-roll** -- the format is simple (3 field types), project already has a binary VDF reader (`findCluckersAppID`), and preserving exact byte layout avoids Steam overwriting our entry. |
| Hand-rolled binary VDF writer | `github.com/TimDeve/valve-vdf-binary` Go library | Has `ParseShortcuts` but limited write support. Same ordering concern. |
| `steam://rungameid/` via `os/exec` | Direct `proton run` on Deck | Loses Steam-managed lifecycle which is the entire fix. Only as fallback for desktop Linux. |
| Automating shortcuts.vdf | Printed instructions only | Eliminates complexity but adds manual step. Acceptable per user decision as fallback. |

### No Installation Needed

All dependencies are already in `go.mod`. No new `go get` commands required.

## Architecture Patterns

### Recommended Code Changes

```
internal/cli/
  steam_linux.go           # MODIFY: improve shortcut creation, remove WINEDLLOVERRIDES
  prep_linux.go            # MODIFY: remove WINEDLLOVERRIDES from help text
  launch_linux.go          # NEW or MODIFY: auto-detect Deck, trigger Steam launch

internal/launch/
  proton_env.go            # MODIFY: remove WINEDLLOVERRIDES=dxgi=n
  proton_env_test.go       # MODIFY: update tests for removed WINEDLLOVERRIDES
  prep.go                  # Already complete, no changes expected
  shortcuts.go             # NEW (Linux-only): binary VDF read/write for shortcuts.vdf
  shortcuts_test.go        # NEW: unit tests for VDF read/write

internal/wine/
  detect.go                # No changes (IsSteamDeck already exists)
  steamdir.go              # No changes (FindSteamInstall already exists)
```

### Pattern 1: Steam-Managed Launch Flow

**What:** On Steam Deck, `cluckers launch` runs the prep pipeline, then invokes `steam steam://rungameid/<appid>` to trigger Steam to launch the non-Steam shortcut with Proton.

**When to use:** When `wine.IsSteamDeck()` returns true and a Cluckers shortcut exists in Steam.

**Example:**
```go
// In cluckers launch on Steam Deck:
// 1. Run prep pipeline (auth, tokens, bootstrap, update, write config)
// 2. Find Cluckers shortcut appid from shortcuts.vdf
// 3. Calculate BPID = (appid << 32) | 0x02000000
// 4. exec: steam steam://rungameid/<BPID>

func launchViaSteam(appID uint32) error {
    bpid := (uint64(appID) << 32) | 0x02000000
    url := fmt.Sprintf("steam://rungameid/%d", bpid)
    cmd := exec.Command("steam", url)
    return cmd.Start() // Don't wait -- Steam manages the game lifecycle
}
```

### Pattern 2: Binary VDF Shortcuts.vdf Writing

**What:** Read existing shortcuts.vdf, append a new shortcut entry, write back.

**When to use:** `cluckers steam add` when no existing Cluckers shortcut is found.

**Binary VDF format:**
```
File structure:
  \x00shortcuts\x00     -- header
  \x00<index>\x00       -- entry index (string: "0", "1", ...)
    \x02appid\x00<4 bytes LE>           -- appid (int32)
    \x01AppName\x00<string>\x00         -- display name
    \x01Exe\x00<string>\x00             -- quoted exe path
    \x01StartDir\x00<string>\x00        -- quoted start directory
    \x01icon\x00<string>\x00            -- icon path (can be empty)
    \x01ShortcutPath\x00<string>\x00    -- shortcut path (empty)
    \x01LaunchOptions\x00<string>\x00   -- launch options string
    \x02IsHidden\x00<4 bytes>           -- boolean as int32
    \x02AllowDesktopConfig\x00<4 bytes> -- boolean as int32
    \x02AllowOverlay\x00<4 bytes>       -- boolean as int32
    \x02OpenVR\x00<4 bytes>             -- boolean as int32
    \x02LastPlayTime\x00<4 bytes>       -- unix timestamp
    \x00tags\x00\x08                    -- empty tags section
  \x08                                  -- end of entry
  \x08                                  -- end of shortcuts
```

**Example shortcut values:**
```go
shortcut := Shortcut{
    AppName:       "Realm Royale (Cluckers)",
    Exe:           fmt.Sprintf(`"%s"`, shmLauncherPath), // ~/.cluckers/bin/shm_launcher.exe
    StartDir:      fmt.Sprintf(`"%s"`, config.BinDir()),
    LaunchOptions: fmt.Sprintf("%s prep && %%command%%", cluckersPath),
    // Leave AppID as 0 -- Steam will assign one on next restart
}
```

**Critical: Steam assigns the appid.** When we write appid=0, Steam will generate a proper appid on restart and write it back to shortcuts.vdf. We then read it from there for `steam://rungameid/` launches. This matches the behavior documented in [ValveSoftware/steam-for-linux#9463](https://github.com/ValveSoftware/steam-for-linux/issues/9463).

### Pattern 3: Launch Mode Auto-Detection

**What:** `cluckers launch` detects the platform and selects the appropriate launch path.

**When to use:** Every launch on Linux.

```go
func selectLaunchMode(cfg *config.Config) LaunchMode {
    if !wine.IsSteamDeck() {
        return DirectProtonMode  // Desktop Linux: existing proton run path
    }
    // Steam Deck: check if shortcut exists
    appID := findExistingShortcutAppID()
    if appID == 0 {
        ui.Warn("No Steam shortcut found. Run 'cluckers steam add' first.")
        return DirectProtonMode  // Fallback
    }
    return SteamManagedMode  // Deck with shortcut: prep + steam://rungameid
}
```

### Pattern 4: Backup Before Modify (shortcuts.vdf)

**What:** Always back up shortcuts.vdf before writing.

**Why:** Steam will DELETE a malformed shortcuts.vdf on restart. If our write corrupts the file, the user loses all their non-Steam shortcuts.

```go
func backupShortcuts(path string) error {
    backupPath := path + ".cluckers-backup"
    data, err := os.ReadFile(path)
    if err != nil {
        return err // No existing file to back up
    }
    return os.WriteFile(backupPath, data, 0644)
}
```

### Anti-Patterns to Avoid

- **Setting WINEDLLOVERRIDES=dxgi=n**: Proton manages DXVK internally. External override causes instant crash. This was proven in Phase 7.1 Deploy 14a.
- **Writing appid ourselves**: Steam generates random appids now. Let Steam assign the appid by writing 0, then read it back after Steam restarts.
- **Modifying shortcuts.vdf while Steam is running**: Steam holds the file. Changes will be overwritten. Warn user to close Steam first.
- **Using `steam -applaunch` for non-Steam games**: Only works for real Steam appids, not non-Steam shortcuts.
- **Trying to bypass Steam Input**: Phase 7.1 proved this is impossible. Work WITH Steam, not around it.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Steam Deck detection | Custom board ID checks | `wine.IsSteamDeck()` | Already exists, tested |
| Steam install detection | Path hardcoding | `wine.FindSteamInstall()` | Already exists with native/Flatpak/Snap support |
| Shortcut appid parsing | Custom binary parser | `FindCluckersAppID()` in `deckconfig.go` | Already tested and working |
| CRC32 for appid | Manual bit manipulation | `hash/crc32` stdlib | Standard CRC32-IEEE, well-tested |
| Proton detection | Manual filesystem scanning | `wine.FindProton()` | Already exists with version sorting |

**Key insight:** Nearly all infrastructure for this phase already exists. The `cluckers prep` command, `shm_launcher.exe` extraction, `launch-config.txt` writing, shortcut appid parsing, Steam detection, and Deck detection are all implemented. The phase is primarily about wiring these together correctly and validating on hardware.

## Common Pitfalls

### Pitfall 1: WINEDLLOVERRIDES=dxgi=n Crash
**What goes wrong:** Game crashes instantly on launch.
**Why it happens:** Proton manages DXVK (dxgi.dll) internally. Setting `WINEDLLOVERRIDES=dxgi=n` tells Wine to use the native dxgi.dll which doesn't exist, causing an immediate crash.
**How to avoid:** Remove `WINEDLLOVERRIDES=dxgi=n` from: (1) `proton_env.go` line 71, (2) `steam_linux.go` line 93, (3) `prep_linux.go` line 20. Also remove from `strippedEnvKeys` or add explicitly stripping it.
**Warning signs:** "Unhandled exception" or segfault immediately after Proton starts. Phase 7.1 Deploy 14a confirmed this.

### Pitfall 2: Steam Overwrites Malformed shortcuts.vdf
**What goes wrong:** User loses all non-Steam game shortcuts after a failed write.
**Why it happens:** Steam validates shortcuts.vdf on startup. If the format is incorrect (wrong terminator bytes, missing fields, malformed entries), Steam deletes the file entirely.
**How to avoid:** Always back up before modifying. Write to a temp file first, validate structure, then rename. Test with an actual Steam installation.
**Warning signs:** Empty non-Steam games library after running `cluckers steam add`.

### Pitfall 3: %command% Not Working for Non-Steam Games
**What goes wrong:** Launch options with `%command%` are appended as arguments instead of replacing the exe.
**Why it happens:** There is a [known Steam bug](https://github.com/ValveSoftware/steam-for-linux/issues/6046) where `%command%` was not interpreted for non-Steam games. However, community reports from 2024-2025 on Steam Deck indicate it DOES work for non-Steam games with Proton compatibility forced.
**How to avoid:** Test on actual Steam Deck. If `%command%` fails, the fallback is to make the shortcut point to a wrapper script instead of shm_launcher.exe directly.
**Warning signs:** Launch options string appears as arguments to the exe instead of being processed.

### Pitfall 4: Shortcut AppID Changes Between Adds
**What goes wrong:** The BPID calculated for `steam://rungameid/` doesn't match after re-adding the shortcut.
**Why it happens:** Steam now [generates random appids](https://github.com/ValveSoftware/steam-for-linux/issues/9463) instead of using the deterministic CRC32 formula. Re-adding generates a new appid.
**How to avoid:** Write appid=0 in shortcuts.vdf, let Steam assign on restart, then read back the assigned appid from shortcuts.vdf. The existing `findCluckersAppID()` already handles reading.
**Warning signs:** "App not found" or Steam launching wrong game.

### Pitfall 5: Steam Not Running When CLI Tries steam://rungameid
**What goes wrong:** The `steam` command hangs or fails because Steam isn't running.
**Why it happens:** On Steam Deck in Game Mode, Steam is always running. On Desktop Mode or desktop Linux, it may not be.
**How to avoid:** Check if Steam is running before attempting `steam://rungameid`. On Deck Game Mode, this is always true. On Desktop, fall back to direct proton run.
**Warning signs:** CLI hangs or prints "Steam not running" error.

### Pitfall 6: Proton Compatibility Not Forced on Shortcut
**What goes wrong:** Steam launches shm_launcher.exe natively (as Linux binary) instead of through Proton, or doesn't launch at all.
**Why it happens:** Non-Steam game shortcuts need "Force the use of a specific Steam Play compatibility tool" enabled in Properties > Compatibility. Without this, Steam won't use Proton for .exe files.
**How to avoid:** Instructions must explicitly tell user to force Proton-GE as compatibility tool. This cannot be set programmatically via shortcuts.vdf -- it's stored in a separate config. Alternatively, this might auto-apply if the shortcut target is a .exe file (Steam may auto-enable Proton).
**Warning signs:** Game doesn't launch, or launches with system Wine instead of Proton-GE.

### Pitfall 7: Deploy 14b Equivalence
**What goes wrong:** Phase 9 repeats Phase 7.1 Deploy 14b and gets the same FAIL result.
**Why it happens:** Deploy 14b was described as "Steam-managed Proton launch" but may not have had all elements: proper shortcut with appid, forced Proton, correct launch options format.
**How to avoid:** Document exactly what was different in Deploy 14b vs the Phase 9 approach. Key differences to verify: (1) Was Proton-GE forced as compatibility tool? (2) Was the shortcut appid non-zero? (3) Was shm_launcher.exe the shortcut target (not cluckers binary)? (4) Were there any WINEDLLOVERRIDES? If Deploy 14b had all these correct, the approach may genuinely not work and secondary approaches must be attempted.
**Warning signs:** Controller drops during ServerTravel even with perfect Steam-managed setup.

## Code Examples

### Removing WINEDLLOVERRIDES=dxgi=n

```go
// proton_env.go -- REMOVE this line:
// "WINEDLLOVERRIDES=dxgi=n",
// The strippedEnvKeys already strips any user-set WINEDLLOVERRIDES.
// Proton manages DXVK internally -- no override needed.

func buildProtonEnvFrom(baseEnv []string, compatDataPath, steamInstallPath, steamGameId string, verbose bool) []string {
    env := filterEnv(baseEnv, strippedEnvKeys...)
    if steamGameId == "" {
        steamGameId = "0"
    }
    env = append(env,
        "STEAM_COMPAT_DATA_PATH="+compatDataPath,
        "STEAM_COMPAT_CLIENT_INSTALL_PATH="+steamInstallPath,
        "SteamGameId="+steamGameId,
        "SteamAppId="+steamGameId,
        // NO WINEDLLOVERRIDES -- Proton manages DXVK internally
    )
    if verbose {
        env = append(env, "PROTON_LOG=1")
    }
    return env
}
```

### Binary VDF Shortcut Entry Writer

```go
// Source: Valve Developer Community wiki + gist.github.com/gablm/2a79355026bde51ac4f516d347fa1cd0

func writeShortcutEntry(w *bytes.Buffer, index int, s *Shortcut) {
    // Entry header: \x00<index>\x00
    w.WriteByte(0x00)
    w.WriteString(strconv.Itoa(index))
    w.WriteByte(0x00)

    // appid (int32, little-endian) -- write 0, let Steam assign
    writeInt32Field(w, "appid", 0)

    // String fields
    writeStringField(w, "AppName", s.AppName)
    writeStringField(w, "Exe", s.Exe)
    writeStringField(w, "StartDir", s.StartDir)
    writeStringField(w, "icon", s.Icon)
    writeStringField(w, "ShortcutPath", "")
    writeStringField(w, "LaunchOptions", s.LaunchOptions)

    // Boolean fields (as int32)
    writeInt32Field(w, "IsHidden", 0)
    writeInt32Field(w, "AllowDesktopConfig", 1)
    writeInt32Field(w, "AllowOverlay", 1)
    writeInt32Field(w, "OpenVR", 0)
    writeInt32Field(w, "LastPlayTime", 0)

    // Empty tags
    w.WriteByte(0x00)
    w.WriteString("tags")
    w.WriteByte(0x00)
    w.WriteByte(0x08)

    // End of entry
    w.WriteByte(0x08)
}

func writeStringField(w *bytes.Buffer, key, value string) {
    w.WriteByte(0x01) // string type
    w.WriteString(key)
    w.WriteByte(0x00)
    w.WriteString(value)
    w.WriteByte(0x00)
}

func writeInt32Field(w *bytes.Buffer, key string, value int32) {
    w.WriteByte(0x02) // int32 type
    w.WriteString(key)
    w.WriteByte(0x00)
    binary.Write(w, binary.LittleEndian, value)
}
```

### Launching via Steam Protocol

```go
// Source: Steam community forums, ArchWiki Steam page

func launchViaSteam(appID uint32) error {
    // Calculate BPID (Big Picture ID) from shortcut appid
    bpid := (uint64(appID) << 32) | 0x02000000
    url := fmt.Sprintf("steam://rungameid/%d", bpid)

    cmd := exec.Command("steam", url)
    // Don't wait for completion -- Steam manages the game lifecycle.
    // The steam command just sends the URL to the running Steam process.
    if err := cmd.Start(); err != nil {
        return &ui.UserError{
            Message:    "Could not launch via Steam",
            Detail:     err.Error(),
            Suggestion: "Make sure Steam is running. On Desktop, start Steam first.",
        }
    }
    return nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Direct `proton run` with env vars | Steam-managed Proton via non-Steam shortcut | Phase 9 (this phase) | Steam manages window tracking, controller state |
| `WINEDLLOVERRIDES=dxgi=n` | No WINEDLLOVERRIDES | Phase 7.1 (Deploy 14a) | Proton manages DXVK internally, override causes crash |
| evdev/uinput proxy | No proxy -- let Steam handle everything | Phase 7.1 (abandoned) | Proxy fundamentally incompatible with Steam Input |
| Deterministic CRC32 appid | Steam-assigned random appid | Steam update ~2023 | Must read appid from shortcuts.vdf after Steam assigns it |
| Manual shortcut setup | Automated shortcuts.vdf writing | Phase 9 (planned) | Eliminates manual Steam UI steps |

**Deprecated/outdated:**
- **evdev proxy approach**: Abandoned Phase 7.1. Creating uinput gamepad kills Steam Input virtual pads.
- **XInput DLL proxy**: Abandoned Phase 7.1. Bypasses Proton's Steam Input IPC.
- **WINEDLLOVERRIDES=dxgi=n**: Causes instant crash. Must be removed from all code paths.
- **Deterministic CRC32 appid calculation**: Steam no longer uses this formula consistently. Read from shortcuts.vdf instead.

## Open Questions

1. **Does Steam auto-enable Proton for .exe shortcuts?**
   - What we know: Phase 7.1 noted that Steam requires "Force compatibility" enabled for Proton to run .exe targets. Some users report Steam auto-detects .exe and enables Proton.
   - What's unclear: Is this automatic on Steam Deck in Game Mode? Does it depend on SteamOS version?
   - Recommendation: Test on hardware. If not automatic, `cluckers steam add` instructions must include the "Force compatibility" step. This cannot be automated via shortcuts.vdf -- it's stored separately.

2. **Was Deploy 14b a proper Steam-managed launch?**
   - What we know: Deploy 14b used `cluckers prep && %command%` with no WINEDLLOVERRIDES and controller dropped during ServerTravel.
   - What's unclear: Did Deploy 14b have Proton forced as compatibility tool? Was the shortcut properly configured with a valid appid? Was shm_launcher.exe the target?
   - Recommendation: Review Deploy 14b setup carefully before repeating. If it was identical to Phase 9's planned approach, the primary approach may genuinely not work, requiring immediate pivot to secondary approaches (Gamescope window tracking, HID injection).

3. **steam://rungameid/ reliability for non-Steam games**
   - What we know: There was a [segfault bug](https://github.com/ValveSoftware/steam-for-linux/issues/9194) in 2023, reportedly fixed. Arguments may not be passed through.
   - What's unclear: Is `steam://rungameid/<BPID>` reliable on current SteamOS (2025/2026)?
   - Recommendation: Test on hardware. Fallback: user launches from Steam UI directly (no CLI trigger). The prep command still works regardless.

4. **Gamescope window tracking details**
   - What we know: Gamescope uses STEAM_GAME X11 property and GAMESCOPECTRL_BASELAYER_APPID for focus tracking. [GitHub issue #8513](https://github.com/ValveSoftware/steam-for-linux/issues/8513) documented 5 specific failures with non-Steam games.
   - What's unclear: Are these issues fixed in current SteamOS? Does a proper non-Steam shortcut with forced Proton set STEAM_GAME correctly?
   - Recommendation: During hardware testing, check X11 properties on the game window (`xdotool search --name "Realm" | xargs xprop`) before and after ServerTravel.

5. **shortcuts.vdf field ordering**
   - What we know: Binary VDF has specific field order per the Valve wiki. Third-party VDF libraries note "order not preserved" as a limitation.
   - What's unclear: Does Steam require exact field ordering, or just correct field types? Will Steam accept entries with different field ordering?
   - Recommendation: Match the exact field order from the official documentation. Write entries byte-by-byte rather than using a generic VDF library. Back up before modifying.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | None (stdlib, no config needed) |
| Quick run command | `go test ./internal/launch/... -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CTRL-03 | WINEDLLOVERRIDES removed from proton env builder | unit | `go test ./internal/launch/ -run TestBuildProtonEnv -count=1` | Yes -- needs modification |
| CTRL-03 | Binary VDF shortcut entry serialization | unit | `go test ./internal/launch/ -run TestWriteShortcut -count=1` | No -- Wave 0 |
| CTRL-03 | Binary VDF shortcuts.vdf read-modify-write roundtrip | unit | `go test ./internal/launch/ -run TestShortcutsRoundtrip -count=1` | No -- Wave 0 |
| CTRL-03 | Existing shortcut detection (findCluckersAppID) | unit | `go test ./internal/launch/ -run TestFindCluckersAppID -count=1` | Yes -- existing |
| CTRL-03 | BPID calculation from appid | unit | `go test ./internal/launch/ -run TestBPIDCalculation -count=1` | No -- Wave 0 |
| CTRL-03 | Launch mode auto-detection (Deck vs desktop) | unit | `go test ./internal/launch/ -run TestLaunchModeDetection -count=1` | No -- Wave 0 |
| CTRL-03 | Controller buttons persist through ServerTravel | manual-only | SSH to Steam Deck, launch game, enter match | N/A -- hardware |

### Sampling Rate
- **Per task commit:** `go test ./internal/launch/... -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green + hardware validation on Steam Deck

### Wave 0 Gaps
- [ ] `internal/launch/shortcuts_test.go` -- covers VDF write, read-modify-write roundtrip, field serialization
- [ ] Update `internal/launch/proton_env_test.go` -- remove WINEDLLOVERRIDES assertions, add no-override assertion

## Sources

### Primary (HIGH confidence)
- [Valve Developer Community - Steam Library Shortcuts](https://developer.valvesoftware.com/wiki/Steam_Library_Shortcuts) -- Binary VDF format specification
- [SteamShortcuts.md Gist](https://gist.github.com/gablm/2a79355026bde51ac4f516d347fa1cd0) -- Detailed binary format with byte values and appid field
- [Steam Shortcut Manager wiki](https://github.com/CorporalQuesadilla/Steam-Shortcut-Manager/wiki/Steam-Shortcuts-Documentation) -- Full field list and entry structure
- Phase 7.1 Deploy 14a/14b results (internal) -- WINEDLLOVERRIDES crash confirmed, clean baseline tested
- Existing codebase (`deckconfig.go`, `prep.go`, `proton_env.go`, `steam_linux.go`) -- All infrastructure already implemented

### Secondary (MEDIUM confidence)
- [ValveSoftware/steam-for-linux#9463](https://github.com/ValveSoftware/steam-for-linux/issues/9463) -- AppID assignment behavior change (random, not CRC32)
- [ValveSoftware/steam-for-linux#8513](https://github.com/ValveSoftware/steam-for-linux/issues/8513) -- Gamescope non-Steam game focus issues (closed as fixed July 2023)
- [ValveSoftware/steam-for-linux#9194](https://github.com/ValveSoftware/steam-for-linux/issues/9194) -- steam://rungameid segfault (fixed Feb 2023)
- [ValveSoftware/steam-for-linux#6046](https://github.com/ValveSoftware/steam-for-linux/issues/6046) -- %command% for non-Steam games (closed as completed 2022)
- [Steam Deck discussions](https://steamcommunity.com/app/1675200/discussions/0/5828254465006618172/) -- %command% confirmed working for non-Steam games with Proton
- [Wakeful-Cloud/vdf](https://github.com/Wakeful-Cloud/vdf) -- Go binary VDF library (evaluated, not recommended -- ordering not preserved)

### Tertiary (LOW confidence)
- [umu-launcher FAQ](https://github.com/Open-Wine-Components/umu-launcher/wiki/Frequently-asked-questions-(FAQ)) -- Alternative launcher approach, not directly applicable but useful context
- [ValveSoftware/gamescope#416](https://github.com/ValveSoftware/gamescope/issues/416) -- GAMESCOPE_FOCUSED_APP behavior details
- [ArchWiki Steam](https://wiki.archlinux.org/title/Steam) -- General Steam on Linux reference

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - No new dependencies, all infrastructure exists
- Architecture: MEDIUM - Steam-managed launch is well-understood but the specific ServerTravel + controller persistence interaction is unproven. Deploy 14b ambiguity is a concern.
- Pitfalls: HIGH - 14 hardware deploys in Phase 7.1 catalogued all failure modes. WINEDLLOVERRIDES crash is definitively proven.
- Binary VDF writing: MEDIUM - Format is documented but field ordering sensitivity is uncertain. Backup-before-modify mitigates risk.
- Hardware validation: LOW - The core thesis (Steam-managed = controller persistence) is untested. Deploy 14b may have already disproven it.

**Research date:** 2026-02-27
**Valid until:** 2026-04-27 (stable Steam/Proton ecosystem, slow-changing binary format)
