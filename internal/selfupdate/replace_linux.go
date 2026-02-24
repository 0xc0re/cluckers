package selfupdate

import (
	"os"

	"github.com/0xc0re/cluckers/internal/ui"
)

// replaceBinary replaces the running executable with the new binary using an
// atomic os.Rename. On Linux, a running binary can be overwritten (the kernel
// keeps the old inode open until the process exits).
func replaceBinary(tmpBin, execPath string) error {
	// Set executable permissions.
	if err := os.Chmod(tmpBin, 0755); err != nil {
		os.Remove(tmpBin)
		return &ui.UserError{
			Message:    "Failed to set permissions on new binary.",
			Detail:     err.Error(),
			Suggestion: "Check filesystem permissions.",
			Err:        err,
		}
	}

	// Atomic rename: replace the current executable.
	if err := os.Rename(tmpBin, execPath); err != nil {
		os.Remove(tmpBin)
		return &ui.UserError{
			Message:    "Failed to replace the current binary.",
			Detail:     err.Error(),
			Suggestion: "Ensure you have write permission to the directory containing the binary.",
			Err:        err,
		}
	}

	return nil
}
