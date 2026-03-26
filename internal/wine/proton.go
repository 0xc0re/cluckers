//go:build linux

package wine

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/0xc0re/cluckers/internal/ui"
)

// FindProton locates a Proton-GE installation. Checks in order:
// 1. Bundled: CLUCKERS_BUNDLED_PROTON env var (set by AppImage AppRun)
// 2. Config override: configOverride param (from config file or CLI flag)
// 3. System scan: existing FindProtonGE() search of known directories
// Returns a UserError with per-distro install instructions if nothing found.
func FindProton(configOverride string) (*ProtonGEInstall, error) {
	return findProton(configOverride, userHome())
}

// findProton is the internal implementation of FindProton, accepting home for testability.
func findProton(configOverride string, home string) (*ProtonGEInstall, error) {
	// 1. Bundled Proton-GE (AppImage mode).
	if bundled := os.Getenv("CLUCKERS_BUNDLED_PROTON"); bundled != "" {
		protonScript := filepath.Join(bundled, "proton")
		if _, err := os.Stat(protonScript); err == nil {
			return &ProtonGEInstall{
				WinePath:  filepath.Join(bundled, "files", "bin", "wine64"),
				ProtonDir: bundled,
			}, nil
		}
		// Bundled Proton-GE declared but proton script not found -- warn and continue.
		ui.Warn("Bundled Proton-GE not found at " + bundled + ", searching system...")
	}

	// 2. Config override (wine_path or proton_path setting).
	if configOverride != "" {
		install, err := resolveConfigOverride(configOverride)
		if err == nil {
			return install, nil
		}
		// Config override invalid -- fall through to system scan.
		ui.Warn("Configured Proton path not valid: " + configOverride + ", searching system...")
	}

	// 3. System scan (existing FindProtonGE).
	installs := FindProtonGE(home)
	if len(installs) > 0 {
		return &installs[0], nil
	}

	// 4. Nothing found -- return helpful error.
	return nil, &ui.UserError{
		Message:    "Proton-GE not found. Proton-GE is required to run Realm Royale.",
		Suggestion: ProtonInstallInstructions(effectiveDistro()),
	}
}

// resolveConfigOverride resolves a config override path to a ProtonGEInstall.
// Handles two forms:
// - wine64 path: /path/to/GE-Proton10-1/files/bin/wine64 -> derives ProtonDir
// - directory path: /path/to/GE-Proton10-1 (with proton script) -> uses directly
func resolveConfigOverride(configOverride string) (*ProtonGEInstall, error) {
	// Check if the path points to a wine64 binary (contains "files/bin/wine64").
	if strings.Contains(configOverride, filepath.Join("files", "bin", "wine64")) {
		protonDir := ProtonBaseDir(configOverride)
		protonScript := filepath.Join(protonDir, "proton")
		if _, err := os.Stat(protonScript); err == nil {
			return &ProtonGEInstall{
				WinePath:  configOverride,
				ProtonDir: protonDir,
			}, nil
		}
	}

	// Check if it's a directory containing a proton script.
	protonScript := filepath.Join(configOverride, "proton")
	if _, err := os.Stat(protonScript); err == nil {
		return &ProtonGEInstall{
			WinePath:  filepath.Join(configOverride, "files", "bin", "wine64"),
			ProtonDir: configOverride,
		}, nil
	}

	return nil, &ui.UserError{
		Message:    "Configured Proton path not valid: " + configOverride,
		Detail:     "Expected a Proton-GE directory with a 'proton' script, or a path to files/bin/wine64",
		Suggestion: "Check your wine_path setting in ~/.cluckers/config/settings.toml",
	}
}

// ProtonScript returns the path to the proton Python script.
func (p ProtonGEInstall) ProtonScript() string {
	return filepath.Join(p.ProtonDir, "proton")
}

// DisplayVersion returns a human-readable version string like "GE-Proton10-1".
func (p ProtonGEInstall) DisplayVersion() string {
	return filepath.Base(p.ProtonDir)
}

// knownDistros lists distro IDs that have specific install instructions.
var knownDistros = map[string]bool{
	"arch": true, "steamos": true,
	"ubuntu": true, "debian": true, "linuxmint": true, "pop": true,
	"fedora": true, "bazzite": true,
	"nixos": true,
}

// effectiveDistro returns the best distro ID for install instructions.
// Checks ID first, then falls back to the first known base in ID_LIKE.
func effectiveDistro() string {
	id := DetectDistro()
	if knownDistros[id] {
		return id
	}
	idLike := DetectDistroLike()
	for _, base := range strings.Fields(idLike) {
		if knownDistros[base] {
			return base
		}
	}
	return id
}

// ProtonInstallInstructions returns per-distro Proton-GE install instructions.
func ProtonInstallInstructions(distro string) string {
	switch distro {
	case "arch", "steamos":
		return "Install Proton-GE via ProtonUp-Qt, or from the AUR:\n" +
			"  yay -S proton-ge-custom-bin\n" +
			"  paru -S proton-ge-custom-bin\n" +
			"  pacman -S protonup-qt  # then use ProtonUp-Qt to install"
	case "ubuntu", "debian", "linuxmint", "pop":
		return "Install Proton-GE via ProtonUp-Qt:\n" +
			"  Download from https://davidotek.github.io/protonup-qt/ or Flathub\n" +
			"  flatpak install flathub net.davidotek.pupgui2"
	case "fedora", "bazzite":
		return "Install Proton-GE via ProtonUp-Qt:\n" +
			"  sudo dnf install protonup-qt\n" +
			"  Or: flatpak install flathub net.davidotek.pupgui2"
	case "nixos":
		return "Install Proton-GE declaratively in your NixOS configuration:\n" +
			"  programs.steam.extraCompatPackages = with pkgs; [ proton-ge-bin ];\n" +
			"  Or install via ProtonUp-Qt into ~/.steam/root/compatibilitytools.d/\n\n" +
			"Note: Cluckers requires steam-run to launch Proton on NixOS.\n" +
			"  Ensure programs.steam.enable = true; is set (provides steam-run)."
	default:
		return "Install Proton-GE from https://github.com/GloriousEggroll/proton-ge-custom\n" +
			"  Or use ProtonUp-Qt: https://davidotek.github.io/protonup-qt/"
	}
}
