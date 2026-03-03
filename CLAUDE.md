# Cluckers - Project Instructions for Claude

## 1. Project Overview

- **Cluckers**: Native CLI launcher for Realm Royale on the Project Crown private server
- **Language**: Go 1.25, single binary, Linux and Windows (amd64), runs game via Wine/Proton-GE on Linux, directly on Windows
- **Module**: `github.com/0xc0re/cluckers`
- **Entry point**: `cmd/cluckers/main.go`
- **CLI framework**: cobra + viper
- **Build**: `go build -o cluckers ./cmd/cluckers`
- **Release**: goreleaser via GitHub Actions on tag push (`v*`)
- **CI**: GitHub Actions (build, test, vet) on all branches, verifies both Linux and Windows builds
- **No CGO**: `CGO_ENABLED=0` for both CI and release builds. Pure Go + embedded binaries.

## 2. Architecture

- **Gateway API**: `https://gateway-dev.project-crown.com` (behind Cloudflare). All API calls are `POST /json/<COMMAND>` with JSON body/response. Commands: `LAUNCHER_HEALTH`, `LAUNCHER_LOGIN_OR_LINK`, `LAUNCHER_REGISTER`, `LAUNCHER_REQUEST_LINK_CODE`, `LAUNCHER_DISCORD_STATUS`, `LAUNCHER_EAC_OIDC_TOKEN`, `LAUNCHER_CONTENT_BOOTSTRAP`, `LAUNCHER_SUPPORTER_BOT_NAME_UPSERT`, `LAUNCHER_SUPPORTER_BOT_NAMES_LIST`, `LAUNCHER_SUPPORTER_BOT_NAME_DELETE`.
- **Game Server (MCTS)**: `157.90.131.105` (direct TCP, separate from gateway)
- **Updater API**: `https://updater.realmhub.io/builds/version.json` (GET, no auth, returns version info with zip URL and BLAKE3 hash)
- **Game**: UE3-based Win64 binary (`ShippingPC-RealmGameNoEditor.exe`), runs under Wine/Proton-GE on Linux, directly on Windows
- **Launch pipeline**: Sequential steps with spinner UI. Shared steps: health check -> auth -> OIDC token -> content bootstrap -> verify game installed -> launch game. Linux adds: detect Proton -> ensure compatdata -> resolve Steam integration (before verify) and deck config (after verify). Windows skips Proton/compatdata/deck steps entirely. The launch pipeline does NOT download or update game files -- users must run `cluckers update` separately.
- **Shared memory**: Game reads content bootstrap via Win32 named shared memory (`OpenFileMapping`). `shm_launcher.exe` (embedded, compiled from C) creates the mapping and launches game as child process. On Linux it runs under Wine; on Windows it runs natively.

## 3. CLI Commands

- `cluckers login` -- Authenticate with gateway, save credentials and cache tokens
- `cluckers register` -- Create a new account, save credentials, request Discord link code with optional poll for linking
- `cluckers launch` -- Full pipeline: auth, tokens, bootstrap, platform setup, game launch
- `cluckers update` -- Check for game updates and download if needed, verify BLAKE3, extract
- `cluckers status` -- Show game, server, gateway status (+ Proton/compatdata on Linux). Compact + verbose modes.
- `cluckers logout` -- Delete encrypted credentials and token cache
- `cluckers self-update` -- Check GitHub releases for a newer launcher binary and download/replace if available
- `cluckers steam add` -- Create .desktop file (Linux) or .bat launcher (Windows) for Steam integration
- `cluckers prep` (Linux only) -- Run auth/tokens/bootstrap/update pipeline and write persistent files for Steam-managed Proton launch
- `cluckers --version` -- Version info (set via ldflags at build time)

## 4. Code Map

### `cmd/cluckers/`
Entry point. Sets version string from ldflags (`version`, `commit`, `date`), calls `cli.Execute()`.

