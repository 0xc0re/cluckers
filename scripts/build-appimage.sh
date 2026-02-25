#!/bin/bash
# build-appimage.sh - Build the Cluckers AppImage with bundled Proton-GE
#
# Usage:
#   ./scripts/build-appimage.sh                  # Build binary from source, then package
#   ./scripts/build-appimage.sh path/to/cluckers # Use pre-built binary
#
# Prerequisites (must be on PATH or in script directory):
#   - linuxdeploy      (https://github.com/linuxdeploy/linuxdeploy/releases)
#   - appimagetool     (https://github.com/AppImage/appimagetool/releases)
#   - zsyncmake        (apt install zsync)
#
# The script will download the type2-runtime if not already present.

set -euo pipefail

# ---------------------------------------------------------------------------
# Colors
# ---------------------------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

info()  { echo -e "${CYAN}[INFO]${NC}  $*"; }
ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
die()   { error "$@"; exit 1; }

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
PROTON_VERSION="GE-Proton10-32"
PROTON_URL="https://github.com/GloriousEggroll/proton-ge-custom/releases/download/${PROTON_VERSION}/${PROTON_VERSION}.tar.gz"
RUNTIME_URL="https://github.com/AppImage/type2-runtime/releases/download/continuous/runtime-x86_64"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BUILD_DIR="$PROJECT_DIR/build/appimage"
APPDIR="$BUILD_DIR/Cluckers.AppDir"
OUTPUT_DIR="$PROJECT_DIR/dist"
PROTON_CACHE="$BUILD_DIR/proton-cache"
RUNTIME_PATH="$BUILD_DIR/type2-runtime-x86_64"

# ---------------------------------------------------------------------------
# Prerequisite checks
# ---------------------------------------------------------------------------
check_prerequisites() {
    local missing=0

    if ! command -v linuxdeploy &>/dev/null && ! command -v linuxdeploy-x86_64.AppImage &>/dev/null; then
        error "linuxdeploy not found."
        echo "  Download from: https://github.com/linuxdeploy/linuxdeploy/releases/download/continuous/linuxdeploy-x86_64.AppImage"
        echo "  Then: chmod +x linuxdeploy-x86_64.AppImage && sudo mv linuxdeploy-x86_64.AppImage /usr/local/bin/linuxdeploy"
        missing=1
    fi

    if ! command -v appimagetool &>/dev/null && ! command -v appimagetool-x86_64.AppImage &>/dev/null; then
        error "appimagetool not found."
        echo "  Download from: https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage"
        echo "  Then: chmod +x appimagetool-x86_64.AppImage && sudo mv appimagetool-x86_64.AppImage /usr/local/bin/appimagetool"
        missing=1
    fi

    if ! command -v zsyncmake &>/dev/null; then
        error "zsyncmake not found."
        echo "  Install via: sudo apt install zsync  (Debian/Ubuntu)"
        echo "           or: sudo pacman -S zsync    (Arch)"
        echo "           or: sudo dnf install zsync  (Fedora)"
        missing=1
    fi

    if [ "$missing" -eq 1 ]; then
        die "Missing prerequisites. Install the tools listed above and try again."
    fi

    ok "All prerequisites found"
}

# Resolve a command that might be installed as a standalone binary or as an
# AppImage with an architecture suffix (e.g. linuxdeploy-x86_64.AppImage).
resolve_cmd() {
    local name="$1"
    if command -v "$name" &>/dev/null; then
        echo "$name"
    elif command -v "${name}-x86_64.AppImage" &>/dev/null; then
        echo "${name}-x86_64.AppImage"
    else
        die "Cannot resolve command: $name"
    fi
}

# ---------------------------------------------------------------------------
# Step 1: Obtain the binary
# ---------------------------------------------------------------------------
obtain_binary() {
    local binary_arg="${1:-}"

    if [ -n "$binary_arg" ]; then
        if [ ! -f "$binary_arg" ]; then
            die "Pre-built binary not found: $binary_arg"
        fi
        info "Using pre-built binary: $binary_arg"
        cp "$binary_arg" "$APPDIR/usr/bin/cluckers"
        chmod +x "$APPDIR/usr/bin/cluckers"
    else
        info "Building cluckers binary from source..."
        CGO_ENABLED=1 go build -tags gui -o "$APPDIR/usr/bin/cluckers" ./cmd/cluckers
        ok "Binary built"
    fi
}

