# Cluckers Central 1.6.3 — Gateway Protocol, Recovered by Static Analysis

Targets:
- `/home/cstory/src/cluckers/cluckers-central/cluckers-central.exe` (19 MB, PE32+ GUI, Tauri 2.9.0 + reqwest 0.12.15, `LAUNCHER_VERSION=1.6.3`)
- `/home/cstory/src/cluckers/cluckers-central/launcher_smoketest.exe` (4.4 MB, PE32+ console, older API revision)
- `/home/cstory/src/cluckers/cluckers-central/launcher_error.log` (logs from 0.9.94)

Read-only analysis. No file under `/home/cstory/src/cluckers` was modified.

Image base `0x140000000`. Section map used for every address translation below:

| section | VA | vsize | raw ptr | raw size |
|---|---|---|---|---|
| `.text`   | `0x1000`    | `0xc57cc0` | `0x400`     | `0xc57e00` |
| `.rdata`  | `0xc59000`  | `0x5bbe3c` | `0xc58200`  | `0x5bc000` |
| `.data`   | `0x1215000` | `0x5390`   | `0x1214200` | `0x2e00`   |
| `.pdata`  | `0x121b000` | `0xaaf7c`  | `0x1217000` | `0xab000`  |

---

# (a) Endpoint table

Methods were recovered by resolving the `http::Method` statics referenced at each call site. The statics are one-byte discriminants of `http::method::Inner`, zero-padded to 24 bytes, which makes them trivially identifiable in `.rdata`:

| static VA | byte | method |
|---|---|---|
| `0x140d874a8` | 1 | GET |
| `0x140d874f0` | 2 | POST |
| `0x140d874c0` | 3 | PUT |
| `0x140d87508` | 4 | DELETE |
| `0x140d874d8` | 5 | HEAD (updater range probes only, not gateway) |

## Current API, `cluckers-central.exe` 1.6.3

| Method | Path | Auth | Request body | Response fields | Conf. |
|---|---|---|---|---|---|
| GET | `/healthz` | none | none | `status`, `ready_flag`, `string_value` | high |
| POST | `/launcher/v1/session-or-link` | none | `{user_name, password}` | session object | high |
| POST | `/launcher/v1/session/refresh` | none | `{refresh_token}` | session object, tokens rotate | high |
| POST | `/launcher/v1/account` | none | `{user_name, password, email}` | session object | high |
| POST | `/launcher/v1/password-reset` | none | `{user_name}` | `request_id`, `access_token`, `text_value` | high |
| POST | `/launcher/v1/launch-auth` | Bearer + `x-realm-client-build` | `{}` empty object | `account_id`, `session_id`, `launch_expiration_datetime`, `launch_expires_at_unix`, `custom_value_1`, `custom_value_2`, `expiration_datetime`, plus `launch_token` and `portal_info_1` | high |
| GET | `/launcher/v1/supporter/bot-names` | Bearer | none | untyped JSON, no named fields deserialized | high |
| PUT | `/launcher/v1/supporter/bot-names/{slot}` | Bearer | `{bot_name}` | untyped JSON | high |
| DELETE | `/launcher/v1/supporter/bot-names/{slot}` | Bearer | none | untyped JSON | high |

Path-length evidence, taken from the length immediate that follows each pointer store, so none of these are guesses from `strings` boundaries:

| path | string VA | `lea` site | length |
|---|---|---|---|
| `/launcher/v1/session-or-link` | `0x140c64c1a` | `0x140172acb` | `0x1c` = 28 |
| `/launcher/v1/session/refresh` | `0x140c7232a` | `0x1401c8b1f` | `0x1c` = 28 |
| `/launcher/v1/launch-auth` | `0x140c721e9` | `0x1401c978d` | `0x18` = 24 |
| `/launcher/v1/account` | `0x140c64e60` | `0x14014f5ed` | `0x14` = 20 |
| `/launcher/v1/password-reset` | `0x140c64d08` | `0x140133598` | `0x1b` = 27 |
| `/healthz` | `0x140c64e08` | `0x14010e4ea` | `0x8` = 8 |
| `/launcher/v1/supporter/bot-names` | `0x140c64d60` | `0x140163cd6` | 32 |
| `/launcher/v1/supporter/bot-names/` | `0x140c64d99` | via `format!` | 33 |

