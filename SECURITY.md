# Security Policy

Cluckers is a native Linux CLI launcher for Realm Royale on the Project Crown private server. While this is a game launcher and not a financial or enterprise application, we take security seriously. User credentials and network communication are protected with industry-standard cryptographic primitives.

## Reporting a Vulnerability

**Please do not open public GitHub issues for security vulnerabilities.**

If you discover a security issue in Cluckers, report it privately using one of these methods:

1. **GitHub Private Vulnerability Reporting** (preferred): Navigate to the [Security tab](https://github.com/0xc0re/cluckers/security) of the `0xc0re/cluckers` repository and select "Report a vulnerability." This keeps the report confidential until a fix is available.

2. **GitHub Issue with minimal detail**: Open a GitHub issue titled `[SECURITY]` with a brief, non-specific description requesting private contact. Do not include exploit details in the public issue.

We aim to acknowledge reports within 72 hours and provide an initial assessment within one week.

## Supported Versions

Only the latest release is actively supported with security updates.

| Version | Supported |
|---------|-----------|
| Latest release | Yes |
| Older releases | No |

Users are encouraged to update to the latest release. Cluckers can check for updates via `cluckers status`.

## Security Model

### Credential Storage

User credentials are encrypted at rest using **NaCl secretbox** (XSalsa20-Poly1305 authenticated encryption).

- **Key derivation**: A 32-byte encryption key is derived using **scrypt** (N=32768, r=8, p=1) from the machine's unique ID and an application-scoped salt.
- **Encryption**: Each encryption operation generates a cryptographically random 24-byte nonce. The ciphertext format is `[nonce (24 bytes) | sealed data]`.
- **File permissions**: The credential file is written with `0600` permissions (owner read/write only).
- **Machine binding**: Credentials are bound to the machine where they were saved. They cannot be decrypted on a different machine because the key is derived from the machine ID.
- **Credential removal**: Users can delete stored credentials at any time with `cluckers logout`, which removes the encrypted credential file from disk.

### Network Communication

- All gateway API communication uses **HTTPS** to `gateway-dev.project-crown.com`, which sits behind **Cloudflare**.
- The gateway client uses `retryablehttp` with a **15-second timeout** and automatic retry with backoff (up to 3 retries).
- Game server connections are direct TCP to the MCTS server, separate from the gateway.

### No System Keyring Dependency

Cluckers deliberately avoids D-Bus and system keyring integrations. This is a design choice to ensure compatibility with **Steam Deck Gaming Mode** and other headless Linux environments where D-Bus services may not be available.

### Embedded Binaries

Two assets are **embedded in the Go binary at build time** via `go:embed`. They are not downloaded at runtime, eliminating a class of supply-chain and tampering risks.

- **`shm_launcher.exe`** -- Win32 helper for named shared memory under Wine. Built from source (`tools/shm_launcher.c`) in CI/release workflows via mingw-w64. The C source is available in the repository for audit.
- **`controller_neptune_config.vdf`** -- Steam Deck controller layout configuration for Realm Royale. A plain-text Valve Data Format file.

### Access Tokens

Access tokens obtained during login are held **in memory only** for the duration of the launch session. They are not written to disk. Once the launcher process exits, the tokens are gone.

## Threat Model and Scope

### In Scope

The following areas are within the security scope of the Cluckers project:

- **Credential storage** -- encryption, key derivation, file permissions, and machine binding
- **Network communication** -- TLS usage, request handling, and response validation
- **Binary integrity** -- embedded assets, build reproducibility, and release signatures (checksums)
- **Local privilege escalation** -- ensuring the launcher does not require or request elevated privileges

### Out of Scope

The following are managed by other parties and are outside the scope of this project:

- **Game server security** -- managed by the Project Crown team
- **Wine/Proton vulnerabilities** -- managed by the Wine and Proton-GE projects
- **Cloudflare configuration** -- managed by the Project Crown infrastructure team

If you discover a vulnerability in one of these out-of-scope areas, please report it to the responsible party directly.

## Dependencies

Cluckers uses well-known, widely audited Go libraries for its security-critical operations:

- **golang.org/x/crypto** -- NaCl secretbox (XSalsa20-Poly1305) and scrypt key derivation
- **hashicorp/go-retryablehttp** -- HTTP client with retry and backoff for gateway communication
- **denisbrodbeck/machineid** -- cross-platform machine ID retrieval for key derivation

If you become aware of a known vulnerability in any of these dependencies that affects Cluckers, please report it using the process described above.
