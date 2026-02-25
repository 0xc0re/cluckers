//go:build linux

package wine

import (
	"os"
	"path/filepath"
)

// steamInstallDirs returns the ordered list of directories to check for a Steam installation.
// Most common first: native, then Flatpak, then Snap.
func steamInstallDirs(home string) []string {
	return []string{
		// Native Steam (XDG data dir).
		filepath.Join(home, ".local", "share", "Steam"),
		// Native Steam symlinks (often point to ~/.local/share/Steam).
		filepath.Join(home, ".steam", "steam"),
		filepath.Join(home, ".steam", "root"),
		// Flatpak Steam.
		filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", "data", "Steam"),
		// Snap Steam.
		filepath.Join(home, "snap", "steam", "common", ".local", "share", "Steam"),
	}
}

// FindSteamInstall locates the Steam root directory by scanning known installation paths.
// Returns the first valid Steam directory found, or "" if none found.
func FindSteamInstall() string {
	return findSteamInstall(userHome())
}

// findSteamInstall is the internal implementation of FindSteamInstall, accepting home for testability.
func findSteamInstall(home string) string {
	seen := make(map[string]bool) // Deduplicate by resolved path.

	for _, dir := range steamInstallDirs(home) {
		resolved := resolveReal(dir)
		if seen[resolved] {
			continue
		}
		seen[resolved] = true

		if isSteamDir(resolved) {
			return resolved
		}
	}
	return ""
}

// isSteamDir checks whether a directory contains a valid Steam installation
// by looking for marker files: steam.sh or ubuntu12_32/steamclient.so.
func isSteamDir(dir string) bool {
	// Check for steam.sh (present in all Steam installations).
	if _, err := os.Stat(filepath.Join(dir, "steam.sh")); err == nil {
		return true
	}
	// Check for ubuntu12_32/steamclient.so (Steam runtime library).
	if _, err := os.Stat(filepath.Join(dir, "ubuntu12_32", "steamclient.so")); err == nil {
		return true
	}
	return false
}
