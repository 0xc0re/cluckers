//go:build linux

package game

import "os"

// prepareTarget ensures the target file is writable before extraction overwrites it.
// On Linux, open() with O_WRONLY checks file permission bits and returns EACCES
// for files with mode 0444 (no owner write bit), even for the file owner.
// Files extracted from zip archives often have read-only permissions, so we must
// clear the read-only bit before overwriting during a re-extraction.
func prepareTarget(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return // File doesn't exist yet, nothing to do.
	}
	if info.Mode().Perm()&0200 == 0 {
		_ = os.Chmod(path, 0644)
	}
}
