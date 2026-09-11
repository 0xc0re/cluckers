# Cluckers client map + live gateway probe report

Repo: `/home/cstory/src/cluckers` (module `github.com/0xc0re/cluckers`). The `cluckers-central/` subdirectory was excluded from all greps and reads.

All file paths below are absolute. Line numbers are from the working tree at commit `f51ec81`.

---

## Section 1 — Live gateway probes (read-only; no credentials sent)

All probes used `curl` against `https://api.project-crown.com` unless noted.

### 1.1 Route existence matrix

| Path | Method | Status | `allow` header |
|---|---|---|---|
| `/healthz` | GET | 200 | — |
| `/launcher/v1/session-or-link` | GET | 405 | `POST` |
| `/launcher/v1/session` | GET | 405 | `POST,DELETE` |
| `/launcher/v1/session/refresh` | GET | 405 | `POST` |
| `/launcher/v1/account` | GET | 405 | `POST` |
| `/launcher/v1/discord/link/code` | GET | 405 | `POST` |
| `/launcher/v1/discord/link` | GET | 401 | — |
| `/launcher/v1/password-reset` | GET | 405 | `POST` |
| `/launcher/v1/supporter/bot-names` | GET | 401 | — |
| `/launcher/v1/content-bootstrap` | GET | 401 | — |
| `/launcher/v1/launch-authsrc` | GET, POST, PUT, OPTIONS | 404 | none, `content-length: 0` |

### 1.2 Key conclusions from the matrix

**Content bootstrap still exists.** It returns 401 `missing_bearer`, not 404. The official 1.6.3 launcher stopped referencing it, but the route is still served. Our dependence on it is not automatically broken.

**`/launcher/v1/session` now allows DELETE.** This is new and is almost certainly server-side session revocation. We never call it; `cluckers logout` is purely local.

**`/launcher/v1/session/refresh` is live and functional.** POST only.

**`/launcher/v1/launch-authsrc` is not deployed.** 404 with an empty body and no `allow` header on every method. I also tried these spellings, all 404:
`/launcher/v1/launch_authsrc`, `/launcher/v1/launch-auth-src`, `/launcher/v1/launch/authsrc`, `/launcher/v1/launch-authsrc/`.
It is either feature-flagged off, gated behind a header we did not send, or not yet shipped server-side.

### 1.3 User-Agent does not matter

`/healthz` returns a byte-identical body for `CluckersCentral/1.2.54`, `CluckersCentral/1.6.3`, and no User-Agent at all. Repeating the `launch-authsrc`, `content-bootstrap`, and `session-or-link` probes under UA 1.6.3 produced identical statuses (404 / 401 / 405). **Our stale UA string is not the cause of the breakage.** Updating it is hygiene, not a fix.

### 1.4 `/healthz` body (leaks service topology)

```json
{"service":"session-gateway","status":"ok",
 "player_core_grpc_endpoint":"http://127.0.0.1:31051",
 "config_service_base_url":"https://config.project-crown.com",
 "shop_service_internal_base_url":"http://127.0.0.1:19088",
 "platform_identity_base_url":"https://auth.project-crown.com",
 "matchmaking_grpc_endpoint":"http://127.0.0.1:31052",
 "map_service_grpc_endpoint":"http://127.0.0.1:31053",
 "instance_manager_grpc_endpoint":"http://127.0.0.1:31086",
 "social_service_grpc_endpoint":"http://127.0.0.1:31055",
 "replay_service_grpc_endpoint":"http://127.0.0.1:31058"}
```

Our `HealthResponse` (`/home/cstory/src/cluckers/internal/gateway/types.go:38`) parses only `service` and `status`; everything else is dropped. `config_service_base_url` and `platform_identity_base_url` are the two the client could plausibly need.

### 1.5 The other two hosts

| Host | `/healthz` | Notes |
|---|---|---|
| `config.project-crown.com` | 200 `{"service":"config-service","status":"ok"}` | `launch-authsrc` 404 on all tried paths |
| `auth.project-crown.com` | **502** | 502 on every path including `/healthz` |

**`auth.project-crown.com` returns Cloudflare 502 on everything.** The platform identity service is down or not publicly routed. This is a live anomaly worth flagging, though we do not currently call that host.

---

## Section 2 — The gateway leaks its request schema (serde)

The backend is Rust with serde. A `POST` of `{}` returns **HTTP 422** with a plain-text body naming the first missing field. This enumerates request shapes without sending any credential.

| Endpoint | 1st required field | 2nd required field |
|---|---|---|
| `/launcher/v1/session/refresh` | `refresh_token` | — |
| `/launcher/v1/session-or-link` | `user_name` | `password` |
| `/launcher/v1/session` | `user_name` | `password` |
| `/launcher/v1/account` | `user_name` | `password` |
| `/launcher/v1/discord/link/code` | `user_name` | not probed |
| `/launcher/v1/password-reset` | `user_name` | none (succeeds) |

Exact 422 body form:

```
Failed to deserialize the JSON body into the target type: missing field `refresh_token` at line 1 column 2
```

I advanced one step by sending `{"user_name":"probe"}` (a nonexistent username, no password) and stopped there. I did not send any password to any endpoint.

### 2.1 Two informative live responses

Dummy refresh token, well-formed RFC 7807:

```json
{"detail":"Launcher refresh session is invalid or expired","status":401,
 "title":"invalid_refresh_session","type":"about:blank"}
```

Password reset with only a username returns **200** and real user-facing instructional text:

```json
{"request_id":"d7d7f074-5133-44c3-9a01-c395322fe044",
 "text_value":"If the account exists, DM this code to the Discord bot, then reply with your new password."}
```

That `text_value` is an instruction the user needs, and our code throws it away (see 6.4).

### 2.2 Further enumeration is possible

The serde error reports one missing field at a time, so the full request schema of any POST endpoint can be walked by supplying dummy values. Advancing past `password` means sending a password value, which I deliberately did not do. Flagging it as an available next step if the lead wants the complete shapes.

---

## Section 3 — Every gateway call site in our client

### 3.1 Path constants

All in one block: `/home/cstory/src/cluckers/internal/auth/login.go:17-24`

```go
pathSessionOrLink    = "/launcher/v1/session-or-link"     // :17
pathSession          = "/launcher/v1/session"             // :18  UNUSED
pathContentBootstrap = "/launcher/v1/content-bootstrap"   // :19
pathAccount          = "/launcher/v1/account"             // :20
pathDiscordLink      = "/launcher/v1/discord/link"        // :21
pathDiscordLinkCode  = "/launcher/v1/discord/link/code"   // :22
pathPasswordReset    = "/launcher/v1/password-reset"      // :23
pathBotNames         = "/launcher/v1/supporter/bot-names" // :24
```

`pathSession` at `:18` is declared and referenced nowhere else in the repo. There is no `/launcher/v1/session` call and no session DELETE.

### 3.2 Call site table