The launch path is genuinely `/launcher/v1/launch-auth`. In a raw `strings` dump it reads as `title/launcher/v1/launch-authsrc\gateway_client.rs` only because `title` and `src\gateway_client.rs` are separately pooled literals with their own independent code references. Hexdump at file `0xc713e0`:

```
00c713e0: 3a20 c000 7469 746c 652f 6c61 756e 6368  : ..title/launch
00c713f0: 6572 2f76 312f 6c61 756e 6368 2d61 7574  er/v1/launch-aut
00c71400: 6873 7263 5c67 6174 6577 6179 5f63 6c69  hsrc\gateway_cli
```

## The launch-auth body is an empty object, not absent

At `0x1401c975a` the `serde_json::Value` tag byte is `0x5` (`Value::Object`) and both `BTreeMap` slots are explicitly zeroed, so the client sends `{}`:

```
1401c975a: mov BYTE PTR  [r13+0x838],0x5   ; Value::Object
1401c9762: mov QWORD PTR [r13+0x840],0x0   ; map root = None
1401c976d: mov QWORD PTR [r13+0x850],0x0   ; map length = 0
1401c978d: lea rax,[rip+0xaa8a55]          # "/launcher/v1/launch-auth"
1401c979b: mov QWORD PTR [r13+0x878],0x18
```

`/healthz` writes tag `0x0` (`Value::Null`) instead and takes the GET path, which is how the shared helper distinguishes the two.

## Older API, present only in `launcher_smoketest.exe`

| Method | Path | Auth | Request | Response |
|---|---|---|---|---|
| POST | `/launcher/v1/session` | none | `{user_name, password}` | session object without refresh fields |
| POST | `/launcher/v1/account` | none | `{user_name, password, email}` | session object |
| POST | `/launcher/v1/discord/link/code` | Bearer | `{user_name}` | `code` |
| GET | `/launcher/v1/discord/link` | Bearer | none | `linked_flag`, `portal_userid`, `portal_username`, `string_value`; values `LINKED` / `NOT_LINKED` |

Smoketest environment variables and defaults, from `.rdata` at file offset `2885120`:

```
LAUNCHER_GATEWAY_BASE_URL
LAUNCHER_SMOKE_USER  LAUNCHER_SMOKE_PASS  LAUNCHER_SMOKE_EMAIL
access_token  zzzlauncher_smoke  https://api.project-crown.com  zzzlauncher_smoke@example.com
```

The smoketest session field table at `2896648` **lacks** the refresh fields, and is exactly the shape our Go client currently models:

```
linked_flag access_token expiration_datetime account_id user_name custom_message
custom_value_1 custom_value_2 custom_value_3 custom_value_4 text_value portal_info_1
```

## Session response shape, 1.6.3

Field table at file offset `0xd9c922`, split by resolving every code reference that points into the pool:

| field | string VA |
|---|---|
| `linked_flag` | pool start |
| `access_token` | |
| `expiration_datetime` | |
| `refresh_token` | `0x140d9d722` |
| `refresh_expiration_datetime` | `0x140d9d72f` |
| `access_expires_at_unix` | `0x140d9d74a` |
| `refresh_expires_at_unix` | `0x140d9d760` |
| `account_id` | `0x140d9d777` |
| `user_name` | `0x140d9d781` |
| `custom_message` | `0x140d9d78a` |
| `custom_value_1` .. `custom_value_4` | `0x140d9d798` .. `0x140d9d7c2` |
| `text_value` | `0x140d9d7d0` |
| `portal_info_1` | `0x140d9d7da` |

---

# (b) Headers and User-Agent

| header | value | sent on |
|---|---|---|
| `Authorization: Bearer <access_token>` | literal `Bearer ` at `0x140d87520` | bot-names list, bot-name upsert, bot-name delete, launch-auth |
| `x-realm-client-build` | dotted numeric game build version, see below | launch-auth only |
| `Content-Type: application/json` | reqwest `.json()` | all POST and PUT bodies |
| User-Agent | **not sent at all** | n/a |

No Authorization header on login, register, password reset, or session refresh. No HMAC, signature, nonce, or timestamp scheme exists anywhere in either binary.

