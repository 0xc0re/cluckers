package selfupdate

import (
	"fmt"
	"os"

	"github.com/0xc0re/cluckers/internal/ui"
)

// replaceBinary replaces the running executable with the new binary. On Windows,
// a running .exe cannot be overwritten (the OS holds a file lock). However,
// Windows allows renaming a running .exe. The strategy is:
//  1. Rename the running exe out of the way (execPath -> execPath.old)
//  2. Rename the new binary into place (tmpBin -> execPath)
//  3. Attempt cleanup of the .old file (will likely fail since it is still running)
func replaceBinary(tmpBin, execPath string) error {
	oldPath := execPath + ".old"

	// Remove any leftover .old file from a previous update.
	os.Remove(oldPath)

	// Rename the running exe out of the way. Windows allows renaming a
	// running executable, just not writing to or deleting it.
	if err := os.Rename(execPath, oldPath); err != nil {
		os.Remove(tmpBin)
		return &ui.UserError{
			Message:    "Failed to rename the running binary out of the way.",
			Detail:     err.Error(),
			Suggestion: "Ensure you have write permission to the directory containing the binary.",
			Err:        err,
		}
	}

	// Move the new binary into place.
	if err := os.Rename(tmpBin, execPath); err != nil {
		// Attempt rollback: move the old binary back.
		if rbErr := os.Rename(oldPath, execPath); rbErr != nil {
			// Rollback also failed; report both errors.
			return &ui.UserError{
				Message:    "Failed to place new binary and rollback also failed.",
				Detail:     fmt.Sprintf("rename new: %v; rollback: %v", err, rbErr),
				Suggestion: fmt.Sprintf("The old binary is at %s -- rename it back manually.", oldPath),
				Err:        err,
			}
		}
		os.Remove(tmpBin)
		return &ui.UserError{
			Message:    "Failed to place new binary (rolled back to previous version).",
			Detail:     err.Error(),
			Suggestion: "Ensure you have write permission to the directory containing the binary.",
			Err:        err,
		}
	}

	// Attempt cleanup of the .old file. This will likely fail because the
	// process is still running from that executable, which is fine.
	if err := os.Remove(oldPath); err != nil {
		// Expected on Windows -- the .old file will be cleaned up on the
		// next self-update run via CleanupOldBinary().
		fmt.Printf("Note: %s will be cleaned up on next run.\n", oldPath)
	}

	return nil
}