### `internal/cli/`
Cobra command definitions. Platform-specific behavior uses `_linux.go` / `_windows.go` file naming.
- `root.go`: Root command, persistent flags (`--verbose`, `--gateway`), loads config in PersistentPreRunE. Package-level `Cfg *config.Config`.
- `login.go`: `login` subcommand, authenticates with gateway, saves credentials and caches tokens. Uses saved credentials if available, otherwise prompts for username/password.
- `launch.go`: `launch` subcommand, delegates to `launch.Run()`.
- `status.go`: `status` subcommand, shared game/gateway checks and print logic. `protonStatusResult` and `compatdataStatusResult` structs. Calls `platformStatusCheck()` for Proton/compatdata status.
- `status_linux.go`: Linux `platformStatusCheck()` -- Proton detection and compatdata verification.
- `status_windows.go`: Windows `platformStatusCheck()` -- returns nil (no Proton/compatdata).
- `update.go`: `update` subcommand, version check + download + extract pipeline.
- `logout.go`: `logout` subcommand, deletes credentials + token cache.
- `steam.go`: `steam add` subcommand, shared Cobra command definition. Calls `runSteamAdd()`.
- `steam_linux.go`: Linux `runSteamAdd()` -- creates `.desktop` file, detects Steam Deck.
- `steam_windows.go`: Windows `runSteamAdd()` -- creates `.bat` launcher, prints Steam add instructions.
- `register.go`: `register` subcommand, creates account via `auth.Register()`, saves credentials, requests Discord link code, polls for Discord linking status.
- `selfupdate.go`: `self-update` subcommand, checks GitHub releases via `selfupdate` package, downloads and replaces binary.
- `prep_linux.go`: Linux-only `prep` subcommand, runs full pipeline then writes persistent config for Steam-managed launch.
- `root_gui.go`: GUI build tag. Sets root command `RunE` to launch GUI if display available, with terminal detach support.
- `root_nogui.go`: Non-GUI build tag. Root command shows CLI help (default cobra behavior).
- `detach_linux.go`: Linux `detachSysProcAttr()` for background GUI launch.
- `detach_windows.go`: Windows `detachSysProcAttr()` for background GUI launch.

### `internal/config/`
Configuration and paths. Platform-specific `DataDir()` uses `_linux.go` / `_windows.go` file naming.
- `config.go`: `Config` struct (Gateway, WinePath, GameDir, HostX, Verbose). Loaded via viper from config file (optional). Precedence: CLI flag > config file > default.
- `paths.go`: `ConfigDir()`, `CacheDir()`, `ConfigFile()`, `CredentialsFile()`, `EnsureDir()`.
- `paths_linux.go`: `DataDir()` -- `CLUCKERS_HOME` env or `~/.cluckers`.
- `paths_windows.go`: `DataDir()` -- `CLUCKERS_HOME` env or `%LOCALAPPDATA%\cluckers`.

### `internal/gateway/`
HTTP client for Project Crown gateway.
- `client.go`: `Client` struct with retryablehttp (3 retries, 500ms-5s backoff, 15s timeout). `Post()` method for JSON POST to `/json/<command>`. `HealthCheck()`. User-Agent: `CluckersCentral/1.1.68`. Returns `*ui.UserError` on failures.
- `types.go`: Request/response types (`LoginRequest`, `LoginResponse`, `OIDCTokenResponse`, `BootstrapResponse`, `GenericRequest`, `RegisterRequest`, `RegisterResponse`, `LinkCodeRequest`, `LinkCodeResponse`, `DiscordStatusResponse`, `BotNameUpsertRequest`, `BotNameDeleteRequest`, `BotNameResponse`). `FlexBool` custom type handles bool/number/string JSON variants.

### `internal/auth/`
Authentication and credential management.
- `login.go`: `Login()` (LAUNCHER_LOGIN_OR_LINK), `GetOIDCToken()` (LAUNCHER_EAC_OIDC_TOKEN, tries PORTAL_INFO_1 -> STRING_VALUE -> TEXT_VALUE), `GetContentBootstrap()` (LAUNCHER_CONTENT_BOOTSTRAP, base64-decodes PORTAL_INFO_1, fixes padding, returns raw bytes with BPS1 magic header).
- `credentials.go`: `SaveCredentials()` / `LoadCredentials()` / `DeleteCredentials()`. JSON marshal -> NaCl secretbox encrypt -> write to `credentials.enc` (0600 perms). Machine-bound (key from machine ID).
- `register.go`: `Register()` (LAUNCHER_REGISTER), `RequestLinkCode()` (LAUNCHER_REQUEST_LINK_CODE), `CheckDiscordStatus()` (LAUNCHER_DISCORD_STATUS).
- `cache.go`: `TokenCache` struct with `AccessToken`, `OIDCToken`, `Username`, `CachedAt`. TTLs: access=24h, OIDC=55min. Stored as JSON in cache dir `tokens.json` (0600 perms).