## User-Agent: there is none

No `CluckersCentral/` literal exists in either binary. An exhaustive scan of the 19 MB image for any `name/x.y.z` shaped literal returns exactly one hit, `tauri-plugin-updater/2.7.1`, which belongs to the launcher self-updater and never touches the gateway client. reqwest 0.12.15 sends no default agent unless `.user_agent()` is called on the builder, and no such call is configured.

**Our Go client's `CluckersCentral/1.2.54` is fabricated and matches nothing on the wire.**

## `x-realm-client-build`: the exact value

**It is the installed game build version in dotted numeric form, the `GameVersion.dat` style value such as `0.39.6969.0`. It is not the launcher version `1.6.3`.**

Three independent lines of evidence:

**1. The launcher version is never referenced from the launch path.** The literal `1.6.3` lives at VA `0x140d8cad8` (file `0xd8bcd8`, stored as `1.6.3// LAUNCHER_VERSION=`). It has exactly one code reference in the whole binary, at `0x140677ac4` inside function `0x140677820`, which is not the launch function and not the header helper.

**2. The validator accepts an arbitrary number of dot-separated integer components, which strict semver would not.** The value is passed through `0x1406c3820` before the header is set. That function memchr's for `.` (`cmp BYTE PTR [rax+rdx*1],0x2e`, and `mov cl,0x2e; call 0x140ba3fc0`), loops over every segment rather than stopping at three, range-checks each byte as an ASCII digit with the `add al,0xc6; cmp al,0xf6` idiom, strips a leading `+` or `-`, and accumulates with `mul r14` where `r14 = 10` under an overflow check. That is per-segment `u64::from_str` over an unbounded number of components. The `semver-1.0.26` crate is present in the image but its error strings (`empty string, expected a semver version`, `expected comma after`, `invalid leading zero in`) belong to the Tauri updater plugin, not to this path. A four-component value like `0.39.6969.0` parses fine here and would be rejected by semver.

**3. The failure message names the game, not the launcher.** The string pooled immediately after the header name at `0x140d9d7e7` is:

```
Installed game build is missing or malformed. Verify or repair the game before launching.
```

Related managed state is `verifiedBuildState`, bound after local verification, with sibling strings `[Launch] Refusing unverified game build`, `[Files] Failed to bind verified game build`, and `Verified build lacks GameVersion.dat hash metadata`.

The header is the **only** custom header in the binary. The setter `0x1406c80a0` has exactly one caller:

```
1406c3707: mov QWORD PTR [rsp+0x28],rbx   ; value len
1406c370c: mov QWORD PTR [rsp+0x20],rdi   ; value ptr
1406c3716: lea r8,[rip+0x6da0ca]          # 0x140d9d7e7  "x-realm-client-build"
1406c3721: mov r9d,0x14                   ; name length 20
1406c372a: call 0x1406c80a0
```

and that wrapper `0x1406c3690` is itself called from exactly one place, `0x1401c9a6d`, sitting between the launch-auth method load at `0x1401c991a` and its response handling. The value arrives as a coroutine parameter read at `0x1401c96a2` and `0x1401c96bd` from `[r13+0x798]` and `[r13+0x7a0]`, a `(ptr, len)` pair with no writer inside the function, so it is supplied by the `launch_game` caller. The raw string is forwarded unchanged; the parse is a validation gate, not a transform.

One caveat worth flagging. The frontend passes `version: t.latestVersion || "local"` to `launch_game`, and the literal `"local"` would fail this integer validator. That is further evidence the header is fed from the verified build state rather than straight from that argument, but I could not prove the exact struct field without deeper tracing.

---

# (c) Discord link flow, PIN flow, and websocket

## Where the frontend JavaScript came from and how to get it again

The Tauri v2 asset table is an array of 32-byte `{key_ptr, key_len, data_ptr, data_len}` records in `.rdata` at file offset `0xd5cc28`. Four assets, each a **raw brotli stream** with no container or framing.

