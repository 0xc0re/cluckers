// Package assets embeds binary assets for the cluckers launcher.
package assets

import _ "embed"

// SHMLauncherExe is the embedded shm_launcher.exe Win32 helper binary.
// It creates named shared memory for the content bootstrap, then launches the game.
//
//go:embed shm_launcher.exe
var SHMLauncherExe []byte
