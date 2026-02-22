package launch

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cstory/cluckers/internal/ui"
)

// protonGEPaths lists known Proton-GE wine64 binary locations, checked in order.
var protonGEPaths = []string{
	// System-wide (e.g., AUR proton-ge-custom-bin)
	"/usr/share/steam/compatibilitytools.d/proton-ge-custom/files/bin/wine64",
	// User install (~/.steam)
	filepath.Join(userHome(), ".steam", "root", "compatibilitytools.d", "proton-ge-custom", "files", "bin", "wine64"),
	// User install (~/.local/share/Steam)
	filepath.Join(userHome(), ".local", "share", "Steam", "compatibilitytools.d", "proton-ge-custom", "files", "bin", "wine64"),
}

// userHome returns the user's home directory, falling back to /tmp if unavailable.
func userHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp"
	}
	return home
}

// FindWine locates a Wine binary. Checks in order:
// 1. configOverride (from config file or CLI flag)
// 2. Known Proton-GE paths
// 3. System wine via PATH
// Returns a UserError with per-distro install instructions if nothing found.
func FindWine(configOverride string) (string, error) {
	// If user explicitly configured a Wine path, use it.
	if configOverride != "" {
		if _, err := os.Stat(configOverride); err != nil {
			return "", &ui.UserError{
				Message:    "Configured Wine binary not found: " + configOverride,
				Detail:     err.Error(),
				Suggestion: "Check your wine_path setting in ~/.cluckers/config/settings.toml",
			}
		}
		return configOverride, nil
	}

	// Check Proton-GE paths.
	for _, p := range protonGEPaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// Check system wine in PATH.
	if p, err := exec.LookPath("wine"); err == nil {
		return p, nil
	}

	// Nothing found -- return helpful error.
	distro := DetectDistro()
	instructions := WineInstallInstructions(distro)
	return "", &ui.UserError{
		Message:    "Wine not found. Wine or Proton-GE is required to run Realm Royale.",
		Suggestion: instructions,
	}
}

// IsProtonGE returns true if the Wine binary path indicates Proton-GE.
func IsProtonGE(winePath string) bool {
	return strings.Contains(winePath, "proton-ge")
}

// LinuxToWinePath converts a Linux absolute path to a Wine Z: drive path.
// Non-absolute paths are returned as-is.
func LinuxToWinePath(path string) string {
	if strings.HasPrefix(path, "/") {
		return "Z:" + strings.ReplaceAll(path, "/", "\\")
	}
	return path
}

// DetectDistro reads /etc/os-release and returns the ID field value.
// Returns "unknown" if the file cannot be read or the ID field is missing.
func DetectDistro() string {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return "unknown"
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			return strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
		}
	}
	return "unknown"
}

// WineInstallInstructions returns per-distro Wine install commands.
func WineInstallInstructions(distro string) string {
	switch distro {
	case "arch", "steamos":
		return "Install Wine: sudo pacman -S wine\n  Or install Proton-GE via ProtonUp-Qt for best compatibility."
	case "ubuntu", "debian", "linuxmint", "pop":
		return "Install Wine: sudo apt install wine\n  Or install Proton-GE via ProtonUp-Qt for best compatibility."
	case "fedora":
		return "Install Wine: sudo dnf install wine\n  Or install Proton-GE via ProtonUp-Qt for best compatibility."
	default:
		return "Install Wine or Proton-GE (https://github.com/GloriousEggroll/proton-ge-custom)"
	}
}
