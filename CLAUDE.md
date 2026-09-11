# Cluckers - Project Instructions for Claude

## 1. Project Overview

- **Cluckers**: Native CLI launcher for Realm Royale on the Project Crown private server
- **Language**: Go 1.26, single binary, Linux and Windows (amd64), runs game via Wine/Proton-GE on Linux, directly on Windows
- **Module**: `github.com/0xc0re/cluckers`
- **Entry point**: `cmd/cluckers/main.go`
- **CLI framework**: cobra + viper
- **Build**: `go build -o cluckers ./cmd/cluckers`
- **Release**: goreleaser via GitHub Actions on tag push (`v*`)
- **CI**: GitHub Actions (build, test, vet) on all branches, verifies both Linux and Windows builds
- **No CGO**: `CGO_ENABLED=0` for both CI and release builds. Pure Go + embedded binaries.

## 2. Architecture

- **Gateway API** (v1.6.3 protocol): `https://api.project-crown.com` (behind Cloudflare). RESTful JSON API under `/launcher/v1/*`. Success is signalled by HTTP 2xx (there is no `SUCCESS` field); errors are RFC 7807 problem+json (`detail`, `title`, `status`), surfaced as `*ui.UserError` with typed `Status`/`Code`. Bearer auth (`Authorization: Bearer <access_token>`) is used only by `launch-auth` and the supporter bot-name calls. Endpoints: `GET /healthz`, `POST /launcher/v1/session-or-link` (login, body `{user_name,password}`), `POST /launcher/v1/session/refresh` (body `{refresh_token}`, no bearer), `POST /launcher/v1/launch-auth` (bearer, body `{}`, header `x-realm-client-build: <installed game version>`; returns `launch_token` + `portal_info_1`), `POST /launcher/v1/account` (register, same reply shape as login), `POST /launcher/v1/password-reset` (reply `request_id`, `access_token` = reset code, `text_value` = instructions), `GET|PUT|DELETE /launcher/v1/supporter/bot-names[/{slot}]` (bearer). The old `content-bootstrap` and `discord/link*` endpoints are no longer used. The official launcher sends no User-Agent; ours (`gateway.UserAgent`) is informational. Session response fields: `account_id`, `user_name`, `session_id`, `access_token` (`lpt_v1_...`), `expiration_datetime`, `access_expires_at_unix`, `refresh_token`, `refresh_expiration_datetime`, `refresh_expires_at_unix`, `linked_flag`, `custom_message` (supporter tier), `custom_value_1..4`, `text_value`, `portal_info_1`. **2xx login semantics**: `linked_flag != 1` means the account is not Discord-linked and `access_token` carries the LINK CODE (DM it to the Project Crown bot, re-POST `session-or-link` every 3 s until linked); `linked_flag == 1` with `text_value` `PIN_REQUIRED`/`PIN_INVALID` and no token means the server is in developer-only mode (hard stop, the client cannot send a PIN). See `docs/reverse-engineering/cluckers-central-1.6.3/` for the full protocol analysis.
- **Game Server (MCTS)**: `157.90.131.105` (the game now resolves the server itself; `-hostx` is no longer passed)
- **Updater API**: `https://updater.realmhub.io/builds/version.json` (GET, no auth). Returns `base_url`, `manifest_url`, and `gameversion_dat_*` fields. The per-version manifest (`manifest_url`) lists every game file with its relative path, BLAKE3 hash, and size; each file is fetched from `base_url/<path>`. (The old single-`game.zip` scheme with `zip_url`/`zip_blake3`/`zip_size` is gone.) Delta-patch, repair-index, and minisign fields are present in the API but not yet consumed by the client.
- **Game**: UE3-based Win64 binary (`ShippingPC-RealmGameNoEditor.exe`), runs under Wine/Proton-GE on Linux, directly on Windows
- **Launch pipeline**: Sequential steps with spinner UI. Shared steps: health check -> auth (`auth.EnsureSession`: cached access token -> refresh token -> password login; CLI runs the Discord link flow inline) -> verify game installed -> launch authorization (`POST /launcher/v1/launch-auth`, retried once after a token refresh on 401) -> platform steps -> launch game. Linux adds: detect Proton -> ensure compatdata -> resolve Steam integration, then deck config. Windows adds a display-config step (borderless fullscreen INI patch). The launch pipeline does NOT download or update game files -- users must run `cluckers update` separately.
- **Game launch args**: `-user=<name> -token_file=<path> -Language=INT -dx11 -seekfreeloadingpcconsole -nohomedir -content_bootstrap_size=<len> -content_bootstrap_shm=<name>`. The **launch token** from `launch-auth` (NOT the session access token) is written to a temp file passed via `-token_file`; it is short-lived (`launch_expires_at_unix`, ~10 minutes observed live on 2026-09-11; the official launcher deletes its file after 15 minutes). There is no `-eac_oidc_token` or `-hostx` anymore. **`-seekfreeloadingpcconsole` MUST be a single token** — splitting it into `-seekfreeloading -pcconsole` makes UE3 read cooked content from `CookedPC` (empty) instead of `CookedPCConsole`, so it tries to compile shaders and crashes on the missing `UE3ShaderCompileWorker.exe`. Verified against the official launcher's command line.
- **Shared memory**: Game reads content bootstrap via Win32 named shared memory (`OpenFileMapping`). `shm_launcher.exe` (embedded, compiled from C) creates the mapping and launches game as child process. On Linux it runs under Wine; on Windows it runs natively.

