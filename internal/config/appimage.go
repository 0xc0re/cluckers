package config

import "os"

// IsAppImage returns true if running inside an AppImage.
func IsAppImage() bool {
	return os.Getenv("APPIMAGE") != ""
}

// AppImagePath returns the path to the AppImage file itself,
// or empty string if not running as AppImage.
func AppImagePath() string {
	return os.Getenv("APPIMAGE")
}

// AppDir returns the mount point of the AppImage's squashfs,
// or empty string if not running as AppImage.
func AppDir() string {
	return os.Getenv("APPDIR")
}

// BundledProtonPath returns the path to bundled Proton-GE set by AppRun,
// or empty string if not available.
func BundledProtonPath() string {
	return os.Getenv("CLUCKERS_BUNDLED_PROTON")
}
