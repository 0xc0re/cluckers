package selfupdate

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestIsNewer(t *testing.T) {
	tests := []struct {
		name           string
		currentVersion string
		latestTag      string
		want           bool
	}{
		{"patch bump", "0.4.0", "v0.4.1", true},
		{"same version", "0.4.1", "v0.4.1", false},
		{"current is newer", "0.5.0", "v0.4.1", false},
		{"dev build", "dev", "v0.4.1", false},
		{"minor bump", "0.4.1", "v0.5.0", true},
		{"major bump", "0.4.1", "v1.0.0", true},
		{"current major is higher", "1.0.0", "v0.9.9", false},
		{"with v prefix on current", "v0.4.0", "v0.4.1", true},
		{"both without v prefix", "0.3.0", "0.4.0", true},
		{"two-segment versions", "1.0", "v1.1", true},
		{"latest is shorter", "0.4.1", "v0.5", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNewer(tt.currentVersion, tt.latestTag)
			if got != tt.want {
				t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.currentVersion, tt.latestTag, got, tt.want)
			}
		})
	}
}

func TestFindAsset(t *testing.T) {
	info := &ReleaseInfo{
		TagName: "v0.4.1",
		Assets: []Asset{
			{Name: "cluckers_0.4.1_linux_amd64.tar.gz", BrowserDownloadURL: "https://example.com/linux.tar.gz", Size: 5000000},
			{Name: "cluckers_0.4.1_windows_amd64.zip", BrowserDownloadURL: "https://example.com/windows.zip", Size: 6000000},
			{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums.txt", Size: 256},
		},
	}

	archive, checksums, err := FindAsset(info)
	if err != nil {
		t.Fatalf("FindAsset() returned unexpected error: %v", err)
	}

	if archive == nil {
		t.Fatal("FindAsset() returned nil archive")
	}

	// Verify we get the correct platform asset.
	switch runtime.GOOS {
	case "linux":
		if archive.Name != "cluckers_0.4.1_linux_amd64.tar.gz" {
			t.Errorf("Expected linux asset, got %q", archive.Name)
		}
	case "windows":
		if archive.Name != "cluckers_0.4.1_windows_amd64.zip" {
			t.Errorf("Expected windows asset, got %q", archive.Name)
		}
	}

	if checksums == nil {
		t.Fatal("FindAsset() returned nil checksums (expected non-nil)")
	}
	if checksums.Name != "checksums.txt" {
		t.Errorf("Expected checksums.txt, got %q", checksums.Name)
	}
}

func TestFindAssetMissing(t *testing.T) {
	info := &ReleaseInfo{
		TagName: "v0.4.1",
		Assets: []Asset{
			// No matching asset for current platform.
			{Name: "cluckers_0.4.1_darwin_amd64.tar.gz", BrowserDownloadURL: "https://example.com/darwin.tar.gz", Size: 5000000},
		},
	}

	archive, _, err := FindAsset(info)
	if err == nil {
		t.Fatal("FindAsset() should return error when platform asset is missing")
	}
	if archive != nil {
		t.Error("FindAsset() should return nil archive when not found")
	}
}

func TestFindAssetNoChecksums(t *testing.T) {
	// Construct an asset that matches the current platform.
	version := "0.4.1"
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	name := "cluckers_" + version + "_" + runtime.GOOS + "_" + runtime.GOARCH + "." + ext

	info := &ReleaseInfo{
		TagName: "v" + version,
		Assets: []Asset{
			{Name: name, BrowserDownloadURL: "https://example.com/archive", Size: 5000000},
		},
	}

	archive, checksums, err := FindAsset(info)
	if err != nil {
		t.Fatalf("FindAsset() returned unexpected error: %v", err)
	}
	if archive == nil {
		t.Fatal("FindAsset() returned nil archive")
	}
	if checksums != nil {
		t.Error("FindAsset() should return nil checksums when checksums.txt is not present")
	}
}

func TestReleaseInfoAssetName(t *testing.T) {
	info := &ReleaseInfo{TagName: "v0.4.1"}
	name := info.assetName()

	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	expected := "cluckers_0.4.1_" + runtime.GOOS + "_" + runtime.GOARCH + "." + ext

	if name != expected {
		t.Errorf("assetName() = %q, want %q", name, expected)
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input string
		want  []int
	}{
		{"0.4.1", []int{0, 4, 1}},
		{"1.0.0", []int{1, 0, 0}},
		{"0.5", []int{0, 5}},
		{"abc", []int{0}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseVersion(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("parseVersion(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("parseVersion(%q)[%d] = %d, want %d", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestAssetNameAppImageMode(t *testing.T) {
	t.Setenv("APPIMAGE", "/tmp/Cluckers-x86_64.AppImage")

	info := &ReleaseInfo{TagName: "v0.5.0"}
	name := info.assetName()

	if name != "Cluckers-x86_64.AppImage" {
		t.Errorf("assetName() in AppImage mode = %q, want %q", name, "Cluckers-x86_64.AppImage")
	}
}

func TestAssetNameStandardMode(t *testing.T) {
	// Ensure APPIMAGE is not set (t.Setenv unsets on cleanup).
	t.Setenv("APPIMAGE", "")

	info := &ReleaseInfo{TagName: "v0.5.0"}
	name := info.assetName()

	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	expected := "cluckers_0.5.0_" + runtime.GOOS + "_" + runtime.GOARCH + "." + ext

	if name != expected {
		t.Errorf("assetName() in standard mode = %q, want %q", name, expected)
	}
}

func TestFindAssetAppImageMode(t *testing.T) {
	t.Setenv("APPIMAGE", "/tmp/Cluckers-x86_64.AppImage")

	info := &ReleaseInfo{
		TagName: "v0.5.0",
		Assets: []Asset{
			{Name: "cluckers_0.5.0_linux_amd64.tar.gz", BrowserDownloadURL: "https://example.com/linux.tar.gz", Size: 5000000},
			{Name: "Cluckers-x86_64.AppImage", BrowserDownloadURL: "https://example.com/appimage", Size: 200000000},
			{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums.txt", Size: 256},
		},
	}

	archive, checksums, err := FindAsset(info)
	if err != nil {
		t.Fatalf("FindAsset() returned unexpected error: %v", err)
	}
	if archive == nil {
		t.Fatal("FindAsset() returned nil archive")
	}
	if archive.Name != "Cluckers-x86_64.AppImage" {
		t.Errorf("FindAsset() in AppImage mode returned %q, want %q", archive.Name, "Cluckers-x86_64.AppImage")
	}
	if checksums == nil {
		t.Error("FindAsset() should return checksums when available")
	}
}

func TestReplaceAppImage(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a fake "downloaded" AppImage.
	downloadedPath := filepath.Join(tmpDir, "downloaded.AppImage")
	newContent := []byte("new-appimage-content")
	if err := os.WriteFile(downloadedPath, newContent, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Create a fake "running" AppImage path.
	appImagePath := filepath.Join(tmpDir, "Cluckers-x86_64.AppImage")
	if err := os.WriteFile(appImagePath, []byte("old-content"), 0755); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	t.Setenv("APPIMAGE", appImagePath)

	if err := replaceAppImage(downloadedPath); err != nil {
		t.Fatalf("replaceAppImage() returned unexpected error: %v", err)
	}

	// Verify the AppImage was replaced.
	data, err := os.ReadFile(appImagePath)
	if err != nil {
		t.Fatalf("failed to read replaced AppImage: %v", err)
	}
	if string(data) != string(newContent) {
		t.Errorf("AppImage content = %q, want %q", string(data), string(newContent))
	}

	// Verify executable permissions.
	info, err := os.Stat(appImagePath)
	if err != nil {
		t.Fatalf("failed to stat replaced AppImage: %v", err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Error("replaced AppImage should be executable")
	}
}

func TestReplaceAppImageNoEnvVar(t *testing.T) {
	t.Setenv("APPIMAGE", "")

	err := replaceAppImage("/tmp/fake-download")
	if err == nil {
		t.Fatal("replaceAppImage() should return error when APPIMAGE is not set")
	}
}
