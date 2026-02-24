//go:build gui && windows

package gui

// isSteamDeck always returns false on Windows.
func isSteamDeck() bool {
	return false
}
