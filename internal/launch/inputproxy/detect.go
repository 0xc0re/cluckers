//go:build linux

package inputproxy

import (
	"fmt"

	"github.com/kenshaw/evdev"
)

// Steam Input virtual pad identification constants.
const (
	steamInputVID   uint16 = 0x28de // Valve Corporation
	steamInputPID   uint16 = 0x11ff // Steam Virtual Gamepad
	maxEventDevices        = 20     // Scan event0 through event19
)

// FindSteamInputPad scans /dev/input/event* for the Steam Input virtual pad
// by matching VID=0x28de PID=0x11ff. Returns the device path on success.
func FindSteamInputPad() (string, error) {
	pads := FindAllSteamInputPads()
	if len(pads) == 0 {
		return "", fmt.Errorf("Steam Input virtual pad not found -- is Steam running?")
	}
	return pads[0], nil
}

// FindAllSteamInputPads scans /dev/input/event* for all Steam Input virtual
// pads matching VID=0x28de PID=0x11ff. Returns all matching device paths.
func FindAllSteamInputPads() []string {
	var pads []string
	for i := 0; i < maxEventDevices; i++ {
		path := fmt.Sprintf("/dev/input/event%d", i)
		dev, err := evdev.OpenFile(path)
		if err != nil {
			continue
		}
		id := dev.ID()
		dev.Close()
		if id.Vendor == steamInputVID && id.Product == steamInputPID {
			pads = append(pads, path)
		}
	}
	return pads
}
