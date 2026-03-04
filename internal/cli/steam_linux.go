//go:build linux

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xc0re/cluckers/internal/auth"
	"github.com/0xc0re/cluckers/internal/config"
	"github.com/0xc0re/cluckers/internal/launch"
	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/0xc0re/cluckers/internal/wine"
)

// runSteamAdd creates a Steam shortcut for Cluckers.
// On Steam Deck, it automates shortcuts.vdf writing so the user doesn't need
// to manually add a non-Steam game.
// On desktop Linux, it creates a .desktop file and prints manual Steam instructions.
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

	if wine.IsSteamDeck() {
		return runSteamAddDeck(exePath)
	}
	return runSteamAddDesktop(exePath)
}

// runSteamAddDeck automates Steam shortcut creation on Steam Deck by writing
// directly to shortcuts.vdf. Falls back to desktop-style manual instructions
// if Steam is not found.
func runSteamAddDeck(exePath string) error {
	steamDir := wine.FindSteamInstall()
	if steamDir == "" {
		ui.Warn("Steam installation not found, falling back to manual setup")
		return runSteamAddDesktop(exePath)
	}

	// Extract shm_launcher.exe to bin dir.
	shmDest := filepath.Join(config.BinDir(), "shm_launcher.exe")
	if err := config.EnsureDir(config.BinDir()); err != nil {
		return fmt.Errorf("creating bin directory: %w", err)
	}
	if err := launch.ExtractSHMLauncherTo(shmDest); err != nil {
		return fmt.Errorf("extracting shm_launcher.exe: %w", err)
	}

	// Find shortcuts.vdf in userdata.
	shortcutsPath, err := findShortcutsVDF(steamDir)
	if err != nil {
		return err
	}

	// Read existing shortcuts.vdf (may not exist).
	var existingData []byte
	if data, readErr := os.ReadFile(shortcutsPath); readErr == nil {
		existingData = data
	}

	// Check if a Cluckers shortcut already exists.
	if existingData != nil {
		if appID := launch.FindCluckersAppID(existingData); appID != 0 {
			ui.Success(fmt.Sprintf("Shortcut already exists (app ID: %d)", appID))
			fmt.Println()
			printDeckPostSetupInstructions()
			return nil
		}
	}

	// Back up existing shortcuts.vdf before modification.
	if existingData != nil {
		backupPath := shortcutsPath + ".cluckers-backup"
		if err := os.WriteFile(backupPath, existingData, 0644); err != nil {
			ui.Warn(fmt.Sprintf("Could not create backup at %s: %s", backupPath, err))
		}
	}

	// Build the shortcut entry.
	s := &launch.Shortcut{
		AppName:       "Realm Royale (Cluckers)",
		Exe:           fmt.Sprintf(`"%s"`, shmDest),
		StartDir:      fmt.Sprintf(`"%s"`, config.BinDir()),
		LaunchOptions: fmt.Sprintf("%s prep && %%command%%", exePath),
	}

	// Write the new shortcut to VDF.
	newData, err := launch.AddShortcutToVDF(existingData, s)
	if err != nil {
		return &ui.UserError{
			Message:    "Could not create shortcut entry",
			Detail:     err.Error(),
			Suggestion: "Try adding the shortcut manually in Steam.",
		}
	}

	if err := os.WriteFile(shortcutsPath, newData, 0644); err != nil {
		return &ui.UserError{
			Message:    "Could not write shortcuts.vdf",
			Detail:     err.Error(),
			Suggestion: fmt.Sprintf("Backup saved at %s.cluckers-backup. Try adding the shortcut manually in Steam.", shortcutsPath),
		}
	}

	ui.Success("Shortcut added to Steam!")
	fmt.Println()
	printDeckPostSetupInstructions()
	return nil
}

// findShortcutsVDF locates the shortcuts.vdf file in Steam's userdata directory.
// If multiple userdata dirs exist, uses the first one found.
// Creates the path structure if no shortcuts.vdf exists yet.
func findShortcutsVDF(steamDir string) (string, error) {
	pattern := filepath.Join(steamDir, "userdata", "*", "config", "shortcuts.vdf")
	matches, _ := filepath.Glob(pattern)
	if len(matches) > 0 {
		return matches[0], nil
	}

	// No shortcuts.vdf found. Look for any userdata directory.
	userdataPattern := filepath.Join(steamDir, "userdata", "*")
	userdataDirs, _ := filepath.Glob(userdataPattern)

	var configDir string
	if len(userdataDirs) > 0 {
		configDir = filepath.Join(userdataDirs[0], "config")
	} else {
		return "", &ui.UserError{
			Message:    "No Steam userdata directory found",
			Detail:     fmt.Sprintf("Searched: %s/userdata/*/config/shortcuts.vdf", steamDir),
			Suggestion: "Make sure you have logged into Steam at least once.",
		}
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("creating Steam config directory: %w", err)
	}

	return filepath.Join(configDir, "shortcuts.vdf"), nil
}

// printDeckPostSetupInstructions prints the one-time setup steps for Steam Deck.
func printDeckPostSetupInstructions() {
	fmt.Println("  Required one-time setup:")
	fmt.Println()
	fmt.Println("  1. Restart Steam (or switch to Desktop Mode and back)")
	fmt.Println("  2. Find \"Realm Royale (Cluckers)\" in your Non-Steam games")
	fmt.Println("  3. Right-click > Properties > Compatibility")
	fmt.Println("  4. Check \"Force the use of a specific Steam Play compatibility tool\"")
	fmt.Println("  5. Select your Proton-GE version")
	fmt.Println()
	fmt.Println("  After setup, launch from Steam or run: cluckers launch")
	fmt.Println()
}

// runSteamAddDesktop creates a .desktop file and prints manual Steam instructions
// for desktop Linux users.
func runSteamAddDesktop(exePath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %w", err)
	}

	appsDir := filepath.Join(home, ".local", "share", "applications")
	if err := os.MkdirAll(appsDir, 0755); err != nil {
		return fmt.Errorf("could not create applications directory: %w", err)
	}

	desktopPath := filepath.Join(appsDir, "cluckers.desktop")

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

	if err := os.WriteFile(desktopPath, []byte(content), 0755); err != nil {
		return fmt.Errorf("could not write .desktop file: %w", err)
	}

	ui.Success("Created " + desktopPath)
	fmt.Println()

	// Warn if no credentials saved — desktop shortcut has Terminal=false,
	// so login prompts won't work when launched from Steam.
	creds, _ := auth.LoadCredentials()
	if creds == nil {
		ui.Warn("No saved credentials found. Run `cluckers login` in a terminal before launching from Steam.")
		fmt.Println()
	}

	fmt.Println("  Next steps:")
	fmt.Println()
	fmt.Println("  1. Open Steam in Desktop Mode")
	fmt.Println("  2. Go to Games > Add a Non-Steam Game to My Library")
	fmt.Println("  3. Select \"Realm Royale (Cluckers)\" from the list")
	fmt.Println("  4. Click \"Add Selected Programs\"")
	fmt.Println()

	return nil
}
