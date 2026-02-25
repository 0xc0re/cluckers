#!/bin/sh
# Cluckers installer — download and install the cluckers binary from GitHub Releases.
#
# Usage:
#   curl -sSL https://raw.githubusercontent.com/0xc0re/cluckers/master/install.sh | sh
#   FORMAT=tarball curl -sSL ... | sh      # force tarball instead of AppImage
#   INSTALL_DIR=/usr/local/bin curl -sSL ... | sh
#   sh install.sh

set -eu

# --------------------------------------------------------------------------- #
#  Color / UX helpers
# --------------------------------------------------------------------------- #

if [ -t 1 ]; then
    RED=$(printf '\033[0;31m')
    GREEN=$(printf '\033[0;32m')
    YELLOW=$(printf '\033[0;33m')
    BLUE=$(printf '\033[0;34m')
    BOLD=$(printf '\033[1m')
    RESET=$(printf '\033[0m')
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    BOLD=''
    RESET=''
fi

info()    { printf "${BLUE}::${RESET} %s\n" "$1"; }
success() { printf "${GREEN} %s${RESET}\n" "$1"; }
warn()    { printf "${YELLOW}!! %s${RESET}\n" "$1"; }
error()   { printf "${RED} %s${RESET}\n" "$1" >&2; }
step()    { printf "${BOLD}=> %s${RESET}\n" "$1"; }

# --------------------------------------------------------------------------- #
#  Safety checks
# --------------------------------------------------------------------------- #

if [ "$(id -u)" -eq 0 ]; then
    error "Do not run this installer as root."
    printf "  Run without sudo:  curl -sSL https://raw.githubusercontent.com/0xc0re/cluckers/master/install.sh | sh\n" >&2
    exit 1
fi

# --------------------------------------------------------------------------- #
#  Platform detection
# --------------------------------------------------------------------------- #

OS="$(uname -s)"
ARCH="$(uname -m)"

if [ "$OS" != "Linux" ]; then
    error "Cluckers only supports Linux. Detected OS: $OS"
    exit 1
fi

if [ "$ARCH" != "x86_64" ]; then
    error "Cluckers only supports x86_64 (amd64). Detected architecture: $ARCH"
    exit 1
fi

# Detect distro from /etc/os-release ID field.
DISTRO="unknown"
if [ -f /etc/os-release ]; then
    # shellcheck source=/dev/null
    DISTRO=$(. /etc/os-release && printf '%s' "${ID:-unknown}")
fi

# Detect Steam Deck specifically.
IS_STEAM_DECK=false
if [ "$DISTRO" = "steamos" ]; then
    IS_STEAM_DECK=true
elif [ -d "/home/deck" ]; then
    IS_STEAM_DECK=true
elif [ -f /etc/os-release ] && grep -qi "SteamOS" /etc/os-release 2>/dev/null; then
    IS_STEAM_DECK=true
fi

# --------------------------------------------------------------------------- #
#  Install location
# --------------------------------------------------------------------------- #

DEFAULT_DIR="$HOME/.local/bin"

if [ "$IS_STEAM_DECK" = true ]; then
    # Steam Deck filesystem is read-only; always use ~/.local/bin.
    INSTALL_DIR="${INSTALL_DIR:-$DEFAULT_DIR}"
else
    INSTALL_DIR="${INSTALL_DIR:-$DEFAULT_DIR}"
fi

mkdir -p "$INSTALL_DIR"

INSTALL_PATH="$INSTALL_DIR/cluckers"

# --------------------------------------------------------------------------- #
#  Download tool detection
# --------------------------------------------------------------------------- #

DOWNLOAD_CMD=""
if command -v curl >/dev/null 2>&1; then
    DOWNLOAD_CMD="curl"
elif command -v wget >/dev/null 2>&1; then
    DOWNLOAD_CMD="wget"
else
    error "Neither curl nor wget found. Please install one and try again."
    exit 1
fi

download() {
    # download URL DEST
    if [ "$DOWNLOAD_CMD" = "curl" ]; then
        curl -fsSL -o "$2" "$1"
    else
        wget -qO "$2" "$1"
    fi
}

download_text() {
    # download_text URL  -> stdout
    if [ "$DOWNLOAD_CMD" = "curl" ]; then
        curl -fsSL "$1"
    else
        wget -qO- "$1"
    fi
}

# --------------------------------------------------------------------------- #
#  Format preference
# --------------------------------------------------------------------------- #

FORMAT="${FORMAT:-auto}"
case "$FORMAT" in
    auto|appimage|tarball) ;;
    *)
        error "Unknown FORMAT '$FORMAT'. Use: auto, appimage, or tarball."
        exit 1
        ;;
esac

# --------------------------------------------------------------------------- #
#  Discover latest release
# --------------------------------------------------------------------------- #