## 3. CLI Commands

- `cluckers login` -- Authenticate with gateway, save credentials and cache tokens (access + refresh). If the account is not Discord-linked, prints the link code and polls until linked.
- `cluckers register` -- Create a new account, save credentials, then run the same Discord link flow (the registration reply carries the link code)
- `cluckers reset-password` -- Request a password reset; prints the reset code to DM to the Discord bot plus the server's instructions
- `cluckers launch` -- Full pipeline: auth, tokens, bootstrap, platform setup, game launch
- `cluckers update` -- Check for game updates and download if needed, verify BLAKE3, extract
- `cluckers status` -- Show game, server, gateway status (+ Proton/compatdata on Linux). Compact + verbose modes.
- `cluckers logout` -- Delete encrypted credentials and token cache
- `cluckers self-update` -- Check GitHub releases for a newer launcher binary and download/replace if available
- `cluckers steam add` -- Create .desktop file (Linux) or .bat launcher (Windows) for Steam integration
- `cluckers prep` (Linux only) -- Run auth/tokens/bootstrap/update pipeline and write persistent files for Steam-managed Proton launch
- `cluckers logs` -- Print log file path; `--tail` shows last 50 lines
- `cluckers --version` -- Version info (set via ldflags at build time)

## 4. Code Map

### `cmd/cluckers/`
Entry point. Sets version string from ldflags (`version`, `commit`, `date`), calls `cli.Execute()`.

### `internal/cli/`
Cobra command definitions. Platform-specific behavior uses `_linux.go` / `_windows.go` file naming.
- `root.go`: Root command, persistent flags (`--verbose`, `--gateway`), loads config in PersistentPreRunE. Package-level `Cfg *config.Config`.
- `login.go`: `login` subcommand, authenticates via `launch.LoginInteractive` (login + inline Discord link flow), saves credentials and caches tokens. Uses saved credentials if available, otherwise prompts for username/password.
- `launch.go`: `launch` subcommand, delegates to `launch.Run()`.
- `status.go`: `status` subcommand, shared game/gateway checks and print logic. `protonStatusResult` and `compatdataStatusResult` structs. Calls `platformStatusCheck()` for Proton/compatdata status.
- `status_linux.go`: Linux `platformStatusCheck()` -- Proton detection and compatdata verification.
- `status_windows.go`: Windows `platformStatusCheck()` -- returns nil (no Proton/compatdata).
- `update.go`: `update` subcommand, version check + download + extract pipeline.
- `logout.go`: `logout` subcommand, deletes credentials + token cache.
- `steam.go`: `steam add` subcommand, shared Cobra command definition. Calls `runSteamAdd()`.
- `steam_linux.go`: Linux `runSteamAdd()` -- creates `.desktop` file, detects Steam Deck.
- `steam_windows.go`: Windows `runSteamAdd()` -- creates `.bat` launcher, prints Steam add instructions.
- `register.go`: `register` subcommand, creates account via `auth.Register()`, saves credentials, and runs `launch.WaitForLinkInteractive` when the reply says the account is not linked.
- `resetpassword.go`: `reset-password` subcommand, sends a password reset request to the gateway.
- `selfupdate.go`: `self-update` subcommand, checks GitHub releases via `selfupdate` package, downloads and replaces binary.
- `logs.go`: `logs` subcommand, prints log file path or tails last 50 lines.
- `prep_linux.go`: Linux-only `prep` subcommand, runs full pipeline then writes persistent config for Steam-managed launch.
- `root_gui.go`: GUI build tag. Sets root command `RunE` to launch GUI if display available, with terminal detach support.
- `root_nogui.go`: Non-GUI build tag. Root command shows CLI help (default cobra behavior).
- `detach_linux.go`: Linux `detachSysProcAttr()` for background GUI launch.
- `detach_windows.go`: Windows `detachSysProcAttr()` for background GUI launch.

