package config

import (
	"os"
	"path/filepath"
)

// DataDir returns the base data directory for Cluckers.
// Uses CLUCKERS_HOME env var if set, otherwise %LOCALAPPDATA%\cluckers.
// Falls back to ~/.cluckers if LOCALAPPDATA is not set.
func DataDir() string {
	if env := os.Getenv("CLUCKERS_HOME"); env != "" {
		return env
	}
	if appdata := os.Getenv("LOCALAPPDATA"); appdata != "" {
		return filepath.Join(appdata, "cluckers")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("C:\\", "cluckers")
	}
	return filepath.Join(home, ".cluckers")
}