### `internal/crypto/`
NaCl secretbox encryption.
- `secretbox.go`: `DeriveKey()` (machine ID + scrypt N=32768,r=8,p=1 -> 32-byte key), `Encrypt()` (random 24-byte nonce + secretbox.Seal), `Decrypt()` (extract nonce + secretbox.Open). App salt: `cluckers-credential-encryption-v1`.

### `internal/launch/`
Game launch orchestration. Platform-specific behavior uses `_linux.go` / `_windows.go` file naming.
- `pipeline.go`: Shared pipeline infrastructure -- `LaunchState` struct, `Step` struct, `Run()` loop, signal handling, shared steps (health, auth, OIDC, bootstrap, verify game installed, launch). Version check and download steps are defined here but only used by the prep pipeline. Calls `platformSteps()` and `platformPostSteps()` for platform-specific steps.
- `pipeline_linux.go`: `platformSteps()` returns Proton detect/ensure/resolve steps. `platformPostSteps()` returns deck config step. Contains stepDetectProton, stepEnsureCompatdata, stepResolveSteamIntegration, stepDeckConfig.
- `pipeline_windows.go`: `platformSteps()` and `platformPostSteps()` return empty slices (no Proton/compatdata/deck).
- `process.go`: `LaunchConfig` struct definition (shared). Fields: ProtonScript, ProtonDir, CompatDataPath, SteamInstallPath, SteamGameId, GameDir, Username, AccessToken, OIDCTokenPath, ContentBootstrap, HostX, Verbose.
- `process_linux.go`: `LaunchGame()` -- Proton-based launch with shm_launcher via `proton run`, LinuxToWinePath conversions, STEAM_COMPAT_DATA_PATH/STEAM_COMPAT_CLIENT_INSTALL_PATH env vars.
- `process_windows.go`: `LaunchGame()` -- Direct native launch, shm_launcher.exe runs natively, no path conversions or Wine env vars.
- `shm.go`: `ExtractSHMLauncher()` (writes embedded exe to temp), `WriteBootstrapFile()` (writes bootstrap bytes to temp). Cross-platform.
- `deckconfig.go`: Linux-only (`//go:build linux`). `PatchDeckConfig()`, `PatchDeckInputConfig()`, `deployDeckControllerLayout()`. Steam Deck specific.

### `internal/game/`
Game file management.
- `version.go`: `FetchVersionInfo()` (GET updater API, 15s timeout), `NeedsUpdate()` (compares GameVersion.dat BLAKE3 hash), `LocalVersion()`, `GameDir()`, `GameExePath()`.
- `download.go`: `DownloadGameZip()` (HTTP Range resume, progress bar, ~5.3GB), `VerifyBLAKE3()`, `DownloadAndVerify()` (download + verify, deletes corrupt).
- `diskspace_linux.go`: `checkDiskSpace()` using syscall.Statfs.
- `diskspace_windows.go`: `checkDiskSpace()` using GetDiskFreeSpaceExW.
- `extract.go`: `ExtractZip()` (zip-slip protection, progress counter, removes zip after extraction). Calls `prepareTarget()` before each file overwrite.
- `extract_linux.go`: `prepareTarget()` no-op (Unix allows owner overwrite regardless).
- `extract_windows.go`: `prepareTarget()` clears read-only attribute via `os.Chmod` before overwrite.

### `internal/wine/`
Proton-GE detection, compatdata management, and Steam integration. **Linux-only** (all files have `//go:build linux`).
- `detect.go`: `FindProtonGE()` (scans ~10 standard directories + symlink-resolved dirs, sorted newest first), `IsProtonGE()`, `LinuxToWinePath()` (/ -> Z:\), `DetectDistro()` (reads /etc/os-release ID), `IsSteamDeck()`, `userHome()`, `resolveReal()`, `ProtonBaseDir()`.
- `proton.go`: `FindProton()` (configOverride > bundled > system scan), `ProtonInstallInstructions()` (per-distro), `ProtonGEInstall.ProtonScript()`, `ProtonGEInstall.DisplayVersion()`.
- `compatdata.go`: `CompatdataPath()` (returns ~/.cluckers/compatdata), `CompatdataHealthy()` (checks pfx/drive_c exists).
- `steamdir.go`: `FindSteamInstall()` (detects Steam root directory via known install paths).

