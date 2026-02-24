# Project State

Last activity: 2026-02-24 - Completed Phase 04 Plan 01: Fyne GUI Foundation

## Current Phase Execution

- **Phase:** 04-cross-platform-gui
- **Current Plan:** 2 of 5
- **Last Completed:** 04-01-PLAN.md (Fyne GUI foundation)
- **Last Session:** 2026-02-24T15:51:58Z
- **Stopped At:** Completed 04-01-PLAN.md

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 10 | Resolve Steam Deck controller input not recognized by Realm Royale | 2026-02-23 | 5409539 | [10-resolve-steam-deck-controller-input-not-](./quick/10-resolve-steam-deck-controller-input-not-/) |
| 11 | Create CLAUDE.md and code map | 2026-02-23 | n/a (no git commit) | [11-create-claude-md-and-code-map](./quick/11-create-claude-md-and-code-map/) |
| 13 | Resolve controller input loss in match (XInputEnable + WINEDLLOVERRIDES fix) | 2026-02-23 | e95ea1b | [13-resolve-controller-input-loss-in-match](./quick/13-resolve-controller-input-loss-in-match/) |
| 14 | Launch-readiness code review: remove committed binary, fix docs, deduplicate isSteamDeck | 2026-02-24 | 7db50fd | [14-perform-a-thorough-review-of-the-code-an](./quick/14-perform-a-thorough-review-of-the-code-an/) |
| 15 | Add Windows build target with cross-platform code split | 2026-02-24 | 8cdd503 | [15-make-a-windows-version-no-wine-needed-ju](./quick/15-make-a-windows-version-no-wine-needed-ju/) |
| 16 | Implement self-update command for launcher binary replacement via GitHub releases | 2026-02-24 | a534a0c | [16-implement-self-update-we-have-update-for](./quick/16-implement-self-update-we-have-update-for/) |
| 17 | Split token cache into independent per-token TTL timestamps | 2026-02-24 | e3f7dec | [17-login-and-launch-should-refresh-oidc-tok](./quick/17-login-and-launch-should-refresh-oidc-tok/) |
| 19 | Update docs and install script for Windows support | 2026-02-24 | 6f943ae | [19-implement-windows-builds-and-releases-up](./quick/19-implement-windows-builds-and-releases-up/) |
| 20 | Resolve code scanning alert (add CI workflow permissions) | 2026-02-24 | e5ec06c | [20-resolve-code-scanning-results-from-githu](./quick/20-resolve-code-scanning-results-from-githu/) |

## Accumulated Context

### Roadmap Evolution
- Phase 4 added: cross platform gui

### Decisions
- All GUI package files use //go:build gui tag to keep CLI-only build path clean (CGO_ENABLED=0)
- Steam Deck detection in GUI package is independent of wine package (uses DMI board vendor)
- GUI binary: CGO_ENABLED=1 go build -tags gui; CLI-only binary: CGO_ENABLED=0 go build (unchanged)
- Fyne v2.7.3 selected as GUI framework (most mature Go GUI, cross-platform, works on SteamOS)