### `internal/config/`
Configuration and paths. Platform-specific `DataDir()` uses `_linux.go` / `_windows.go` file naming.
- `config.go`: `Config` struct (Gateway, WinePath, GameDir, HostX, Verbose). Loaded via viper from config file (optional). Precedence: CLI flag > config file > default.
- `paths.go`: `ConfigDir()`, `CacheDir()`, `BinDir()`, `LogDir()`, `TmpDir()`, `ConfigFile()`, `CredentialsFile()`, `EnsureDir()`.
- `paths_linux.go`: `DataDir()` -- `CLUCKERS_HOME` env or `~/.cluckers`.
- `paths_windows.go`: `DataDir()` -- `CLUCKERS_HOME` env or `%LOCALAPPDATA%\cluckers`.

### `internal/gateway/`
HTTP client for Project Crown gateway.
- `client.go`: `Client` struct with retryablehttp (3 retries, 500ms-5s backoff, 15s timeout). REST client. `Do(ctx, method, path, bearer, body, result)` / `DoWithHeaders(..., headers, ...)` are the core request methods (success=2xx, RFC 7807 errors -> `*ui.UserError` with `Status`/`Code`, optional `Authorization: Bearer`; pass `struct{}{}` as body to send `{}`). `HealthCheck()` hits `GET /healthz`. `UserAgent` constant. Verbose logs redact `password`, `access_token`, `refresh_token`, `launch_token`, `portal_info_1`, `text_value`.
- `types.go`: Request/response types (`LoginRequest`, `SessionResponse`, `RefreshRequest`, `LaunchAuthResponse`, `RegisterRequest`, `PasswordResetRequest`, `PasswordResetResponse`, `BotNameUpsertRequest`, `HealthResponse`). `FlexBool` custom type handles bool/number/string JSON variants (e.g. `linked_flag`).

