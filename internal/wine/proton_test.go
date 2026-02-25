//go:build linux

package wine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFindProtonBundled verifies that CLUCKERS_BUNDLED_PROTON env var has highest priority.
func TestFindProtonBundled(t *testing.T) {
	tmp := t.TempDir()

	// Create a fake bundled Proton-GE installation.
	bundledDir := filepath.Join(tmp, "bundled", "GE-Proton10-1")
	os.MkdirAll(filepath.Join(bundledDir, "files", "bin"), 0755)
	os.WriteFile(filepath.Join(bundledDir, "proton"), []byte("#!/usr/bin/env python3\n"), 0755)
	os.WriteFile(filepath.Join(bundledDir, "files", "bin", "wine64"), []byte("fake"), 0755)

	t.Setenv("CLUCKERS_BUNDLED_PROTON", bundledDir)
	// Set HOME to empty dir so system scan finds nothing.
	t.Setenv("HOME", filepath.Join(tmp, "empty"))

	install, err := FindProton("")
	if err != nil {
		t.Fatalf("FindProton returned error: %v", err)
	}
	if install.ProtonDir != bundledDir {
		t.Errorf("ProtonDir = %q, want %q", install.ProtonDir, bundledDir)
	}
	if install.WinePath != filepath.Join(bundledDir, "files", "bin", "wine64") {
		t.Errorf("WinePath = %q, want %q", install.WinePath, filepath.Join(bundledDir, "files", "bin", "wine64"))
	}
}

// TestFindProtonBundledInvalidFallsThrough verifies that invalid bundled path falls through.
func TestFindProtonBundledInvalidFallsThrough(t *testing.T) {
	tmp := t.TempDir()

	// Bundled dir set but has no proton script.
	t.Setenv("CLUCKERS_BUNDLED_PROTON", filepath.Join(tmp, "nonexistent"))
	// Set HOME to empty dir so system scan finds nothing.
	t.Setenv("HOME", filepath.Join(tmp, "empty"))

	_, err := FindProton("")
	if err == nil {
		t.Fatal("FindProton should return error when bundled is invalid and nothing else found")
	}
}

// TestFindProtonConfigOverrideWine64Path verifies config override with wine64 path.
func TestFindProtonConfigOverrideWine64Path(t *testing.T) {
	tmp := t.TempDir()

	// Create a fake Proton-GE installation pointed to by wine64 path.
	protonDir := filepath.Join(tmp, "GE-Proton10-1")
	os.MkdirAll(filepath.Join(protonDir, "files", "bin"), 0755)
	os.WriteFile(filepath.Join(protonDir, "proton"), []byte("#!/usr/bin/env python3\n"), 0755)
	os.WriteFile(filepath.Join(protonDir, "files", "bin", "wine64"), []byte("fake"), 0755)

	t.Setenv("CLUCKERS_BUNDLED_PROTON", "")
	t.Setenv("HOME", filepath.Join(tmp, "empty"))

	wine64Path := filepath.Join(protonDir, "files", "bin", "wine64")
	install, err := FindProton(wine64Path)
	if err != nil {
		t.Fatalf("FindProton returned error: %v", err)
	}
	if install.ProtonDir != protonDir {
		t.Errorf("ProtonDir = %q, want %q", install.ProtonDir, protonDir)
	}
	if install.WinePath != wine64Path {
		t.Errorf("WinePath = %q, want %q", install.WinePath, wine64Path)
	}
}

// TestFindProtonConfigOverrideDirectory verifies config override with directory path.
func TestFindProtonConfigOverrideDirectory(t *testing.T) {
	tmp := t.TempDir()

	// Create a fake Proton-GE installation pointed to by directory.
	protonDir := filepath.Join(tmp, "GE-Proton10-1")
	os.MkdirAll(filepath.Join(protonDir, "files", "bin"), 0755)
	os.WriteFile(filepath.Join(protonDir, "proton"), []byte("#!/usr/bin/env python3\n"), 0755)
	os.WriteFile(filepath.Join(protonDir, "files", "bin", "wine64"), []byte("fake"), 0755)

	t.Setenv("CLUCKERS_BUNDLED_PROTON", "")
	t.Setenv("HOME", filepath.Join(tmp, "empty"))

	install, err := FindProton(protonDir)
	if err != nil {
		t.Fatalf("FindProton returned error: %v", err)
	}
	if install.ProtonDir != protonDir {
		t.Errorf("ProtonDir = %q, want %q", install.ProtonDir, protonDir)
	}
	if install.WinePath != filepath.Join(protonDir, "files", "bin", "wine64") {
		t.Errorf("WinePath = %q, want %q", install.WinePath, filepath.Join(protonDir, "files", "bin", "wine64"))
	}
}

