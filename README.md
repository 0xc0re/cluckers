# Cluckers

Native launcher for Realm Royale on the Project Crown private server, for Linux and Windows.

> **Note:** Cluckers is a community project and is not developed or maintained by
> the Project Crown team. It relies on Project Crown's public gateway API, which
> may change without notice. If the API changes, Cluckers may stop working until
> it is updated to match.

## What It Does

- Authenticates with the Project Crown gateway
- Downloads and manages game files with auto-update
- Launches Realm Royale (via Wine/Proton-GE on Linux, directly on Windows)
- Stores credentials encrypted on disk (no system keyring needed)
- Self-updates the launcher binary from GitHub Releases
- Works on desktop Linux, Steam Deck, and Windows

## Prerequisites

### Linux

- Wine or Proton-GE
- winetricks (for prefix setup with system Wine; not needed with Proton-GE)
- A Project Crown account

### Windows

- A Project Crown account

No additional dependencies are required on Windows. The game runs natively.

## Install

### Quick install -- Linux

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

### Quick install -- Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/0xc0re/cluckers/master/install.ps1 | iex
```

This downloads the latest release binary to `%LOCALAPPDATA%\cluckers\bin\cluckers.exe`
and adds it to your user PATH.

You can also run the script directly:

```powershell
powershell -ExecutionPolicy Bypass -File install.ps1
```

### Manual download -- Linux

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

### Manual download -- Windows

1. Go to [GitHub Releases](https://github.com/0xc0re/cluckers/releases)
2. Download the `cluckers_*_windows_amd64.zip` file
3. Extract the zip and place `cluckers.exe` somewhere in your PATH (e.g. `%LOCALAPPDATA%\cluckers\bin\`)

### Build from source

Requires Go 1.25+ and mingw-w64 (for cross-compiling the embedded SHM helper):

```bash
git clone https://github.com/0xc0re/cluckers.git
cd cluckers
x86_64-w64-mingw32-gcc -o assets/shm_launcher.exe tools/shm_launcher.c -municode
go build -o cluckers ./cmd/cluckers
```

To build for Windows from Linux:

```bash
GOOS=windows go build -o cluckers.exe ./cmd/cluckers
```

## Usage

### `cluckers launch`

Authenticate and launch the game. Prompts for credentials on first run, saves
them for future use.

### `cluckers login`

Authenticate with the Project Crown gateway and save credentials without
launching the game.

### `cluckers update`

Check for game updates and download if available.

### `cluckers self-update`

Check for launcher updates and download the latest version from GitHub Releases,
replacing the current binary in place.

### `cluckers status`

Show game, server, and gateway status. On Linux, also shows Wine and prefix
status. Add `-v` for verbose output with additional details.

### `cluckers logout`

Remove saved credentials and cached tokens.

### `cluckers steam add`

Add Cluckers as a non-Steam game in your Steam library. On Linux, creates a
`.desktop` file (useful for launching from Gaming Mode on Steam Deck). On
Windows, creates a `.bat` launcher and prints instructions for adding it to Steam.

### `cluckers --version`

Show version info.

**Note:** On Windows, Wine-related status output and prefix management commands
are not applicable and are automatically skipped.

## Configuration

### Linux

Config file: `~/.cluckers/config/settings.toml` (optional, created manually).

```toml
gateway = "https://gateway-dev.project-crown.com"
wine_path = ""       # auto-detected if empty (Linux only)
wine_prefix = ""     # defaults to ~/.cluckers/prefix/ (Linux only)
game_dir = ""        # defaults to ~/.cluckers/game/
verbose = false
```

### Windows

Config file: `%LOCALAPPDATA%\cluckers\config\settings.toml` (optional, created manually).

```toml
gateway = "https://gateway-dev.project-crown.com"
game_dir = ""        # defaults to %LOCALAPPDATA%\cluckers\game\
verbose = false
```

The `wine_path` and `wine_prefix` settings are Linux-only and have no effect on Windows.

### Common options

CLI flags: `--gateway`, `-v/--verbose`

Environment variable: `CLUCKERS_HOME` overrides the base data directory
(`~/.cluckers` on Linux, `%LOCALAPPDATA%\cluckers` on Windows).

## Directory Structure

### Linux

Created at runtime:

```
~/.cluckers/
  config/
    settings.toml        # optional config
    credentials.enc      # encrypted login credentials
  cache/
    tokens.json          # cached auth tokens
  game/                  # game files (managed by update command)
  prefix/                # Wine prefix (auto-created on launch)
```

### Windows

Created at runtime:

```
%LOCALAPPDATA%\cluckers\
  config\
    settings.toml        # optional config
    credentials.enc      # encrypted login credentials
  cache\
    tokens.json          # cached auth tokens
  game\                  # game files (managed by update command)
    Realm-Royale\
      Binaries\
        Win64\
          ShippingPC-RealmGameNoEditor.exe
        GameVersion.dat  # local version marker
```

## Steam Deck

1. Install via the quick install script (works in Desktop Mode terminal or SSH)
2. Install Proton-GE via ProtonUp-Qt from the Discover store
3. Run `cluckers steam add` to add it to your Steam library
4. In Steam, find "Realm Royale (Cluckers)" and launch it

The launcher auto-detects Steam Deck and configures display settings (fullscreen
1280x800). Proton-GE is auto-detected from Steam's `compatibilitytools.d`
directory.

**Controller support:** Controller input on Steam Deck is not currently
supported. Keyboard and mouse work normally.

## Windows

On Windows, Realm Royale runs natively without Wine or any compatibility layer.
The launcher handles authentication, game downloads, and updates the same way as
on Linux. `cluckers steam add` creates a `.bat` launcher for adding the game to
your Steam library on Windows.

## License

License: TBD
