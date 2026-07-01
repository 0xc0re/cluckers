//go:build linux

package game

import "os"

// prepareTarget ensures the target file is writable before a sync overwrites it.
// On Linux, open() with O_WRONLY checks file permission bits and returns EACCES
// for files with mode 0444 (no owner write bit), even for the file owner.
// Previously-installed game files may have read-only permissions, so we clear
// the read-only bit before overwriting them during a re-sync.
func prepareTarget(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return // File doesn't exist yet, nothing to do.
	}
	if info.Mode().Perm()&0200 == 0 {
		_ = os.Chmod(path, 0644)
	}
}