| Function | file:line | Method + path | Bearer | Request type | Response type |
|---|---|---|---|---|---|
| `Login` | `/home/cstory/src/cluckers/internal/auth/login.go:43` | POST `/launcher/v1/session-or-link` | no | `gateway.LoginRequest` | `gateway.SessionResponse` |
| `GetContentBootstrap` | `/home/cstory/src/cluckers/internal/auth/login.go:77` | GET `/launcher/v1/content-bootstrap` | yes | none | `gateway.BootstrapResponse` |
| `Register` | `/home/cstory/src/cluckers/internal/auth/register.go:20` | POST `/launcher/v1/account` | no | `gateway.RegisterRequest` | `gateway.SessionResponse` |
| `RequestLinkCode` | `/home/cstory/src/cluckers/internal/auth/register.go:52` | POST `/launcher/v1/discord/link/code` | no | `gateway.LinkCodeRequest` | `gateway.LinkCodeResponse` |
| `CheckDiscordStatus` | `/home/cstory/src/cluckers/internal/auth/register.go:76` | GET `/launcher/v1/discord/link` | yes | none | `gateway.DiscordStatusResponse` |
| `RequestPasswordReset` | `/home/cstory/src/cluckers/internal/auth/resetpassword.go:14` | POST `/launcher/v1/password-reset` | no | `gateway.PasswordResetRequest` | `gateway.PasswordResetResponse` (discarded) |
| `ListBotNames` | `/home/cstory/src/cluckers/internal/auth/supporter.go:15` | GET `/launcher/v1/supporter/bot-names` | yes | none | `[]string` |
| `UpsertBotName` | `/home/cstory/src/cluckers/internal/auth/supporter.go:25` | PUT `.../bot-names/{slot}` | yes | `gateway.BotNameUpsertRequest` | none |
| `DeleteBotName` | `/home/cstory/src/cluckers/internal/auth/supporter.go:33` | DELETE `.../bot-names/{slot}` | yes | none | none |
| `HealthCheck` | `/home/cstory/src/cluckers/internal/gateway/client.go:191` | GET `/healthz` | no | none | `gateway.HealthResponse` |

Bot-name slots are 1-indexed and formatted into the path at `supporter.go:26` and `:34` via `fmt.Sprintf("%s/%d", pathBotNames, slot)`.

### 3.3 Transport

`Client.Do(ctx, method, path, bearer, body, result)` at `/home/cstory/src/cluckers/internal/gateway/client.go:85`.

- Client construction `client.go:56-69`: retryablehttp, `RetryMax = 3` (`:58`), `RetryWaitMin = 500ms` (`:59`), `RetryWaitMax = 5s` (`:60`), `HTTPClient.Timeout = 15s` (`:62`), logger suppressed (`:61`).
- Headers: `Accept: application/json` (`:103`), `User-Agent` (`:104`), `Content-Type: application/json` only when a body exists (`:106`), `Authorization: Bearer <token>` when bearer is non-empty (`:109`).
- `UserAgent = "CluckersCentral/1.2.54"` at `client.go:19`.
- Verbose logging redacts `password`, `access_token`, `portal_info_1`, `text_value` via `sensitiveKeys` (`client.go:22-27`) and `sanitizeJSON` (`:31-46`), called at `:134` and `:138`.
- Success is any 2xx (`:141`); `result == nil` or an empty body skips decoding (`:145`).

### 3.4 Current JSON tags — request types

```go
// /home/cstory/src/cluckers/internal/gateway/types.go:45   LoginRequest
UserName string `json:"user_name"`   // :46
Password string `json:"password"`    // :47

// types.go:80   RegisterRequest
UserName string `json:"user_name"`   // :81
Password string `json:"password"`    // :82
Email    string `json:"email"`       // :83

// types.go:87   LinkCodeRequest
UserName string `json:"user_name"`   // :88
Password string `json:"password"`    // :89

// types.go:107  PasswordResetRequest
UserName string `json:"user_name"`   // :108

// types.go:119  BotNameUpsertRequest   (slot is in the path)
BotName string `json:"bot_name"`     // :120
```

### 3.5 Current JSON tags — response types

```go
// types.go:38   HealthResponse
Service string `json:"service"`   // :39
Status  string `json:"status"`    // :40

// types.go:54   SessionResponse  (session-or-link, session, account)
AccountID          json.Number `json:"account_id"`          // :55
UserName           string      `json:"user_name"`           // :56
SessionID          string      `json:"session_id"`          // :57
AccessToken        string      `json:"access_token"`        // :58
ExpirationDatetime string      `json:"expiration_datetime"` // :59
LinkedFlag         FlexBool    `json:"linked_flag"`         // :60
CustomMessage      string      `json:"custom_message"`      // :61
CustomValue1       json.Number `json:"custom_value_1"`      // :62
TextValue          string      `json:"text_value"`          // :63
PortalInfo1        string      `json:"portal_info_1"`       // :64

// types.go:69   BootstrapResponse
AccountID          json.Number `json:"account_id"`          // :70
SessionID          string      `json:"session_id"`          // :71
Version            json.Number `json:"version"`             // :72
CustomValue1       json.Number `json:"custom_value_1"`      // :73
CustomValue2       json.Number `json:"custom_value_2"`      // :74
ExpirationDatetime string      `json:"expiration_datetime"` // :75
PortalInfo1        string      `json:"portal_info_1"`       // :76

// types.go:93   LinkCodeResponse
Code        string   `json:"code"`         // :94
AccessToken string   `json:"access_token"` // :95
LinkedFlag  FlexBool `json:"linked_flag"`  // :96

// types.go:100  DiscordStatusResponse
LinkedFlag     FlexBool `json:"linked_flag"`      // :101
PortalUserID   string   `json:"portal_userid"`    // :102
PortalUsername string   `json:"portal_username"`  // :103

// types.go:112  PasswordResetResponse
RequestID string `json:"request_id"`  // :113
TextValue string `json:"text_value"`  // :114
```

`FlexBool` at `types.go:13` with `UnmarshalJSON` at `:15` accepts bool, float64 (non-zero true), and string (via `strconv.ParseBool`, warning on parse failure at `:28`); anything else becomes false.

**There is no `refresh_token` field anywhere in our types.** That is the largest structural gap against the new API.

`SessionResponse` is unmarshalled in exactly two places: `/home/cstory/src/cluckers/internal/auth/login.go:46` and `/home/cstory/src/cluckers/internal/auth/register.go:27`.

### 3.6 Error classification and 401 handling

Non-2xx path: `client.go:141` → `errorFromResponse` at `client.go:162`.

1. HTML body (Cloudflare/nginx page) → "Gateway unreachable" `UserError` (`:164-170`), Detail is `fmt.Sprintf("HTTP %d from %s %s", ...)` at `:167`.
2. RFC 7807 parse (`problemDetails` struct at `:72-77`, fields `detail`/`title`/`status`/`type`) → `UserError{Message: pd.Detail, Detail: fmt.Sprintf("HTTP %d (%s)", status, pd.Title)}` at `:178-181`.
3. Fallback → `UserError{Message: fmt.Sprintf("Gateway error (HTTP %d).", status)}` at `:184-187`.