### `internal/ui/`
Terminal output helpers.
- `output.go`: `Success()`, `Warn()`, `Error()`, `Info()`, `Verbose()` with color (fatih/color).
- `errors.go`: `UserError` struct (Message, Detail, Suggestion, Err). `FormatError()` formats based on verbose mode. Implements `error` interface and `Unwrap()`.
- `prompt.go`: `PromptUsername()` (reads line), `PromptPassword()` (hidden input via x/term). Both check `term.IsTerminal()`.
- `spinner.go`: `StepSpinner` wraps briandowns/spinner. `StartStep()`, `Stop()`, `Success()`, `Fail()`. Non-TTY fallback prints plain text.

### `internal/gui/`
Fyne-based graphical user interface. Built only with `gui` build tag.
- `app.go`: GUI entry point. `Run()` checks credentials, shows login or main view. System tray support (desktop only), close-to-tray when game running. Screen navigation: login -> register -> Discord linking -> main -> settings -> launch progress.
- `theme.go`: `cluckersTheme` custom dark theme (Material green primary, dark backgrounds).
- `detect.go` / `detect_linux.go` / `detect_windows.go`: `CanShowGUI()` display detection. `deck_linux.go` / `deck_windows.go`: `isSteamDeck()` detection.
- `assets/`: Embedded logo resource.

### `internal/gui/screens/`
GUI screen implementations.
- `login.go`: `MakeLoginScreen()` -- username/password form, inline error display, Enter-to-submit, Create Account button.
- `register.go`: `MakeRegisterScreen()` -- username/password/email form, Discord link code flow with `showDiscordLinking()` polling view.
- `main.go`: `MakeMainView()` -- launch button, game management (verify/update/repair) with progress bars, supporter bot names section (auto-detected), community links, settings/logout buttons.
- `settings.go`: `MakeSettingsView()` -- gateway URL, verbose mode, game directory, Proton path (Linux only). Persists via viper TOML.
- `launch_progress.go`: `MakeLaunchProgressView()` -- pipeline step list with live status updates, cancel button.

### `internal/gui/widgets/`
Reusable GUI components.
- `step_list.go`: `StepListWidget` -- vertical list of pipeline steps with status icons (pending/running/done/failed/skipped).

### `internal/selfupdate/`
Launcher self-update via GitHub releases.
- `selfupdate.go`: Checks latest release tag, compares semantic versions, downloads platform-appropriate archive, verifies checksums, replaces binary.
- `replace_linux.go` / `replace_windows.go`: Platform-specific binary replacement.

### `assets/`
Embedded binary assets.
- `embed.go`: `//go:embed shm_launcher.exe` and `//go:embed controller_neptune_config.vdf`. Two embedded assets: the SHM launcher helper and the Steam Deck controller layout VDF.
- `controller_neptune_config.vdf`: Steam Deck (Neptune) controller layout for Realm Royale.

### `tools/`
Build-time source files (not embedded directly).
- `shm_launcher.c`: C source for the SHM launcher. Build: `x86_64-w64-mingw32-gcc -o assets/shm_launcher.exe tools/shm_launcher.c -municode`

## 5. Key Dependencies

- `spf13/cobra` + `spf13/viper` -- CLI framework + config
- `hashicorp/go-retryablehttp` -- HTTP client with retry/backoff
- `fatih/color` -- Terminal colors
- `briandowns/spinner` -- Terminal spinners
- `schollz/progressbar/v3` -- Download progress bars
- `zeebo/blake3` -- BLAKE3 hashing for file integrity
- `denisbrodbeck/machineid` -- Machine ID for key derivation
- `golang.org/x/crypto` -- NaCl secretbox + scrypt
- `golang.org/x/term` -- Terminal detection + password input

## 6. Conventions and Patterns