| published path | file offset | compressed | decompressed | extracted to |
|---|---|---|---|---|
| `/assets/index-DG_AOtdV.js`  | `0xd59e3e` | 11748  | 49928 | `scratchpad/fe_index.js` |
| `/assets/index-TMGXzNC_.css` | `0xc7363e` | 4603   | 20498 | `scratchpad/fe_index.css` |
| `/index.html`                | `0xc74844` | 317    | 780   | `scratchpad/fe_index.html` |
| `/finallogo.png`             | `0xc7498f` | 939158 | n/a   | not extracted, irrelevant to the protocol |

This host has no Python `brotli` module, no brotli CLI, and no binwalk, but Node 22 is installed and `zlib.brotliDecompressSync` decodes raw brotli directly. Reproduce with:

```bash
node -e '
const fs=require("fs"),zlib=require("zlib");
const d=fs.readFileSync("/home/cstory/src/cluckers/cluckers-central/cluckers-central.exe");
for(const [n,o,l] of [["index.js",0xd59e3e,11748],["index.css",0xc7363e,4603],["index.html",0xc74844,317]])
  fs.writeFileSync("fe_"+n, zlib.brotliDecompressSync(d.subarray(o,o+l)));'
```

All three text assets are already extracted to `/tmp/claude-1000/-home-cstory-src-cluckers/7d0de5db-04af-4783-b2f5-29516da50788/scratchpad/`. The bundle is hand-rolled vanilla TypeScript built by Vite, no framework, one module, one state object, string-template rendering. No source maps are embedded, so behaviour is fully recoverable but original identifier names are not.

## The PIN request key

**The IPC argument is named `pin`. There is no gateway JSON key, because this build never puts it on the wire.**

The frontend does pass it. Verbatim from `fe_index.js`:

```js
const o={userName:e,password:a};
n&&(o.pin=n);
const i=x(await l("launcher_login_or_link",o));
```

The Rust side declares and deserializes it. The Tauri argument-name table at `.rdata` `0xc637c1` reads `authStatepasswordpin...`, and the only code reference to that `pin` literal in the entire image is the IPC deserializer:

```
140171e18: lea rax,[rip+0xaf27aa]        # 0x140c645c9  "pin"
140171e26: mov QWORD PTR [rbp+0xb0],0x3  ; length 3
```

The request body built in the same function contains exactly two keys, and the allocation sites prove it. A three-byte `"pin"` key would need an `alloc(3, 1)`; the function has none:

```
140172862: mov ecx,0x9                   ; alloc 9
14017287a: movabs rcx,0x6d616e5f72657375 ; "user_nam"
140172887: mov BYTE PTR [rax+0x8],0x65   ; + 'e'  -> "user_name"
...
14017296b: mov ecx,0x8                   ; alloc 8
140172983: movabs rcx,0x64726f7773736170 ; "password"
```

Every `mov ecx,<small>` in the whole function `0x140171b20`-`0x14017354a` is `0x9`, `0x8`, and two `0x1` in unwind paths. The function makes no `format!` call other than its two log lines, so the PIN is not concatenated into `user_name` or `password` either. The URL is assembled from two pieces only, base plus path, with the path length hardcoded at `0x1c`, so it is not a query parameter. The function never calls the header setter at `0x1406c80a0`, so it is not a header.

**Wire body is exactly `{"user_name": ..., "password": ...}`.** Confidence is high on the mechanics and medium on the interpretation. Either the PIN gate is vestigial in 1.6.3 or this is a live bug in the official client.

## What the PIN actually is, and how a user gets one

**It is a developer access PIN for dev-only server mode, typed by the user. It is neither requested from the Discord bot nor pushed to the user.** No length, format, default, or hint exists anywhere in the binary, so it is validated entirely server-side.

The screen is gated on the login reply, not on any bot interaction:

```js
const f=b(i,"LINKED_FLAG")===1,g=(d(i,"STRING_VALUE")||"").trim();
if(f&&(g==="PIN_REQUIRED"||g==="PIN_INVALID")){
  t.loginToken="",
  t.pinStatus={text:g==="PIN_INVALID"?"Invalid PIN. Try again.":"Enter the developer PIN to continue.",
               tone:g==="PIN_INVALID"?"danger":"info"},
  t.view="pin",h(),await S(),await y();return}
```

So the trigger is `LINKED_FLAG === 1` **and** `STRING_VALUE` of either `PIN_REQUIRED` or `PIN_INVALID`. Both sentinels appear only in the JavaScript and never in the Rust image, which confirms they are server-supplied `STRING_VALUE` strings rather than client-generated states.

