//go:build gui && linux

package gui

import (
	"os"
	"strings"
)

// isSteamDeck returns true if running on a Steam Deck.
// Checks DMI board vendor for "Valve" and the /home/deck directory.
func isSteamDeck() bool {
	// Check board vendor (works in SteamOS and Desktop Mode).
	data, err := os.ReadFile("/sys/devices/virtual/dmi/id/board_vendor")
	if err == nil && strings.TrimSpace(string(data)) == "Valve" {
		return true
	}

	// Fallback: check for SteamOS distro via os-release.
	data, err = os.ReadFile("/etc/os-release")
	if err == nil && strings.Contains(string(data), "ID=steamos") {
		return true
	}

	// Fallback: check for deck home directory.
	if _, err := os.Stat("/home/deck"); err == nil {
		return true
	}

	return false
}