**The HTTP status is preserved only as formatted text inside `UserError.Detail`.** No typed status field exists.

`classifyTokenError` at `/home/cstory/src/cluckers/internal/auth/login.go:101` recovers it by string matching:

```go
if strings.Contains(ue.Detail, "HTTP 401") || strings.Contains(ue.Detail, "HTTP 403") {   // :104
    return &ui.UserError{
        Message:    message + ": " + ue.Message,
        Suggestion: "Your session may have expired. Try logging out and back in.",        // :107
        Err:        ErrTokenRejected,                                                     // :108
    }
}
```

`ErrTokenRejected` is defined at `login.go:30`.

**`classifyTokenError` is called from exactly one place: `login.go:80`, inside `GetContentBootstrap`.** Only the content-bootstrap call can ever produce `ErrTokenRejected`. `Login`, `Register`, `RequestLinkCode`, `CheckDiscordStatus`, `ListBotNames`, `UpsertBotName`, `DeleteBotName`, and `RequestPasswordReset` all return unclassified `*ui.UserError`.

### 3.7 Base64 decoding of the bootstrap

`decodeBase64Resilient` at `login.go:118` strips whitespace (`:120`), pads to a multiple of 4 (`:124-126`), then tries in order: StdEncoding padded (`:129`), URLEncoding padded (`:134`), RawStdEncoding (`:139`), RawURLEncoding (`:144`). An empty `portal_info_1` returns `(nil, nil)` and is explicitly not an error (`login.go:83-85`).

---

## Section 4 — Launch pipeline

### 4.1 Step construction

`buildSteps` at `/home/cstory/src/cluckers/internal/launch/pipeline.go:173`:

1. `Checking gateway` → `stepHealthCheck` (`pipeline.go:190`) — warns and continues on failure (`:193`)
2. `Authenticating` → `stepAuthenticate` (`pipeline.go:203`)
3. `Requesting content bootstrap` → `stepBootstrap` (`pipeline.go:323`)
4. `platformSteps(state)` (`:179`)
5. `Verifying game installation` → `stepVerifyGameInstalled` (`pipeline.go:449`)
6. `platformPostSteps(state)` (`:183`)
7. `platformLaunchStep()` (`:184`)

Linux `platformLaunchStep` at `/home/cstory/src/cluckers/internal/launch/pipeline_linux.go:140` returns `stepLaunchGameLinux` (`:148`), which branches: on Steam Deck with a shortcut appid it calls `stepWriteLaunchConfig` then `launchViaSteam` (`:149-156`); otherwise it falls through to the direct `proton run` path. Windows `platformLaunchStep` at `/home/cstory/src/cluckers/internal/launch/pipeline_windows.go:22` returns `stepLaunchGame` directly.

Windows `platformPostSteps` (`pipeline_windows.go:28`) is **not** empty as CLAUDE.md claims; it returns `Configuring display` → `stepWindowsDisplayConfig` (`:34`), which patches `RealmSystemSettings.ini` for borderless fullscreen and makes INI files writable.

Entry points: `Run` (`pipeline.go:70`), `RunWithReporter` (`:77`), `RunWithReporterAndCreds` (`:126`, GUI), `StepNames` (`:161`, GUI pre-render).

Signal handling at `pipeline.go:95-104`: a dedicated channel (not `ctx.Done()`) removes the token temp file and `os.Exit(130)`.

### 4.2 Exact game argument list

From `/home/cstory/src/cluckers/internal/launch/process_linux.go:46-53` and `/home/cstory/src/cluckers/internal/launch/process_windows.go:42-49`:

```
-user=<username>
-token_file=<path>
-Language=INT
-dx11
-seekfreeloadingpcconsole
-nohomedir
```

Then, appended **only when `cfg.ContentBootstrap` is non-empty** (`process_linux.go:77-80`, `process_windows.go:71-74`):

```
-content_bootstrap_size=<len(cfg.ContentBootstrap)>
-content_bootstrap_shm=Local\realm_content_bootstrap_<pid>
```

Notes:
- The size is `len(...)` at runtime, **not** the hardcoded 136 that CLAUDE.md section 2 states.
- On Linux `-token_file` is passed through `wine.LinuxToWinePath(cfg.TokenPath)` (`process_linux.go:48`); on Windows it is the native path (`process_windows.go:44`).
- The SHM name is built with `os.Getpid()` at `process_linux.go:76` / `process_windows.go:70`.
- `-seekfreeloadingpcconsole` must stay one token. `prep_test.go:150-165` asserts the split form never appears.
- No `-hostx`, no `-eac_oidc_token`.

### 4.3 Argument-order discrepancy between launch and prep

`stepWriteLaunchConfig` at `/home/cstory/src/cluckers/internal/launch/prep.go:118-130` emits the **same set in a different order**:

```
-user=<username>
-token_file=<Z:\...>
-Language=INT
-dx11
-content_bootstrap_size=<n>        <-- before seekfree here
-seekfreeloadingpcconsole
-nohomedir
-content_bootstrap_shm=<name>      <-- last here
```

Runtime puts both bootstrap args after `-nohomedir`; prep interleaves them. Same set, different order. Worth normalizing during migration.

Prep also uses a **fixed** SHM name, `Local\realm_content_bootstrap_cluckers` (`prep.go:21`), not PID-based.

### 4.4 Token file

`writeTokenFile` at `/home/cstory/src/cluckers/internal/launch/pipeline.go:505`:

- `config.TmpDir()` ensured (`:506-509`)
- `os.CreateTemp(tmpDir, "realm_token_*.txt")` (`:511`)
- writes the raw token string, no trailing newline (`:516`)
- `os.Chmod(..., 0600)` (`:527`)
- returns `(path, cleanup, err)`; cleanup removes the file (`:532`)

Called from `stepLaunchGame` at `pipeline.go:479`, with `defer tokenCleanup()` at `:483` and `state.SetTokenTempFile(tokenPath)` at `:486` so the signal handler can clean up when `os.Exit` bypasses defers. `SetTokenTempFile`/`TokenTempFile` use an `atomic.Pointer[string]` (`pipeline.go:45`, `:49`, `:54`) because the signal goroutine races the pipeline goroutine.

### 4.5 Content bootstrap flow, gateway to shared memory

1. Gateway returns base64 in `portal_info_1` (`BootstrapResponse.PortalInfo1`, `types.go:76`).
2. `GetContentBootstrap` decodes it at `/home/cstory/src/cluckers/internal/auth/login.go:87`; returns raw BPS1 bytes.
3. `stepBootstrap` stores them on `state.Bootstrap` at `pipeline.go:358`. `nil` only warns (`:360`).
4. `WriteBootstrapFile` at `/home/cstory/src/cluckers/internal/launch/shm.go:49` writes `realm_bootstrap_*.bin` under `config.TmpDir()`, chmod 0600 (`:71`).
5. `ExtractSHMLauncher` at `shm.go:13` writes the embedded `assets.SHMLauncherExe` to `shm_launcher_*.exe`, chmod 0755 (`:35`).
6. Linux: `buildProtonCommand` at `/home/cstory/src/cluckers/internal/launch/proton_env.go:99` assembles

   ```
   <protonScript> run <shmPath> <Z:\bootstrapPath> <shmName> <Z:\gameExe> <gameArgs...>
   ```

   `shmPath` stays a Linux path (Proton converts it); `bootstrapPath` and `gameExe` go through `wine.LinuxToWinePath` (`:107`, `:109`). Without a bootstrap it degrades to `<protonScript> run <gameExe> <gameArgs...>` (`:114`). On NixOS the whole thing is wrapped in `steam-run` (`:120-124`, detection at `:134`).