## PIN_INVALID handling specifically

`PIN_INVALID` is handled identically to `PIN_REQUIRED` in control flow, differing only in the message and tone:

- Message becomes `Invalid PIN. Try again.` instead of `Enter the developer PIN to continue.`
- Tone becomes `danger` instead of `info`.
- `t.loginToken` is cleared in both cases, so any partial session is discarded.
- The view is set to `pin` in both cases, so an invalid PIN re-renders the same screen rather than bouncing to the login form.

There is **no retry counter, no lockout, and no backoff** on the client. There is also no PIN expiry string anywhere in either binary. The gate appears twice in the bundle, once on the initial login result and once on the Discord link-polling result, with identical logic.

Submission handler and validation, again verbatim:

```js
async function Rt(){const e=T("dev_pin");
  if(!t.username||!t.password){t.pinStatus={text:"Go back and sign in first.",tone:"warning"},...;return}
  if(!e){t.pinStatus={text:"Enter the developer PIN.",tone:"warning"},...;return}
  t.pinStatus={text:"Checking PIN…",tone:"info"},...
```

The input element id is `dev_pin`, and Enter is bound to the same handler. Retry re-invokes **the same** `launcher_login_or_link` command with `{userName, password, pin}`. There is no separate PIN submit command and no separate endpoint.

## The Discord link flow is separate

Keyed on `LINKED_FLAG !== 1`, in which case the same reply's `ACCESS_TOKEN` field carries a **link code** rather than a token:

```js
const A=d(i,"ACCESS_TOKEN");
if(!f){t.linkCode=A, t.supporterTierName="none", ...}
```

The user sends that code to the bot by direct message. The bot user ID is hardcoded at `.rdata` `0xd8b778` as `1404860983419211839`, opened via `discord://-/users/`. The client then re-polls `launcher_login_or_link` on a three second interval, gated on `!document.hidden && view==="link"`. Requesting a new code is just another call to the same command. UI copy includes `Link codes expire automatically.`, `DM the code to the bot before it expires. We'll auto-check every 3 seconds.`, and `Not linked yet. DM the active code and wait for auto-check.` Password reset uses the identical mechanic and returns its code in `access_token`.

Note the ordering consequence: the PIN gate and the link gate are mutually exclusive, because one requires `LINKED_FLAG === 1` and the other requires `LINKED_FLAG !== 1`.

## Websocket and `ws_base_url`

The URL is built in Rust from these literals at `0xd8b630`:

```
00d8b630: 7773 733a 2f2f 7773 3a2f 2fc0 c001 3ac0   wss://ws://...:.
00d8b640: 032f 7773 00c0 c003 2f77 7300             ./ws..../ws.
```

giving `wss://` or `ws://`, an optional `:port`, and the path `/ws`, so the endpoint is `<scheme>://<host>[:<port>]/ws`. With the default gateway that is `wss://api.project-crown.com/ws`.

The frontend fetches it through the IPC layer rather than constructing it:

```js
async function Yt(){if(!G)try{const e=await l("get_api_config");
  if(!(e!=null&&e.ws_base_url))return; G=String(e.ws_base_url)}catch{return}
  be();try{m=new WebSocket(G)}catch{Ut();return}
  m.onopen=()=>{W=800,t.gatewayStatus={text:"Server: Connected",...}}
```

Exactly one message type is handled:

```js
if((typeof n?.type=="string"?n.type:"")==="gateway_status"){
  const o=(typeof n.ready=="number"?n.ready:0)===1,
        i=typeof n.online=="boolean"?n.online:!0;
```

so the payload is `{"type":"gateway_status","ready":0|1,"online":bool}`. It maps to `Server: Online` when ready is 1, `Server: Degraded` when ready is not 1, and `Server: Offline` when online is false. The offline case force-logs-out any non-auth view with `Server offline. Please try again later.`

Reconnect backoff starts at 800 ms, multiplies by 1.8 per attempt, and caps at 60000 ms. A separate interval re-runs `launcher_health` every 60 s whenever the socket is not in the OPEN state. **No token is sent on the socket and the client never writes to it**, so it is a pure server-push status feed.

