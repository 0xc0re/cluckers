//go:build gui && linux

package gui

import "github.com/0xc0re/cluckers/internal/wine"

// isSteamDeck returns true if running on a Steam Deck.
// Delegates to wine.IsSteamDeck() which checks DMI board vendor, SteamOS distro, and /home/deck.
func isSteamDeck() bool {
	return wine.IsSteamDeck()
}