// TestFindProtonSystemScan verifies system scan finds Proton-GE when no bundled or config.
func TestFindProtonSystemScan(t *testing.T) {
	tmp := t.TempDir()

	// Create a fake Proton-GE at a system scan location.
	compatDir := filepath.Join(tmp, ".local", "share", "Steam", "compatibilitytools.d")
	protonDir := filepath.Join(compatDir, "GE-Proton10-1")
	os.MkdirAll(filepath.Join(protonDir, "files", "bin"), 0755)
	os.WriteFile(filepath.Join(protonDir, "proton"), []byte("#!/usr/bin/env python3\n"), 0755)
	os.WriteFile(filepath.Join(protonDir, "files", "bin", "wine64"), []byte("fake"), 0755)

	t.Setenv("CLUCKERS_BUNDLED_PROTON", "")
	t.Setenv("HOME", tmp)

	install, err := FindProton("")
	if err != nil {
		t.Fatalf("FindProton returned error: %v", err)
	}
	if install.ProtonDir != protonDir {
		t.Errorf("ProtonDir = %q, want %q", install.ProtonDir, protonDir)
	}
}

// TestFindProtonBundledOverridesSystem verifies bundled takes priority over system scan.
func TestFindProtonBundledOverridesSystem(t *testing.T) {
	tmp := t.TempDir()

	// Create both bundled and system scan installations.
	bundledDir := filepath.Join(tmp, "bundled", "GE-Proton10-1")
	os.MkdirAll(filepath.Join(bundledDir, "files", "bin"), 0755)
	os.WriteFile(filepath.Join(bundledDir, "proton"), []byte("#!/usr/bin/env python3\n"), 0755)
	os.WriteFile(filepath.Join(bundledDir, "files", "bin", "wine64"), []byte("fake"), 0755)

	compatDir := filepath.Join(tmp, ".local", "share", "Steam", "compatibilitytools.d")
	systemDir := filepath.Join(compatDir, "GE-Proton9-27")
	os.MkdirAll(filepath.Join(systemDir, "files", "bin"), 0755)
	os.WriteFile(filepath.Join(systemDir, "proton"), []byte("#!/usr/bin/env python3\n"), 0755)
	os.WriteFile(filepath.Join(systemDir, "files", "bin", "wine64"), []byte("fake"), 0755)

	t.Setenv("CLUCKERS_BUNDLED_PROTON", bundledDir)
	t.Setenv("HOME", tmp)

	install, err := FindProton("")
	if err != nil {
		t.Fatalf("FindProton returned error: %v", err)
	}
	// Should use bundled, not system.
	if install.ProtonDir != bundledDir {
		t.Errorf("ProtonDir = %q, want bundled %q", install.ProtonDir, bundledDir)
	}
}

// TestFindProtonNotFoundArch verifies per-distro error messages for Arch Linux.
func TestFindProtonNotFoundArch(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLUCKERS_BUNDLED_PROTON", "")
	t.Setenv("HOME", filepath.Join(tmp, "empty"))

	_, err := FindProton("")
	if err == nil {
		t.Fatal("FindProton should return error when nothing found")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "Proton-GE not found") {
		t.Errorf("error message should contain 'Proton-GE not found', got: %s", errMsg)
	}
}

// TestFindProtonNotFoundInstructions verifies install instructions are per-distro.
func TestProtonInstallInstructionsArch(t *testing.T) {
	instructions := ProtonInstallInstructions("arch")
	if !strings.Contains(instructions, "ProtonUp-Qt") {
		t.Errorf("arch instructions should mention ProtonUp-Qt, got: %s", instructions)
	}
	if !strings.Contains(instructions, "pacman") || !strings.Contains(instructions, "yay") || !strings.Contains(instructions, "paru") {
		t.Errorf("arch instructions should mention pacman/yay/paru, got: %s", instructions)
	}
}

