# Cluckers Central 1.6.3 — launch & updater reverse-engineering report

Target: `/home/cstory/src/cluckers/cluckers-central/cluckers-central.exe`
(19,836,208 bytes, PE32+ x86-64, Tauri 2.9.0 + wry 0.53.4 + tao 0.34.5, reqwest 0.12.15, tokio 1.44.2)

Build machine paths leak as `C:\Users\hetznerbuildpc\.cargo\registry\...`.
Rust source modules present: `src\main.rs`, `src\gateway_client.rs`, `src\delta_update.rs`,
`src\repair.rs`, `src\content_sig.rs`.

Version literal: `LAUNCHER_VERSION=1.6.3` (log header emitted as `// LAUNCHER_VERSION=<v>`).

Method: read-only static analysis (`strings`, `grep -a`, `dd`, `xxd`, `objdump -d`, `python3`)
plus unauthenticated `curl` GETs against the public updater host.

---

## Reading the string evidence

Rust format strings in this binary are stored length-prefixed, with `\xc0` marking a format
placeholder and the following byte being the argument index. This matters because it lets us
recover the *exact* literal including whether a leading `-` is present.

Verification of the encoding against known-length strings:

| Prefix byte | Literal | Length |
| --- | --- | --- |
| `\x1e` | `Local\realm_content_bootstrap_` | 30 |
| `\x0a` | `-Language=` | 10 |
| `\x06` | `-user=` | 6 |
| `\x0c` | `-token_file=` | 12 |
| `\x17` | `-content_bootstrap_shm=` | 23 |
| `\x07` | `Bearer ` | 7 |

All six match exactly, so the encoding is confirmed.

---

## (a) Exact game command line

### Raw evidence

Hexdump at file offset 13033380 (`.rdata`), the argument-construction literal table:

```
00000250: ffff ff1e 4c6f 6361 6c5c 7265 616c 6d5f  ....Local\realm_
00000260: 636f 6e74 656e 745f 626f 6f74 7374 7261  content_bootstra
00000270: 705f c000 245b 4c61 756e 6368 5d20 4761  p_..$[Launch] Ga
00000280: 6d65 206c 616e 6775 6167 653a 2063 6f6e  me language: con
00000290: 6669 6775 7265 643d 27c0 0d27 2065 6666  figured='..' eff
000002a0: 6563 7469 7665 3d27 c001 2700 0572 6573  ective='..'..res
000002b0: 783d c000 0572 6573 793d c000 7265 7378  x=...resy=..resx
000002c0: 3d31 3932 3072 6573 793d 3130 3830 2d73  =1920resy=1080-s
000002d0: 6565 6b66 7265 656c 6f61 6469 6e67 7063  eekfreeloadingpc
000002e0: 636f 6e73 6f6c 652d 6e6f 686f 6d65 6469  console-nohomedi
000002f0: 722d 6478 3131 0a2d 4c61 6e67 7561 6765  r-dx11.-Language
00000300: 3dc0 0006 2d75 7365 723d c000 0c2d 746f  =...-user=...-to
00000310: 6b65 6e5f 6669 6c65 3dc0 0017 2d63 6f6e  ken_file=...-con
00000320: 7465 6e74 5f62 6f6f 7473 7472 6170 5f73  tent_bootstrap_s
00000330: 686d 3dc0 002d 636f 6e74 656e 745f 626f  hm=..-content_bo
00000340: 6f74 7374 7261 705f 7369 7a65 3d31 3336  otstrap_size=136
00000350: 2a5b 4c61 756e 6368 5d20 4170 7065 6e64  *[Launch] Append
00000360: 696e 6720 6465 765f 636f 6d6d 616e 645f  ing dev_command_
00000370: 6c69 6e65 2061 7267 733a 20c0 0020 5b4c  line args: .. [L
```

Decoded literal sequence, in rodata order:

```
1e "Local\realm_content_bootstrap_" c0 00
24 "[Launch] Game language: configured='" c0 0d "' effective='" c0 01 "'"
05 "resx=" c0 00
05 "resy=" c0 00
   "resx=1920"  "resy=1080"  "-seekfreeloadingpcconsole"  "-nohomedir"  "-dx11"
0a "-Language=" c0 00
06 "-user=" c0 00
0c "-token_file=" c0 00
17 "-content_bootstrap_shm=" c0 00
   "-content_bootstrap_size=136"
2a "[Launch] Appending dev_command_line args: " c0 00
```

### Resulting command line

```
<realm_root>\Binaries\Win64\ShippingPC-RealmGameNoEditor.exe
    resx=<width>
    resy=<height>
    -seekfreeloadingpcconsole
    -nohomedir
    -dx11
    -Language=<LANG>
    -user=<username>
    -token_file=<%TEMP%\realm_launcher_token_XXXX.txt>
    -content_bootstrap_shm=Local\realm_content_bootstrap_<random>
    -content_bootstrap_size=136
    [dev_command_line args, if dev mode enabled]
    [extraArgs, passed from the UI via the launch_game Tauri command]
```

**`resx=` / `resy=` carry NO leading dash.** The length prefix is `\x05`, and `-resx=` would
be 6 bytes. The built-in defaults `resx=1920` / `resy=1080` are likewise dashless.

`-seekfreeloadingpcconsole` remains a single token, matching our existing note in CLAUDE.md.

### How each value is sourced

| Argument | Source |
| --- | --- |
| `resx=` / `resy=` | Launcher settings; defaults `resx=1920`, `resy=1080` |
| `-seekfreeloadingpcconsole`, `-nohomedir`, `-dx11` | Hardcoded constants |
| `-Language=` | Saved `language` setting (`get_game_language` / `save_game_language` Tauri commands), validated against a localization file (see below) |
| `-user=` | `USER_NAME` from the gateway session response |
| `-token_file=` | Temp file containing `LAUNCH_TOKEN` from `/launcher/v1/launch-auth` |
| `-content_bootstrap_shm=` | `Local\realm_content_bootstrap_{}` with a random suffix |
| `-content_bootstrap_size=136` | Hardcoded literal, NOT a format string |

### Language validation

```
"RealmGame" "Localization"      (a [&str; 3] path slice)
05 "Lang_" c0 04 ".dat"
29 "[Launch] WARNING: Localization file for '" c0 13 "' is zero-length ('" c0 14 "'). Falling back to " c0
29 "[Launch] WARNING: Localization file for '" c0 13 "' is unavailable ('" c0 ...  "). Falling back to " c0
```

The launcher probes `<realm_root>/RealmGame/Localization/Lang_<LANG>.dat`. If it is missing or
zero length, it falls back to a default. The fallback constant is almost certainly `INT`
(a bare 3-byte `INT` literal exists in rodata at file offset 14202795), but it sits adjacent to
unrelated Discord URL strings so this is inference, not proof.

