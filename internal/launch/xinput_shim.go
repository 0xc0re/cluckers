//go:build linux

package launch

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xc0re/cluckers/assets"
)

// DeployXInputShim copies the embedded xinput1_3_cache.dll to the game's
// Win64 binary directory as xinput1_3.dll. This overrides UE3's default
// XInput loading and caches device state across ServerTravel re-enumeration.
//
// Combined with WINEDLLOVERRIDES=xinput1_3=n,b, Proton loads our native DLL
// first. The shim then loads Proton's builtin xinput from the system directory,
// preserving Steam Input IPC while adding a caching layer.
//
// The operation is idempotent: if the destination file already exists and
// has the same size as the embedded binary, the write is skipped.
func DeployXInputShim(gameDir string) error {
	win64Dir := filepath.Join(gameDir, "Binaries", "Win64")
	if _, err := os.Stat(win64Dir); err != nil {
		return fmt.Errorf("game Win64 directory not found: %w", err)
	}

	destPath := filepath.Join(win64Dir, "xinput1_3.dll")

	// Idempotent: skip if file exists and size matches embedded binary.
	info, err := os.Stat(destPath)
	if err == nil && info.Size() == int64(len(assets.XInputCacheDLL)) {
		return nil // Already up to date.
	}

	// Back up any existing xinput1_3.dll (could be a previous version or
	// something from the game itself).
	if err == nil {
		backupPath := destPath + ".bak"
		if backErr := os.Rename(destPath, backupPath); backErr != nil {
			// Non-fatal: proceed with overwrite.
			_ = backErr
		}
	}

	if err := os.WriteFile(destPath, assets.XInputCacheDLL, 0644); err != nil {
		return fmt.Errorf("writing xinput1_3.dll: %w", err)
	}

	return nil
}
