//go:build linux

package game

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareTarget_NonexistentFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "does-not-exist.txt")

	// Should not panic on nonexistent file.
	prepareTarget(path)
}

func TestPrepareTarget_ReadOnlyFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "readonly.txt")

	if err := os.WriteFile(path, []byte("data"), 0444); err != nil {
		t.Fatal(err)
	}

	prepareTarget(path)

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	perm := info.Mode().Perm()
	if perm != 0644 {
		t.Errorf("file mode = %o, want %o", perm, 0644)
	}
}

func TestPrepareTarget_WritableFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "writable.txt")

	if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	prepareTarget(path)

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	perm := info.Mode().Perm()
	if perm != 0644 {
		t.Errorf("file mode = %o, want %o", perm, 0644)
	}
}