### Legacy / absent arguments

The strings `-token=`, `-eac_oidc_token=`, and `-eac_oidc_token_file=` DO appear, but only in a
log-scrubbing table at file offset 14097719, each paired with `<redacted>`:

```
-content_bootstrap_shm=<redacted>-eac_oidc_token_file=<redacted>-eac_oidc_token=<redacted>-token_file=<redacted>-token=<redacted>src\main.rs
```

This is a sanitizer that redacts secrets before writing the command line to the log. None of
these appear in the argument builder. **1.6.3 emits only `-token_file=`.**

`-hostx` does not appear anywhere in the binary. Confirmed removed.

### Launch log strings (complete set)

```
[Launch] Initiating authenticated game launch,
[Launch] Requested launch context: version='{}' install_path='{}'
[Launch] Selected install path appears versioned. Using parent install root: '{}' (from '{}')
[Launch] Effective install root for resolution: '{}'
[Launch] Initial path decision: using requested version folder '{}' (direct EAC not present at install root)
[Launch] Initial path decision: using direct install root '{}' because RealmEAC.exe exists there
[Launch] Content root decision: realm_root='{}' reason={}
[Launch] Executable decision: chosen='{}' reason='{}' primary='{}' exists={} secondary='{}' exists={}
[Launch] Skipping launcher-side INI validation and injection; game manages config directly.
[Launch] Scheduling launch token file cleanup in 15 minutes
[Launch] Game language: configured='{}' effective='{}'
[Launch] Appending dev_command_line args: {}
[Launch] Spawning game process: {}
[Launch] Spawning game process with args: {}
[Launch] Working directory: {}
[Launch] Game process spawned with PID: {}
[Launch] ERROR: {}
[Launch] Failed to fetch launch auth artifacts
[Launch] Failed to write launcher token file {}
[Launch] Failed to base64-decode content bootstrap
[Launch] Failed to lock settings for dev command line
[Launch] Failed to lock settings for game language
[Launch] Refusing unverified game build
[Launch] WARNING: Localization file for '{}' is zero-length ('{}'). Falling back to {}
```

Error strings:

```
Invalid content bootstrap size: {} (expected 136)
Gateway returned an empty content bootstrap
Gateway returned an empty launch token
Gateway response missing PORTAL_INFO_1 (content bootstrap)
Gateway response missing LAUNCH_TOKEN
Account too new for this client; update launcher
Required game version '{}' is not installed at '{}'. Run Download / repair.
Game executable not found in selected install root: {}
Game executable not found (required): {}
Invalid realm_root working directory (does not exist): {}
Failed to spawn game process: {}
```

---

## (b) Shared memory mechanism

### No helper executable

The 1.6.3 launcher creates the mapping **in-process**. Evidence:

- The file contains exactly one `MZ\x90\x00` signature (itself). No embedded PE.
- Zero occurrences of the string `shm_launcher`.
- `CreateFileMappingW`, `MapViewOfFile`, and `UnmapViewOfFile` are all imported directly.

Import table entries:

```
00c59290  CreateFileMappingW
00c596e8  MapViewOfFile
00c596e0  UnmapViewOfFile
```

`CreateFileMappingW` has four call sites. Three (`0x1406be6d9`, `0x1406be713`, `0x1406be981`)
pass a real file handle with `lpName = NULL` — these are the `memmap2` crate used by the delta
updater (`mmap donor`). Only one call site creates a *named* section.

### The named-section call site

Function at `0x140522a40`:

```asm
140522a40:  push %rbp ...                    ; prologue
140522a64:  mov  0x90(%rbp),%rdi             ; rdi = payload length
140522a6b:  test %rdi,%rdi
140522a6e:  je   0x140522ac7                 ; -> "Bootstrap shm: empty bytes"
140522a70:  mov  %rdi,%rax
140522a73:  shr  $0x20,%rax
140522a77:  je   0x140522b11                 ; length must fit in 32 bits
140522a7d:  ...                              ; -> "Bootstrap shm: size too large"

140522b11:  mov  %r9,%rbx                    ; rbx = source bytes
140522b34:  call 0x1403789c0                 ; build UTF-16 (wide) name
140522b39:  mov  -0x18(%rbp),%r12            ; r12 = wide name ptr
140522b3d:  mov  %r12,0x28(%rsp)             ; arg6 lpName
140542b42:  mov  %edi,0x20(%rsp)             ; arg5 dwMaximumSizeLow = length
140522b46:  mov  $0xffffffffffffffff,%rcx    ; arg1 hFile = INVALID_HANDLE_VALUE
140522b4d:  xor  %edx,%edx                   ; arg2 lpAttributes = NULL
140522b4f:  mov  $0x4,%r8d                   ; arg3 flProtect = PAGE_READWRITE
140522b55:  xor  %r9d,%r9d                   ; arg4 dwMaximumSizeHigh = 0
140522b58:  call CreateFileMappingW
140522b5d:  test %rax,%rax
140522b60:  je   0x140522bce                 ; -> CreateFileMappingW failed (err={})
140522b62:  mov  %rax,%r14                   ; r14 = HANDLE
140522b6a:  mov  %rax,%rcx
140522b6d:  mov  $0x2,%edx                   ; FILE_MAP_WRITE
140522b72:  xor  %r8d,%r8d
140522b75:  xor  %r9d,%r9d
140522b78:  call MapViewOfFile               ; (h, FILE_MAP_WRITE, 0, 0, 0)
140522b80:  je   0x140522c0f                 ; -> MapViewOfFile failed (err={})
140522b86:  mov  %rax,%r15
140522b8c:  mov  %rbx,%rdx                   ; src
140522b8f:  mov  %rdi,%r8                    ; len
140522b92:  call memcpy
140522b97:  mov  %r15,%rcx
140522b9a:  call UnmapViewOfFile
140522b9f:  mov  %r14,0x8(%rsi)              ; *** HANDLE stored in returned struct ***
140522ba3:  movabs $0x8000000000000000,%rax
140522bad:  mov  %rax,(%rsi)                 ; Ok-variant tag
```

**The view is unmapped immediately, but the HANDLE is retained in the returned struct.** The
section object therefore lives exactly as long as the launcher process holds that handle. The
launcher must stay alive for the game to call `OpenFileMapping` successfully. This is corroborated
by `[Launch] Scheduling launch token file cleanup in 15 minutes`, which implies a long-lived
process after spawn.

Error messages recovered from `.rdata`:

```
0x140d7da48  "Bootstrap shm: empty bytes"                             (len 0x1a)
0x140d7dac3  "Bootstrap shm: size too large"                          (len 0x1d)
0x140d7da62  "Bootstrap shm: CreateFileMappingW failed (err={})"      (len 0x2e)
0x140d7da95  "Bootstrap shm: MapViewOfFile failed (err={})"           (len 0x29)
```