7. Windows: `shm_launcher.exe` runs natively with positional args `<bootstrap_file> <shm_name> <game_exe> [game_args...]` (`process_windows.go:78-84`). No bootstrap means launching the game exe directly (`:86-88`).

Proton environment from `buildProtonEnvFrom` at `proton_env.go:56`: strips `LD_LIBRARY_PATH`, `WINEPREFIX`, `WINE`, `WINEDLLOVERRIDES`, `WINEFSYNC`, `WINEESYNC` (`:18-25`), then appends `STEAM_COMPAT_DATA_PATH`, `STEAM_COMPAT_CLIENT_INSTALL_PATH`, `SteamGameId`, `SteamAppId` (`:69-74`), plus `PROTON_LOG=1` when verbose (`:77`). `SteamGameId` defaults to `"0"` (`:60-62`).

### 4.6 `shm_launcher.c`

`/home/cstory/src/cluckers/tools/shm_launcher.c`:

- `wmain` at `:85`. If `argc < 4` it falls back to `read_config_file` (`:33`), which reads `launch-config.txt` from the executable's own directory (`:43-50`), one argument per line, max 64 args (`:25`), max 4096 chars per line (`:26`).
- Reads the bootstrap file via `CreateFileW`/`ReadFile` (`:112`, `:134`); rejects size 0 (`:120`).
- **`CreateFileMappingW(INVALID_HANDLE_VALUE, NULL, PAGE_READWRITE, 0, fileSize, shm_name)` at `:145`** — this is the anonymous named mapping the game opens with `OpenFileMapping`. A file path does not work.
- `MapViewOfFile` (`:153`), `memcpy` (`:161`).
- Builds a quoted command line into a 32768-wide buffer (`:167-176`), args from effective index 3 onward.
- `CreateProcessW` (`:184`), `WaitForSingleObject(..., INFINITE)` (`:194`), propagates the game's exit code as its own (`:211`).
- Cleans up mapping and handles at `:200-209`.

Failure strings it prints (`CreateFileMapping failed`, `MapViewOfFile failed`, etc.) are what `shmBridgeError` at `proton_env.go:159` pattern-matches on: `createfilemapping`, `openfilemapping`, `shm_launcher`, `shared memory` (`:165-170`).

### 4.7 What `prep` persists

`stepWriteLaunchConfig` at `/home/cstory/src/cluckers/internal/launch/prep.go:79` writes four artifacts:

| Artifact | Path | Mode | Line |
|---|---|---|---|
| Bootstrap bytes | `<cache>/bootstrap.bin` | 0600 | `:98-101` |
| Access token (bare string) | `<cache>/token.txt` | 0600 | `:104-107` |
| SHM helper | `<bin>/shm_launcher.exe` | 0755 | `:110-113` |
| Argument file | `<bin>/launch-config.txt` | 0644 | `:137-140` |

`ExtractSHMLauncherTo` at `prep.go:149` is idempotent by size comparison (`:151`).

`launch-config.txt` line order (`prep.go:118-130`): Wine bootstrap path, `prepSHMName`, Wine game exe path, then the eight game args listed in 4.3.

**Prep hard-fails on an empty bootstrap** at `prep.go:80-85` ("Content bootstrap is required for prep mode"). The normal launch path only warns. If the gateway ever stops serving the bootstrap, `cluckers prep` and Steam Deck Steam-managed launch break outright while plain `cluckers launch` degrades gracefully.

Prep pipeline step list, `buildPrepSteps` at `prep.go:58`: health, auth, bootstrap, platformSteps, `Checking game version` (`stepCheckVersion`, `pipeline.go:369`), `Downloading game update` (`stepDownloadGame`, `pipeline.go:405`), platformPostSteps, `Writing launch config`. The version-check and download steps exist only for prep; the launch pipeline never downloads.

CLI wrapper: `/home/cstory/src/cluckers/internal/cli/prep_linux.go:21` calls `launch.RunPrep(cmd.Context(), Cfg)`. Linux only.

### 4.8 `LaunchConfig`

`/home/cstory/src/cluckers/internal/launch/process.go:4-16`: `ProtonScript`, `ProtonDir`, `CompatDataPath`, `SteamInstallPath`, `SteamGameId`, `GameDir`, `Username`, `AccessToken`, `TokenPath`, `ContentBootstrap []byte`, `Verbose`. Populated at `pipeline.go:488-500`.

Note both `AccessToken` and `TokenPath` are carried; only `TokenPath` reaches the game.

---

## Section 5 — Token cache

`/home/cstory/src/cluckers/internal/auth/cache.go` in full:

```go
AccessTokenTTL = 45 * time.Minute                       // :18

type TokenCache struct {                                 // :24
    AccessToken    string    `json:"access_token"`        // :25
    Username       string    `json:"username"`            // :26
    AccessCachedAt time.Time `json:"access_cached_at"`    // :27
}
```

- Path: `filepath.Join(config.CacheDir(), "tokens.json")` (`:32`)
- `AccessTokenValid()` (`:36`): false on empty token; otherwise `time.Since(AccessCachedAt) < AccessTokenTTL`
- `LoadTokenCache()` (`:45`): returns `(nil, nil)` for a missing file (`:48-50`) **and for corrupt JSON** (`:55-58`)
- `SaveTokenCache()` (`:66`): 0600, `MarshalIndent`; the caller must stamp `AccessCachedAt`
- `ClearTokenCache()` (`:81`): idempotent

Eight save sites, all stamping the timestamp: `cli/login.go:74`, `cli/register.go:51`, `gui/screens/login.go:99`, `gui/screens/register.go:108`, `gui/screens/main.go:354`, `launch/pipeline.go:257`, `:310`, `:347`.

### 5.1 Where refresh must hook in

**No refresh token, no absolute expiry, no session id is cached.** The 45-minute TTL is a client-side guess; the server's `expiration_datetime` (`types.go:59`) is parsed into the struct but never read.

A refresh migration needs:
- `TokenCache.RefreshToken` plus a real `ExpiresAt time.Time` derived from `expiration_datetime`
- `SessionResponse.RefreshToken string \`json:"refresh_token"\``
- an `auth.RefreshSession(ctx, client, refreshToken)` calling POST `/launcher/v1/session/refresh`
- a single `EnsureAccessToken(ctx, client)` helper to replace the three duplicated validity checks at `pipeline.go:222`, `pipeline.go:237`, and `gui/screens/main.go:334`

The natural insertion point is the top of `stepAuthenticate` (`pipeline.go:203`).

