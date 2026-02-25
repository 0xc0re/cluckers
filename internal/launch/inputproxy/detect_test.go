//go:build linux

package inputproxy

import (
	"testing"
)

// TestSteamInputVIDPID verifies the target device identification constants.
// Wrong VID/PID = proxy connects to wrong device or fails to find Steam Input pad.
func TestSteamInputVIDPID(t *testing.T) {
	if steamInputVID != 0x28de {
		t.Errorf("Steam Input VID: got 0x%x, want 0x28de", steamInputVID)
	}
	if steamInputPID != 0x11ff {
		t.Errorf("Steam Input PID: got 0x%x, want 0x11ff", steamInputPID)
	}
}

// TestFindSteamInputPadNotFound verifies that FindSteamInputPad returns an error
// when no matching device exists (which is the case on non-Steam Deck systems).
func TestFindSteamInputPadNotFound(t *testing.T) {
	// On non-Deck systems, no device with VID=0x28de PID=0x11ff should exist.
	// The function should return a descriptive error.
	_, err := FindSteamInputPad()
	if err == nil {
		t.Skip("Steam Input pad found -- running on Steam Deck, skipping not-found test")
	}
	// Verify the error message is user-friendly.
	if err.Error() == "" {
		t.Error("expected non-empty error message when device not found")
	}
}

// TestMaxDeviceScan verifies the scan range covers event0 through event19.
func TestMaxDeviceScan(t *testing.T) {
	if maxEventDevices != 20 {
		t.Errorf("maxEventDevices: got %d, want 20", maxEventDevices)
	}
}