### Size check and name generation

Caller at `0x1401ce560`:

```asm
1401ce60e:  cmp  $0x88,%r12                  ; 0x88 = 136
1401ce615:  jne  0x1401ce845                 ; -> "Invalid content bootstrap size: {} (expected 136)"
1401ce622:  call 0x140228580                 ; random byte generator
1401ce63f:  call 0x1404d74b0                 ; format the random value
1401ce652:  lea  0xaa074a(%rip),%rdx         ; 0x140c6eda3 = "Local\realm_content_bootstrap_{}"
1401ce667:  call 0x1400fe070                 ; format!()
1401ce691:  movq $0x88,0x20(%rsp)            ; size = 136
1401ce6a8:  call 0x140522a40                 ; create the mapping
```

Function `0x140228580` reads bytes one at a time from a 0x40-byte buffer with refill logic —
the signature of a buffered CSPRNG (`rand::ThreadRng` / `getrandom`). **The suffix is random per
launch, not the process id.**

### Payload format

Unchanged. The size is hardcoded at 136 (`0x88`) in both the check and the argument literal.
No `BPS2` or any other magic variant appears in the binary. The payload arrives base64-encoded
in `portal_info_1` and is base64-decoded before being written into the section
(`[Launch] Failed to base64-decode content bootstrap`).

### What goes in the token file

**Not the gateway access token.** Sequence from the launch path:

```
PORTAL_INFO_1        <- response key for the content bootstrap
LAUNCH_TOKEN         <- response key for the game token
launch_token         <- serde field name
portal_info_1        <- serde field name
realm_launcher_token_{}.txt     <- temp filename template
[Launch] Failed to write launcher token file {}
[Launch] Scheduling launch token file cleanup in 15 minutes
Gateway returned an empty launch token
Gateway response missing LAUNCH_TOKEN
```

Both artifacts come from a single call to `/launcher/v1/launch-auth`. The file is named
`realm_launcher_token_<random>.txt`, and is scheduled for deletion 15 minutes after launch.

---

## (c) Install layout, path resolution, working directory

### Default install roots

```
Games/Cluckers
./Cluckers
```

Both appear as adjacent literals at file offset ~14204843, immediately after the settings
field-name table.

### Settings file

From the shipped `launcher_error.log` (written by an older 0.9.94 build, but the path is stable):

```
[Settings] Settings path: "C:\\Users\\Chris\\AppData\\Roaming\\com.cluckers-central.app\\cluckers_launcher_settings.json"
```

Tauri identifier: `com.cluckers-central.app`, product name `Cluckers Central Launcher`.

### Path resolution algorithm

The launcher uses `RealmEAC.exe` as a layout sentinel to decide whether `install_path` already
points at the game root or at a parent holding versioned subfolders:

```
[Launch] Requested launch context: version='{}' install_path='{}'
[Launch] Selected install path appears versioned. Using parent install root: '{}' (from '{}')
[Launch] Effective install root for resolution: '{}'
[Launch] Initial path decision: using direct install root '{}' because RealmEAC.exe exists there
[Launch] Initial path decision: using requested version folder '{}' (direct EAC not present at install root)
```

Sentinel paths checked, both literals present:

```
Realm-Royale/Binaries/Win64/RealmEAC.exe
Binaries/Win64/RealmEAC.exe
```

Content-root reasons:

```
versioned path is treated as Realm-Royale root
versioned path contains Realm-Royale subfolder
```

Executable selection, with a primary and a secondary candidate:

```
Binaries/Win64/ShippingPC-RealmGameNoEditor.exe
primary candidate exists at version/direct root
primary missing, secondary candidate exists under resolved realm_root
no candidate exists; defaulting to primary for error reporting
[Launch] Executable decision: chosen='{}' reason='{}' primary='{}' exists={} secondary='{}' exists={}
```

### Working directory

Set to the resolved `realm_root`, validated before spawn:

```
Invalid realm_root working directory (does not exist): {}
[Launch] Working directory: {}
```

### Environment variables

None are set for the child process. The binary imports one `SetEnvironmentVariableW` and one
`GetEnvironmentVariableW`. The only launcher-relevant env var is `CC_UPDATER_BASE_URL` (updater
host override). Zero occurrences of `SteamAppId`, `WINEPREFIX`, or `current_dir` as a string.

### INI handling

Removed in 1.6.3:

```
[Launch] Skipping launcher-side INI validation and injection; game manages config directly.
```

### Verified-build gate

Before launching, the launcher binds a previously verified `GameVersion.dat` and refuses if
anything drifted:

```
No verified game build is available. Verify or repair the game before launching.
GameVersion.dat changed after verification. Verify or repair the game before launching.
GameVersion.dat moved outside the verified install after verification.
The selected game install changed after verification. Verify it again before launching.
GameVersion.dat changed while the verified build was being bound
GameVersion.dat metadata has an invalid BLAKE3 hash
Failed to resolve verified GameVersion.dat at '{}': {}
Failed to resolve the verified game install at '{}': {}
[Launch] Refusing unverified game build
```

The frontend tracks this as `verifiedBuildState`.

### Tauri command surface

```
get_app_version  get_api_config  get_install_path  save_install_path
get_dev_command_line_settings  save_dev_command_line_args  get_dev_bypass_settings
get_game_language  save_game_language
launcher_login_or_link  launcher_register  launcher_request_password_reset
launcher_health
launcher_supporter_bot_names_list  launcher_supporter_bot_name_upsert
launcher_supporter_bot_name_delete
check_for_game_updates  start_download  launch_game
open_discord_dm  open_discord_server  open_kofi
```

`launch_game` takes `installPath`, `username`, `extraArgs`.
`start_download` takes `baseUrl`, `downloadState`.
Shared frontend state keys: `authState`, `accessToken`, `apiState`, `logState`,
`verifiedBuildState`, `downloadState`.

---

## (d) Anti-cheat

The game ships EasyAntiCheat, but **the launcher does not launch through an EAC bootstrapper**.

- No `start_protected_game` string anywhere in the binary.
- The chosen executable is always `Binaries/Win64/ShippingPC-RealmGameNoEditor.exe`.
- `RealmEAC.exe` is used purely as (1) a layout sentinel and (2) a repair precondition.

Repair precondition string:

```
no packaged same-version install to repair (Realm-Royale/.../RealmEAC.exe absent)
```

EAC files present in the live manifest:

```
Realm-Royale/Binaries/Win64/RealmEAC.exe
Realm-Royale/Binaries/Win64/eac_server64.dll
Realm-Royale/Binaries/Win64/EasyAntiCheat/EasyAntiCheat_EOS_Setup.exe
Realm-Royale/Binaries/Win64/EasyAntiCheat/Settings.json
Realm-Royale/Binaries/Win64/EasyAntiCheat/Certificates/base.bin
Realm-Royale/Binaries/Win64/EasyAntiCheat/Certificates/base.cer
Realm-Royale/Binaries/Win64/EasyAntiCheat/Licenses/{Apache-2.0,Licenses,MIT}.txt
Realm-Royale/Binaries/Win64/EasyAntiCheat/Localization/*.cfg   (20 locales)
Realm-Royale/Binaries/Win64/EasyAntiCheat/Splash{Realm,Screen}.png
```