Existing re-auth is one place only: `stepBootstrap` at `pipeline.go:323-365` — on `ErrTokenRejected` (`:326`) it clears the cache (`:333`), re-logs in (`:337`), re-caches (`:344-351`), retries once (`:353`), and fails hard on a second failure (`:355`).

---

## Section 6 — CLI and GUI consumers

### 6.1 CLI

**`/home/cstory/src/cluckers/internal/cli/login.go`** — `gateway.NewClient` (`:18`), `auth.LoadCredentials` (`:21`), prompts (`:33`, `:37`), `auth.Login` (`:44`). On failure with saved creds it warns (`:47`), re-prompts (`:48`, `:52`), retries once (`:56`). Then `auth.SaveCredentials` (`:66`), builds `TokenCache` (`:71-75`), `auth.SaveTokenCache` (`:77`), success message (`:81`). **Never reads `result.Linked`.** No Discord flow exists here despite `register.go:61` promising one.

**`/home/cstory/src/cluckers/internal/cli/register.go`** — prompts (`:21`, `:25`, `:29`), `auth.Register` (`:35`), `auth.SaveCredentials` (`:43`), `auth.SaveTokenCache` (`:48-55`), `auth.RequestLinkCode` (`:58`). Link-code failure warns and returns nil (`:60-61`). Code printed at **`:69`** via `fmt.Printf("  Your verification code: %s\n", code)`. Poll loop `:73-99`: ticker **5s** (`:74`), timeout **5 min** (`:76`), `auth.CheckDiscordStatus` (`:88`), errors swallowed with `continue` (`:90-91`), timeout exits 0 with an info message (`:83-86`), success at `:93-97`. No attempt counter.

**`/home/cstory/src/cluckers/internal/cli/resetpassword.go`** — `HealthCheck` (`:20`), prompt (`:24`), `auth.RequestPasswordReset` (`:30`).

**`/home/cstory/src/cluckers/internal/cli/logout.go`** — `auth.DeleteCredentials` (`:14`), `auth.ClearTokenCache` (`:17`). Purely local; never calls DELETE `/launcher/v1/session`.

**`/home/cstory/src/cluckers/internal/cli/status.go`** — `checkAuthStatus` at `:84-111` reads only local state: `auth.LoadCredentials` (`:87`), `auth.LoadTokenCache` (`:96`), `cache.AccessTokenValid()` (`:101`), remaining TTL (`:103`). Result struct `authStatusResult` at `:67-74` has no linked/PIN/pending field. Display at `:193-199` (compact) and `:271-286` (verbose). **Never contacts the gateway for auth**, so a server-revoked token still reports OK for up to 45 minutes.

**`/home/cstory/src/cluckers/internal/cli/root.go`** — no auth calls. `PersistentPreRunE` loads config into package var `Cfg` (`:20-27`). Flags at `:33-39`: `--verbose`, `--gateway`, `--game-version`. `SilenceUsage`/`SilenceErrors` at `:28-29` push rendering to `cmd/cluckers/main.go:28-41`. Subcommands self-register via `init()`.

**`/home/cstory/src/cluckers/internal/cli/steam_linux.go:201`** — `auth.LoadCredentials` only to warn when writing a `.desktop` shortcut.

### 6.2 GUI

**`/home/cstory/src/cluckers/internal/gui/app.go`** — entry at `:68-79`: `auth.LoadCredentials`; if username and password are both non-empty it goes **straight to `showMainView` (`:75`) with no server call and no token validation**; otherwise `showLoginScreen` (`:78`). Logout closure at `:137-143`.

Screen map (no state machine type; mutually recursive `show*` functions each ending in `w.SetContent`, no history stack):

| Screen | `show*` | Builder |
|---|---|---|
| Login | `app.go:95` | `screens/login.go:28` |
| Forgot password | `app.go:108` | `screens/forgot_password.go:25` |
| Register | `app.go:118` | `screens/register.go:29` |
| Discord linking | **none — bypasses app.go** | `screens/register.go:176`, sets content itself at `:241` |
| Main | `app.go:129` | `screens/main.go:56` |
| Launch progress | `app.go:158` | `screens/launch_progress.go:31` |
| Settings | `app.go:189` | `screens/settings.go:22` |

**`/home/cstory/src/cluckers/internal/gui/screens/login.go`** — `doLogin` at `:63-108`, `auth.Login` at `:80` with `context.Background()` (not window-scoped). Error path `:81-88` sets an inline label via `formatGUIError`. `auth.SaveCredentials` (`:91`), `auth.SaveTokenCache` (`:96-102`), `onSuccess` (`:105`). **Only `result.AccessToken` is read (`:98`).**

**`/home/cstory/src/cluckers/internal/gui/screens/register.go`** — `auth.Register` (`:89`), `auth.RequestLinkCode` (`:114`), `showDiscordLinking` (`:131`). The linking view at `:176-284` is the only intermediate screen in the app: code label `:192`, copy button `:195-197`, status label `:201`, cancellable ctx `:205`, "Continue Without Linking" `:208-211`, `w.SetContent` `:241`. Poll goroutine `:244-283`: ticker **5s** (`:246`), timeout **5 min** (`:248`), `auth.CheckDiscordStatus` (`:265`), errors `continue` (`:267-268`), timeout path sleeps 2s (`:259`), success sleeps 1500ms (`:275`).

**`/home/cstory/src/cluckers/internal/gui/screens/main.go`** — bot names handler `:321-390`: cache fast path (`:333-334`), inline `auth.Login` fallback (`:341`), re-cache (`:351-356`), `auth.UpsertBotName` per slot (`:375`), errors via `dialog.ShowError` (`:378`) with no retry. Supporter probe goroutine `:411-441`: `auth.ListBotNames` (`:425`) with a 15s timeout (`:423`); any error including 401 silently hides the section (`:427-429`).

**`/home/cstory/src/cluckers/internal/gui/screens/launch_progress.go`** — calls no auth functions. `launch.StepNames` (`:45`), `launch.RunWithReporterAndCreds` (`:65`), single `onError(err)` (`:68`). All auth happens inside the pipeline; the GUI can only surface an error, with no affordance for an interactive step.

**`/home/cstory/src/cluckers/internal/gui/screens/forgot_password.go:75`** — `auth.RequestPasswordReset`, then an information dialog (`:86-90`).

### 6.3 `linked_flag`, `custom_message`, `text_value` — exhaustive

All occurrences outside `cluckers-central/`:

`linked_flag` / `LinkedFlag` / `Linked`:
- `internal/gateway/types.go:60`, `:96`, `:101` (three struct fields)
- `internal/auth/login.go:36` (`LoginResult.Linked` declaration), `:65` (the only write)
- `internal/auth/register.go:82` (`return bool(resp.LinkedFlag)` from `DiscordStatusResponse` — the only consumed linked flag)
- tests: `internal/auth/login_test.go:200`, `:213-214`; `internal/auth/register_test.go:118`

`custom_message` / `CustomMessage`: **exactly one occurrence in the entire repo**, the field declaration at `internal/gateway/types.go:61`.