---

# (d) Tauri command list

Twenty-two commands, cross-checked between the Rust ident tables at `.rdata` `0xc637a0` and `0xc707ca` and the recovered bundle. Argument objects are exactly as the frontend passes them.

| command | arguments |
|---|---|
| `launcher_health` | none |
| `launcher_login_or_link` | `{userName, password}`, plus `pin` only when the PIN view is active |
| `launcher_register` | `{userName, password, email}` |
| `launcher_request_password_reset` | `{userName}` |
| `launcher_supporter_bot_names_list` | `{accessToken}` |
| `launcher_supporter_bot_name_upsert` | `{accessToken, slotIndex, botName}` |
| `launcher_supporter_bot_name_delete` | `{accessToken, slotIndex}` |
| `check_for_game_updates` | none, returns `{latestVersion, baseUrl, status, currentVersion}` |
| `start_download` | `{version:"latest", installPath, baseUrl}` |
| `launch_game` | `{version, installPath, username, accessToken, extraArgs}` |
| `get_app_version` | none |
| `get_api_config` | none, returns `{http_base_url, ws_base_url}` |
| `get_install_path` / `save_install_path` | none / `{path}` |
| `get_game_language` / `save_game_language` | none / `{language}` |
| `get_dev_command_line_settings` / `save_dev_command_line_args` | none, returns `{enabled, args}` / `{args}` |
| `get_dev_bypass_settings` | none, returns `{bypass_verification, bypass_file_checks}` |
| `open_discord_dm`, `open_discord_server`, `open_kofi` | none |

`apiState`, `authState`, `logState`, `downloadState`, `verifiedBuildState`, `state`, `appHandle`, and `window` are Tauri managed-state injections, not caller arguments.

Events the frontend listens for: `download-progress` carrying `{percentage, status_text, speed_mbs}`, and `verification-progress` carrying `{progress, status_text}` where progress runs 0 to 1.

Tauri plugin IPC in use: `plugin:dialog|open`, `plugin:process|restart`, `plugin:updater|check|download|install|download_and_install`, `plugin:event|listen|unlisten`, `plugin:resources|close`.

The frontend reads the gateway envelope in SCREAMING_CASE (`e.SUCCESS===1`, `LINKED_FLAG`, `STRING_VALUE`, `ACCESS_TOKEN`, `USER_NAME`, `CUSTOM_MESSAGE`, `CUSTOM_VALUE_1..4`, `TEXT_VALUE`, `PORTAL_INFO_1`). The Rust layer maps those from the snake_case JSON in section (a), so the uppercase names are an internal IPC convention and not a wire format.

## Refresh token handling

**The frontend never sees the refresh token.** The substring `refresh` appears **zero times** in `fe_index.js`. The only credential the UI holds is `t.loginToken`, which is always the `ACCESS_TOKEN` value. Refresh is therefore entirely transparent to the UI.

**When refresh fires.** The standalone refresh helper at `0x1400c6dc0` has exactly three callers, all of them the Bearer-authenticated supporter operations:

```
0x1401632b0  launcher_supporter_bot_names_list
0x14018ca90  launcher_supporter_bot_name_upsert
0x1401911c0  launcher_supporter_bot_name_delete
```

The launch path does not call that helper; it has refresh inlined, with its own path reference at `0x1401c8b1f` and method load at `0x1401c8cb2`. So refresh is attempted on demand around bearer calls, not on a timer.

**Tokens rotate on refresh.** The session deserializer that owns the `refresh_token` field is function `0x1406bead0`, 11392 bytes, and it is called from all of `0x1400c6dc0` (refresh), `0x14014e7b0` (register), `0x140171b20` (login), `0x1401c7fa0` (launch), and `0x1406c55c0`. Since refresh parses into the same full session type, **the refresh response carries a fresh `refresh_token` and `refresh_expiration_datetime` alongside the new access token.** Plan for rotation.

**Tokens are not persisted.** They live in the `cluckers_central::LauncherAuthState` managed state. The settings file field list, pooled at file offset `14204392`, contains no token fields at all:

```
install_path selected_version dev_command_line dev_command_line_args
dev_bypass_verification dev_bypass_file_checks prefer_delta prefer_smart_repair language
```