- **Error handling**: Use `*ui.UserError` for user-facing errors (Message + Detail + Suggestion). Return `fmt.Errorf` wrapping for internal errors. All gateway errors are wrapped as UserError with suggestions.
- **Verbose output**: Gated by `Config.Verbose` / `-v` flag. Use `ui.Verbose(msg, isVerbose)`.
- **Idempotent operations**: Compatdata preparation, deck config patching, and controller layout deployment all check current state before acting.
- **Graceful degradation**: Health check warns but continues. Missing bootstrap warns but continues. Token cache failures are non-fatal.
- **File permissions**: Credentials and token cache use 0600. Directories use 0700 (EnsureDir) or 0755.
- **Path resolution**: `config.DataDir()` respects `CLUCKERS_HOME` env var. Default: `~/.cluckers` (Linux) or `%LOCALAPPDATA%\cluckers` (Windows).
- **Testing**: Tests use `t.TempDir()` + `t.Setenv("CLUCKERS_HOME", tmp)` pattern to isolate file operations.
- **Commit messages**: Conventional commits (`feat:`, `fix:`, `docs:`, `refactor:`, `ci:`, `chore:`). Goreleaser groups changelog by prefix.
- **No CGO**: `CGO_ENABLED=0` for both CI and release builds. Pure Go + embedded binaries.
- **Build tags**: Platform-specific code uses file naming convention (`_linux.go`, `_windows.go`). The `internal/wine/` package uses `//go:build linux` comment tags since all files are Linux-only.

## 7. Runtime Directory Structure

### Linux
```
~/.cluckers/
  config/
    settings.toml        # optional TOML config
    credentials.enc      # NaCl secretbox encrypted JSON {username, password}
  cache/
    tokens.json          # {access_token, oidc_token, username, cached_at}
  game/                  # Game files (managed by update command)
    Realm-Royale/
      Binaries/
        Win64/
          ShippingPC-RealmGameNoEditor.exe
        GameVersion.dat  # Local version marker
      RealmGame/
        Config/
          RealmSystemSettings.ini  # Patched on Steam Deck
  compatdata/            # Proton compatibility data (auto-created on first launch)
```

### Windows
```
%LOCALAPPDATA%\cluckers\
  config\
    settings.toml        # optional TOML config
    credentials.enc      # NaCl secretbox encrypted JSON {username, password}
  cache\
    tokens.json          # {access_token, oidc_token, username, cached_at}
  game\                  # Game files (managed by update command)
    Realm-Royale\
      Binaries\
        Win64\
          ShippingPC-RealmGameNoEditor.exe
        GameVersion.dat  # Local version marker
```

## 8. Build Instructions

```bash
# Build shm_launcher.exe from source (requires mingw-w64)
# NOTE: shm_launcher.exe is not committed to git. Build it before running go build.
x86_64-w64-mingw32-gcc -o assets/shm_launcher.exe tools/shm_launcher.c -municode

# Standard build (Linux)
go build -o cluckers ./cmd/cluckers

# Windows cross-compile
GOOS=windows go build -o cluckers.exe ./cmd/cluckers

# Run tests
go test ./...

# Vet (both platforms)
go vet ./...
GOOS=windows go vet ./...
```

## 9. Critical Domain Knowledge

- **Content bootstrap**: Comes from `LAUNCHER_CONTENT_BOOTSTRAP` endpoint (NOT from login response PORTAL_INFO_1 which is a cosmetics list). 136 bytes with BPS1 magic header, base64-encoded.
- **Shared memory requirement**: Game uses `OpenFileMapping()`. Passing a file path does NOT work. Must use `CreateFileMappingW(INVALID_HANDLE_VALUE, ...)` via shm_launcher.exe.
- **Proton-GE compatdata**: Proton-GE auto-manages its prefix via `proton run` in the compatdata directory. No manual prefix creation, winetricks, or DLL verification needed.
- **Steam Deck controller**: Controller fix deferred to v1.2+. INI patching (removing Count bXAxis/bYAxis) and Steam Input controller layout VDF are deployed but do not fully resolve controller drop on ServerTravel. Input proxy approach (evdev, XInput DLL) was abandoned -- see Phase 7.1 summary.
- **`-hostx` flag**: Required game arg pointing to MCTS game server IP (157.90.131.105), NOT the gateway.

## 10. Security Notes

- Credentials encrypted with NaCl secretbox (XSalsa20-Poly1305)
- Key derived from machine ID via scrypt (machine-bound, non-portable)
- Access tokens in memory only, not persisted (only cached token hashes)
- No system keyring dependency (works in Steam Deck Gaming Mode)
- See SECURITY.md for full threat model
