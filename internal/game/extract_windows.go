//go:build windows

package game

import "os"

// prepareTarget ensures the target file is writable before extraction overwrites it.
// On Windows, files extracted from zip with mode 0444 get the read-only attribute,
// which prevents subsequent extractions from overwriting them.
func prepareTarget(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return // File doesn't exist yet, nothing to do.
	}
	if info.Mode().Perm()&0200 == 0 {
		_ = os.Chmod(path, 0644)
	}
}
