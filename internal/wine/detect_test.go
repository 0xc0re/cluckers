//go:build linux

package wine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsSteamDeckCheck(t *testing.T) {
	tests := []struct {
		name           string
		boardVendor    string
		distroID       string
		deckHomeExists bool
		want           bool
	}{
		{
			name:        "Valve DMI on SteamOS",
			boardVendor: "Valve", distroID: "steamos", deckHomeExists: true,
			want: true,
		},
		{
			name:        "Valve DMI on Bazzite",
			boardVendor: "Valve", distroID: "bazzite", deckHomeExists: false,
			want: true,
		},
		{
			name:        "Valve DMI with trailing newline",
			boardVendor: "Valve\n", distroID: "unknown", deckHomeExists: false,
			want: true,
		},
		{
			name:        "SteamOS without DMI",
			boardVendor: "", distroID: "steamos", deckHomeExists: false,
			want: true,
		},
		{
			name:        "bazzite-deck on ROG Ally with /home/deck",
			boardVendor: "ASUSTeK", distroID: "bazzite", deckHomeExists: true,
			want: true,
		},
		{
			name:        "Bazzite on desktop without /home/deck",
			boardVendor: "ASUSTeK", distroID: "bazzite", deckHomeExists: false,
			want: false,
		},
		{
			name:        "Generic Linux desktop",
			boardVendor: "LENOVO", distroID: "fedora", deckHomeExists: false,
			want: false,
		},
		{
			name:        "Empty everything",
			boardVendor: "", distroID: "unknown", deckHomeExists: false,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSteamDeckCheck(tt.boardVendor, tt.distroID, tt.deckHomeExists)
			if got != tt.want {
				t.Errorf("isSteamDeckCheck(%q, %q, %v) = %v, want %v",
					tt.boardVendor, tt.distroID, tt.deckHomeExists, got, tt.want)
			}
		})
	}
}

func TestParseOSRelease(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantID   string
		wantLike string
	}{
		{
			name:     "SteamOS",
			input:    "NAME=\"SteamOS\"\nID=steamos\nID_LIKE=arch\n",
			wantID:   "steamos",
			wantLike: "arch",
		},
		{
			name:     "Bazzite",
			input:    "NAME=Bazzite\nID=bazzite\nID_LIKE=\"fedora\"\n",
			wantID:   "bazzite",
			wantLike: "fedora",
		},
		{
			name:     "Ubuntu with multiple ID_LIKE",
			input:    "ID=linuxmint\nID_LIKE=\"ubuntu debian\"\n",
			wantID:   "linuxmint",
			wantLike: "ubuntu debian",
		},
		{
			name:     "Fedora no ID_LIKE",
			input:    "ID=fedora\nVERSION_ID=39\n",
			wantID:   "fedora",
			wantLike: "",
		},
		{
			name:     "Empty",
			input:    "",
			wantID:   "",
			wantLike: "",
		},
		{
			name:     "Quoted ID",
			input:    "ID=\"arch\"\n",
			wantID:   "arch",
			wantLike: "",
		},
		{
			name:     "NixOS",
			input:    "NAME=NixOS\nID=nixos\nVERSION_ID=\"24.11pre\"\n",
			wantID:   "nixos",
			wantLike: "",
		},
		{
			name:     "openSUSE Tumbleweed",
			input:    "NAME=\"openSUSE Tumbleweed\"\nID=opensuse-tumbleweed\nID_LIKE=\"opensuse suse\"\n",
			wantID:   "opensuse-tumbleweed",
			wantLike: "opensuse suse",
		},
		{
			name:     "openSUSE Leap",
			input:    "NAME=\"openSUSE Leap\"\nID=opensuse-leap\nID_LIKE=\"suse opensuse\"\n",
			wantID:   "opensuse-leap",
			wantLike: "suse opensuse",
		},
		{
			name:     "Gentoo",
			input:    "NAME=Gentoo\nID=gentoo\n",
			wantID:   "gentoo",
			wantLike: "",
		},
		{
			name:     "Void Linux",
			input:    "NAME=\"Void Linux\"\nID=void\n",
			wantID:   "void",
			wantLike: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := parseOSRelease(strings.NewReader(tt.input))
			if info.ID != tt.wantID {
				t.Errorf("parseOSRelease ID = %q, want %q", info.ID, tt.wantID)
			}
			if info.IDLike != tt.wantLike {
				t.Errorf("parseOSRelease IDLike = %q, want %q", info.IDLike, tt.wantLike)
			}
		})
	}
}

