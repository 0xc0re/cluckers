//go:build linux

package game

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// createTestZip creates a zip file at dir/test.zip with the given filename->content pairs.
func createTestZip(t *testing.T, dir string, files map[string]string) string {
	t.Helper()
	zipPath := filepath.Join(dir, "test.zip")

	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("creating zip file: %v", err)
	}

	w := zip.NewWriter(f)
	for name, content := range files {
		entry, err := w.Create(name)
		if err != nil {
			t.Fatalf("creating zip entry %s: %v", name, err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatalf("writing zip entry %s: %v", name, err)
		}
	}

	if err := w.Close(); err != nil {
		t.Fatalf("closing zip writer: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("closing zip file: %v", err)
	}
	return zipPath
}

func TestExtractZip_OverwriteReadOnly(t *testing.T) {
	tmp := t.TempDir()
	zipDir := filepath.Join(tmp, "zips")
	destDir := filepath.Join(tmp, "dest")
	if err := os.MkdirAll(zipDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}

	// First extraction: create files with original content.
	originalFiles := map[string]string{
		"testfile.txt":       "original",
		"subdir/nested.txt":  "nested-original",
	}
	zipPath1 := createTestZip(t, zipDir, originalFiles)
	if err := ExtractZip(zipPath1, destDir); err != nil {
		t.Fatalf("first extraction failed: %v", err)
	}

	// Chmod extracted files to read-only (0444).
	file1 := filepath.Join(destDir, "testfile.txt")
	file2 := filepath.Join(destDir, "subdir", "nested.txt")
	if err := os.Chmod(file1, 0444); err != nil {
		t.Fatalf("chmod file1: %v", err)
	}
	if err := os.Chmod(file2, 0444); err != nil {
		t.Fatalf("chmod file2: %v", err)
	}

	// Second extraction: overwrite with updated content.
	updatedFiles := map[string]string{
		"testfile.txt":       "updated",
		"subdir/nested.txt":  "nested-updated",
	}
	zipPath2 := createTestZip(t, zipDir, updatedFiles)
	if err := ExtractZip(zipPath2, destDir); err != nil {
		t.Fatalf("second extraction (overwrite read-only) failed: %v", err)
	}

	// Verify contents were updated.
	got1, err := os.ReadFile(file1)
	if err != nil {
		t.Fatalf("reading file1: %v", err)
	}
	if string(got1) != "updated" {
		t.Errorf("file1 content = %q, want %q", got1, "updated")
	}

	got2, err := os.ReadFile(file2)
	if err != nil {
		t.Fatalf("reading file2: %v", err)
	}
	if string(got2) != "nested-updated" {
		t.Errorf("file2 content = %q, want %q", got2, "nested-updated")
	}
}

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
