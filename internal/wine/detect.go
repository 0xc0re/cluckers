package wine

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/0xc0re/cluckers/internal/ui"
)

// protonVersionRe extracts the numeric version from GE-Proton directory names.
// Matches patterns like "GE-Proton9-22", "GE-Proton10-1".
var protonVersionRe = regexp.MustCompile(`GE-Proton(\d+)-(\d+)`)

// ProtonGEInstall represents a discovered Proton-GE installation.
type ProtonGEInstall struct {
	WinePath  string // Full path to the wine64 binary.
	ProtonDir string // Root of the Proton-GE installation (contains files/ and default_pfx).
}

// Version returns a sortable version string extracted from the Proton directory name.
// For versioned installs (GE-ProtonX-Y), returns a zero-padded string like "00009-00022".
// For non-versioned installs (proton-ge-custom), returns "00000-00000" (lowest priority).
func (p ProtonGEInstall) Version() string {
	name := filepath.Base(p.ProtonDir)
	m := protonVersionRe.FindStringSubmatch(name)
	if m == nil {
		// Non-versioned (system package): lowest priority.
		return "00000-00000"
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	return padVersion(major, minor)
}

// padVersion formats major-minor as zero-padded sortable string.
func padVersion(major, minor int) string {
	return strings.Join([]string{
		padInt(major),
		padInt(minor),
	}, "-")
}

func padInt(n int) string {
	s := strconv.Itoa(n)
	for len(s) < 5 {
		s = "0" + s
	}
	return s
}

// protonSearchDirs returns the directories to scan for Proton-GE installations.
func protonSearchDirs(home string) []string {
	return []string{
		// System-wide (e.g., AUR proton-ge-custom-bin).
		"/usr/share/steam/compatibilitytools.d",
		// Native Steam user install (~/.steam/root).
		filepath.Join(home, ".steam", "root", "compatibilitytools.d"),
		// Native Steam user install (~/.steam/steam).
		filepath.Join(home, ".steam", "steam", "compatibilitytools.d"),
		// XDG data dir Steam.
		filepath.Join(home, ".local", "share", "Steam", "compatibilitytools.d"),
		// Flatpak Steam.
		filepath.Join(home, ".var", "app", "com.valvesoftware.Steam",
			"data", "Steam", "compatibilitytools.d"),
		// Snap Steam.
		filepath.Join(home, "snap", "steam", "common", ".steam",
			"steam", "compatibilitytools.d"),
		// ProtonUp-Qt Flatpak — when installed via Flatpak Discover on Steam Deck,
		// ProtonUp-Qt writes to its own data dir which Steam reads via symlink.
		filepath.Join(home, ".var", "app", "net.davidotek.pupgui2",
			"data", "Steam", "compatibilitytools.d"),
		// Steam Deck native Steam internal path (sometimes differs from symlink).
		filepath.Join(home, ".local", "share", "Steam", "steamapps", "common",
			"Proton - GE", "compatibilitytools.d"),
	}
}

// symlinkResolvedDirs returns additional search directories by resolving
// ~/.steam/root and ~/.steam/steam symlinks. On Steam Deck, these are often
// symlinks to ~/.local/share/Steam but may point elsewhere.
func symlinkResolvedDirs(home string) []string {
	symlinks := []string{
		filepath.Join(home, ".steam", "root"),
		filepath.Join(home, ".steam", "steam"),
	}

	var dirs []string
	for _, link := range symlinks {
		resolved, err := filepath.EvalSymlinks(link)
		if err != nil {
			continue
		}
		// Only add if it resolved to something different from the original.
		if resolved != link {
			dirs = append(dirs, filepath.Join(resolved, "compatibilitytools.d"))
		}
	}
	return dirs
}

// FindProtonGE scans standard directories for Proton-GE installations and returns
// them sorted by version descending (newest first).
func FindProtonGE(home string) []ProtonGEInstall {
	var installs []ProtonGEInstall
	seen := make(map[string]bool) // Deduplicate by WinePath (symlinked dirs may overlap).

	// Combine static search dirs with symlink-resolved dirs.
	searchDirs := protonSearchDirs(home)
	searchDirs = append(searchDirs, symlinkResolvedDirs(home)...)

	for _, dir := range searchDirs {
		// Check system package: proton-ge-custom/files/bin/wine64
		sysPath := filepath.Join(dir, "proton-ge-custom", "files", "bin", "wine64")
		if _, err := os.Stat(sysPath); err == nil {
			real := resolveReal(sysPath)
			if !seen[real] {
				seen[real] = true
				installs = append(installs, ProtonGEInstall{
					WinePath:  sysPath,
					ProtonDir: filepath.Join(dir, "proton-ge-custom"),
				})
			}
		}

		// Check ProtonUp-Qt versioned: GE-Proton*/files/bin/wine64
		pattern := filepath.Join(dir, "GE-Proton*", "files", "bin", "wine64")
		matches, _ := filepath.Glob(pattern)
		for _, m := range matches {
			real := resolveReal(m)
			if !seen[real] {
				seen[real] = true
				installs = append(installs, ProtonGEInstall{
					WinePath:  m,
					ProtonDir: filepath.Dir(filepath.Dir(filepath.Dir(m))),
				})
			}
		}
	}

	// Sort by version descending (newest first).
	sort.Slice(installs, func(i, j int) bool {
		return installs[i].Version() > installs[j].Version()
	})

	return installs
}

// resolveReal attempts to resolve a path to its real (symlink-resolved) form.
// Falls back to the original path on error.
func resolveReal(p string) string {
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		return p
	}
	return r
}

// ProtonBaseDir derives the Proton base directory (containing files/ and default_pfx)
// from a wine64 binary path by going up 3 parent directories.
// Example: /path/to/GE-Proton10-1/files/bin/wine64 -> /path/to/GE-Proton10-1
func ProtonBaseDir(winePath string) string {
	return filepath.Dir(filepath.Dir(filepath.Dir(winePath)))
}

// FindWine locates a Wine binary. Checks in order:
// 1. configOverride (from config file or CLI flag)
// 2. Proton-GE installations (newest version first)
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

	// Check Proton-GE installations (sorted newest first).
	home := userHome()
	installs := FindProtonGE(home)
	if len(installs) > 0 {
		return installs[0].WinePath, nil
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
// Matches both proton-ge-custom (system package) and GE-Proton* (versioned) paths.
func IsProtonGE(winePath string) bool {
	return strings.Contains(winePath, "proton-ge") || strings.Contains(winePath, "GE-Proton")
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

// userHome returns the user's home directory, falling back to /tmp if unavailable.
func userHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp"
	}
	return home
}