### `internal/auth/`
Authentication and credential management.
- `login.go`: REST path constants, `Login()` (`POST /launcher/v1/session-or-link`) and `sessionResultFrom()` which interprets a 2xx session reply: `ErrNotLinked` (+ `*NotLinkedError{LinkCode}`) when `linked_flag != 1`, `ErrPinRequired` on the developer-only gate, otherwise a `LoginResult` with access/refresh tokens and expiries (`expiryFrom` prefers `*_expires_at_unix`, falls back to RFC 3339). `classifyTokenError()` maps HTTP 401/403 to `ErrTokenRejected` via the typed status. `decodeBase64Resilient()` handles std/url-safe/padded/unpadded base64.
- `launchauth.go`: `LaunchAuth(ctx, client, accessToken, clientBuild)` (`POST /launcher/v1/launch-auth`, bearer, body `{}`, `x-realm-client-build` header when non-empty) returning `LaunchAuthResult{Bootstrap, LaunchToken, LaunchExpiresAt}`. Empty `launch_token` is an error; missing `portal_info_1` yields a nil bootstrap.
- `session.go`: `RefreshSession()` (`POST /launcher/v1/session/refresh`, `ErrRefreshRejected` on 401/403) and `EnsureSession(ctx, client, SessionRequest)` -> `Session{Username, AccessToken, Cache, Source}`: cached access token -> refresh -> password login; never prompts; `ErrNoCredentials` when it would need a password it does not have.
- `link.go`: `WaitForLink()` polls `Login` every 3 s (5 min timeout) until the account is Discord-linked, calling `onCode` when the server rotates the link code.
- `credentials.go`: `SaveCredentials()` / `LoadCredentials()` / `DeleteCredentials()`. JSON marshal -> NaCl secretbox encrypt -> write to `credentials.enc` (0600 perms). Machine-bound (key from machine ID).
- `register.go`: `Register()` (`POST /launcher/v1/account`), returns the same `LoginResult`/`ErrNotLinked` outcomes as `Login`.
- `resetpassword.go`: `RequestPasswordReset()` -> `PasswordResetResult{RequestID, Code, Message}`.
- `cache.go`: `TokenCache` struct with `AccessToken`, `RefreshToken`, `Username`, `AccessCachedAt`, `AccessExpiresAt`, `RefreshExpiresAt`. Validity uses the server expiry (minus 60 s skew) when known, else the legacy 45-min TTL from `AccessCachedAt` (old `tokens.json` files keep working). `NewTokenCache()`, `InvalidateAccess()` (drop access, keep refresh), `AccessRemaining()`, `RefreshTokenValid()`. Stored as JSON in cache dir `tokens.json` (0600 perms).
- `supporter.go`: `ListBotNames()`/`UpsertBotName()`/`DeleteBotName()` for the `/launcher/v1/supporter/bot-names[/{slot}]` endpoints (bearer auth, 401 -> `ErrTokenRejected`).

### `internal/crypto/`
NaCl secretbox encryption.
- `secretbox.go`: `DeriveKey()` (machine ID + scrypt N=32768,r=8,p=1 -> 32-byte key), `Encrypt()` (random 24-byte nonce + secretbox.Seal), `Decrypt()` (extract nonce + secretbox.Open). App salt: `cluckers-credential-encryption-v1`.

### `internal/launch/`
Game launch orchestration. Platform-specific behavior uses `_linux.go` / `_windows.go` file naming.
- `pipeline.go`: Shared pipeline infrastructure -- `LaunchState` struct (incl. `LaunchToken`, `LaunchExpiresAt`), `Step` struct, `Run()` loop, signal handling, shared steps (health, `stepAuthenticate` via `auth.EnsureSession` + terminal link flow, verify game installed, `stepLaunchAuth` which resolves the installed build via `game.InstalledVersion` and retries once after renewing the session on 401, launch). `stepLaunchGame` writes the launch token to a temp file (`writeTokenFile`) passed via `-token_file`. Version check and download steps are defined here but only used by the prep pipeline. Calls `platformSteps()` and `platformPostSteps()` for platform-specific steps.
- `interactive.go`: CLI-only helpers `LoginInteractive()` / `WaitForLinkInteractive()` / `PrintLinkCode()` used by the pipeline and the `login`/`register` commands.
- `pipeline_test.go`: httptest-backed tests for `stepAuthenticate` and `stepLaunchAuth` (cache hit, unlinked/PIN outcomes in GUI mode, 401 -> refresh -> retry) and `writeTokenFile`.
- `prep.go` (Linux): `RunPrep()` / `buildPrepSteps()` / `stepWriteLaunchConfig()` -- writes `bootstrap.bin`, `token.txt` (launch token; warns with its expiry), `shm_launcher.exe`, `launch-config.txt` for Steam-managed launch.
- `proton_env.go`, `gamelog.go`, `shortcuts.go`, `reporter*.go`: Proton command/env assembly, game log tailing on failure, Steam shortcuts.vdf parsing, CLI/GUI progress reporters.
- `pipeline_linux.go`: `platformSteps()` returns Proton detect/ensure/resolve steps. `platformPostSteps()` returns deck config step. `stepLaunchGameLinux` dispatches to Steam-managed launch on Deck (writes the prep config with a fresh launch token first). Contains stepDetectProton, stepEnsureCompatdata, stepResolveSteamIntegration, stepDeckConfig.
- `pipeline_windows.go`: `platformSteps()` returns an empty slice; `platformPostSteps()` returns the display-config step (`stepWindowsDisplayConfig`, borderless fullscreen INI patch).
- `process.go`: `LaunchConfig` struct definition (shared). Fields: ProtonScript, ProtonDir, CompatDataPath, SteamInstallPath, SteamGameId, GameDir, Username, LaunchToken, TokenPath (file passed via `-token_file`), ContentBootstrap, Verbose.
- `process_linux.go`: `LaunchGame()` -- Proton-based launch with shm_launcher via `proton run`, LinuxToWinePath conversions, STEAM_COMPAT_DATA_PATH/STEAM_COMPAT_CLIENT_INSTALL_PATH env vars.
- `process_windows.go`: `LaunchGame()` -- Direct native launch, shm_launcher.exe runs natively, no path conversions or Wine env vars.
- `shm.go`: `ExtractSHMLauncher()` (writes embedded exe to temp), `WriteBootstrapFile()` (writes bootstrap bytes to temp). Cross-platform.
- `deckconfig.go`: Linux-only (`//go:build linux`). `PatchDeckConfig()`, `PatchDeckInputConfig()`, `deployDeckControllerLayout()`. Steam Deck specific.

