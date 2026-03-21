//go:build gui && windows

package gui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xc0re/cluckers/internal/config"
	"golang.org/x/sys/windows"
)

// tryLock acquires an exclusive file lock to prevent multiple GUI instances.
// Returns a cleanup function on success, or an error if another instance holds the lock.
func tryLock() (func(), error) {
	lockPath := filepath.Join(config.DataDir(), "cluckers.lock")
	_ = config.EnsureDir(filepath.Dir(lockPath))

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	ol := new(windows.Overlapped)
	err = windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0, 1, 0, ol,
	)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("another instance of Cluckers is already running")
	}

	cleanup := func() {
		windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, ol)
		f.Close()
		os.Remove(lockPath)
	}
	return cleanup, nil
}
