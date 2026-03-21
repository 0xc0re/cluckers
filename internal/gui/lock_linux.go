//go:build gui && linux

package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/0xc0re/cluckers/internal/config"
)

// tryLock acquires an exclusive file lock to prevent multiple GUI instances.
// Returns a cleanup function on success, or an error if another instance holds the lock.
// The lock is automatically released if the process crashes (kernel cleans up flock).
func tryLock() (func(), error) {
	lockPath := filepath.Join(config.DataDir(), "cluckers.lock")
	_ = config.EnsureDir(filepath.Dir(lockPath))

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("another instance of Cluckers is already running")
	}

	cleanup := func() {
		syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
		os.Remove(lockPath)
	}
	return cleanup, nil
}
