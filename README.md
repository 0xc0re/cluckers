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
- Includes a graphical interface (GUI) -- release binaries open the GUI when
  double-clicked or run without arguments on a desktop
- Works on desktop Linux, Steam Deck, and Windows

## Prerequisites

### Linux

Choose either format -- both include the full GUI + CLI single binary:

**AppImage:** Self-contained -- bundles Proton-GE and all graphics libraries. No
prerequisites beyond a Project Crown account. Larger download (~1.5 GB).

**Tarball:** Smaller download. Requires Wine or Proton-GE installed separately,
plus winetricks for prefix setup with system Wine (not needed with Proton-GE).
Also requires OpenGL and X11 graphics libraries, which are pre-installed on most
desktop Linux distributions. If you are on a minimal or server setup, install them:

- **Debian/Ubuntu:** `sudo apt install libgl1-mesa-glx libx11-6 libxrandr2 libxcursor1 libxi6 libxinerama1 libxxf86vm1`
- **Fedora:** `sudo dnf install mesa-libGL libX11 libXrandr libXcursor libXi libXinerama libXxf86vm`
- **Arch:** These are typically pre-installed; if not: `sudo pacman -S mesa libx11 libxrandr libxcursor libxi libxinerama libxxf86vm`

Both formats require a Project Crown account.

### Windows

- A Project Crown account

No additional dependencies are required on Windows. The game runs natively.

## Install

### Quick install -- Linux

```bash
curl -sSL https://raw.githubusercontent.com/0xc0re/cluckers/master/install.sh | sh
```

The installer downloads the latest release to `~/.local/bin/cluckers`. When
available, it prefers the AppImage (which bundles Proton-GE) over the tarball.
To install elsewhere, set `INSTALL_DIR`:

```bash
INSTALL_DIR=/usr/local/bin curl -sSL https://raw.githubusercontent.com/0xc0re/cluckers/master/install.sh | sh
```

**Steam Deck:** Works out of the box. The installer detects SteamOS and places
the binary in `~/.local/bin`. The AppImage bundles Proton-GE, so no additional
setup is needed.

### Quick install -- Windows (PowerShell)

```powershell
(iwr https://raw.githubusercontent.com/0xc0re/cluckers/master/install.ps1).Content | iex
```

This downloads the latest release binary to `%LOCALAPPDATA%\cluckers\bin\cluckers.exe`
and adds it to your user PATH.

You can also run the script directly:

```powershell
powershell -ExecutionPolicy Bypass -File install.ps1
```

### Manual download -- Linux

Grab the latest release from [GitHub Releases](https://github.com/0xc0re/cluckers/releases).

**AppImage:** Single file, bundles Proton-GE and graphics libraries. No Wine or
system library setup needed.

```bash
curl -s https://api.github.com/repos/0xc0re/cluckers/releases/latest \
  | grep "browser_download_url.*Cluckers-x86_64.AppImage\"" \
  | cut -d '"' -f 4 \
  | xargs curl -LO
chmod +x Cluckers-x86_64.AppImage
mv Cluckers-x86_64.AppImage ~/.local/bin/cluckers
```

**Tarball:** Smaller download. Requires Wine or Proton-GE and graphics libraries
(OpenGL, X11) installed on the system -- see prerequisites above.

```bash
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

See [CONTRIBUTING.md](CONTRIBUTING.md) for build prerequisites and instructions.

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

Add Cluckers as a non-Steam game in your Steam library.

**Linux / Steam Deck:** Creates a `.desktop` file at
`~/.local/share/applications/cluckers.desktop`. The `.desktop` file includes
`launch` in its Exec line, so no additional launch options are needed in Steam.
After running the command, open Steam in Desktop Mode, go to Games > Add a
Non-Steam Game, and select "Realm Royale (Cluckers)" from the list. On Steam
Deck, switch back to Game Mode afterwards -- the game appears in the Non-Steam
section.

**Windows:** Prints step-by-step instructions and shows the exact path to
`cluckers.exe`. The default install location is
`%LOCALAPPDATA%\cluckers\bin\cluckers.exe`. In Steam, browse to and add the
executable, then right-click the entry in your library, open Properties, and set
**Launch Options** to: `launch`. Without this launch option, opening the entry
in Steam will show the GUI instead of launching the game directly. Optionally
rename the entry to "Realm Royale" in Properties.

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

## Adding to Steam

### Linux / Steam Deck

1. Run `cluckers steam add` -- this creates a `.desktop` file at
   `~/.local/share/applications/cluckers.desktop`
2. Open Steam in Desktop Mode
3. Go to Games > Add a Non-Steam Game to My Library
4. Select "Realm Royale (Cluckers)" from the list and click Add
5. (Steam Deck) Switch back to Game Mode -- the game appears in the Non-Steam
   section of your library

No launch option is needed on Linux. The `.desktop` file already includes the
`launch` command.

### Windows

1. Run `cluckers steam add` to see the exact path to your `cluckers.exe`
   (default: `%LOCALAPPDATA%\cluckers\bin\cluckers.exe`)
2. Open Steam > Games > Add a Non-Steam Game to My Library
3. Click Browse, change the file filter to "All Files (\*.\*)"
4. Navigate to the cluckers.exe location, select it, and click Open
5. Click "Add Selected Programs"
6. Right-click "cluckers" in your library > Properties
7. Set **Launch Options** to: `launch`
8. (Optional) Rename the entry to "Realm Royale" in Properties

**Important:** The `launch` option in step 7 is required. Without it, opening
the entry in Steam will show the graphical settings UI instead of launching the
game directly, because release binaries include the GUI.

## Steam Deck

The quick install script works in Desktop Mode terminal or SSH. When available,
it installs the AppImage which bundles Proton-GE -- no additional setup needed.

The launcher auto-detects Steam Deck and configures display settings (fullscreen
1280x800).

**Controller support:** Controller input on Steam Deck is not currently
supported. Keyboard and mouse work normally.

## Windows

On Windows, Realm Royale runs natively without Wine or any compatibility layer.
The launcher handles authentication, game downloads, and updates the same way as
on Linux. After downloading game files, the launcher automatically configures
borderless fullscreen and makes game settings writable so in-game changes persist.

Double-clicking `cluckers.exe` opens the graphical interface where you can log
in, adjust settings, and launch the game. All CLI commands (`cluckers launch`,
`cluckers update`, etc.) still work from a terminal.

To add the game to Steam, see the "Adding to Steam" section above or run
`cluckers steam add` for step-by-step instructions.

## License

License: TBD
