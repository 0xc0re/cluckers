//go:build gui && linux

package gui

import "os"

// CanShowGUI returns true if a display server (X11 or Wayland) is available.
// On Linux, a GUI cannot be shown in headless environments (e.g., SSH sessions,
// containers, CI) where neither DISPLAY nor WAYLAND_DISPLAY is set.
func CanShowGUI() bool {
	if os.Getenv("DISPLAY") != "" {
		return true
	}
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return true
	}
	return false
}