func TestExtraCompatToolsDirsEmpty(t *testing.T) {
	t.Setenv("STEAM_EXTRA_COMPAT_TOOLS_PATHS", "")
	dirs := extraCompatToolsDirs()
	if len(dirs) != 0 {
		t.Errorf("extraCompatToolsDirs() returned %d dirs, want 0", len(dirs))
	}
}

func TestExtraCompatToolsDirsSingle(t *testing.T) {
	t.Setenv("STEAM_EXTRA_COMPAT_TOOLS_PATHS", "/nix/store/abc-proton-ge-bin")
	dirs := extraCompatToolsDirs()
	if len(dirs) != 1 || dirs[0] != "/nix/store/abc-proton-ge-bin" {
		t.Errorf("extraCompatToolsDirs() = %v, want [/nix/store/abc-proton-ge-bin]", dirs)
	}
}

func TestExtraCompatToolsDirsMultiple(t *testing.T) {
	t.Setenv("STEAM_EXTRA_COMPAT_TOOLS_PATHS", "/nix/store/abc-proton-ge-bin:/nix/store/def-proton-ge-bin")
	dirs := extraCompatToolsDirs()
	if len(dirs) != 2 {
		t.Errorf("extraCompatToolsDirs() returned %d dirs, want 2", len(dirs))
	}
}

func TestExtraCompatToolsDirsTrailingColon(t *testing.T) {
	t.Setenv("STEAM_EXTRA_COMPAT_TOOLS_PATHS", "/nix/store/abc-proton-ge-bin:")
	dirs := extraCompatToolsDirs()
	if len(dirs) != 1 {
		t.Errorf("extraCompatToolsDirs() returned %d dirs, want 1 (trailing colon ignored)", len(dirs))
	}
}

// TestFindProtonGEExtraCompatPaths verifies that STEAM_EXTRA_COMPAT_TOOLS_PATHS
// is scanned for Proton-GE installations (NixOS declarative installs).
func TestFindProtonGEExtraCompatPaths(t *testing.T) {
	tmp := t.TempDir()
	emptyHome := filepath.Join(tmp, "empty")
	os.MkdirAll(emptyHome, 0755)

	// Create a Proton-GE install at a Nix-store-like path.
	nixStoreDir := filepath.Join(tmp, "nix", "store", "abc-proton-ge-bin")
	protonDir := filepath.Join(nixStoreDir, "GE-Proton10-33")
	os.MkdirAll(filepath.Join(protonDir, "files", "bin"), 0755)
	os.WriteFile(filepath.Join(protonDir, "proton"), []byte("#!/usr/bin/env python3\n"), 0755)
	os.WriteFile(filepath.Join(protonDir, "files", "bin", "wine64"), []byte("fake"), 0755)

	t.Setenv("STEAM_EXTRA_COMPAT_TOOLS_PATHS", nixStoreDir)

	installs := FindProtonGE(emptyHome)
	if len(installs) == 0 {
		t.Fatal("FindProtonGE should find Proton-GE via STEAM_EXTRA_COMPAT_TOOLS_PATHS")
	}
	if installs[0].ProtonDir != protonDir {
		t.Errorf("ProtonDir = %q, want %q", installs[0].ProtonDir, protonDir)
	}
}
