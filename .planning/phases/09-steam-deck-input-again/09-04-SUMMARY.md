---
phase: 09-steam-deck-input-again
plan: 04
subsystem: input
tags: [xinput, proton, wine, steam-input, dll-shim, steam-deck]

requires:
  - phase: 09-03
    provides: Hardware testing infrastructure and baseline controller behavior confirmation
provides:
  - "Definitive proof that XInput native DLL override approach is incompatible with Proton's architecture"
  - "Three distinct failure modes documented with hardware evidence"
affects: [controller, steam-deck, proton]

tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified: []

key-decisions:
  - "XInput caching shim approach FAILED — reverted entirely"
  - "Proton builtin DLLs require Wine's unixlib bridge, cannot be loaded as native copies"
  - "CTRL-03 remains unresolved — deferred beyond v1.1"

patterns-established:
  - "Anti-pattern: Never attempt native DLL override of Proton builtins that use Steam Input IPC"
  - "Anti-pattern: Wine DLL overrides apply to ALL LoadLibrary calls by name, including from within the override DLL itself"
  - "Anti-pattern: Proton overwrites WINEDLLOVERRIDES env var — cannot inject overrides via launch options"

requirements-completed: []

duration: 45min
completed: 2026-02-28
---

# Plan 04: XInput Caching Shim — FAILED

**XInput native DLL override approach proven incompatible with Proton's builtin DLL architecture across three distinct failure modes on Steam Deck hardware**

## Self-Check: FAILED

CTRL-03 NOT satisfied. The XInput caching shim approach does not work due to fundamental incompatibilities between Wine's native DLL override mechanism and Proton's builtin DLL architecture.

## Performance

- **Duration:** ~45 min
- **Tasks:** 1/2 (Task 1 built & tested, Task 2 hardware validation FAILED)
- **Files modified:** 0 (all changes reverted)

## Hardware Testing Results

Three approaches tested on Steam Deck (GE-Proton10-32), all failed:

### Failure 1: Recursive DLL Loading (crash)

- **Approach:** `LoadLibraryExW("xinput1_3.dll", NULL, LOAD_LIBRARY_SEARCH_SYSTEM32)`
- **Expected:** Wine loads Proton's builtin from system directory
- **Actual:** Wine's DLL override (`native,builtin`) intercepts ALL loads by name, returns handle to our own shim → infinite recursion → crash
- **Evidence:** Game process immediately went `<defunct>`

### Failure 2: Renamed Copy (non-functional)

- **Approach:** Copy Proton's builtin `xinput1_3.dll` to game dir as `xinput1_3_real.dll`, shim loads by renamed name
- **Expected:** Renamed copy loads as separate DLL, Steam Input IPC preserved
- **Actual:** Both DLLs loaded (confirmed via `/proc/<pid>/maps`), but renamed copy loaded as NATIVE DLL — Wine's unixlib PE↔Unix bridge NOT established. Steam Input IPC requires builtin loader status. Controller non-functional in-game (works in menu via different input path).
- **Evidence:** `xinput1_3_real.dll` visible in process maps at `6ffff8e40000`, controller dead in-game

### Failure 3: WINEDLLOVERRIDES Environment Variable (ignored)

- **Approach:** Set `WINEDLLOVERRIDES=xinput1_3=n,b` in Steam launch options
- **Expected:** Proton passes override to Wine process
- **Actual:** Proton's `proton` script constructs WINEDLLOVERRIDES internally, overwriting any user-set value. Process env showed Proton's own overrides only (dxgi, d3d11, d3d12, etc.)
- **Workaround:** Used Wine registry `[Software\\Wine\\AppDefaults\\ShippingPC-RealmGameNoEditor.exe\\DllOverrides]` instead — this DID work for loading our native DLL

## Root Cause Analysis

**Proton's builtin xinput1_3.dll uses Wine's "unixlib" architecture.** The PE-side DLL contains stub functions that call through to Unix-side implementations via `__wine_unix_call`. This bridge is only established when Wine loads the DLL as a "builtin" through its internal loader. Loading the same PE file as a "native" DLL (from the game directory, regardless of filename) does NOT establish the unixlib connection.

The Steam Input IPC chain:
```
game.exe → xinput1_3.dll (builtin, PE) → __wine_unix_call → xinput.so (Unix) → winebus.sys → SDL2 → Steam Input → hardware
```

A native copy breaks at `__wine_unix_call` — the Unix-side thunks aren't registered.

## Why This Differs From the Plan's Hypothesis

The plan hypothesized that `LOAD_LIBRARY_SEARCH_SYSTEM32` would bypass the native override and load the builtin. This is incorrect because:
1. Wine's DLL override mechanism operates at a higher level than Windows' library search order
2. The override applies to all loads of a DLL by name, regardless of search flags
3. Even if the builtin could be loaded by path, the override logic intercepts first

## Decisions Made

- **Reverted all code changes** — the approach is fundamentally broken, not just buggy
- **CTRL-03 deferred** — no viable DLL-level approach exists within Proton's architecture
- **Registry override works** but is moot since the loaded DLL can't function as native

## What Would Be Needed (Future Reference)

To fix the ServerTravel controller drop, these approaches remain theoretically possible:
1. **Proton source modification** — Add caching directly into Wine's xinput1_3 builtin (requires custom Proton build)
2. **winebus.sys patch** — Prevent the HID device re-enumeration that causes the transient disconnect
3. **UE3 engine-level fix** — Modify the game binary to retry XInput after ServerTravel (binary patching)
4. **Steam Input API** — Use Steam Input API directly instead of XInput (requires game code changes)

None of these are practical for the Cluckers launcher project scope.

## Task Commits

1. **Task 1: XInput caching shim** — `efd8f70` (feat, then reverted)
2. **Revert** — `571e527` (revert: remove all shim code)

## Next Phase Readiness

- CTRL-03 remains open — not achievable via launcher-side DLL shimming
- Phase 9 infrastructure (Plans 01-03) remains useful for future Steam Deck work
- v1.1 milestone should evaluate whether CTRL-03 can be descoped

---
*Phase: 09-steam-deck-input-again*
*Completed: 2026-02-28*
