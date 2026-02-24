package config

import (
	"os"
	"path/filepath"
)

// DataDir returns the base data directory for Cluckers.
// Uses CLUCKERS_HOME env var if set, otherwise ~/.cluckers.
func DataDir() string {
	if env := os.Getenv("CLUCKERS_HOME"); env != "" {
		return env
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback; should not happen on Linux.
		return filepath.Join("/tmp", ".cluckers")
	}
	return filepath.Join(home, ".cluckers")
}