The settings file itself is `cluckers_launcher_settings.json` under `%APPDATA%\com.cluckers-central.app`. So both tokens are in-memory only and are lost when the launcher exits. Related failure strings are `[Gateway] launcher session refresh failed`, `Launcher session expired. Please log in again.`, and `Login required.`

---

# (e) Errors, base URLs, environment, and game launch

## Errors

RFC 7807 as assumed. The client parses `detail` and `title`. Pooled literals at file `0xc63e7c`:

```
detail  request failed   HTTP    :    title   JSON parse failed:
Launcher portal    Launcher portal request failed:
```

Rendered failures read `Launcher portal request failed: HTTP <code>: <detail>`.

The frontend maps exactly one status code and passes everything else through verbatim:

```js
f[1]==="401"?"Invalid username or password.":f[2].trim()
```

parsed with `/HTTP\s+(\d{3})[^:]*:\s*(.+)$/`. **There is no 429 or Retry-After handling anywhere.** The `Rate limited (login). Try again in a minute.` text in the 0.9.94 log is a server-supplied `detail` rendered as-is.

Server-behaviour leaks worth knowing:

- `Account too new for this client; update launcher`, pooled next to `account_id_overflow`, so the client-version gate is an account-ID range test.
- `Gateway response missing LAUNCH_TOKEN`, `Gateway response missing PORTAL_INFO_1 (content bootstrap)`, `Gateway returned an empty content bootstrap`, `Gateway returned an empty launch token`.
- `[Launch] Scheduling launch token file cleanup in 15 minutes`.
- Staged rollout gates `machine not in delta_rollout_percent=` and `machine not in repair_rollout_percent=`.
- `GameVersion.dat mismatch. Repair/update required.` and `Missing GameVersion.dat hash metadata on server.`
- Anti-tamper surfacing: `[Repair][Guard5] extraneous binary in monitored dir (SURFACED, removal suppressed):`, over monitored directories `/binaries/` and `/easyanticheat/`.
- Frontend file states from `check_for_game_updates.status`: `upToDate`, `updateRequired`, `notInstalled`, `corruptOrModified`.

## Base URLs and environment

| purpose | value | offset |
|---|---|---|
| gateway default | `https://api.project-crown.com` | `0xd8b889` |
| gateway env, primary | `LAUNCHER_GATEWAY_BASE_URL` | `0xd8b80b` |
| gateway env, fallback | `API_BASE_URL` | `0xd8b8a6` |
| updater default | `https://updater.realmhub.io` | `0xd6c53c` |
| updater mirrors | `https://updater.realmhub.io,https://updater.project-crown.com` | `0xd6c558` |
| updater env | `CC_UPDATER_BASE_URL` | `0xd6c521` |
| self-update manifests | `https://updater.realmhub.io/update.json`, `https://updater.project-crown.com/update-crown.json` | `0xc73538` |
| Discord server | `https://discord.gg/realmroyale` | `0xd8bdf0` |
| Ko-fi | `https://ko-fi.com/projectcrown/tiers` | `0xd8cdeb` |
| dev CLI env | `CC_DEV_COMMAND_LINE` | `0xd8c961` |
| vite dev origin | `http://localhost:1420` | `0xc7340f` |

`gateway-dev.project-crown.com` no longer appears in the binary. The 0.9.94 log in this directory shows it was the old hardcoded default, so dev pointing now goes exclusively through the two environment variables.

## Game launch

Argument literals recovered from the launch function:

```
resx=1920  resy=1080  -seekfreeloadingpcconsole  -nohomedir  -dx11
-content_bootstrap_size=136
```

Log-redaction patterns, which reveal the full argument vocabulary the client knows:

```
-content_bootstrap_shm=<redacted>  -eac_oidc_token_file=<redacted>
-eac_oidc_token=<redacted>  -token_file=<redacted>  -token=<redacted>
```

The token written to the `-token_file` path is the **`launch_token` from launch-auth**, not the session access token, and it is deleted after 15 minutes. Game languages offered: `INT, CHN, CHT, DEU, ESL, ESN, FRA, JPN, KOR, POL, POR, RUS, TUR`, default `INT`. The dev launch-args panel is gated on `get_dev_command_line_settings().enabled` and states `Appended to the default args. Auth/bootstrap args are blocked.`