Also present: `Realm-Royale/Binaries/Win64/EOSSDK-Win64-Shipping.dll` (19,256,272 bytes).

### Integrity guard over EAC directories

A repair-time guard monitors two directory substrings for unexpected binaries:

```
/binaries/
/easyanticheat/
```

against a whitelist of known executable names (recovered from a 16-byte-strided rodata table,
stored in reversed half-chunks):

```
easyanticheat_setup.exe
easyanticheat_eos.exe
easyanticheat.exe
realmgame-win64-shipping.exe
```

The guard only reports; it never deletes:

```
[Repair][Guard5] extraneous file (kept, surfaced): {}
[Repair][Guard5] extraneous binary in monitored dir (SURFACED, removal suppressed): {}
[Repair][Guard5] surfaced {}/ extraneous file(s) (surface-only; no removals)
```

---

## (e) Updater API

### `GET https://updater.realmhub.io/builds/version.json` (fetched live)

```json
{
    "latest_version": "0.39.6969.0",
    "base_url": "https://updater.realmhub.io/builds/0.39.6969.0",
    "manifest_url": "https://updater.realmhub.io/builds/manifest-v0.39.6969.0.json",
    "gameversion_dat_path": "Realm-Royale/Binaries/GameVersion.dat",
    "gameversion_dat_blake3": "9bebfc833dc08de9910891e61b0cf40e700862a2dd6a65ddfad4ea41136c0f92",
    "gameversion_dat_size": 19,
    "patch_threshold_bytes": 16777216,
    "delta_enabled": true,
    "delta_rollout_percent": 10,
    "repair_index_url": "https://updater.realmhub.io/builds/manifest-repair-v0.39.6969.0.json",
    "repair_enabled": true,
    "repair_rollout_percent": 10,
    "content_sig_scheme": "minisign"
}
```

Note: `zip_url`, `zip_blake3`, `zip_size`, and `patch_index_url` are **absent from the live
response** even though the launcher's deserializer accepts all of them.

### Full `VersionInfo` field set accepted by the launcher

Recovered from the serde field-name table at file offset ~14204581:

```
latest_version  base_url  manifest_url  zip_url  zip_blake3  zip_size
gameversion_dat_path  gameversion_dat_blake3  gameversion_dat_size
patch_index_url  patch_threshold_bytes
delta_enabled  delta_rollout_percent
repair_index_url  repair_enabled  repair_rollout_percent
content_sig_scheme
```

Adjacent local-settings fields (persisted in `cluckers_launcher_settings.json`):

```
install_path  selected_version  language
dev_command_line  dev_command_line_args
dev_bypass_verification  dev_bypass_file_checks
prefer_delta  prefer_smart_repair
```

Frontend status enum (Rust side `UpToDate / UpdateRequired / NotInstalled / CorruptOrModified`):

```
upToDate  updateRequired  notInstalled  corruptOrModified
bypassVerification  bypassFileChecks
hash  size  status  currentVersion  latestVersion  baseUrl  enabled  args
percentage  speed_mbs
```

### `GET https://updater.realmhub.io/builds/manifest-v0.39.6969.0.json` (fetched live)

244,111 bytes. Head:

```json
{
  "schema": 1,
  "version": "0.39.6969.0",
  "files": [
    {
      "path": "Realm-Royale/Binaries/GameVersion.dat",
      "hash": "9bebfc833dc08de9910891e61b0cf40e700862a2dd6a65ddfad4ea41136c0f92",
      "size": 19
    },
    {
      "path": "Realm-Royale/Binaries/Win64/APEX_ClothingCHECKED_x64.dll",
      "hash": "9416077d9ab6d885af2a3a4b68f30fea21547dcb9d4c8690bf2f9f171f050cd8",
      "size": 1855136
    },
    ...
  ]
}
```

Structure:

| Property | Value |
| --- | --- |
| Top-level keys | `schema`, `version`, `files` |
| Schema | 1 |
| File count | 1257 |
| File entry keys | `path`, `hash`, `size` (exactly these three) |
| Total bytes | 8,054,269,355 |
| Top-level directory | `Realm-Royale` (single root, all paths prefixed) |

Executables in the manifest:

```
Realm-Royale/Binaries/Win64/ShippingPC-RealmGameNoEditor.exe
Realm-Royale/Binaries/Win64/RealmEAC.exe
Realm-Royale/Binaries/Win64/EasyAntiCheat/EasyAntiCheat_EOS_Setup.exe
```

INI files in the manifest (25 total), notably including a shipped `RealmSystemSettings.ini`:

```
Realm-Royale/Engine/Config/{BaseEngine,BaseGame,BaseGameStats,BaseInput,BaseLightmass,BaseSystemSettings,BaseUI,ConsoleVariables}.ini
Realm-Royale/RealmGame/Cloud/CloudStorage.ini
Realm-Royale/RealmGame/Config/{DefaultEngine,DefaultGame,DefaultInput,DefaultInputDefaults,DefaultLightmass,DefaultLock,DefaultSystemCompat,DefaultSystemSettings,DefaultUI}.ini
Realm-Royale/RealmGame/Config/{RealmEngine,RealmGame,RealmInput,RealmLightmass,RealmSystemSettings,RealmUI,ShadowPatch}.ini
```

### `GET https://updater.realmhub.io/builds/manifest-repair-v0.39.6969.0.json` (fetched live)

```json
{
  "to_version": "0.39.6969.0",
  "algo": "blake3-bao-outboard",
  "threshold_bytes": 16777216,
  "group_size": 262144,
  "files": [
    {
      "path": "Realm-Royale/Binaries/Win64/EOSSDK-Win64-Shipping.dll",
      "hash": "b9a0e81b547bba0424e815e433cf74a4bcc52fa23c514704cb1d2909df43ec61",
      "size": 19256272,
      "outboard_path": "0.39.6969.0/Realm-Royale/Binaries/Win64/EOSSDK-Win64-Shipping.dll.bao",
      "outboard_size": 1203464
    },
    {
      "path": "Realm-Royale/Binaries/Win64/ShippingPC-RealmGameNoEditor.exe",
      "hash": "fdd3096bb6018354c6a7f1be3113a3c50e68ec67f3e7b8a6cbb07c0c9c3f819e",
      "size": 48017408,
      "outboard_path": "0.39.6969.0/Realm-Royale/Binaries/Win64/ShippingPC-RealmGameNoEditor.exe.bao",
      "outboard_size": 3001032
    },
    ...
  ]
}
```

