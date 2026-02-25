//go:build linux

package wine

import (
	"os"
	"path/filepath"

	"github.com/0xc0re/cluckers/internal/config"
)

// CompatdataHealthy checks if the Proton compatdata directory looks valid.
// Returns true when compatdata exists and pfx/drive_c is present as a directory.
func CompatdataHealthy(compatdataPath string) bool {
	driveC := filepath.Join(compatdataPath, "pfx", "drive_c")
	info, err := os.Stat(driveC)
	return err == nil && info.IsDir()
}

// CompatdataPath returns the path to the Proton compatdata directory.
// This is where Proton stores its Wine prefix (at <compatdata>/pfx/).
func CompatdataPath() string {
	return filepath.Join(config.DataDir(), "compatdata")
}
