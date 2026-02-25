//go:build linux

package wine

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFindSteamInstallNative verifies native Steam installation is detected via ~/.local/share/Steam.
func TestFindSteamInstallNative(t *testing.T) {
	tmp := t.TempDir()

	// Create a fake native Steam installation.
	steamDir := filepath.Join(tmp, ".local", "share", "Steam")
	os.MkdirAll(steamDir, 0755)
	os.WriteFile(filepath.Join(steamDir, "steam.sh"), []byte("#!/bin/bash\n"), 0755)

	got := findSteamInstall(tmp)
	if got != steamDir {
		t.Errorf("findSteamInstall() = %q, want %q", got, steamDir)
	}
}

// TestFindSteamInstallFlatpak verifies Flatpak Steam installation is detected.
func TestFindSteamInstallFlatpak(t *testing.T) {
	tmp := t.TempDir()

	// Create a fake Flatpak Steam installation with steamclient.so marker.
	steamDir := filepath.Join(tmp, ".var", "app", "com.valvesoftware.Steam", "data", "Steam")
	os.MkdirAll(filepath.Join(steamDir, "ubuntu12_32"), 0755)
	os.WriteFile(filepath.Join(steamDir, "ubuntu12_32", "steamclient.so"), []byte("fake"), 0644)

	got := findSteamInstall(tmp)
	if got != steamDir {
		t.Errorf("findSteamInstall() = %q, want %q", got, steamDir)
	}
}

// TestFindSteamInstallSnap verifies Snap Steam installation is detected.
func TestFindSteamInstallSnap(t *testing.T) {
	tmp := t.TempDir()

	// Create a fake Snap Steam installation with steam.sh marker.
	steamDir := filepath.Join(tmp, "snap", "steam", "common", ".local", "share", "Steam")
	os.MkdirAll(steamDir, 0755)
	os.WriteFile(filepath.Join(steamDir, "steam.sh"), []byte("#!/bin/bash\n"), 0755)

	got := findSteamInstall(tmp)
	if got != steamDir {
		t.Errorf("findSteamInstall() = %q, want %q", got, steamDir)
	}
}

// TestFindSteamInstallNotFound verifies empty string when no Steam installation exists.
func TestFindSteamInstallNotFound(t *testing.T) {
	tmp := t.TempDir()
	emptyHome := filepath.Join(tmp, "empty")
	os.MkdirAll(emptyHome, 0755)

	got := findSteamInstall(emptyHome)
	if got != "" {
		t.Errorf("findSteamInstall() = %q, want empty string", got)
	}
}

// TestFindSteamInstallDedup verifies symlink deduplication -- same dir via two paths returns once.
func TestFindSteamInstallDedup(t *testing.T) {
	tmp := t.TempDir()

	// Create native Steam installation at ~/.local/share/Steam.
	steamDir := filepath.Join(tmp, ".local", "share", "Steam")
	os.MkdirAll(steamDir, 0755)
	os.WriteFile(filepath.Join(steamDir, "steam.sh"), []byte("#!/bin/bash\n"), 0755)

	// Create ~/.steam/steam as a symlink pointing to the same directory.
	dotSteam := filepath.Join(tmp, ".steam")
	os.MkdirAll(dotSteam, 0755)
	os.Symlink(steamDir, filepath.Join(dotSteam, "steam"))

	// findSteamInstall should return the first match and skip the symlink duplicate.
	got := findSteamInstall(tmp)
	if got != steamDir {
		t.Errorf("findSteamInstall() = %q, want %q (first in order)", got, steamDir)
	}
}

// TestFindSteamInstallPrefersFirst verifies priority ordering -- native before Flatpak.
func TestFindSteamInstallPrefersFirst(t *testing.T) {
	tmp := t.TempDir()

	// Create native Steam installation (higher priority).
	nativeDir := filepath.Join(tmp, ".local", "share", "Steam")
	os.MkdirAll(nativeDir, 0755)
	os.WriteFile(filepath.Join(nativeDir, "steam.sh"), []byte("#!/bin/bash\n"), 0755)

	// Create Flatpak Steam installation (lower priority).
	flatpakDir := filepath.Join(tmp, ".var", "app", "com.valvesoftware.Steam", "data", "Steam")
	os.MkdirAll(flatpakDir, 0755)
	os.WriteFile(filepath.Join(flatpakDir, "steam.sh"), []byte("#!/bin/bash\n"), 0755)

	got := findSteamInstall(tmp)
	if got != nativeDir {
		t.Errorf("findSteamInstall() = %q, want %q (native has higher priority)", got, nativeDir)
	}
}

// TestIsSteamDirSteamSh verifies detection via steam.sh marker file.
func TestIsSteamDirSteamSh(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "steam.sh"), []byte("#!/bin/bash\n"), 0755)

	if !isSteamDir(tmp) {
		t.Error("isSteamDir should return true when steam.sh exists")
	}
}

// TestIsSteamDirSteamclient verifies detection via ubuntu12_32/steamclient.so marker.
func TestIsSteamDirSteamclient(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, "ubuntu12_32"), 0755)
	os.WriteFile(filepath.Join(tmp, "ubuntu12_32", "steamclient.so"), []byte("fake"), 0644)

	if !isSteamDir(tmp) {
		t.Error("isSteamDir should return true when ubuntu12_32/steamclient.so exists")
	}
}

// TestIsSteamDirEmpty verifies false for empty directory.
func TestIsSteamDirEmpty(t *testing.T) {
	tmp := t.TempDir()

	if isSteamDir(tmp) {
		t.Error("isSteamDir should return false for empty directory")
	}
}