GITHUB_API="https://api.github.com/repos/0xc0re/cluckers/releases/latest"

step "Checking latest cluckers release..."

RELEASE_JSON=$(download_text "$GITHUB_API") || {
    error "Failed to fetch release information from GitHub."
    printf "  URL: %s\n" "$GITHUB_API" >&2
    exit 1
}

# Extract version tag (e.g. "v0.1.0") — POSIX-safe parsing.
LATEST_TAG=$(printf '%s' "$RELEASE_JSON" | sed -n 's/.*"tag_name" *: *"\([^"]*\)".*/\1/p' | head -1)
LATEST_VERSION=$(printf '%s' "$LATEST_TAG" | sed 's/^v//')

if [ -z "$LATEST_VERSION" ]; then
    error "Could not determine latest release version."
    exit 1
fi

# Extract download URLs — prefer AppImage, fall back to tar.gz.
APPIMAGE_URL=$(printf '%s' "$RELEASE_JSON" | sed -n 's/.*"browser_download_url" *: *"\([^"]*Cluckers-x86_64\.AppImage\)".*/\1/p' | head -1)
TARBALL_URL=$(printf '%s' "$RELEASE_JSON" | sed -n 's/.*"browser_download_url" *: *"\([^"]*cluckers_[^"]*linux_amd64\.tar\.gz\)".*/\1/p' | head -1)
CHECKSUMS_URL=$(printf '%s' "$RELEASE_JSON" | sed -n 's/.*"browser_download_url" *: *"\([^"]*checksums\.txt\)".*/\1/p' | head -1)

# Apply format preference.
case "$FORMAT" in
    appimage) TARBALL_URL="" ;;
    tarball)  APPIMAGE_URL="" ;;
    # auto: keep both, AppImage preferred (existing behavior).
esac

if [ -z "$APPIMAGE_URL" ] && [ -z "$TARBALL_URL" ]; then
    error "Could not find linux_amd64 release asset."
    printf "  Check: https://github.com/0xc0re/cluckers/releases/latest\n" >&2
    exit 1
fi

# Track which format we install.
INSTALLED_APPIMAGE=false

info "Latest version: $LATEST_VERSION"

# --------------------------------------------------------------------------- #
#  Idempotency — check existing install
# --------------------------------------------------------------------------- #

if [ -f "$INSTALL_PATH" ]; then
    CURRENT_VERSION=$("$INSTALL_PATH" --version 2>/dev/null | head -1 | sed 's/.*version //' | sed 's/ .*//' || printf '')
    if [ "$CURRENT_VERSION" = "$LATEST_VERSION" ]; then
        success "cluckers $LATEST_VERSION is already installed and up to date."
        printf "  Location: %s\n" "$INSTALL_PATH"
        exit 0
    fi
    if [ -n "$CURRENT_VERSION" ]; then
        info "Updating cluckers from $CURRENT_VERSION to $LATEST_VERSION"
    fi
fi

# --------------------------------------------------------------------------- #
#  Download and verify
# --------------------------------------------------------------------------- #

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

if [ -n "$APPIMAGE_URL" ]; then
    # ---------- AppImage path (preferred) ----------
    step "Downloading Cluckers AppImage $LATEST_VERSION..."
    download "$APPIMAGE_URL" "$TMPDIR/Cluckers-x86_64.AppImage"

    if [ -n "$CHECKSUMS_URL" ]; then
        step "Verifying checksum..."
        download "$CHECKSUMS_URL" "$TMPDIR/checksums.txt"

        SHA_CMD=""
        if command -v sha256sum >/dev/null 2>&1; then
            SHA_CMD="sha256sum"
        elif command -v shasum >/dev/null 2>&1; then
            SHA_CMD="shasum -a 256"
        fi

        if [ -n "$SHA_CMD" ]; then
            EXPECTED=$(grep "Cluckers-x86_64\.AppImage" "$TMPDIR/checksums.txt" | awk '{print $1}')
            if [ -n "$EXPECTED" ]; then
                ACTUAL=$($SHA_CMD "$TMPDIR/Cluckers-x86_64.AppImage" | awk '{print $1}')
                if [ "$EXPECTED" != "$ACTUAL" ]; then
                    error "Checksum verification failed!"
                    printf "  Expected: %s\n" "$EXPECTED" >&2
                    printf "  Got:      %s\n" "$ACTUAL" >&2
                    exit 1
                fi
                success "Checksum verified."
            else
                warn "No AppImage checksum found in checksums.txt; skipping verification."
            fi
        else
            warn "No sha256sum or shasum found; skipping checksum verification."
        fi
    else
        warn "No checksums.txt found in release; skipping verification."
    fi

    # Install the AppImage directly (no extraction needed).
    step "Installing to $INSTALL_DIR..."
    mv "$TMPDIR/Cluckers-x86_64.AppImage" "$INSTALL_PATH"
    chmod +x "$INSTALL_PATH"
    INSTALLED_APPIMAGE=true