`text_value` / `TextValue`:
- `internal/gateway/types.go:63` (SessionResponse), `:114` (PasswordResetResponse)
- `internal/gateway/client.go:26` — a redaction key in `sensitiveKeys`, not a field read

**Explicit answer: NO.** `SessionResponse.CustomMessage` and `SessionResponse.TextValue` are read **nowhere** outside their declarations. There are zero selector expressions on a `SessionResponse` for either, in production or test code. `SessionResponse.AccountID`, `SessionID`, `ExpirationDatetime`, `CustomValue1`, and `PortalInfo1` are likewise never read from a `SessionResponse`.

`LoginResult.Linked` is dead: every caller (`cli/login.go:44`, `gui/screens/login.go:80`, `gui/screens/main.go:341`, `pipeline.go:247`, `:292`, `:337`) reads only `.AccessToken` and `.Username`. `LinkCodeResponse.AccessToken` (`types.go:95`) and `.LinkedFlag` (`:96`) are also never read — `RequestLinkCode` returns only `resp.Code` (`register.go:70`).

**Consequence:** if the server signals a PIN-required state via `linked_flag`, `custom_message`, or `text_value` on the session response, all three channels are discarded before reaching any caller.

### 6.4 Concrete existing bug

`RequestPasswordReset` (`internal/auth/resetpassword.go:14-21`) decodes into `PasswordResetResponse` at `:19-20` and **discards it**. The live probe (section 2.1) proves the server returns actionable text there:

> "If the account exists, DM this code to the Discord bot, then reply with your new password."

Our `cluckers reset-password` never shows the user that instruction. `RequestID` is also lost.

### 6.5 No PIN / MFA state machine exists

Case-insensitive greps across `internal/` (excluding `cluckers-central/`):

- **pin** → only game-version pinning: `cli/root.go:35` (`--game-version`), `config.PinnedVersion` (`cli/status.go:129`, `:136`, `:147`; `gui/screens/settings.go:45-47`), `game/version.go:152-154`, `game/manifest.go:78`, `internal/game/pin_test.go`
- **verify / verification** → game-file verification (`cli/update.go:63-74`, `gui/screens/main.go:85`, `:163`, `:170-174`), selfupdate checksums, Proton prefix checks, and the `Verifying game installation` step
- **pending** → only `gui/widgets/step_list.go:24`, `:35` (step status enum)
- **code** → only the Discord link code and `resp.StatusCode`
- **challenge, mfa, otp, 2fa, two-step** → **zero hits**

`auth.Login` returns `(*LoginResult, error)` with no third "needs more input" outcome. The Discord link flow is the only poll-until-server-confirms pattern in the codebase and is the template to copy.

### 6.6 Three required insertion points for a PIN step

1. **`/home/cstory/src/cluckers/internal/gui/screens/login.go:80-88`** — the `auth.Login` branch point. Needs a third outcome (typed sentinel checked with `errors.Is`, or a widened `LoginResult`) invoking a new `onPinRequired` callback. `MakeLoginScreen` (`login.go:28`) gains a fifth parameter; its only call site is `app.go:96`.
2. **`/home/cstory/src/cluckers/internal/gui/app.go:95-104`** — add `showPinScreen`. Model it on `screens/register.go:176-284` but route through app.go rather than repeating the `w.SetContent` shortcut at `register.go:241`.
3. **`/home/cstory/src/cluckers/internal/gui/app.go:68-79`** — the auto-skip path. Saved credentials bypass the server entirely, so a user who newly needs a PIN lands on Main with no valid session. Either log in here and branch, or handle it downstream at `screens/main.go:341` and inside `pipeline.go:247` — the latter has no UI affordance at all.

Also note `app.go:129` / `screens/main.go:56` thread `username, password` in plaintext through every screen. Anything PIN-derived has no existing carrier and must join that chain or move into `TokenCache`.

### 6.7 Credentials on disk

`/home/cstory/src/cluckers/internal/auth/credentials.go:16-20`:

```go
type Credentials struct {
    Username string `json:"username"`   // :18
    Password string `json:"password"`   // :19
}
```

Two fields, password plaintext inside the NaCl-encrypted blob. No token, no expiry, no flags, no schema version.

- `SaveCredentials(username, password string) error` (`:24`) — marshal (`:27`), `crypto.DeriveKey` (`:32`), `crypto.Encrypt` (`:37`), write 0600 (`:47`)
- `LoadCredentials() (*Credentials, error)` (`:57`) — **returns `(nil, nil)` on not-exist** (`:60-62`), a first-run sentinel every caller handles (`cli/login.go:28`, `cli/status.go:91`, `gui/app.go:73`, `pipeline.go:222`, `:231`, `cli/steam_linux.go:202`). Key-derivation failure → `UserError` (`:68-73`); decrypt failure → `UserError` (`:78-83`); unmarshal failure → plain wrapped error (`:87`, inconsistently not a `UserError`)
- `DeleteCredentials() error` (`:96`) — idempotent (`:98`)

---

## Section 7 — Updater

### 7.1 Constants and timeouts

| Item | Value | Location |
|---|---|---|
| `UpdaterURL` | `https://updater.realmhub.io/builds/version.json` | `internal/game/version.go:20` |
| `GameVersionDatRelPath` | `Realm-Royale/Binaries/GameVersion.dat` | `internal/game/version.go:24` |
| version.json timeout | 15s | `internal/game/version.go:39` |
| manifest timeout | 30s | `internal/game/manifest.go:35` |
| game-file client | `&http.Client{}`, **no timeout by design** | `internal/game/sync.go:151` |
| `manifestSchema` | `1` | `internal/game/manifest.go:14` |

Both metadata fetches use `http.DefaultClient` (`version.go:52`, `manifest.go:48`) and are explicitly unauthenticated (comments at `version.go:37`, `manifest.go:33`).

### 7.2 `VersionInfo` — complete, six fields

```go
// internal/game/version.go:27
LatestVersion        string `json:"latest_version"`          // :28
BaseURL              string `json:"base_url"`               // :29
ManifestURL          string `json:"manifest_url"`           // :30
GameVersionDatPath   string `json:"gameversion_dat_path"`   // :31
GameVersionDatBLAKE3 string `json:"gameversion_dat_blake3"` // :32
GameVersionDatSize   int64  `json:"gameversion_dat_size"`   // :33
```

**Ignored API fields, confirmed absent from the struct:** `delta_enabled`, `patch_threshold_bytes`, `repair_index_url`, `content_sig_scheme`. No `delta`, `patch`, `repair`, `sig`, or `minisign` identifier exists anywhere in `internal/game/`. The deferral is documented at `/home/cstory/src/cluckers/docs/superpowers/specs/2026-06-29-manifest-updater-design.md:36-38` and `:159`, not in the code. Nothing sets `DisallowUnknownFields`, so adding them later is backward-compatible. Legacy `zip_url` / `zip_blake3` / `zip_size` are gone.

### 7.3 `Manifest` and `ManifestFile`