## Supporter tiers, fully client-visible

Tier comes from `CUSTOM_VALUE_1` as a number, falling back to a name map over `CUSTOM_MESSAGE`: `cluck_overlord`/`overlord` is 4, `cluck_commander`/`commander`/`supporter` is 3, `cluck_soldier`/`soldier` is 2, `cluckling` is 1. `CUSTOM_VALUE_2` is slots total, `CUSTOM_VALUE_3` slots used, `CUSTOM_VALUE_4` announcement seconds, `TEXT_VALUE` the announcement text, and `PORTAL_INFO_1` a JSON string array of bot names. The bot-names UI renders only at tier 3 or above. Slot count is 2 at tier 4, 1 at tier 3, otherwise 0. The name field is `maxlength="32"`.

---

# (f) Contradictions with our Go client, and open questions

## Contradictions

1. **`/launcher/v1/content-bootstrap` no longer exists.** The substring appears zero times in the 1.6.3 binary. `POST /launcher/v1/launch-auth` replaces it and returns the bootstrap **and** a separate launch token.
2. **The game receives a launch token, not the access token.** Our client passes the session access token via `-token_file`.
3. **Refresh tokens exist and rotate.** We cache only an access token on a 45-minute TTL. The real client carries `refresh_token`, `refresh_expiration_datetime`, both `*_expires_at_unix` fields, and a dedicated refresh endpoint whose response is a full session object.
4. **Our User-Agent is fabricated.** The real client sends none.
5. **`x-realm-client-build` is required on launch-auth** and we do not send it.
6. **Discord link endpoints are gone from the current API.** Linking rides on `session-or-link`, with the link code delivered in `ACCESS_TOKEN` when `linked_flag` is not 1.
7. **Bot-name slot operations are PUT and DELETE on `/{slot}`**, not a single upsert shape. The PUT body is `{bot_name}`.
8. **Environment override order is `LAUNCHER_GATEWAY_BASE_URL`, then `API_BASE_URL`.** We only expose a `--gateway` flag.
9. **Game arguments differ**, notably `resx`/`resy` and `-content_bootstrap_size=136` as a separate token.
10. **A websocket status channel exists** at `/ws` that we do not implement at all.

## Open questions

- Whether the server still honours a `pin` field on `session-or-link`, given this client cannot send one. Only a live request settles it.
- The exact struct field feeding `x-realm-client-build`. I proved it is a dotted numeric game build and not the launcher version, but not which `verifiedBuildState` member supplies it.
- Whether `get_api_config` derives `ws_base_url` from the gateway base URL or from the updater manifest. Both `http_base_url` and `ws_base_url` sit in the same serde blob at `0xd8bdf0` alongside `base_url` and `manifest_url`, so either source fits the strings. The resolver was not disassembled.
- Response shapes for bot-names GET and PUT, which the client deserializes as untyped JSON.
- Server-side rate-limit thresholds and PIN expiry, which have no client-side representation at all.

---

# Artifacts

All under `/tmp/claude-1000/-home-cstory-src-cluckers/7d0de5db-04af-4783-b2f5-29516da50788/scratchpad/`:

| file | contents |
|---|---|
| `fe_index.js` | decompressed frontend bundle, 49928 bytes |
| `fe_index.css` | decompressed stylesheet, 20498 bytes |
| `fe_index.html` | decompressed Vite shell, 780 bytes |
| `cc_a.txt`, `cc_u.txt` | ASCII and UTF-16LE strings with offsets, main exe |
| `st_a.txt`, `st_u.txt` | same for the smoketest |
| `login_fn.asm` | full disassembly of `launcher_login_or_link`, `0x140171b20`-`0x14017354a` |
| `launch_fn.asm` | full disassembly of the launch function, `0x1401c7fa0`-`0x1401d0c58` |
| `fnstr.py` | lists every string, 8/4/2-byte immediate, and method static referenced inside the function containing a given address |
| `split.py` | splits a concatenated serde literal pool by resolving code reference points |
| `pool.py` | finds `(ptr,len)` arrays pointing into a given pool |
| `refs.py`, `calls.py`, `fields.py` | reference and call-target scanners |
