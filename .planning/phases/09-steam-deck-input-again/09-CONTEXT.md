# Phase 9: Steam Deck Input (again) - Context

**Gathered:** 2026-02-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Solve the Steam Deck controller button loss during UE3 ServerTravel (lobby-to-match transition). Buttons must persist through the D3D window recreation. This phase delivers a working solution validated on actual Steam Deck hardware.

Out of scope: new CLI commands beyond what's needed for this fix, GUI changes, Windows platform changes.

</domain>

<decisions>
## Implementation Decisions

### Attack vector
- **Primary approach: Steam-managed Proton launch** — `cluckers prep` writes auth/token/bootstrap files, then Steam launches `shm_launcher.exe` via Proton as a non-Steam game shortcut. Steam/Gamescope fully manage the game lifecycle, which is how real Steam games avoid the ServerTravel controller disconnect.
- **Secondary: Gamescope window tracking** — If Steam-managed launch alone doesn't fix controller, investigate X11 property manipulation or Gamescope configuration to ensure the new D3D window is recognized after ServerTravel.
- **Last resort: HID feature report injection** — Reverse-engineer Steam's HID commands to controller firmware to keep game mode active during ServerTravel. Only attempted if first two approaches fail.
- Multiple approaches can be tried within this phase — the goal is "do whatever it takes" to get controller working.

### Launch architecture
- `cluckers prep` already exists and is fully implemented (writes bootstrap.bin, oidc-token.txt, shm_launcher.exe, launch-config.txt to ~/.cluckers/bin/ and ~/.cluckers/cache/).
- `shm_launcher.c` already supports reading launch-config.txt — no C code changes expected.
- Steam shortcut needs to be created pointing to `~/.cluckers/bin/shm_launcher.exe` with Proton-GE forced as compatibility tool.
- The prep command's help text references `WINEDLLOVERRIDES=dxgi=n` which is KNOWN TO CRASH — must be removed. Launch options should be: `/path/to/cluckers prep && %command%` with NO WINEDLLOVERRIDES.

### Claude's Discretion
- **Launch mode selection**: Whether `cluckers launch` auto-detects Steam Deck and uses Steam-managed mode, or keeps both direct-proton and Steam-managed as separate paths. Recommendation: auto-detect Deck hardware, use Steam-managed on Deck, direct proton on desktop Linux.
- **CLI triggers Steam vs prep-only**: Whether `cluckers launch` programmatically triggers the Steam shortcut (via `steam://rungameid/`) or the user launches from Steam. Recommendation: CLI triggers Steam launch after prep, for a seamless single-command experience.
- **Shortcut automation vs instructions**: Whether `cluckers steam add` modifies `shortcuts.vdf` directly or prints instructions. Recommendation: automate shortcuts.vdf modification — the VDF binary format is well-documented and this eliminates manual steps.
- **Fallback behavior**: Whether direct `proton run` remains as a fallback. Recommendation: keep both modes — direct proton run continues working for desktop Linux users who don't need Steam-managed launch.

### Scope and success criteria
- **Hard requirement**: Controller buttons (A/B/X/Y, bumpers, triggers) persist through ServerTravel on Steam Deck. CTRL-03 must be fully satisfied.
- **Hardware validation required**: Phase 9 is NOT complete until tested on actual Steam Deck hardware and controller works through a lobby-to-match transition.
- **Multi-approach persistence**: If the primary approach (Steam-managed launch) doesn't fully fix controller, continue with secondary and tertiary approaches within this phase. Only stop when it works or all viable options are exhausted.
- **One-time manual setup acceptable**: If automating shortcuts.vdf proves too fragile, falling back to printed instructions for initial Steam shortcut setup is acceptable. After initial setup, all subsequent launches should be automatic.

</decisions>

<specifics>
## Specific Ideas

- The existing `cluckers prep` pipeline is the foundation — it handles auth/tokens/bootstrap/update and writes persistent files that shm_launcher.exe reads.
- Deploy 11 finding: when launching through Steam, Proton uses Steam's OWN compatdata (`steamapps/compatdata/<appid>/`), not `~/.cluckers/compatdata`. This means any Wine prefix patches from earlier phases won't be present — but that's fine since we're NOT doing DLL overrides or prefix manipulation.
- The prep command's `WINEDLLOVERRIDES=dxgi=n` in the help text is a known crash trigger — must be cleaned up.
- The entire thesis: by letting Steam manage the Proton lifecycle, Gamescope should track the game window through ServerTravel and keep Steam Input in game mode. This is untested — hardware validation will confirm or deny.

</specifics>

<deferred>
## Deferred Ideas

- Windows Steam integration improvements — separate concern, not related to Deck controller
- External USB/BT controller documentation — only if all approaches fail and we need a workaround doc
- Steam community bug report for ServerTravel — worth doing regardless but not a code deliverable

</deferred>

---

*Phase: 09-steam-deck-input-again*
*Context gathered: 2026-02-27*
