package game

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/0xc0re/cluckers/internal/ui"
)

// checkDiskSpace verifies that the filesystem containing dir has at least
// requiredBytes of free space. Uses GetDiskFreeSpaceExW on Windows.
func checkDiskSpace(dir string, requiredBytes int64) error {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx := kernel32.NewProc("GetDiskFreeSpaceExW")

	dirPtr, err := syscall.UTF16PtrFromString(dir)
	if err != nil {
		// If we can't check, proceed anyway (non-critical).
		return nil
	}

	var freeBytesAvailable uint64
	ret, _, _ := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(dirPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		0,
		0,
	)
	if ret == 0 {
		// If we can't check, proceed anyway (non-critical).
		return nil
	}

	if int64(freeBytesAvailable) < requiredBytes {
		requiredGB := float64(requiredBytes) / (1024 * 1024 * 1024)
		availableGB := float64(freeBytesAvailable) / (1024 * 1024 * 1024)
		return &ui.UserError{
			Message:    fmt.Sprintf("Not enough disk space. Need at least %.1f GB free, have %.1f GB.", requiredGB, availableGB),
			Suggestion: "Free up disk space or configure a different game directory.",
		}
	}

	return nil
}
