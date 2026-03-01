// Package assets embeds binary assets for the cluckers launcher.
package assets

import _ "embed"

// SHMLauncherExe is the embedded shm_launcher.exe Win32 helper binary.
// It creates named shared memory for the content bootstrap, then launches the game.
//
//go:embed shm_launcher.exe
var SHMLauncherExe []byte

// ControllerLayout is the Steam Deck (Neptune) controller layout VDF for
// Realm Royale. Maps Deck controls to keyboard/mouse bindings so the game
// works without native XInput controller support.
//
//go:embed controller_neptune_config.vdf
var ControllerLayout []byte
