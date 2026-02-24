//go:build windows

package launch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xc0re/cluckers/internal/ui"
)

// platformSteps returns Windows-specific pipeline steps.
// On Windows, no Wine detection, prefix creation, or prefix verification is needed.
func platformSteps(_ *LaunchState) []Step {
	return []Step{}
}

// platformPostSteps returns Windows-specific post-download steps.
// On Windows, we patch display settings for borderless fullscreen and make INI files writable.
func platformPostSteps(_ *LaunchState) []Step {
	return []Step{
		{Name: "Configuring display", Fn: stepWindowsDisplayConfig},
	}
}

// stepWindowsDisplayConfig patches RealmSystemSettings.ini for borderless fullscreen
// on Windows. The game zip ships with Fullscreen=false and FullscreenWindowed=false,
// causing the game to launch in a small window. This step sets FullscreenWindowed=True
// (borderless windowed mode automatically uses the desktop resolution, so ResX/ResY
// are left unchanged).
//
// Also makes all .ini files in the Config directory writable (0644) so the game
// can persist user settings changes in-game.
//
// Idempotent: skips write if already patched.
func stepWindowsDisplayConfig(_ context.Context, state *LaunchState) error {
	iniPath := filepath.Join(state.GameDir, "Realm-Royale", "RealmGame", "Config", "RealmSystemSettings.ini")

	data, err := os.ReadFile(iniPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Game not yet downloaded -- nothing to patch.
			return nil
		}
		return fmt.Errorf("reading RealmSystemSettings.ini: %w", err)
	}

	original := string(data)
	output := original

	// Patch for borderless windowed mode.
	output = strings.Replace(output, "FullscreenWindowed=false", "FullscreenWindowed=True", 1)
	// Disable exclusive fullscreen (borderless windowed is preferred).
	output = strings.Replace(output, "Fullscreen=True", "Fullscreen=false", 1)

	if output != original {
		// Ensure file is writable before writing -- game zip extracts as 0444.
		ensureWritableWin(iniPath)

		if err := os.WriteFile(iniPath, []byte(output), 0644); err != nil {
			return fmt.Errorf("writing RealmSystemSettings.ini: %w", err)
		}

		ui.Verbose("Patched display settings for borderless fullscreen", state.Config.Verbose)
	} else {
		ui.Verbose("Display settings already configured, skipping", state.Config.Verbose)
	}

	// Make all INI files writable so the game can save user preferences.
	configDir := filepath.Join(state.GameDir, "Realm-Royale", "RealmGame", "Config")
	makeINIsWritableWin(configDir)

	return nil
}

// ensureWritableWin sets a file to 0644 if it exists and is not already writable.
func ensureWritableWin(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if info.Mode().Perm()&0200 == 0 {
		os.Chmod(path, 0644)
	}
}

// makeINIsWritableWin ensures all .ini files in the config directory are writable (0644).
// The game zip extracts files as read-only, which prevents the game from saving
// user preferences (graphics options, input settings, etc.).
func makeINIsWritableWin(configDir string) {
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".ini") {
			ensureWritableWin(filepath.Join(configDir, entry.Name()))
		}
	}
}
