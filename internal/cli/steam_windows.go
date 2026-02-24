//go:build windows

package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xc0re/cluckers/internal/ui"
)

// runSteamAdd creates a .bat launcher script and prints instructions for adding
// Cluckers to Steam on Windows.
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

	// Create a .bat wrapper next to the executable for easier Steam integration.
	exeDir := filepath.Dir(exePath)
	batPath := filepath.Join(exeDir, "cluckers-launch.bat")
	batContent := fmt.Sprintf("@echo off\r\n\"%s\" launch\r\n", exePath)

	if err := os.WriteFile(batPath, []byte(batContent), 0755); err != nil {
		return fmt.Errorf("could not write .bat file: %w", err)
	}

	ui.Success("Created " + batPath)
	fmt.Println()
	fmt.Println("  Next steps:")
	fmt.Println()
	fmt.Println("  1. Open Steam")
	fmt.Println("  2. Go to Games > Add a Non-Steam Game to My Library")
	fmt.Println("  3. Click Browse")
	fmt.Println("  4. Navigate to: " + exeDir)
	fmt.Println("  5. Select \"cluckers-launch.bat\" and click Open")
	fmt.Println("  6. Click \"Add Selected Programs\"")
	fmt.Println()
	return nil
}
