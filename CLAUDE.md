# Cluckers

Native Linux CLI launcher for Realm Royale on the Project Crown private server.
Go 1.25+, single static binary, zero runtime deps for users (`CGO_ENABLED=0`).
Handles auth, game file management, Wine prefix setup, and game launch.

## Build and Test

```bash
go build -o cluckers ./cmd/cluckers          # build binary
go test ./...                                  # run all tests
go vet ./...                                   # lint
go test ./internal/crypto/ -run TestSecretbox  # single test example
```

Build tags: none (pure Go + go:embed). Version set via ldflags at build time (goreleaser pattern).

## Project Structure

```
cmd/cluckers/      main entry point
internal/
  cli/             Cobra commands (launch, update, status, logout)
  auth/            credential storage (NaCl secretbox encrypted)
  gateway/         API client (retryablehttp, POST /json/<CMD>)
  game/            version checking, download, extraction
  wine/            Wine/Proton-GE detection, prefix verification
  launch/          launch pipeline (shm bridge, process management)
  config/          Viper config, paths (~/.cluckers/)
  crypto/          NaCl secretbox encryption
  ui/              terminal output (spinners, colors, UserError pattern)
assets/            embedded shm_launcher.exe via go:embed
```

## Conventions

- **Error handling**: Use `*ui.UserError{Message, Detail, Suggestion}` for user-facing errors, stdlib errors for internal.
- **Config**: Viper with TOML file at `~/.cluckers/config/settings.toml`. CLI flags override via `BindPFlag` in `init()`.
- **API client**: `gateway.NewClient(url, verbose)` with retryablehttp. All endpoints are `POST /json/<COMMAND>`.
- **Verbose output**: Gate behind `cfg.Verbose` or `ui.Verbose(msg, cfg.Verbose)`. Never print debug info by default.
- **Version**: Set via ldflags at build time (goreleaser pattern).
- **CLUCKERS_HOME**: Env var overrides `~/.cluckers` base directory.
- **Wine prefix**: Path resolved once and stored in `LaunchState.PrefixPath`.
- **Game exe path**: Use `game.GameExePath(gameDir)` for consistency across pipeline and process.

## Do NOT

- Use CGO (must be pure Go, static binary)
- Use jsonwebtoken (CommonJS issues with Wine/Edge runtime context -- use jose if JWT needed)
- Hardcode game server IP (comes from config, default 157.90.131.105)
- Print verbose/debug output without checking `cfg.Verbose`
- Use D-Bus or system keyring (not available on Steam Deck Gaming Mode)

## Key Architecture Notes

- Gateway API is behind Cloudflare at `gateway-dev.project-crown.com`.
- Game server (MCTS) is direct connection at `157.90.131.105`, separate from gateway.
- `shm_launcher.exe` creates Win32 named shared memory for content bootstrap -- game reads it via `OpenFileMapping()`.
- Content bootstrap (136 bytes, BPS1 magic) comes from `LAUNCHER_CONTENT_BOOTSTRAP` endpoint, NOT login response.
- EAC is disabled server-side, null client works.
