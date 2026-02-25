# Contributing to Cluckers

## Development

### Prerequisites

- Go 1.25+
- mingw-w64 (for cross-compiling the embedded SHM helper binary)
- For GUI builds: CGO-capable toolchain + platform graphics libraries
  - Linux: `libgl1-mesa-dev`, `xorg-dev`, `libxkbcommon-dev`
  - Windows cross-compile from Linux: `gcc-mingw-w64-x86-64`

### Building from Source

The embedded `shm_launcher.exe` binary is **not committed to git**. You must build it from C source before compiling the Go project.

```bash
git clone https://github.com/0xc0re/cluckers.git
cd cluckers

# 1. Build the embedded SHM helper (requires mingw-w64)
x86_64-w64-mingw32-gcc -o assets/shm_launcher.exe tools/shm_launcher.c -municode

# 2. CLI-only build (no CGO, no GUI)
CGO_ENABLED=0 go build -o cluckers ./cmd/cluckers

# 3. GUI build (Linux -- requires CGO + graphics libs)
CGO_ENABLED=1 go build -tags gui -o cluckers ./cmd/cluckers

# 4. GUI build (Windows cross-compile from Linux)
CGO_ENABLED=1 GOOS=windows CC=x86_64-w64-mingw32-gcc go build -tags gui -o cluckers.exe ./cmd/cluckers

# 5. CLI-only Windows cross-compile (no CGO)
GOOS=windows go build -o cluckers.exe ./cmd/cluckers
```

Release binaries from GitHub are always built with the `gui` tag. Without `gui`, the binary is CLI-only (no graphical interface).

### Running Tests

```bash
go test ./...                    # tests run with CGO_ENABLED=0
go vet ./...                     # vet Linux
GOOS=windows go vet ./...        # vet Windows cross-platform
```

### Project Structure

```
cmd/cluckers/         Entry point, sets version via ldflags
internal/cli/         Cobra commands, platform-specific via _linux.go/_windows.go
internal/config/      Configuration and paths
internal/gateway/     HTTP client for Project Crown gateway
internal/auth/        Authentication and credential management
internal/crypto/      NaCl secretbox encryption
internal/launch/      Game launch orchestration (platform-specific)
internal/game/        Game file management (version, download, extract)
internal/wine/        Wine/Proton-GE detection and prefix management (Linux-only)
internal/ui/          Terminal output helpers (colors, spinners, prompts, errors)
assets/               Embedded binaries (shm_launcher.exe, controller VDF)
tools/                Build-time source files (shm_launcher.c)
```

### CI/CD

- CI runs on all branches and PRs: builds Linux + Windows (both with `gui` tag), tests (`CGO_ENABLED=0`), vets both platforms
- Releases via goreleaser on tag push (`v*`): produces three artifacts:
  - `cluckers_*_linux_amd64.tar.gz` -- Linux GUI+CLI binary
  - `cluckers_*_windows_amd64.zip` -- Windows GUI+CLI binary
  - `Cluckers-x86_64.AppImage` -- Linux GUI+CLI binary with bundled Proton-GE and graphics libraries
- Changelog grouped by conventional commit prefix

### Commit Conventions

Conventional commits: `feat:`, `fix:`, `docs:`, `refactor:`, `ci:`, `chore:`. Goreleaser groups changelog entries by prefix. Merge commits and `chore:` commits are excluded from the changelog.

### Platform-Specific Code

- File naming convention: `_linux.go` / `_windows.go` for platform-specific behavior
- `internal/wine/` uses `//go:build linux` comment tags (all files are Linux-only)
- GUI code uses `//go:build gui` tag; CLI-only builds exclude GUI entirely
- All builds use `CGO_ENABLED=0` except GUI builds which require CGO

### Key Dependencies

| Package | Purpose |
|---------|---------|
| `spf13/cobra` + `spf13/viper` | CLI framework + config |
| `hashicorp/go-retryablehttp` | HTTP client with retry/backoff |
| `fatih/color` | Terminal colors |
| `briandowns/spinner` | Terminal spinners |
| `schollz/progressbar/v3` | Download progress bars |
| `zeebo/blake3` | BLAKE3 hashing for file integrity |
| `denisbrodbeck/machineid` | Machine ID for key derivation |
| `golang.org/x/crypto` | NaCl secretbox + scrypt |
| `golang.org/x/term` | Terminal detection + password input |

### Security

Credentials are encrypted at rest with NaCl secretbox (XSalsa20-Poly1305), with keys derived from the machine ID via scrypt (machine-bound, non-portable). No system keyring dependency -- works in Steam Deck Gaming Mode and other headless environments.

See [SECURITY.md](SECURITY.md) for the full threat model.