else
    # ---------- Tarball fallback (older releases without AppImage) ----------
    step "Downloading cluckers $LATEST_VERSION..."
    download "$TARBALL_URL" "$TMPDIR/cluckers.tar.gz"

    if [ -n "$CHECKSUMS_URL" ]; then
        step "Verifying checksum..."
        download "$CHECKSUMS_URL" "$TMPDIR/checksums.txt"

        SHA_CMD=""
        if command -v sha256sum >/dev/null 2>&1; then
            SHA_CMD="sha256sum"
        elif command -v shasum >/dev/null 2>&1; then
            SHA_CMD="shasum -a 256"
        fi

        if [ -n "$SHA_CMD" ]; then
            EXPECTED=$(grep "cluckers_.*linux_amd64\.tar\.gz" "$TMPDIR/checksums.txt" | awk '{print $1}')
            ACTUAL=$($SHA_CMD "$TMPDIR/cluckers.tar.gz" | awk '{print $1}')
            if [ "$EXPECTED" != "$ACTUAL" ]; then
                error "Checksum verification failed!"
                printf "  Expected: %s\n" "$EXPECTED" >&2
                printf "  Got:      %s\n" "$ACTUAL" >&2
                exit 1
            fi
            success "Checksum verified."
        else
            warn "No sha256sum or shasum found; skipping checksum verification."
        fi
    else
        warn "No checksums.txt found in release; skipping verification."
    fi

    step "Installing to $INSTALL_DIR..."
    tar xzf "$TMPDIR/cluckers.tar.gz" -C "$TMPDIR"

    if [ -f "$TMPDIR/cluckers" ]; then
        mv "$TMPDIR/cluckers" "$INSTALL_PATH"
    else
        error "Binary not found in archive. Contents:"
        ls -la "$TMPDIR" >&2
        exit 1
    fi
    chmod +x "$INSTALL_PATH"
fi

# Verify the installed binary.
INSTALLED_VERSION=$("$INSTALL_PATH" --version 2>/dev/null | head -1 || printf '')
if [ -z "$INSTALLED_VERSION" ]; then
    warn "Installed binary did not respond to --version, but file exists."
    INSTALLED_VERSION="$LATEST_VERSION"
fi

# --------------------------------------------------------------------------- #
#  Wine / Proton-GE detection
# --------------------------------------------------------------------------- #

WINE_STATUS=""

check_proton_ge() {
    # Check all standard Proton-GE locations (mirrors detect.go logic).
    SEARCH_DIRS="
        $HOME/.steam/root/compatibilitytools.d
        $HOME/.steam/steam/compatibilitytools.d
        $HOME/.local/share/Steam/compatibilitytools.d
        $HOME/.var/app/com.valvesoftware.Steam/data/Steam/compatibilitytools.d
        /usr/share/steam/compatibilitytools.d
        $HOME/snap/steam/common/.steam/steam/compatibilitytools.d
        $HOME/.var/app/net.davidotek.pupgui2/data/Steam/compatibilitytools.d
        $HOME/.local/share/Steam/steamapps/common/Proton - GE/compatibilitytools.d
    "

    # Add symlink-resolved paths for ~/.steam/root and ~/.steam/steam.
    for link in "$HOME/.steam/root" "$HOME/.steam/steam"; do
        if [ -L "$link" ]; then
            resolved=$(readlink -f "$link" 2>/dev/null || true)
            if [ -n "$resolved" ] && [ "$resolved" != "$link" ]; then
                SEARCH_DIRS="$SEARCH_DIRS
        $resolved/compatibilitytools.d"
            fi
        fi
    done

    for base_dir in $SEARCH_DIRS; do
        # Check proton-ge-custom (system package).
        if [ -f "$base_dir/proton-ge-custom/files/bin/wine64" ]; then
            printf '%s' "$base_dir/proton-ge-custom"
            return 0
        fi

        # Check GE-Proton* (ProtonUp-Qt versioned installs).
        if [ -d "$base_dir" ]; then
            for d in "$base_dir"/GE-Proton*/files/bin/wine64; do
                if [ -f "$d" ]; then
                    # Return the Proton dir (three levels up from wine64).
                    printf '%s' "$(dirname "$(dirname "$(dirname "$d")")")"
                    return 0
                fi
            done
        fi
    done
    return 1
}

# If we installed the AppImage, Proton-GE is bundled inside it.
if [ "$INSTALLED_APPIMAGE" = true ]; then
    WINE_STATUS="Proton-GE bundled in AppImage"
else
    PROTON_DIR=""
    if PROTON_DIR=$(check_proton_ge); then
        WINE_STATUS="Proton-GE found: $(basename "$PROTON_DIR")"
    elif command -v wine >/dev/null 2>&1; then
        WINE_STATUS="System Wine found: $(wine --version 2>/dev/null || printf 'wine')"
    else
        WINE_STATUS="not found"
    fi
