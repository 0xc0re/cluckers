//go:build linux

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/0xc0re/cluckers/internal/wine"
)

// runSteamAdd creates a .desktop file and prints instructions for Steam integration.
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

	// Determine .desktop file location.
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %w", err)
	}

	appsDir := filepath.Join(home, ".local", "share", "applications")
	if err := os.MkdirAll(appsDir, 0755); err != nil {
		return fmt.Errorf("could not create applications directory: %w", err)
	}

	desktopPath := filepath.Join(appsDir, "cluckers.desktop")

	// Build the .desktop file content.
	content := strings.Join([]string{
		"[Desktop Entry]",
		"Name=Realm Royale (Cluckers)",
		"Comment=Launch Realm Royale via Cluckers Central",
		"Exec=" + exePath + " launch",
		"Type=Application",
		"Categories=Game;",
		"Terminal=false",
		"",
	}, "\n")

	// Write the .desktop file (idempotent -- overwrites if it exists).
	if err := os.WriteFile(desktopPath, []byte(content), 0755); err != nil {
		return fmt.Errorf("could not write .desktop file: %w", err)
	}

	ui.Success("Created " + desktopPath)
	fmt.Println()

	// Detect Steam Deck for tailored instructions.
	isSteamDeck := wine.IsSteamDeck()

	fmt.Println("  Next steps:")
	fmt.Println()
	fmt.Println("  1. Open Steam in Desktop Mode")
	fmt.Println("  2. Go to Games > Add a Non-Steam Game to My Library")
	fmt.Println("  3. Select \"Realm Royale (Cluckers)\" from the list")
	fmt.Println("  4. Click \"Add Selected Programs\"")

	if isSteamDeck {
		fmt.Println()
		fmt.Println("  Steam Deck:")
		fmt.Println("  5. Switch back to Game Mode")
		fmt.Println("  6. The game will appear in the Non-Steam section of your library")
	}

	fmt.Println()
	return nil
}
