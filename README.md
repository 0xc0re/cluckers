# Cluckers

Native Linux launcher for Realm Royale on the Project Crown private server.

> **Note:** Cluckers is a community project and is not developed or maintained by
> the Project Crown team. It relies on Project Crown's public gateway API, which
> may change without notice. If the API changes, Cluckers may stop working until
> it is updated to match.

## What It Does

- Authenticates with the Project Crown gateway
- Downloads and manages game files with auto-update
- Sets up Wine prefix with required dependencies
- Launches Realm Royale under Wine/Proton-GE
- Stores credentials encrypted on disk (no system keyring needed)
- Works on desktop Linux and Steam Deck

## Prerequisites

- Wine or Proton-GE (runtime)
- winetricks (for prefix setup with system Wine; not needed with Proton-GE)
- A Project Crown account

## Install

### Quick install (recommended)

```bash
curl -sSL https://raw.githubusercontent.com/0xc0re/cluckers/master/install.sh | sh
```

This downloads the latest release binary to `~/.local/bin/cluckers`. To install
elsewhere, set `INSTALL_DIR`:

```bash
INSTALL_DIR=/usr/local/bin curl -sSL https://raw.githubusercontent.com/0xc0re/cluckers/master/install.sh | sh
```

**Steam Deck:** Works out of the box. The installer detects SteamOS and places
the binary in `~/.local/bin`. If you don't have Proton-GE installed, grab
ProtonUp-Qt from the Discover store and install the latest GE-Proton version.

### Manual download

Grab the latest release from [GitHub Releases](https://github.com/0xc0re/cluckers/releases):

```bash
# Download latest release
curl -s https://api.github.com/repos/0xc0re/cluckers/releases/latest \
  | grep "browser_download_url.*tar.gz\"" \
  | cut -d '"' -f 4 \
  | xargs curl -LO
tar xzf cluckers_*.tar.gz
chmod +x cluckers
mv cluckers ~/.local/bin/  # or wherever you prefer
```

### Build from source

Requires Go 1.25+ and mingw-w64 (for cross-compiling the embedded SHM helper):

```bash
git clone https://github.com/0xc0re/cluckers.git
cd cluckers
x86_64-w64-mingw32-gcc -o assets/shm_launcher.exe tools/shm_launcher.c -municode
go build -o cluckers ./cmd/cluckers
```

## Usage

### `cluckers login`

Authenticate with the Project Crown gateway and save credentials for future
launches. Uses saved credentials if available, otherwise prompts for username
and password.

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

### `cluckers steam add`

Add Cluckers as a non-Steam game in your Steam library. Creates a `.desktop` file
so Steam can find it. Useful for launching from Steam's Gaming Mode on Steam Deck.

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

1. Install via the quick install script (works in Desktop Mode terminal or SSH)
2. Install Proton-GE via ProtonUp-Qt from the Discover store
3. Run `cluckers steam add` to add it to your Steam library
4. In Steam, find "Realm Royale (Cluckers)" and launch it

**Controller input:** Controller support on Steam Deck works automatically. The
launcher patches game configuration files to prevent input mode auto-switching
and deploys a Steam controller layout. On first launch, right-click "Realm Royale
(Cluckers)" in Steam > Properties > Controller and select the **"Gamepad with
Joystick Trackpad"** template.

Do NOT set `STEAM_INPUT_DISABLE=1` or other SDL environment variables. Steam Input
must remain active to forward controller inputs to the virtual Xbox 360 pad that
Wine reads via XInput.

Proton-GE is auto-detected from Steam's `compatibilitytools.d` directory.

## License

License: TBD
