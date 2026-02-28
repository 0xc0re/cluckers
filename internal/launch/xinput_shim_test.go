//go:build linux

package launch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0xc0re/cluckers/assets"
)

func TestDeployXInputShim_CreatesFile(t *testing.T) {
	gameDir := t.TempDir()
	win64Dir := filepath.Join(gameDir, "Binaries", "Win64")
	if err := os.MkdirAll(win64Dir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := DeployXInputShim(gameDir); err != nil {
		t.Fatalf("DeployXInputShim() = %v", err)
	}

	destPath := filepath.Join(win64Dir, "xinput1_3.dll")
	info, err := os.Stat(destPath)
	if err != nil {
		t.Fatalf("xinput1_3.dll not created: %v", err)
	}
	if info.Size() != int64(len(assets.XInputCacheDLL)) {
		t.Errorf("size = %d, want %d", info.Size(), len(assets.XInputCacheDLL))
	}
}

func TestDeployXInputShim_Idempotent(t *testing.T) {
	gameDir := t.TempDir()
	win64Dir := filepath.Join(gameDir, "Binaries", "Win64")
	if err := os.MkdirAll(win64Dir, 0755); err != nil {
		t.Fatal(err)
	}

	// First deploy.
	if err := DeployXInputShim(gameDir); err != nil {
		t.Fatalf("first DeployXInputShim() = %v", err)
	}

	destPath := filepath.Join(win64Dir, "xinput1_3.dll")
	info1, _ := os.Stat(destPath)
	modTime1 := info1.ModTime()

	// Second deploy should skip (same size).
	if err := DeployXInputShim(gameDir); err != nil {
		t.Fatalf("second DeployXInputShim() = %v", err)
	}

	info2, _ := os.Stat(destPath)
	if info2.ModTime() != modTime1 {
		t.Error("second deploy modified file, should be idempotent")
	}
}

func TestDeployXInputShim_BackupsExistingFile(t *testing.T) {
	gameDir := t.TempDir()
	win64Dir := filepath.Join(gameDir, "Binaries", "Win64")
	if err := os.MkdirAll(win64Dir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a pre-existing xinput1_3.dll with different content.
	destPath := filepath.Join(win64Dir, "xinput1_3.dll")
	oldContent := []byte("old xinput dll content")
	if err := os.WriteFile(destPath, oldContent, 0644); err != nil {
		t.Fatal(err)
	}

	if err := DeployXInputShim(gameDir); err != nil {
		t.Fatalf("DeployXInputShim() = %v", err)
	}

	// Backup should exist.
	backupPath := destPath + ".bak"
	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("backup not created: %v", err)
	}
	if string(backupData) != string(oldContent) {
		t.Error("backup content does not match original")
	}

	// New file should be the embedded DLL.
	newData, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("new file not readable: %v", err)
	}
	if len(newData) != len(assets.XInputCacheDLL) {
		t.Errorf("new file size = %d, want %d", len(newData), len(assets.XInputCacheDLL))
	}
}

func TestDeployXInputShim_MissingGameDir(t *testing.T) {
	err := DeployXInputShim("/nonexistent/game/dir")
	if err == nil {
		t.Fatal("expected error for missing game directory")
	}
}