The repair path uses BLAKE3 Bao outboard trees plus HTTP range requests to localize and splice
only the damaged 256 KiB groups of large files:

```
.bao.blockpart
local length {} != manifest size {} (not bao-eligible)
outboard download failed: {}
slice pre-verify failed for [{}
range-get [{}) failed: {}
expected 206, got HTTP {}
short range body: got {} expected {}
no bad groups localized though file hash mismatched
mandatory whole-file re-verify failed
[Repair][Bao] {} repaired via {} group(s)
[Repair][Bao] {} fell back to whole-file: {}
```

`manifest-patch-v0.39.6969.0.json` returns **404** — the delta/patch index is not currently
published, matching `patch_index_url` being absent from `version.json`.

### Content signing (minisign)

Both signature files return HTTP 200.

`GET https://updater.realmhub.io/builds/version.json.minisig`:

```
untrusted comment: cluckers content signature
RUQoJP0ExlFzVz9pp8wxeVbzh5O64Rw8NPugVHX8dOaBcsSH9lYG8c5AKuxaqmor9NDku4iAb/v9BXMhjHvRxT25994YNlXX0Qs=
trusted comment: file:version.json
SADUfhsd7ms/Frwc5nL1L/rkWKSlG4QA30GxvY8A6hvSebnPqamj9cgsaxBzEnU9pib6/y/rY8zhkQwmZnKiDQ==
```

`GET https://updater.realmhub.io/builds/manifest-v0.39.6969.0.json.minisig`:

```
untrusted comment: cluckers content signature
RUQoJP0ExlFzVwndHyGgEApwUtPJ6gHGDqbWEyyz6vvgzRIi8Dw7NUe7ysXjHH7mXOpDr7BoSJvsTlBI6Jo1W+QkxvzWGL6/ggk=
trusted comment: file:manifest-v0.39.6969.0.json
vCwjlDSYl35FKmULtL126DIRPQW+Sz7gSlW4+86oaV4jVSONQ4ZTwFHzYLDsf3c59H6hTgx3KB9FBxEQy6d0Cg==
```

The public key is embedded **twice** in the binary (once for the Tauri self-updater, once for
`src\content_sig.rs`), base64-of-the-whole-file:

```
dW50cnVzdGVkIGNvbW1lbnQ6IG1pbmlzaWduIHB1YmxpYyBrZXk6IDU3NzM1MUM2MDRGRDI0MjgKUldRb0pQMEV4bEZ6VjgzS2pNVXU5SDFHcjlYaEN0Qllmazd3ZUtrQklMc0RSOE0xd3FPVVo4bk8K
```

which decodes to:

```
untrusted comment: minisign public key: 577351C604FD2428
RWQoJP0ExlFzV83KjMUu9H1Gr9XhCtBYfk7weKkBILsDR8M1wqOUZ8nO
```

Signature enforcement strings (the launcher refuses, it does not warn):

```
content pubkey utf8: {} / content pubkey base64 decode: {} / content pubkey parse: {}
signature verify: {} / signature parse: {}
minisign  .minisig  .minisig?
[Delta] REFUSE: version.json signature: {}
[Delta] REFUSE: manifest signature: {}
[Delta] REFUSE: patch index signature: {}
[Delta] content signatures OK (signing_active={})
[Repair] REFUSE: version.json signature: {}
[Repair] REFUSE: manifest signature: {}
[Repair] REFUSE: repair index signature: {}
[Repair] content signatures OK (signing_active={})
content signature (version.json): {}
content signature (manifest): {}
content signature (patch index): {}
content signature (repair index): {}
```

### Multi-host fetch with cache busting

```
CC_UPDATER_BASE_URL
https://updater.realmhub.io
https://updater.realmhub.io,https://updater.project-crown.com     <- allowlist
/builds/version.json?t={}                                          <- cache buster
all updater hosts failed ({})
{}: HTTP {}
```

`updater.project-crown.com` is a mirror used for both content and self-update.

### Download engine

The per-file download path supports segmented/range downloads with resume:

```
Segmented download requires a known Content-Length
Incomplete segmented download: {}
segmented download error: {}
bytes={}          bytes=0-0        range probe request error: {}
[Download] {} attempt {} (resuming)
[Download] FAILED {} after {} tries
chunk: {}        stalled: no data for {}
.part            .repairing        .staging        .diffstaging       .appliedversion
```

Marker files used: `.repairing`, `.staging`, `.diffstaging`, `.part`, `.part.zstdelta`,
`.bao.blockpart`, `.appliedversion`.

Path-traversal guards are present throughout:

```
path-traversal reject for '{}': {}
path-traversal reject for GameVersion.dat '{}': {}
path-traversal reject repair '{}': {}
path-traversal reject sentinel '{}': {}
```

The zip path still exists as a fallback but requires `zip_url`, which is no longer served:

```
game-{}.zip
[Download][Zip] Begin. {} zip_url='{}' zip_path='{}' zip_blake3_present={} zip_size={}
[Download] ERROR: missing manifest_url and zip_url.
Error: Missing manifest_url/zip_url in version.json
```

Cross-version file reuse (donor installs) is implemented:

```
[Reuse] Reused {} files from existing versions.
[Reuse] Failed to place {} from donor: {}
[Delta] Selected donor '{}'
no donor install found
machine not in delta_rollout_percent={}
machine not in repair_rollout_percent={}
```

### Comparison to our Go implementation

| Aspect | Official 1.6.3 | Our Go (`internal/game/`) | Verdict |
| --- | --- | --- | --- |
| version.json URL | `/builds/version.json?t=<n>` across 2 hosts | `version.go:20`, single host, no cache buster | Minor gap |
| `VersionInfo` fields | 17 fields | 6 fields, all names correct | Compatible |
| Manifest schema | `schema`/`version`/`files[{path,hash,size}]` | Identical in `manifest.go` | Exact match |
| Per-file URL | `base_url` + `/` + `path` | `sync.go:208`, same | **Verified working** |
| `content_sig_scheme` | Enforced, refuses on bad signature | Ignored entirely | **Security gap** |
| `repair_index_url` | Bao outboard range repair | Ignored | Feature gap |
| `patch_index_url` / delta | zstd delta patching | Ignored | Feature gap (currently unpublished anyway) |
| `selected_version` | Version pinning setting | We have `PinVersionInfo` | Compatible |
| Local marker | `Realm-Royale/Binaries/GameVersion.dat` | `version.go:24`, same constant | Exact match |
| Sync marker | `.repairing` / `.staging` | `.cluckers-syncing` | Ours is private, fine |

Live verification performed:

```
$ curl -r 0-15 https://updater.realmhub.io/builds/0.39.6969.0/Realm-Royale/Binaries/GameVersion.dat
status=206 len=16
00000000: 0000 0000 0001 0001 4057 0d00 0039 1b27
```