```go
// internal/game/manifest.go:18
Schema  int            `json:"schema"`   // :19
Version string         `json:"version"`  // :20
Files   []ManifestFile `json:"files"`    // :21

// internal/game/manifest.go:26
Path string `json:"path"`   // :27  relative, forward slashes
Hash string `json:"hash"`   // :28  BLAKE3 hex
Size int64  `json:"size"`   // :29
```

Schema validation at `manifest.go:81`: `if m.Schema != manifestSchema && m.Schema != 0`. Schema 0 (absent) is accepted as the legacy schema-less format a version pin may fall back to (`:77-80`). Other values produce a `UserError` telling the user to run `cluckers self-update` (`:82-87`). Empty file list rejected at `:89-95`; non-200 at `:59-65`.

### 7.4 Signatures

```go
func FetchVersionInfo(ctx context.Context) (*VersionInfo, error)                          // version.go:38
func ResolveVersionInfo(ctx context.Context, pinned string) (*VersionInfo, error)         // version.go:144
func NeedsUpdate(gameDir string, remote *VersionInfo) (bool, error)                       // version.go:87
func ResolveNeedsUpdate(ctx context.Context, gameDir string, info *VersionInfo) (bool, *Manifest, error) // version.go:204
func LocalVersion(gameDir string) string                                                  // version.go:218
func GameDir() string                                                                     // version.go:230
func GameExePath(gameDir string) string                                                   // version.go:235
func PinVersionInfo(latest *VersionInfo, version string) *VersionInfo                     // version.go:129
func NeedsUpdateFromManifest(gameDir string, m *Manifest) (bool, error)                   // version.go:165
func FetchManifest(ctx context.Context, info *VersionInfo) (*Manifest, error)             // manifest.go:34
func SyncManifest(ctx context.Context, info *VersionInfo, m *Manifest, gameDir string, onProgress ProgressFunc) error // sync.go:52
func IsSyncIncomplete(gameDir string) bool                                                // sync.go:35
type ProgressFunc func(downloaded, total int64)                                           // sync.go:31
```

`GameDir()` = `filepath.Join(config.DataDir(), "game")`. `GameExePath` = `gameDir/Realm-Royale/Binaries/Win64/ShippingPC-RealmGameNoEditor.exe`.

`ResolveNeedsUpdate` (`:204-215`) short-circuits: with a non-empty `GameVersionDatBLAKE3` it calls `NeedsUpdate` and returns a **nil manifest**; otherwise (pinned, hash unknown) it fetches and returns the manifest for reuse. `cli/update.go:50-57` handles the nil case.

`PinVersionInfo` derives pinned URLs by `strings.ReplaceAll` on the version token in `BaseURL`/`ManifestURL` and **clears** `GameVersionDatBLAKE3` and `GameVersionDatSize` (`:135-138`).

`LocalVersion` hardcodes its own dat path instead of using `GameVersionDatRelPath` (`:219`) and returns `"not installed"` or `"present (N bytes)"`.

`ProgressFunc == nil` means draw a terminal progress bar; non-nil is throttled to 250ms (`sync.go:316-350`). The CLI passes nil (`cli/update.go:59`).

### 7.5 Sync internals

| Item | Value | Location |
|---|---|---|
| marker filename | `.cluckers-syncing` | `sync.go:24` |
| worker pool | `syncWorkers = 8` | `sync.go:27` |
| traversal guard | `safeJoin` | `sync.go:108-115` |

Marker lifecycle: written `"syncing"` 0644 before downloads (`:87-90`); left in place on error so the next run re-syncs (`:92-94`); removed only after downloads and the stale pass succeed (`:100-102`). `IsSyncIncomplete` is an `os.Stat` (`:35-38`), consulted first by `NeedsUpdate` (`version.go:89`) and `NeedsUpdateFromManifest` (`version.go:166`). `removeStale` skips the marker (`:292-294`).

```go
func safeJoin(base, rel string) (string, error) {
    dest := filepath.Join(base, filepath.FromSlash(rel))
    r, err := filepath.Rel(base, dest)
    if err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
        return "", fmt.Errorf("path %q escapes base directory", rel)
    }
    return dest, nil
}
```

Applied to **every** manifest entry up front in the diff loop (`:67-74`), before any download; a violation aborts the whole sync as a "corrupted or tampered manifest". It is purely lexical and does not resolve symlinks, so a pre-existing symlink inside `gameDir` could still redirect a write.

Other details: download URL is `strings.TrimRight(baseURL, "/") + "/" + j.file.Path` (`:208`); each file goes to an `os.CreateTemp(dir, ".dl-*")` sibling, is BLAKE3-verified, then `os.Rename`d (`:244-278`); `checkDiskSpace` runs before the marker is written (`:82`); workers drain the channel on cancellation so the feeder never blocks (`:177-179`); `removeStale` deletes any regular file not in the manifest (`:284-303`), sweeping aborted `.dl-*` leftovers.

### 7.6 `Config`

`/home/cstory/src/cluckers/internal/config/config.go:39-46`, defaults `:54-59`:

| Field | viper key | Type | Default | Line |
|---|---|---|---|---|
| `Gateway` | `gateway` | string | `https://api.project-crown.com` (ldflags-overridable) | `:54`, var `:13` |
| `HostX` | `hostx` | string | `157.90.131.105` (ldflags-overridable) | `:55`, var `:14` |
| `Verbose` | `verbose` | bool | `false` | `:56` |
| `WinePath` | `wine_path` | string | `""` | `:57` |
| `GameDir` | `game_dir` | string | `""` | `:58` |
| `PinnedVersion` | `pinned_version` | string | `""` | `:59` |

Config file `settings.toml` in `config.ConfigDir()` (`:62-64`); missing file is not an error (`:67-76`). Precedence CLI flag > file > default, handled by viper (`:51`). `SetBuildDefaults(gateway, hostx)` at `:29-36`. Path helpers: `CacheDir` (`paths.go:14`), `BinDir` (`:19`), `LogDir` (`:24`), `TmpDir` (`:32`). `DataDir()` honors `$CLUCKERS_HOME`, else `~/.cluckers` (`paths_linux.go:10-11`) or `%LOCALAPPDATA%\cluckers`.

---

## Section 8 — Tests, build status, logs, git

### 8.1 Test inventory (23 files)

```
internal/gateway/client_test.go
internal/auth/login_test.go        register_test.go    cache_test.go
internal/auth/credentials_test.go  resetpassword_test.go
internal/crypto/secretbox_test.go
internal/game/manifest_test.go     pin_test.go   sync_test.go   extract_test.go
internal/launch/prep_test.go       shortcuts_test.go  gamelog_test.go
internal/launch/shm_test.go        proton_env_test.go deckconfig_test.go
internal/selfupdate/selfupdate_test.go
internal/config/appimage_test.go
internal/wine/detect_test.go       steamdir_test.go   proton_test.go   compatdata_test.go
internal/ui/logging_test.go
```

### 8.2 Conventions

