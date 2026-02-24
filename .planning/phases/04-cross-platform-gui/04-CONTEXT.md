# Phase 4: Cross-Platform GUI - Context

**Gathered:** 2026-02-24
**Status:** Ready for planning

<domain>
## Phase Boundary

Build a cross-platform graphical interface for the Cluckers launcher that works on Linux, Windows, and Steam Deck. The GUI becomes the default mode when running `cluckers` with no subcommand. All existing CLI functionality is preserved and accessible via subcommands. The GUI replaces the need for terminal access on Steam Deck.

</domain>

<decisions>
## Implementation Decisions

### Screen layout & features
- Login-first flow: if not logged in, show login screen. Once authenticated, transition to main view
- Main view matches the Windows Project Crown launcher's feature set: launch button, Discord link, donate/support link, game verify/download/repair options, bot name setting for supporters (API-driven)
- Full feature parity with CLI: login, launch, update, status, steam add, logout, settings — all accessible from the GUI

### Visual identity & feel
- Modern minimal design — clean, simple, fast-loading
- Cluckers logo only — no game art or Project Crown branding. Tool-like, not game-themed
- Fullscreen on Steam Deck, windowed on desktop
- Steam Deck compatible — eliminates need for terminal in Gaming Mode

### Launch experience
- Step-by-step progress: each pipeline step (health → auth → OIDC → bootstrap → version check → game start) shown with checkmarks as they complete
- Launcher closes when the game launches (user re-opens to play again)

### CLI coexistence
- Single binary: `cluckers` opens GUI by default, `cluckers launch`, `cluckers update`, etc. still work as CLI subcommands
- Auto-detect headless environments (no display available) and fall back to CLI mode automatically — no error, just works
- Full feature parity: everything CLI does, GUI does

### Claude's Discretion
- Color scheme (dark theme, system-adaptive, or custom)
- Settings UI scope (which config options to expose vs leave in TOML file)
- Download progress presentation during launch (inline vs dedicated view)
- Error display approach (inline in step list vs dialog/modal)
- GUI framework/toolkit selection
- Window size for desktop mode
- Exact layout and component arrangement

</decisions>

<specifics>
## Specific Ideas

- Reference: Windows Project Crown launcher has Discord links, donate/support links, verify/download/repair game options, and a bot name setter for supporters (API-driven)
- "Modern minimal" — think Steam/Heroic Launcher style but simpler. Not a game-art-heavy launcher
- Must use the `frontend-design` skill during implementation for high design quality
- Steam Deck is a key target: fullscreen mode, no terminal required, works in Gaming Mode as a non-Steam game

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 04-cross-platform-gui*
*Context gathered: 2026-02-24*