Our URL construction scheme is correct against the live CDN.

---

## (f) INI presets in `assets/inis/`

### Files

| Preset | Size | Lines | Encoding |
| --- | --- | --- | --- |
| low | 50,354 | 1047 | ASCII |
| medium | ~50 KB | 1047 | ASCII |
| high | ~50 KB | 1047 | ASCII |
| ultra | 104,052 | 1059 | **UTF-16LE with BOM** |

Each folder holds exactly one file, `RealmSystemSettings.ini`. All four have 36 sections.
Low/medium/high have 974 keys; ultra has 985.

### Differences

103 keys differ across the four presets, 76 of them in `[SystemSettings]`.

Scalar quality knobs:

| Key | low | medium | high | ultra |
| --- | --- | --- | --- | --- |
| MaxShadowResolution | 32 | 128 | 512 | 1024 |
| MinShadowResolution | 16 | 32 | 32 | 1024 |
| ShadowFadeResolution | 16 | 128 | 128 | 1024 |
| ShadowTexelsPerPixel | 2.0 | 2.0 | 2.0 | 1024.0 |
| ShadowFilterQualityBias | 0 | 0 | 0 | 4 |
| MaxWholeSceneDominantShadowResolution | 1280 | 1280 | 1280 | 2048 |
| MaxAnisotropy | 1 | 4 | 4 | 16 |
| MaxMultisamples | 1 | 1 | 4 | 16 |
| FXAAQuality | 0 | 0 | 0 | 4 |
| ScreenPercentage | 100.0 | 100.0 | 200.0 | 200.0 |
| DetailMode | 1 | 1 | 4 | 4 |
| MaterialQualityLevel | 2 | 2 | 2 | 0 |
| TexturePoolSize | 450 | 450 | 450 | 15000 |
| MaxActiveDecals | 50 | 50 | 50 | 100 |
| DecalCullDistanceScale | 0.6 | 0.6 | 0.6 | 2.4 |
| MaxDrawDistanceScale | 1.0 | 1.0 | 1.0 | 10.2 |
| MeshScreenPixelAreaThreshold | 196.0 | 196.0 | 196.0 | 100.0 |
| OctreeScreenPixelAreaThreshold | 392.0 | 392.0 | 392.0 | 200.0 |
| ParticleLODBias | 4 | 1 | 1 | -1 |
| SpeedTreeLODBias | 4 | 2 | 2 | -1 |
| SkeletalMeshLODBias | 0 | 0 | 0 | -1 |
| StaticMeshLODBias | 0 | 0 | 0 | -1 |
| PerfScalingBias | 0.1 | 0.1 | 0.1 | 0.0 |
| MobileShadowTextureResolution | 1120 | 1120 | 1120 | 2048 |
| ApexGRBGPUMemSceneSize | 128 | 128 | 128 | 256 |
| ApexGRBGPUMemTempDataSize | 128 | 128 | 128 | 256 |

Boolean feature toggles:

| Key | low | medium | high | ultra |
| --- | --- | --- | --- | --- |
| AmbientOcclusion | False | False | False | True |
| AllowImageReflections | False | False | False | True |
| AllowSubsurfaceScattering | False | False | False | True |
| AllowScreenDoorFade | False | False | False | True |
| bAllowLightShafts | False | False | False | True |
| bAllowPostprocessMLAA | False | False | False | True |
| ApexGRBEnable | false | false | false | True |
| CompositeDynamicLights | True | True | True | False |
| DropParticleDistortion | True | True | True | False |
| bUseLowQualMaterials | **True** | False | False | False |
| UseHighQualityBloom | absent | absent | absent | True |

Notable: **`bUseLowQualMaterials` is the only `[SystemSettings]` key separating low from medium.**

Texture groups: every `TEXTUREGROUP_*` entry (roughly 30 of them) is rewritten per preset.
Ultra uniformly sets `LODBias=-1`, `LODBiasTexturePack=-1`, `MipFilter=Linear`, and
`MinLODSize`/`MaxLODSize` of 2048–4096. Example:

```
TEXTUREGROUP_Character
  low   = (MinLODSize=16,  MaxLODSize=64,   MaxLODSizeTexturePack=1024, LODBias=4, ..., MipFilter=Point)
  med   = (MinLODSize=128, MaxLODSize=512,  MaxLODSizeTexturePack=1024, LODBias=1, ..., MipFilter=Point)
  high  = (MinLODSize=128, MaxLODSize=1024, MaxLODSizeTexturePack=1024, LODBias=1, ..., MipFilter=Point)
  ultra = (MinLODSize=4096,MaxLODSize=4096, MaxLODSizeTexturePack=4096, LODBias=-1,..., MipFilter=Linear)
```

Ultra-only additions: `TEXTUREGROUP_UIStreamable` (in `[SystemSettings]`, all five
`[SystemSettingsBucketN]`, and `[TextureSettingsSpectator]`), `TEXTUREGROUP_TitleScreenPreview`,
`NumPartialInstallBuckets=2`, and `UseHighQualityBloom=True`.

Ultra also sets `MaxShadowResolution=4096` in every `[SystemSettingsBucketN]`, whereas
low/medium/high use 256–512.

The ultra file contains a mojibake key (`潍楢敬潐瑳牐捯獥䉳畬䅲潭湵t` = "MobilePostProcessBlurAmount"
read with the wrong endianness) — a defect in the shipped asset itself, not in my decoding.

### The launcher does not use these files

Zero references in the binary to `inis`, `RealmSystemSettings`, `RealmEngine.ini`, a
`RealmGame/Config` write path, or any quality-preset name (`low`/`medium`/`high`/`ultra` as a
setting value). This is consistent with the log line:

```
[Launch] Skipping launcher-side INI validation and injection; game manages config directly.
```

**`assets/inis/` is dead weight in 1.6.3**, left over from a version that copied presets into
`RealmGame/Config/`.

---

## (g) Self-update and Discord

### Self-update

Uses the Tauri updater plugin (`tauri-plugin-updater/2.7.1`) with two endpoints:

```
https://updater.realmhub.io/update.json
https://updater.project-crown.com/update-crown.json
allowDownloadAndInstall
{{current_version}}  {{target}}  {{arch}}
```

`GET https://updater.realmhub.io/update.json` (fetched live):

