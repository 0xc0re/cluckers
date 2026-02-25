//go:build linux

package wine

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCompatdataHealthyValid verifies true when pfx/drive_c exists as directory.
func TestCompatdataHealthyValid(t *testing.T) {
	tmp := t.TempDir()
	compatdata := filepath.Join(tmp, "compatdata")
	os.MkdirAll(filepath.Join(compatdata, "pfx", "drive_c"), 0755)

	if !CompatdataHealthy(compatdata) {
		t.Error("CompatdataHealthy should return true when pfx/drive_c exists as directory")
	}
}

// TestCompatdataHealthyMissing verifies false when compatdata doesn't exist.
func TestCompatdataHealthyMissing(t *testing.T) {
	tmp := t.TempDir()
	compatdata := filepath.Join(tmp, "nonexistent")

	if CompatdataHealthy(compatdata) {
		t.Error("CompatdataHealthy should return false when compatdata doesn't exist")
	}
}

// TestCompatdataHealthyNoPfx verifies false when pfx/ is missing.
func TestCompatdataHealthyNoPfx(t *testing.T) {
	tmp := t.TempDir()
	compatdata := filepath.Join(tmp, "compatdata")
	os.MkdirAll(compatdata, 0755)
	// No pfx/ subdirectory.

	if CompatdataHealthy(compatdata) {
		t.Error("CompatdataHealthy should return false when pfx/ is missing")
	}
}

// TestCompatdataHealthyNoDriveC verifies false when pfx/ exists but drive_c is missing.
func TestCompatdataHealthyNoDriveC(t *testing.T) {
	tmp := t.TempDir()
	compatdata := filepath.Join(tmp, "compatdata")
	os.MkdirAll(filepath.Join(compatdata, "pfx"), 0755)
	// pfx/ exists but no drive_c.

	if CompatdataHealthy(compatdata) {
		t.Error("CompatdataHealthy should return false when drive_c is missing")
	}
}

// TestCompatdataHealthyDriveCIsFile verifies false when drive_c is a file not a directory.
func TestCompatdataHealthyDriveCIsFile(t *testing.T) {
	tmp := t.TempDir()
	compatdata := filepath.Join(tmp, "compatdata")
	os.MkdirAll(filepath.Join(compatdata, "pfx"), 0755)
	os.WriteFile(filepath.Join(compatdata, "pfx", "drive_c"), []byte("not a dir"), 0644)

	if CompatdataHealthy(compatdata) {
		t.Error("CompatdataHealthy should return false when drive_c is a file")
	}
}

// TestCompatdataPath verifies that CompatdataPath returns correct path.
func TestCompatdataPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLUCKERS_HOME", tmp)

	want := filepath.Join(tmp, "compatdata")
	if got := CompatdataPath(); got != want {
		t.Errorf("CompatdataPath() = %q, want %q", got, want)
	}
}
