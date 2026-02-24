//go:build windows

package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xc0re/cluckers/internal/ui"
)

// runSteamAdd prints instructions for adding Cluckers to Steam on Windows.
// Steam cannot properly add .bat files as non-Steam games, so we instruct the
// user to add cluckers.exe directly with "launch" as the launch argument.
func runSteamAdd() error {
	// Resolve the cluckers binary path.
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("could not resolve executable path: %w", err)
	}

	exeDir := filepath.Dir(exePath)

	ui.Success("Steam integration ready")
	fmt.Println()
	fmt.Println("  Executable: " + exePath)
	fmt.Println()
	fmt.Println("  To add Cluckers to Steam:")
	fmt.Println()
	fmt.Println("  1. Open Steam")
	fmt.Println("  2. Go to Games > Add a Non-Steam Game to My Library")
	fmt.Println("  3. Click Browse, change file filter to \"All Files (*.*)\"")
	fmt.Println("  4. Navigate to: " + exeDir)
	fmt.Println("  5. Select \"cluckers.exe\" and click Open")
	fmt.Println("  6. Click \"Add Selected Programs\"")
	fmt.Println("  7. Right-click \"cluckers\" in your library > Properties")
	fmt.Println("  8. Set LAUNCH OPTIONS to: launch")
	fmt.Println("  9. (Optional) Rename to \"Realm Royale\" in Properties")
	fmt.Println()
	return nil
}
