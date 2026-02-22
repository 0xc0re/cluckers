# Cluckers

Native Linux launcher for Realm Royale on the Project Crown private server.

## What It Does

- Authenticates with the Project Crown gateway
- Downloads and manages game files with auto-update
- Sets up Wine prefix with required dependencies
- Launches Realm Royale under Wine/Proton-GE
- Stores credentials encrypted on disk (no system keyring needed)
- Works on desktop Linux and Steam Deck

## Prerequisites

- Go 1.25+ (build only)
- Wine or Proton-GE (runtime)
- winetricks (for prefix setup)
- A Project Crown account

## Install

```
git clone https://github.com/0xc0re/cluckers.git
cd cluckers
go build -o cluckers ./cmd/cluckers
sudo cp cluckers /usr/local/bin/  # optional
```

## Usage

### `cluckers launch`

Authenticate and launch the game. Prompts for credentials on first run, saves
them for future use.

### `cluckers update`

Check for game updates and download if available.

### `cluckers status`

Show Wine, prefix, game, server, and gateway status. Add `-v` for verbose
output with additional details.

### `cluckers logout`

Remove saved credentials.

### `cluckers --version`

Show version info.

## Configuration

Config file: `~/.cluckers/config/settings.toml` (optional, created manually).

```toml
gateway = "https://gateway-dev.project-crown.com"
hostx = "your.server.ip"
wine_path = ""       # auto-detected if empty
wine_prefix = ""     # defaults to ~/.cluckers/prefix/
game_dir = ""        # defaults to ~/.cluckers/game/
verbose = false
```

CLI flags: `--gateway`, `-v/--verbose`

Environment variable: `CLUCKERS_HOME` overrides the `~/.cluckers` base
directory.

## Directory Structure

Created at runtime:

```
~/.cluckers/
  config/
    settings.toml        # optional config
    credentials.enc      # encrypted login credentials
  game/                  # game files (managed by update command)
  prefix/                # Wine prefix (auto-created on launch)
```

## Steam Deck

- Add the cluckers binary as a non-Steam game in Desktop Mode
- Set launch options if needed
- Works in Gaming Mode once configured
- Proton-GE is auto-detected from Steam's compatibilitytools.d

## License

License: TBD