### `internal/game/`
Game file management.
- `version.go`: `FetchVersionInfo()` (GET updater API, 15s timeout), `NeedsUpdate()` (compares GameVersion.dat BLAKE3 hash; also true if a sync was interrupted), `InstalledVersion(ctx, gameDir, pinned)` (dotted build version for `x-realm-client-build`: `.cluckers-installed-version` marker -> pinned version -> updater hash match, writing the marker), `LocalVersion()`, `GameDir()`, `GameExePath()`.
- `manifest.go`: `Manifest`/`ManifestFile` types, `FetchManifest()` (GET `manifest_url`, validates schema 1). The manifest lists every game file with its relative path, BLAKE3 hash, and size.
- `sync.go`: `SyncManifest()` (manifest-based updater: per-file BLAKE3 diff, bounded parallel worker pool downloading `base_url/<path>` with verify + atomic rename, path-traversal guard, clean-sync deletion of files not in the manifest, aggregated progress; writes the `.cluckers-installed-version` marker on success). `IsSyncIncomplete()` checks the `.cluckers-syncing` marker. `ProgressFunc` type. Replaces the old zip download/extract path.
- `diskspace_linux.go`: `checkDiskSpace()` using syscall.Statfs.
- `diskspace_windows.go`: `checkDiskSpace()` using GetDiskFreeSpaceExW.
- `extract_linux.go` / `extract_windows.go`: `prepareTarget()` clears the read-only bit (via `os.Chmod`) before a sync overwrites an existing file.

### `internal/wine/`
Proton-GE detection, compatdata management, and Steam integration. **Linux-only** (all files have `//go:build linux`).
- `detect.go`: `FindProtonGE()` (scans ~10 standard directories + symlink-resolved dirs, sorted newest first), `IsProtonGE()`, `LinuxToWinePath()` (/ -> Z:\), `DetectDistro()` (reads /etc/os-release ID), `IsSteamDeck()`, `userHome()`, `resolveReal()`, `ProtonBaseDir()`.
- `proton.go`: `FindProton()` (configOverride > bundled > system scan), `ProtonInstallInstructions()` (per-distro), `ProtonGEInstall.ProtonScript()`, `ProtonGEInstall.DisplayVersion()`.
- `compatdata.go`: `CompatdataPath()` (returns ~/.cluckers/compatdata), `CompatdataHealthy()` (checks pfx/drive_c exists).
- `steamdir.go`: `FindSteamInstall()` (detects Steam root directory via known install paths).

