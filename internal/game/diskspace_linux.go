package game

import (
	"fmt"
	"syscall"

	"github.com/0xc0re/cluckers/internal/ui"
)

// checkDiskSpace verifies that the filesystem containing dir has at least
// requiredBytes of free space.
func checkDiskSpace(dir string, requiredBytes int64) error {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		// If we can't check, proceed anyway (non-critical).
		return nil
	}

	availableBytes := int64(stat.Bavail) * int64(stat.Bsize)
	if availableBytes < requiredBytes {
		requiredGB := float64(requiredBytes) / (1024 * 1024 * 1024)
		availableGB := float64(availableBytes) / (1024 * 1024 * 1024)
		return &ui.UserError{
			Message:    fmt.Sprintf("Not enough disk space. Need at least %.1f GB free, have %.1f GB.", requiredGB, availableGB),
			Suggestion: "Free up disk space or configure a different game directory with `cluckers config set game_dir /path/to/dir`.",
		}
	}

	return nil
}
