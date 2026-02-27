//go:build linux

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xc0re/cluckers/internal/config"
	"github.com/0xc0re/cluckers/internal/launch"
	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/0xc0re/cluckers/internal/wine"
)

// runSteamAdd creates a .desktop file and prints instructions for Steam integration.
// On Steam Deck, the shortcut points to shm_launcher.exe so Steam auto-enables Proton,
// and prints %command% launch options for Steam-managed controller persistence.
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

	// Detect Steam Deck for different shortcut behavior.
	isSteamDeck := wine.IsSteamDeck()

	var execLine string
	if isSteamDeck {
		// On Deck: point to shm_launcher.exe so Steam auto-enables Proton.
		shmDest := filepath.Join(config.BinDir(), "shm_launcher.exe")
		if err := config.EnsureDir(config.BinDir()); err != nil {
			return fmt.Errorf("creating bin directory: %w", err)
		}
		if err := launch.ExtractSHMLauncherTo(shmDest); err != nil {
			return fmt.Errorf("extracting shm_launcher.exe: %w", err)
		}
		execLine = "Exec=" + shmDest
	} else {
		execLine = "Exec=" + exePath + " launch"
	}

	// Build the .desktop file content.
	content := strings.Join([]string{
		"[Desktop Entry]",
		"Name=Realm Royale (Cluckers)",
		"Comment=Launch Realm Royale via Cluckers Central",
		execLine,
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

	if isSteamDeck {
		fmt.Println("  Next steps:")
		fmt.Println()
		fmt.Println("  1. Open Steam in Desktop Mode")
		fmt.Println("  2. Go to Games > Add a Non-Steam Game to My Library")
		fmt.Println("  3. Select \"Realm Royale (Cluckers)\" from the list")
		fmt.Println("  4. Click \"Add Selected Programs\"")
		fmt.Println()
		fmt.Println("  5. Right-click the game in your Library > Properties")
		fmt.Println("  6. Set Launch Options to:")
		fmt.Println()
		fmt.Printf("     %s prep && WINEDLLOVERRIDES=dxgi=n %%command%%\n", exePath)
		fmt.Println()
		fmt.Println("  7. Switch back to Game Mode")
		fmt.Println("  8. The game will appear in the Non-Steam section of your library")
	} else {
		fmt.Println("  Next steps:")
		fmt.Println()
		fmt.Println("  1. Open Steam in Desktop Mode")
		fmt.Println("  2. Go to Games > Add a Non-Steam Game to My Library")
		fmt.Println("  3. Select \"Realm Royale (Cluckers)\" from the list")
		fmt.Println("  4. Click \"Add Selected Programs\"")
	}

	fmt.Println()
	return nil
}