Universal: `httptest.NewServer(http.HandlerFunc(...))` with `defer srv.Close()`, injecting `srv.URL` as the gateway base. No interface mocks, no httpmock, no global client swapping. Filesystem tests use `t.TempDir()` + `t.Setenv("CLUCKERS_HOME", tmp)`. Errors asserted with `errors.As(err, &ue)` against `*ui.UserError`, then checking `ue.Message`.

| File | Pattern | Helper |
|---|---|---|
| `internal/gateway/client_test.go` | inline servers, no shared helper | first at `:19` |
| `internal/auth/login_test.go` | shared + inline | `newBootstrapServer` `:30` |
| `internal/auth/register_test.go` | shared + inline | `newJSONServer` `:17` |
| `internal/auth/cache_test.go` | no HTTP; TempDir + Setenv inline | `:61-62`, `:106-107` |
| `internal/auth/resetpassword_test.go` | inline only (does not reuse `newJSONServer`) | `:17`, `:33` |
| `internal/game/manifest_test.go` | inline, injected via `&VersionInfo{ManifestURL: srv.URL}` | `:19`, `:47`, `:59` |
| `internal/launch/prep_test.go` | no HTTP; TempDir + Setenv in a builder; `//go:build linux` | `newTestPrepState` `:19` |

Exact signatures:

```go
func newJSONServer(t *testing.T, status int, resp map[string]interface{}) *httptest.Server // register_test.go:17
func newBootstrapServer(t *testing.T, encoded string) *httptest.Server                     // login_test.go:30
func newTestPrepState(t *testing.T) *LaunchState                                           // prep_test.go:19
type noopReporter struct{}                                                                  // prep_test.go:43
func newFakeUpdater(t *testing.T, files map[string]string) *fakeUpdater                    // sync_test.go:33
func blake3Hex(content string) string                                                       // sync_test.go:18
```

`newJSONServer` is package-scoped and usable from any `package auth` test file — **this is the helper new gateway tests should reuse.** `newFakeUpdater` (type at `sync_test.go:26-32`) wraps a server, auto-builds a matching schema-1 manifest and `VersionInfo`, and records a mutex-guarded `requested map[string]int` for skip/redownload assertions; it uses `t.Cleanup` rather than `defer`, the only file in the repo that does.

No `TestMain`, no `testdata/`, no shared cross-package helper package. Helpers are duplicated per package by convention.

### 8.3 Build status — NOT VERIFIED

**There is no Go toolchain on this machine.** `go` is absent from `PATH` and from `/usr/local/go`, `~/go/bin`, `~/sdk`, `/usr/lib/go*/bin`, `/snap/bin`. Every `PATH` directory was checked individually for an executable named `go`; none found. `go build`, `go vet ./...`, and `GOOS=windows go vet ./...` all fail with `go: command not found`.

**Build status is therefore unverified.** Someone with a toolchain must confirm.

`assets/shm_launcher.exe` **is present** (157,821 bytes, mode 0755) but is **gitignored** at `.gitignore:35` and untracked. `git ls-files assets/` lists only `controller_neptune_config.vdf` and `embed.go`. A fresh clone needs mingw-w64 and the documented `x86_64-w64-mingw32-gcc` command before `go build` will succeed.

`go.mod:3` declares `go 1.26.0`; CLAUDE.md section 1 says Go 1.25. Doc drift.

### 8.4 Logs — NONE EXIST

`~/.cluckers` **does not exist at all** on this machine. No `logs/cluckers.log`, no `cache/tokens.json`, no `config/credentials.enc`. A filesystem search for `cluckers.log` under `/home/cstory` returned nothing.

**I have no record of the failure the user hit.** This box has never run the launcher; it is a dev machine only. The real failure evidence lives on whatever machine they actually play on, and someone should retrieve `~/.cluckers/logs/cluckers.log` (or `%LOCALAPPDATA%\cluckers\logs\cluckers.log`) from there.

### 8.5 `git log --oneline -15`

```
f51ec81 fix(ci): default release gateway to prod api.project-crown.com (#15)
5fe5d29 ci(deps): bump actions/cache from 5 to 6 (#14)
c616d2f ci(deps): bump actions/checkout from 6 to 7 (#13)
77e7430 Merge feat/manifest-updater: manifest-based game updater, version pinning, pre-release hardening
d8357c7 docs: fix stale gateway, token-cache, and wine-prefix references
4cbedc7 test: add gateway client coverage
02fa4ab chore: gofmt logging.go and proton.go
3d79d72 fix: cap checksums.txt read and correct Windows free-space comparison
d0f5260 feat(gui): expose pinned game version in settings
177d3fa fix: synchronize token temp file cleanup with the signal handler
7b95589 docs: drop TODO.plan.txt — launch failure resolved, plan obsolete
5e0253d fix: pass -seekfreeloadingpcconsole as one token so UE3 finds cooked content
6efa37e feat: pin game version / rollback to survive a broken latest build
7e437e5 docs: add TODO.plan.txt — resume dev on remote box; 6744 upstream defect
1d1bf57 feat: surface the game's own error on launch failure
```

Nothing auth-related has landed recently. Working tree is clean apart from the untracked `cluckers-central/`.

### 8.6 CLAUDE.md inaccuracies found while mapping

- Says `-content_bootstrap_size=136` is fixed; code uses `len(bootstrap)` (`process_linux.go:78`).
- Says Windows `platformPostSteps` returns an empty slice; it returns a display-config step (`pipeline_windows.go:28-32`).
- Section 4 code map omits `proton_env.go`, `prep.go`, `gamelog.go`, `shortcuts.go`, `reporter*.go`, `forgot_password.go`.
- Says Go 1.25; `go.mod` says 1.26.0.
- Says `CacheDir`/`LogDir` only; `BinDir` and `TmpDir` also exist.

---

## Priorities for the migration plan

1. **The breakage is not a removed route.** Every endpoint we call still answers (401/405, never 404). Whatever broke is inside a response body or an auth state invisible without credentials. Getting the real `cluckers.log` from the user's play machine is the highest-value next step.
2. **Refresh tokens are the real API change.** `POST /launcher/v1/session/refresh` is live and demands `refresh_token`; `/launcher/v1/session` now allows DELETE. We have no field to receive a refresh token, nowhere to store one, and no revocation on logout. Our 45-minute TTL guess plus full re-login is the likely failure mode.
3. **We discard the fields most likely to carry new state.** `custom_message`, `text_value`, and `linked_flag` all reach `auth.Login` and die there. Widening `LoginResult` is a cheap prerequisite to diagnosing anything else, and immediately fixes the reset-password instruction bug.
4. **`launch-authsrc` cannot be implemented yet.** It 404s on every host, method, and spelling. Do not build against it until the server serves it.
5. **401 classification is too narrow.** Only `GetContentBootstrap` can produce `ErrTokenRejected`, and it does so by string-matching `"HTTP 401"` inside an error message. Give `UserError` a typed status field and classify every authenticated call.
6. **A PIN step needs three GUI insertion points** (`screens/login.go:80`, `app.go:95`, `app.go:68`) and has no CLI equivalent at all in `cluckers login`. The Discord link flow (5s ticker, 5min timeout) is the template.
