//go:build linux

package inputproxy

import (
	"testing"
	"unsafe"
)

// TestStructPacking verifies that Go struct sizes match the Linux kernel ABI.
// These sizes are critical: if they drift, ioctl calls will corrupt memory.
func TestStructPacking(t *testing.T) {
	tests := []struct {
		name     string
		got      uintptr
		expected uintptr
	}{
		{"inputID", unsafe.Sizeof(inputID{}), 8},
		{"uinputSetup", unsafe.Sizeof(uinputSetup{}), 92},
		{"uinputAbsSetup", unsafe.Sizeof(uinputAbsSetup{}), 28},
		{"inputAbsinfo", unsafe.Sizeof(inputAbsinfo{}), 24},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s: got %d bytes, want %d bytes", tt.name, tt.got, tt.expected)
			}
		})
	}
}

// TestIoctlConstants verifies ioctl magic numbers match linux/uinput.h.
// Wrong constants = wrong kernel calls = silent corruption or EINVAL.
func TestIoctlConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      uintptr
		expected uintptr
	}{
		{"uiSetEvBit", uiSetEvBit, 0x40045564},
		{"uiSetKeyBit", uiSetKeyBit, 0x40045565},
		{"uiSetAbsBit", uiSetAbsBit, 0x40045567},
		{"uiDevSetup", uiDevSetup, 0x405c5503},
		{"uiAbsSetup", uiAbsSetup, 0x401c5504},
		{"uiDevCreate", uiDevCreate, 0x5501},
		{"uiDevDestroy", uiDevDestroy, 0x5502},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s: got 0x%x, want 0x%x", tt.name, tt.got, tt.expected)
			}
		})
	}
}

// TestXbox360Buttons verifies all 11 Xbox 360 buttons are registered with correct codes.
// Missing buttons = dead inputs in-game.
func TestXbox360Buttons(t *testing.T) {
	expectedButtons := map[uint16]string{
		0x130: "BTN_A",
		0x131: "BTN_B",
		0x133: "BTN_X",
		0x134: "BTN_Y",
		0x136: "BTN_TL",
		0x137: "BTN_TR",
		0x13a: "BTN_SELECT",
		0x13b: "BTN_START",
		0x13c: "BTN_MODE",
		0x13d: "BTN_THUMBL",
		0x13e: "BTN_THUMBR",
	}

	if len(xbox360Buttons) != 11 {
		t.Fatalf("expected 11 buttons, got %d", len(xbox360Buttons))
	}

	// Build a set of actual button codes.
	actual := make(map[uint16]bool)
	for _, btn := range xbox360Buttons {
		actual[btn] = true
	}

	for code, name := range expectedButtons {
		if !actual[code] {
			t.Errorf("missing button %s (0x%x)", name, code)
		}
	}
}

// TestXbox360Axes verifies all 8 axes with correct ranges from kernel xpad.c.
// Wrong axis ranges = analog sticks/triggers mapped incorrectly.
func TestXbox360Axes(t *testing.T) {
	type axisSpec struct {
		code uint16
		min  int32
		max  int32
		fuzz int32
		flat int32
	}

	expectedAxes := []axisSpec{
		{0x00, -32768, 32767, 16, 128},  // ABS_X
		{0x01, -32768, 32767, 16, 128},  // ABS_Y
		{0x02, 0, 255, 0, 0},            // ABS_Z (left trigger)
		{0x03, -32768, 32767, 16, 128},  // ABS_RX
		{0x04, -32768, 32767, 16, 128},  // ABS_RY
		{0x05, 0, 255, 0, 0},            // ABS_RZ (right trigger)
		{0x10, -1, 1, 0, 0},             // ABS_HAT0X
		{0x11, -1, 1, 0, 0},             // ABS_HAT0Y
	}

	if len(xbox360Axes) != 8 {
		t.Fatalf("expected 8 axes, got %d", len(xbox360Axes))
	}

	for i, expected := range expectedAxes {
		got := xbox360Axes[i]
		if got.Code != expected.code {
			t.Errorf("axis %d: code got 0x%x, want 0x%x", i, got.Code, expected.code)
		}
		if got.Min != expected.min {
			t.Errorf("axis %d (0x%x): min got %d, want %d", i, expected.code, got.Min, expected.min)
		}
		if got.Max != expected.max {
			t.Errorf("axis %d (0x%x): max got %d, want %d", i, expected.code, got.Max, expected.max)
		}
		if got.Fuzz != expected.fuzz {
			t.Errorf("axis %d (0x%x): fuzz got %d, want %d", i, expected.code, got.Fuzz, expected.fuzz)
		}
		if got.Flat != expected.flat {
			t.Errorf("axis %d (0x%x): flat got %d, want %d", i, expected.code, got.Flat, expected.flat)
		}
	}
}

// TestXbox360Identity verifies VID=0x045e (Microsoft), PID=0x028e (Xbox 360), BUS=0x03 (USB).
// Wrong identity = Wine/game won't recognize the virtual device as a gamepad.
func TestXbox360Identity(t *testing.T) {
	if xbox360VID != 0x045e {
		t.Errorf("Xbox 360 VID: got 0x%x, want 0x045e", xbox360VID)
	}
	if xbox360PID != 0x028e {
		t.Errorf("Xbox 360 PID: got 0x%x, want 0x028e", xbox360PID)
	}
	if busUSB != 0x03 {
		t.Errorf("BUS_USB: got 0x%x, want 0x03", busUSB)
	}
}
