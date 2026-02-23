// Package assets embeds binary assets for the cluckers launcher.
package assets

import _ "embed"

// SHMLauncherExe is the embedded shm_launcher.exe Win32 helper binary.
// It creates named shared memory for the content bootstrap, then launches the game.
//
//go:embed shm_launcher.exe
var SHMLauncherExe []byte

// XInputRemapDLL is an XInput index remapping proxy DLL for UE3 on Wine.
// UE3 reserves XInput index 0 for the keyboard and polls indices 1-3.
// Wine assigns the controller to index 0. This proxy remaps game index N
// to real index N-1, bridging the mismatch.
//
//go:embed xinput1_3_remap.dll
var XInputRemapDLL []byte
