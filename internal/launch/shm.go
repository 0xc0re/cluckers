package launch

import (
	"fmt"
	"os"

	"github.com/0xc0re/cluckers/assets"
)

// ExtractSHMLauncher writes the embedded shm_launcher.exe to a temp file and
// returns the path, a cleanup function that removes the file, and any error.
func ExtractSHMLauncher() (path string, cleanup func(), err error) {
	f, err := os.CreateTemp("", "shm_launcher_*.exe")
	if err != nil {
		return "", nil, fmt.Errorf("create temp file for shm_launcher: %w", err)
	}

	if _, err := f.Write(assets.SHMLauncherExe); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", nil, fmt.Errorf("write shm_launcher.exe: %w", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", nil, fmt.Errorf("close shm_launcher temp file: %w", err)
	}

	if err := os.Chmod(f.Name(), 0755); err != nil {
		os.Remove(f.Name())
		return "", nil, fmt.Errorf("chmod shm_launcher temp file: %w", err)
	}

	cleanup = func() {
		os.Remove(f.Name())
	}

	return f.Name(), cleanup, nil
}

// WriteBootstrapFile writes content bootstrap bytes to a temp file and returns
// the path, a cleanup function, and any error. File permissions are set to 0600.
func WriteBootstrapFile(data []byte) (path string, cleanup func(), err error) {
	f, err := os.CreateTemp("", "realm_bootstrap_*.bin")
	if err != nil {
		return "", nil, fmt.Errorf("create temp file for bootstrap: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", nil, fmt.Errorf("write bootstrap data: %w", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", nil, fmt.Errorf("close bootstrap temp file: %w", err)
	}

	if err := os.Chmod(f.Name(), 0600); err != nil {
		os.Remove(f.Name())
		return "", nil, fmt.Errorf("chmod bootstrap temp file: %w", err)
	}

	cleanup = func() {
		os.Remove(f.Name())
	}

	return f.Name(), cleanup, nil
}
