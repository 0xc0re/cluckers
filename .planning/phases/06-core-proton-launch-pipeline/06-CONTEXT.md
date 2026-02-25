# Phase 6: Core Proton Launch Pipeline - Context

**Gathered:** 2026-02-24
**Status:** Ready for planning

<domain>
## Phase Boundary

Replace direct Wine execution with Proton-GE's `proton run` for all Linux launches. Includes Proton detection, automatic prefix management at `~/.cluckers/compatdata/pfx/`, correct environment setup (STEAM_COMPAT_DATA_PATH, PROTON_WINEDLLOVERRIDES, etc.), and verified shm_launcher.exe operation under Proton. Controller/Gamescope integration and codebase cleanup are separate phases (7 and 8).

</domain>

<decisions>
## Implementation Decisions

### Proton detection priority
- Detection order: Bundled (CLUCKERS_BUNDLED_PROTON env) > Config override > System scan of known directories
- When no Proton-GE found: Error with per-distro install instructions, launcher exits (no fallback to system Wine)
- Version display: Always show detected version (e.g., "Detected Proton-GE 9-27"), source path shown only in verbose mode
- Old Proton-GE versions: Warn but allow (e.g., "Proton-GE 7 detected, version 9+ recommended"), non-blocking

### First-launch experience
- Dedicated spinner step for prefix creation: "Preparing Proton environment (first launch only)..." — sets expectations about one-time wait
- Quick prefix health check every launch: verify compatdata directory exists and pfx/drive_c is present
- Corrupted or missing prefix: Auto-recreate with warning ("Proton environment damaged, recreating..."), delete old compatdata and rebuild
- Setup completion: Success checkmark only ("✔ Proton environment ready"), verbose mode shows compatdata path

### Error recovery
- Launch failure: Show Proton log path + 2-3 common fixes (delete compatdata and relaunch, update Proton-GE, verify game files)
- Proton stderr/stdout: Capture output, show last ~10 lines on crash for immediate context, full output in verbose mode
- PROTON_LOG=1: Only enabled when user runs with -v flag, keeps compatdata tidy on normal launches
- SHM bridge failures: Distinct error message separate from general Proton failures — detect shm_launcher exit codes/patterns and show specific guidance ("Shared memory bridge failed — try deleting compatdata and relaunching")

### Claude's Discretion
- Exact Proton environment variable set (beyond those specified in requirements)
- Proton version parsing implementation
- Prefix health check implementation details
- stderr/stdout capture mechanism
- shm_launcher exit code detection approach

</decisions>

<specifics>
## Specific Ideas

- Follow the existing `UserError` pattern (Message + Detail + Suggestion) for all Proton-related errors
- Proton detection should extend the existing `FindProtonGE()` scan approach, adding bundled priority on top
- Spinner step names should feel consistent with existing pipeline steps (health check, auth, OIDC, etc.)
- Per-distro error messages for missing Proton-GE, similar to how current Wine errors give distro-specific guidance

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 06-core-proton-launch-pipeline*
*Context gathered: 2026-02-24*
