//go:build linux

package wine

import (
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
