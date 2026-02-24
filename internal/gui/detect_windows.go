//go:build gui && windows

package gui

// CanShowGUI always returns true on Windows, which always has GUI capability.
func CanShowGUI() bool {
	return true
}