```json
{
    "notes": "Bug fixes and improvements.",
    "platforms": {
        "windows-x86_64": {
            "signature": "dW50cnVzdGVkIGNvbW1lbnQ6IHNpZ25hdHVyZSBmcm9tIHRhdXJpIHNlY3JldCBrZXkKUlVRb0pQMEV4bEZ6VjRyQ0d6bmVaREJEcWR1cStYaDVkSk51dWhPdUxRa0gvNFBvTWczMW9Penh2K0VmMFh5OUV5eElhTjRsWkl0MDlKQnBtWVI2djJtWlRKdndyYWpXcWdRPQp0cnVzdGVkIGNvbW1lbnQ6IHRpbWVzdGFtcDoxNzg1OTg5MTQ3CWZpbGU6Y2x1Y2tlcnMtY2VudHJhbF8xLjYuM194NjQtc2V0dXAuZXhlCks4K01IRkh6a2hSdW1Ta1NpRTRpVlh3cUVkbXR3Z05qMjhVNmZBbjNnR3NXZytlcHdrZ3NrTVk3MXFBeGphTUtoRWJWbVNWVFRBMzE4V0pzdWo2VkNBPT0K",
            "url": "https://updater.realmhub.io/cluckers-central_1.6.3_x64-setup.exe"
        }
    },
    "pub_date": "2026-08-05T21:06:41.446Z",
    "version": "1.6.3"
}
```

`GET https://updater.project-crown.com/update-crown.json` (fetched live) is byte-identical
except for the `url`, which points at `https://updater.project-crown.com/cluckers-central_1.6.3_x64-setup.exe`.

The embedded signature's trusted comment decodes to
`timestamp:1785989147  file:cluckers-central_1.6.3_x64-setup.exe`, signed with the same
minisign key `RWQoJP0ExlFzV83KjMUu9H1Gr9XhCtBYfk7weKkBILsDR8M1wqOUZ8nO` used for content signing.

### Discord

**No Rich Presence.** Zero occurrences of `discord_rpc`, `discord-rich`, `discordapp`, or
`discord.com/api`. No 18-19 digit application id anywhere near a Discord string.

Discord appears only as two shell-open commands, `open_discord_dm` and `open_discord_server`,
alongside `open_kofi` which opens `https://ko-fi.com/projectcrown/tiers`. Error strings:
`Failed to open Discord URL: {}` and `Failed to open Ko-fi URL: {}`.

---

## (h) Gateway endpoints

### Complete endpoint set in 1.6.3

```
/launcher/v1/account
/launcher/v1/launch-auth
/launcher/v1/password-reset
/launcher/v1/session-or-link
/launcher/v1/session/refresh
/launcher/v1/supporter/bot-names
healthz
```

`/launcher/v1/content-bootstrap` **does not exist in this binary.** It has been replaced by
`/launcher/v1/launch-auth`, which returns both the content bootstrap and the launch token.

### `/launcher/v1/launch-auth` — path, method, auth

The path literal is exactly `/launcher/v1/launch-auth`, 24 bytes, at file offset 13046761
(VMA `0x140c721e9`). The adjacent literals are `title` before it and `src\gateway_client.rs`
after it:

```
00000040: 7420 6572 726f 723a 20c0 0074 6974 6c65  t error: ..title
00000050: 2f6c 6175 6e63 6865 722f 7631 2f6c 6175  /launcher/v1/lau
00000060: 6e63 682d 6175 7468 7372 635c 6761 7465  nch-authsrc\gate
00000070: 7761 795f 636c 6965 6e74 2e72 7300 0001  way_client.rs...
```

The earlier reading `/launcher/v1/launch-authsrc` was a concatenation artifact. `title` belongs
to the RFC 7807 error parser (`title` / `detail`), and `src\gateway_client.rs` is the module path.

**Method: POST.** Determined by correlating the method-discriminant byte written immediately
before each path pointer is stored into the request struct. Only one xref exists to the path,
at `0x1401c978d`:

```asm
1401c975a:  movb $0x5,0x838(%r13)             ; method discriminant
1401c978d:  lea  0xaa8a55(%rip),%rax          ; 0x140c721e9 = "/launcher/v1/launch-auth"
1401c9794:  mov  %rax,0x870(%r13)
1401c979b:  movq $0x18,0x878(%r13)            ; length 24
```

Correlation across endpoints with known methods:

| Endpoint | Discriminant | Known method |
| --- | --- | --- |
| `/launcher/v1/session-or-link` | 5 | POST |
| `/launcher/v1/account` (register) | 5 | POST |
| `/launcher/v1/password-reset` | 5 | POST |
| `/launcher/v1/session/refresh` (2 sites) | 5 | POST |
| `/launcher/v1/supporter/bot-names` (list) | 0 | GET |
| **`/launcher/v1/launch-auth`** | **5** | **POST (inferred)** |

Five independent confirmations of `5 = POST` and one of `0 = GET`.

**Auth: Bearer.** The binary contains exactly one bearer format literal, `\x07 "Bearer " \xc0 \x00`
(`Bearer {}`), and two `Authorization` strings. The launch path logs
`[Launch] Initiating authenticated game launch,` before the call, so the access token is present.

**Body: not determinable statically.** No distinct request-field literals appear near the call
site; only the *response* deserializer field list is adjacent. The body is either empty or
composed of fields shared with another struct. This needs a live call to settle.

### `/launcher/v1/launch-auth` response

Deserializer field list at file offset 13046840:

```
account_id  session_id
launch_expiration_datetime  launch_expires_at_unix
custom_value_1  custom_value_2
expiration_datetime
```

plus, referenced from the launch path:

```
portal_info_1   -> base64 content bootstrap, 136 bytes after decode
launch_token    -> written to the -token_file
```

Related error strings:

```
account_id_overflow
Account too new for this client; update launcher
```

suggesting `account_id` is parsed into a bounded integer that newer accounts can exceed.

### Session response (login / refresh)

```
USER_NAME  ACCESS_TOKEN  REFRESH_TOKEN
ACCESS_EXPIRES_AT_UNIX  REFRESH_EXPIRES_AT_UNIX
SUCCESS  LINKED_FLAG  STRING_VALUE
```

Log format:

```
[Gateway] {} reply for user='{}': success={} linked={} string_value='{}' access_token_present={}
```

Real examples from the shipped `launcher_error.log` (an older 0.9.94 build):

```
[Gateway] launcher_login_or_link reply for user='0xc0re': success=0 linked=-1 string_value='Invalid credentials' access_token_present=false
[Gateway] launcher_login_or_link reply for user='0xc0re': success=1 linked=0 string_value='' access_token_present=true
[Gateway] launcher_login_or_link reply for user='0xc0re': success=1 linked=1 string_value='PIN_REQUIRED' access_token_present=false
[Gateway] launcher_login_or_link reply for user='0xc0re': success=0 linked=-1 string_value='Rate limited (login). Try again in a minute.' access_token_present=false
```

Note `PIN_REQUIRED` as a `STRING_VALUE`, and the `pin` parameter on the `launcher_login_or_link`
Tauri command — a login flow step our Go client does not implement.

Session expiry handling:

```
/launcher/v1/session/refresh
[Gateway] launcher session refresh failed
Launcher session expired. Please log in again.
```

