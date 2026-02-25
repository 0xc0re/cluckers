//go:build linux

package inputproxy

import (
	"os"
	"strings"
	"testing"
)

func TestRegistryContent(t *testing.T) {
	// Verify generated .reg file content matches Windows registry format.
	content := winebusRegContent

	// Must use \r\n line endings (Windows registry format)
	if !strings.Contains(content, "\r\n") {
		t.Error("registry content must use \\r\\n line endings")
	}

	// Must start with Windows Registry Editor header
	if !strings.HasPrefix(content, "Windows Registry Editor Version 5.00\r\n") {
		t.Error("registry content must start with 'Windows Registry Editor Version 5.00\\r\\n'")
	}

	// Must contain the winebus registry key
	expectedKey := `[HKEY_LOCAL_MACHINE\System\CurrentControlSet\Services\winebus]`
	if !strings.Contains(content, expectedKey) {
		t.Errorf("registry content must contain key %q", expectedKey)
	}

	// Must contain DisableHidraw=1
	if !strings.Contains(content, `"DisableHidraw"=dword:00000001`) {
		t.Error("registry content must contain DisableHidraw=dword:00000001")
	}

	// Must contain Enable SDL=1
	if !strings.Contains(content, `"Enable SDL"=dword:00000001`) {
		t.Error("registry content must contain Enable SDL=dword:00000001")
	}
}

func TestRegistryFilePath(t *testing.T) {
	// Verify temp file creation and cleanup for .reg file.
	path, cleanup, err := WriteWinebusRegFile()
	if err != nil {
		t.Fatalf("WriteWinebusRegFile() error: %v", err)
	}
	defer cleanup()

	// File must exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("registry file does not exist at %s", path)
	}

	// File must have .reg extension
	if !strings.HasSuffix(path, ".reg") {
		t.Errorf("registry file must have .reg extension, got %s", path)
	}

	// File content must match winebusRegContent
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading registry file: %v", err)
	}
	if string(data) != winebusRegContent {
		t.Error("registry file content does not match winebusRegContent constant")
	}

	// Cleanup should remove the file
	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("cleanup did not remove the registry file")
	}
}