fi

# --------------------------------------------------------------------------- #
#  PATH check — auto-add if missing
# --------------------------------------------------------------------------- #

PATH_OK=true
case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *)
        PATH_OK=false
        ;;
esac

PATH_ADDED=false
if [ "$PATH_OK" = false ]; then
    # Determine the shell RC file to modify.
    SHELL_NAME=$(basename "${SHELL:-/bin/sh}")
    case "$SHELL_NAME" in
        zsh)  RC_FILE="$HOME/.zshrc" ;;
        bash) RC_FILE="$HOME/.bashrc" ;;
        *)    RC_FILE="$HOME/.profile" ;;
    esac

    EXPORT_LINE='export PATH="$HOME/.local/bin:$PATH"'

    # Only append if the line isn't already in the file.
    if [ -f "$RC_FILE" ] && grep -qF '.local/bin' "$RC_FILE" 2>/dev/null; then
        : # Already present in RC file, skip.
    else
        printf '%s\n' "$EXPORT_LINE" >> "$RC_FILE"
        PATH_ADDED=true
    fi
fi

# --------------------------------------------------------------------------- #
#  Summary
# --------------------------------------------------------------------------- #

printf "\n"
printf '%s%s%s\n' "$BOLD" "================================================" "$RESET"
printf '%s  Cluckers installed successfully%s\n' "$BOLD" "$RESET"
printf '%s%s%s\n' "$BOLD" "================================================" "$RESET"
printf "\n"
printf '  %sLocation:%s  %s\n' "$BOLD" "$RESET" "$INSTALL_PATH"
printf '  %sVersion:%s   %s\n' "$BOLD" "$RESET" "$LATEST_VERSION"
if [ "$INSTALLED_APPIMAGE" = true ]; then
    printf '  %sFormat:%s    AppImage (Proton-GE bundled)\n' "$BOLD" "$RESET"
fi

if [ "$WINE_STATUS" = "not found" ]; then
    printf '  %sWine:%s      %sNot found%s\n' "$BOLD" "$RESET" "$YELLOW" "$RESET"
else
    printf '  %sWine:%s      %s%s%s\n' "$BOLD" "$RESET" "$GREEN" "$WINE_STATUS" "$RESET"
fi

printf "\n"

# PATH status.
if [ "$PATH_ADDED" = true ]; then
    success "Added $INSTALL_DIR to PATH in $(basename "$RC_FILE")."
    printf "  Run 'source %s' or open a new terminal to use cluckers.\n" "$RC_FILE"
    printf "\n"
elif [ "$PATH_OK" = false ]; then
    warn "$INSTALL_DIR is not in your PATH (already configured in $(basename "$RC_FILE"), restart your shell)."
    printf "\n"
fi

# Wine install guidance if not found (only applies to tarball installs;
# AppImage bundles Proton-GE so this section is skipped).
if [ "$WINE_STATUS" = "not found" ]; then
    warn "Wine or Proton-GE is required to run Realm Royale."
    printf "\n"
    printf "  %sTip:%s The AppImage version bundles Proton-GE automatically.\n" "$BOLD" "$RESET"
    printf "  If this release has an AppImage, re-run the installer to get it.\n"
    printf "\n"
    if [ "$IS_STEAM_DECK" = true ]; then
        printf '  %sSteam Deck:%s Install ProtonUp-Qt from the Discover store,\n' "$BOLD" "$RESET"
        printf "  then use it to install the latest GE-Proton version.\n"
    else
        case "$DISTRO" in
            arch)
                printf "  Install Proton-GE via ProtonUp-Qt (recommended), or:\n"
                printf "    sudo pacman -S wine\n"
                ;;
            ubuntu|debian|linuxmint|pop)
                printf "  Install Proton-GE via ProtonUp-Qt (recommended), or:\n"
                printf "    sudo apt install wine\n"
                ;;
            fedora)
                printf "  Install Proton-GE via ProtonUp-Qt (recommended), or:\n"
                printf "    sudo dnf install wine\n"
                ;;
            *)
                printf "  Install Proton-GE: https://github.com/GloriousEggroll/proton-ge-custom\n"
                printf "  Or install Wine for your distribution.\n"
                ;;
        esac
    fi
    printf "\n"
fi

# Next steps.
printf '  %sNext steps:%s\n' "$BOLD" "$RESET"
printf "    cluckers            Launch GUI\n"
printf "    cluckers launch     Launch game (CLI)\n"
printf "    cluckers status     Check system readiness\n"
printf "\n"
printf "  On first launch, cluckers will prompt for your Project Crown\n"
printf "  credentials and set up the Wine prefix automatically.\n"
printf "\n"