Error envelope is RFC 7807: `title`, `detail`, plus `{} request failed`, `{} HTTP {}`,
`{} JSON parse failed: {}`, `Launcher portal {}`, `Launcher portal request failed: {}`.

---

## (i) Differences from our Go launcher that would break launching

Ordered by severity.

### 1. Wrong token written to the token file — CRITICAL

We write the gateway **access token** (`internal/launch/pipeline.go:505`, `writeTokenFile`,
via `LaunchConfig.AccessToken` / `TokenPath`).

1.6.3 writes **`LAUNCH_TOKEN`**, a distinct artifact returned by `/launcher/v1/launch-auth`
alongside `portal_info_1`. The launcher keys it separately (`launch_token` vs `access_token`),
gives it its own expiry (`launch_expiration_datetime`, `launch_expires_at_unix`), and has a
dedicated failure mode (`Gateway returned an empty launch token`).

If the game validates this token server-side, our launch fails authentication no matter what
else is correct.

### 2. `/launcher/v1/content-bootstrap` no longer exists — CRITICAL

`internal/auth/login.go:19` defines `pathContentBootstrap = "/launcher/v1/content-bootstrap"`.
That path is absent from the 1.6.3 binary's complete endpoint list. The replacement is a single
`POST /launcher/v1/launch-auth` (bearer) returning both `portal_info_1` and `launch_token`.

The old endpoint may still be served for backwards compatibility, but we would still be missing
the launch token from item 1.

### 3. No content-signature verification — SECURITY

`version.json` advertises `"content_sig_scheme": "minisign"` and both `version.json.minisig` and
`manifest-v<ver>.json.minisig` are live. The official launcher hard-refuses on a bad signature
(`[Delta] REFUSE: manifest signature:`, `[Repair] REFUSE: version.json signature:`).

`internal/game/version.go` and `manifest.go` ignore the field entirely. We accept any manifest
a MITM or compromised CDN serves, then execute the binaries it names. The public key is already
known:

```
RWQoJP0ExlFzV83KjMUu9H1Gr9XhCtBYfk7weKkBILsDR8M1wqOUZ8nO
```

### 4. No verified-build gate before launch — MEDIUM

The official launcher binds a verified `GameVersion.dat` and refuses to launch on any drift.
We check `NeedsUpdate` during `cluckers update` but not at launch time, so a corrupted install
fails inside the game with a worse error.

### 5. Missing `resx=` / `resy=` — MEDIUM

We emit neither (`process_windows.go:42-48`, `process_linux.go:46-52`). The official launcher
always passes them, defaulting to `resx=1920 resy=1080`, **without a leading dash**. If the game
relies on these rather than its own config, the window may open at an unintended resolution.

### 6. Shared-memory handle lifetime — MEDIUM

We spawn `shm_launcher.exe` as an intermediate process which creates the mapping and then execs
the game as a child. The official launcher creates the mapping in its own long-lived process and
keeps the `HANDLE` open.

Our design is functionally equivalent **only if** the helper outlives the game's
`OpenFileMapping` call. Worth confirming `tools/shm_launcher.c` does not close the handle or
exit before the game opens the section.

### 7. `-content_bootstrap_size` computed rather than fixed — LOW

We emit `fmt.Sprintf("-content_bootstrap_size=%d", len(cfg.ContentBootstrap))`. The official
launcher hardcodes `-content_bootstrap_size=136` and rejects any payload that is not exactly
136 bytes (`cmp $0x88,%r12`). Functionally the same when the payload is well formed, but we
would silently pass a wrong size where the official launcher errors out cleanly.

### 8. Refresh tokens unused — LOW

The session response now carries `REFRESH_TOKEN`, `ACCESS_EXPIRES_AT_UNIX`, and
`REFRESH_EXPIRES_AT_UNIX`, and `POST /launcher/v1/session/refresh` exists. Our 45-minute
`TokenCache` TTL with re-login on HTTP 401 (`internal/auth/cache.go`) works but is cruder and
re-sends the password more often than necessary.

### 9. No `PIN_REQUIRED` handling — LOW

`launcher_login_or_link` accepts a `pin` argument and `STRING_VALUE` can be `PIN_REQUIRED`.
Our `auth.Login()` has no path for this, so a 2FA-enabled account cannot log in.

### 10. Single updater host, no cache buster — LOW

Official: `/builds/version.json?t=<n>` across `updater.realmhub.io` and
`updater.project-crown.com`, overridable by `CC_UPDATER_BASE_URL`, with `all updater hosts failed`
only after both. Ours: one host, no cache buster (`version.go:20`).

### Non-breaking differences

- **Argument order.** Ours is `-user -token_file -Language -dx11 -seekfreeloadingpcconsole -nohomedir -content_bootstrap_size -content_bootstrap_shm`; theirs is `resx resy -seekfreeloadingpcconsole -nohomedir -dx11 -Language -user -token_file -content_bootstrap_shm -content_bootstrap_size`. UE3 does not care.
- **Shm name suffix.** We use the process id, they use random bytes. The name is passed explicitly, so either works.
- **Install layout.** We use `~/.cluckers/game/Realm-Royale/`, they use `Games/Cluckers`. Ours is self-consistent.
- **Sync marker.** `.cluckers-syncing` vs `.repairing`/`.staging`. Private to each implementation.

---

## (j) Open questions

1. **Is `LAUNCH_TOKEN` actually different from the `lpt_v1_` access token,** or merely the same
   value under a second key? This is the single fact that decides whether item 1 above is a real
   break or a rename. Needs a live authenticated `POST /launcher/v1/launch-auth`.

2. **What body does `/launcher/v1/launch-auth` take?** No request-field literals sit near the
   call site. Could be an empty body, or could carry a client version / install version.

3. **Are `resx=`/`resy=` emitted unconditionally,** or only when a resolution override is set in
   settings? Both a format string (`resx={}`) and a default constant (`resx=1920`) exist, which
   is consistent with either.

4. **What is the language fallback constant?** A bare `INT` literal exists but sits next to
   unrelated Discord strings, so the association is unproven. Our Go hardcodes `INT`, which
   matches the game's own default regardless.

5. **`RealmSystemSettings.ini` is a manifest-managed game file.** If the game now writes user
   settings there, our clean-sync deletion pass in `internal/game/sync.go` could clobber them on
   every update. Worth checking what clean-sync does to `Realm-Royale/RealmGame/Config/`.

6. **`account_id_overflow` / "Account too new for this client".** Suggests `account_id` is parsed
   into a type that newer accounts overflow. If our Go parses it as a string or int64 we may be
   fine, but the server-side ID range is worth knowing.

7. **Does the game require the launcher to stay alive** beyond the initial `OpenFileMapping`?
   The official design keeps the handle for the whole session. If the game re-opens the mapping
   later, our helper-process approach needs to match that lifetime.