### `internal/ui/`
Terminal output helpers.
- `logging.go`: `InitLogging()`, `CloseLogging()`, `LogWriter()`. File-based logging singleton with 5MB rotation. All `output.go` functions tee to log.
- `output.go`: `Success()`, `Warn()`, `Error()`, `Info()`, `Verbose()` with color (fatih/color). Each function also writes to log file.
- `errors.go`: `UserError` struct (Message, Detail, Suggestion, Err, Status, Code). `IsStatus(codes...)` for HTTP status checks. `FormatError()` formats based on verbose mode. Implements `error` interface and `Unwrap()`.
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
- `login.go`: `MakeLoginScreen()` -- username/password form, inline error display, Enter-to-submit, Create Account button. Routes `ErrNotLinked` to `ShowDiscordLinking`.
- `register.go`: `MakeRegisterScreen()` -- username/password/email form; `ShowDiscordLinking()` view shows the link code and polls `auth.WaitForLink` (also used by the login screen and by a launch that fails with `ErrNotLinked`).
- `forgot_password.go`: `MakeForgotPasswordScreen()` -- shows the reset code (copied to clipboard) and the server's instructions.
- `main.go`: `MakeMainView()` -- launch button, game management (verify/update/repair) with progress bars, supporter bot names section (auto-detected), community links, settings/logout buttons.
- `settings.go`: `MakeSettingsView()` -- gateway URL, verbose mode, game directory, pinned game version, Proton path (Linux only). Persists via viper TOML.
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
    tokens.json          # {access_token, refresh_token, username, access_cached_at, access_expires_at, refresh_expires_at}
  logs/
    cluckers.log         # Rolling log file (rotated at 5MB)
  game/                  # Game files (managed by update command)
    .cluckers-installed-version  # Build version written by the last sync (sent as x-realm-client-build)
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
    tokens.json          # {access_token, refresh_token, username, access_cached_at, access_expires_at, refresh_expires_at}
  logs\
    cluckers.log         # Rolling log file (rotated at 5MB)
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

- **Launch artifacts**: `POST /launcher/v1/launch-auth` (Bearer auth, body `{}`, `x-realm-client-build` header) returns both the content bootstrap (base64 in `portal_info_1`, 136 bytes with BPS1 magic header; NOT the login response `portal_info_1`, which is a supporter bot-name list) and `launch_token`, the value the game reads from `-token_file`. The launch token is short-lived (`launch_expires_at_unix`, ~10 minutes observed), so it is requested as late as possible in the pipeline and `cluckers prep` warns about its expiry. Live-verified 2026-09-11: `x-realm-client-build: 0.39.6969.0` accepted, 136-byte bootstrap, access token ~1 h, refresh token ~30 days, `*_expiration_datetime` formatted as `2026-09-11_19.53.36`.
- **Discord linking**: There is no separate link endpoint any more. An unlinked account's login reply has `linked_flag != 1` and the link code in `access_token`; the user DMs it to the Project Crown bot and the client re-polls `session-or-link` every 3 s (`auth.WaitForLink`).
- **Developer-only PIN gate**: a login reply with `text_value` `PIN_REQUIRED`/`PIN_INVALID` means the server is closed to regular clients; surface `ErrPinRequired` and stop.
- **Shared memory requirement**: Game uses `OpenFileMapping()`. Passing a file path does NOT work. Must use `CreateFileMappingW(INVALID_HANDLE_VALUE, ...)` via shm_launcher.exe.
- **Proton-GE compatdata**: Proton-GE auto-manages its prefix via `proton run` in the compatdata directory. No manual prefix creation, winetricks, or DLL verification needed.
- **Steam Deck controller**: Controller fix deferred to v1.2+. INI patching (removing Count bXAxis/bYAxis) and Steam Input controller layout VDF are deployed but do not fully resolve controller drop on ServerTravel. Input proxy approach (evdev, XInput DLL) was abandoned -- see Phase 7.1 summary.
- **`-hostx` flag**: REMOVED in v1.2 — the game now resolves the MCTS server itself. Historically it was a required game arg pointing to the MCTS game server IP (157.90.131.105), NOT the gateway. The `HostX` config field/flag remains for backward compatibility but is no longer passed to the game.

## 10. Security Notes

- Credentials encrypted with NaCl secretbox (XSalsa20-Poly1305)
- Key derived from machine ID via scrypt (machine-bound, non-portable)
- Access and refresh tokens cached in plaintext JSON (~/.cluckers/cache/tokens.json, 0600 permissions) for session reuse. The access token expires per the server's `access_expires_at_unix` (45-min fallback for legacy files); an expired or rejected access token is renewed with the refresh token before falling back to the saved password.
- No system keyring dependency (works in Steam Deck Gaming Mode)
- See SECURITY.md for full threat model