# ---------------------------------------------------------------------------
# Step 2: Assemble AppDir
# ---------------------------------------------------------------------------
assemble_appdir() {
    info "Assembling AppDir..."

    # Clean previous build
    rm -rf "$APPDIR"

    # Create directory structure
    mkdir -p "$APPDIR/usr/bin"
    mkdir -p "$APPDIR/usr/share/applications"
    mkdir -p "$APPDIR/usr/share/icons/hicolor/256x256/apps"
    mkdir -p "$APPDIR/usr/share/licenses/cluckers"

    # Copy desktop entry
    cp "$PROJECT_DIR/deploy/cluckers.desktop" "$APPDIR/"
    cp "$PROJECT_DIR/deploy/cluckers.desktop" "$APPDIR/usr/share/applications/"

    # Copy icon
    cp "$PROJECT_DIR/internal/gui/assets/cluckers_logo.png" "$APPDIR/cluckers.png"
    cp "$PROJECT_DIR/internal/gui/assets/cluckers_logo.png" \
       "$APPDIR/usr/share/icons/hicolor/256x256/apps/cluckers.png"

    # Create .DirIcon symlink
    ln -sf cluckers.png "$APPDIR/.DirIcon"

    # Copy AppRun
    cp "$PROJECT_DIR/deploy/AppRun" "$APPDIR/AppRun"
    chmod +x "$APPDIR/AppRun"

    # Copy license
    cp "$PROJECT_DIR/LICENSES/PROTON_LICENSE" "$APPDIR/usr/share/licenses/cluckers/PROTON_LICENSE"

    ok "AppDir structure assembled"
}

# ---------------------------------------------------------------------------
# Step 3: Download and cache Proton-GE
# ---------------------------------------------------------------------------
download_proton() {
    mkdir -p "$PROTON_CACHE"

    if [ -d "$PROTON_CACHE/$PROTON_VERSION" ]; then
        info "Using cached Proton-GE: $PROTON_VERSION"
    else
        info "Downloading $PROTON_VERSION..."
        local tarball="$PROTON_CACHE/${PROTON_VERSION}.tar.gz"

        if [ ! -f "$tarball" ]; then
            curl -fSL --progress-bar "$PROTON_URL" -o "$tarball"
        fi

        info "Extracting $PROTON_VERSION..."
        tar -xf "$tarball" -C "$PROTON_CACHE/"

        # Clean up tarball after successful extraction to save disk space
        rm -f "$tarball"
        ok "Proton-GE extracted"
    fi

    # Copy into AppDir preserving symlinks
    info "Copying Proton-GE into AppDir..."
    cp -a "$PROTON_CACHE/$PROTON_VERSION" "$APPDIR/proton"

    # Verify critical file exists
    if [ ! -f "$APPDIR/proton/files/bin/wine64" ]; then
        die "Proton-GE copy failed: $APPDIR/proton/files/bin/wine64 not found"
    fi

    ok "Proton-GE bundled ($(du -sh "$APPDIR/proton" | cut -f1))"
}

# ---------------------------------------------------------------------------
# Step 4: Download type2-runtime if needed
# ---------------------------------------------------------------------------
download_runtime() {
    if [ -f "$RUNTIME_PATH" ]; then
        info "Using cached type2-runtime"
        return
    fi

    info "Downloading type2-runtime..."
    curl -fSL "$RUNTIME_URL" -o "$RUNTIME_PATH"
    chmod +x "$RUNTIME_PATH"
    ok "type2-runtime downloaded"
}

# ---------------------------------------------------------------------------
# Step 5: Bundle shared libraries with linuxdeploy
# ---------------------------------------------------------------------------
bundle_libraries() {
    info "Bundling shared libraries with linuxdeploy..."

    local ld_cmd
    ld_cmd="$(resolve_cmd linuxdeploy)"

    # linuxdeploy bundles Fyne/GL/X11 shared library dependencies and patches RPATH.
    # Do NOT use --output appimage; we use appimagetool separately for type2-runtime.
    "$ld_cmd" --appdir "$APPDIR"

    ok "Shared libraries bundled"
}

# ---------------------------------------------------------------------------
# Step 6: Generate AppImage
# ---------------------------------------------------------------------------
generate_appimage() {
    info "Generating AppImage..."

    mkdir -p "$OUTPUT_DIR"

    local at_cmd
    at_cmd="$(resolve_cmd appimagetool)"

    ARCH=x86_64 "$at_cmd" \
        --runtime-file "$RUNTIME_PATH" \
        -u "gh-releases-zsync|0xc0re|cluckers|latest|Cluckers-x86_64.AppImage.zsync" \
        --comp zstd \
        "$APPDIR" \
        "$OUTPUT_DIR/Cluckers-x86_64.AppImage"

    ok "AppImage generated"
}

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
print_summary() {
    local appimage="$OUTPUT_DIR/Cluckers-x86_64.AppImage"

    echo ""
    echo "==========================================="
    info "AppImage build complete!"
    echo "==========================================="
    echo ""
    echo "  Output:  $appimage"

    if [ -f "$appimage" ]; then
        echo "  Size:    $(du -h "$appimage" | cut -f1)"
    fi

    if [ -f "${appimage}.zsync" ]; then
        echo "  Zsync:   ${appimage}.zsync"
    fi

    echo "  Proton:  $PROTON_VERSION"
    echo ""
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
    info "Cluckers AppImage Builder"
    info "Proton-GE version: $PROTON_VERSION"
    echo ""

    check_prerequisites

    # Assemble AppDir structure first (creates directories)
    assemble_appdir

    # Place binary into AppDir
    obtain_binary "${1:-}"

    # Download and bundle Proton-GE
    download_proton

    # Ensure type2-runtime is available
    download_runtime

    # Bundle shared libraries (Fyne/GL/X11 deps)
    bundle_libraries

    # Generate final AppImage with zsync support
    generate_appimage

    # Print results
    print_summary
}

main "$@"