func TestProtonInstallInstructionsSteamOS(t *testing.T) {
	instructions := ProtonInstallInstructions("steamos")
	if !strings.Contains(instructions, "ProtonUp-Qt") {
		t.Errorf("steamos instructions should mention ProtonUp-Qt, got: %s", instructions)
	}
}

func TestProtonInstallInstructionsUbuntu(t *testing.T) {
	instructions := ProtonInstallInstructions("ubuntu")
	if !strings.Contains(instructions, "ProtonUp-Qt") {
		t.Errorf("ubuntu instructions should mention ProtonUp-Qt, got: %s", instructions)
	}
}

func TestProtonInstallInstructionsDebian(t *testing.T) {
	instructions := ProtonInstallInstructions("debian")
	if !strings.Contains(instructions, "ProtonUp-Qt") {
		t.Errorf("debian instructions should mention ProtonUp-Qt, got: %s", instructions)
	}
}

func TestProtonInstallInstructionsFedora(t *testing.T) {
	instructions := ProtonInstallInstructions("fedora")
	if !strings.Contains(instructions, "ProtonUp-Qt") {
		t.Errorf("fedora instructions should mention ProtonUp-Qt, got: %s", instructions)
	}
}

func TestProtonInstallInstructionsDefault(t *testing.T) {
	instructions := ProtonInstallInstructions("unknown")
	if !strings.Contains(instructions, "github") || !strings.Contains(instructions, "GloriousEggroll") {
		t.Errorf("default instructions should link to GE GitHub, got: %s", instructions)
	}
}

// TestProtonScript verifies the ProtonScript method.
func TestProtonScript(t *testing.T) {
	install := ProtonGEInstall{
		ProtonDir: "/opt/proton/GE-Proton10-1",
		WinePath:  "/opt/proton/GE-Proton10-1/files/bin/wine64",
	}
	want := "/opt/proton/GE-Proton10-1/proton"
	if got := install.ProtonScript(); got != want {
		t.Errorf("ProtonScript() = %q, want %q", got, want)
	}
}

// TestDisplayVersion verifies the DisplayVersion method.
func TestDisplayVersion(t *testing.T) {
	tests := []struct {
		protonDir string
		want      string
	}{
		{"/opt/proton/GE-Proton10-1", "GE-Proton10-1"},
		{"/home/user/.steam/compatibilitytools.d/GE-Proton9-27", "GE-Proton9-27"},
		{"/usr/share/steam/compatibilitytools.d/proton-ge-custom", "proton-ge-custom"},
	}
	for _, tt := range tests {
		install := ProtonGEInstall{ProtonDir: tt.protonDir}
		if got := install.DisplayVersion(); got != tt.want {
			t.Errorf("DisplayVersion() for %q = %q, want %q", tt.protonDir, got, tt.want)
		}
	}
}

// TestFindProtonOldVersionAllowed verifies old Proton-GE versions are returned (warn but allow).
func TestFindProtonOldVersionAllowed(t *testing.T) {
	tmp := t.TempDir()

	// Create an old Proton-GE installation.
	compatDir := filepath.Join(tmp, ".local", "share", "Steam", "compatibilitytools.d")
	protonDir := filepath.Join(compatDir, "GE-Proton7-55")
	os.MkdirAll(filepath.Join(protonDir, "files", "bin"), 0755)
	os.WriteFile(filepath.Join(protonDir, "proton"), []byte("#!/usr/bin/env python3\n"), 0755)
	os.WriteFile(filepath.Join(protonDir, "files", "bin", "wine64"), []byte("fake"), 0755)

	t.Setenv("CLUCKERS_BUNDLED_PROTON", "")
	t.Setenv("HOME", tmp)

	install, err := FindProton("")
	if err != nil {
		t.Fatalf("FindProton should succeed for old versions (warn but allow), got error: %v", err)
	}
	if install.DisplayVersion() != "GE-Proton7-55" {
		t.Errorf("DisplayVersion() = %q, want %q", install.DisplayVersion(), "GE-Proton7-55")
	}
}
