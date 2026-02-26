//go:build linux

// Package inputproxy implements an evdev-to-uinput proxy for Steam Deck
// controller input. It creates a virtual Xbox 360 gamepad via /dev/uinput,
// reads events from the Steam Input virtual pad, and forwards them with
// dead reckoning to hold button state during ServerTravel transitions.
package inputproxy

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

// --- Kernel ABI structs ---
// These must exactly match the Linux kernel structures in size and layout.
// See: include/uapi/linux/uinput.h, include/uapi/linux/input.h

// inputID matches struct input_id (8 bytes).
type inputID struct {
	Bustype uint16
	Vendor  uint16
	Product uint16
	Version uint16
}

// uinputSetup matches struct uinput_setup (92 bytes).
type uinputSetup struct {
	ID           inputID
	Name         [80]byte
	FFEffectsMax uint32
}

// inputAbsinfo matches struct input_absinfo (24 bytes: 6 x int32).
type inputAbsinfo struct {
	Value      int32
	Minimum    int32
	Maximum    int32
	Fuzz       int32
	Flat       int32
	Resolution int32
}

// uinputAbsSetup matches struct uinput_abs_setup (28 bytes).
type uinputAbsSetup struct {
	Code uint16
	_    [2]byte // padding to align AbsInfo
	Info inputAbsinfo
}

// linuxInputEvent matches struct input_event (24 bytes on 64-bit).
type linuxInputEvent struct {
	Sec   uint64
	Usec  uint64
	Type  uint16
	Code  uint16
	Value int32
}

// --- Ioctl constants from linux/uinput.h and linux/input.h ---

const (
	uiSetEvBit   uintptr = 0x40045564 // UI_SET_EVBIT
	uiSetKeyBit  uintptr = 0x40045565 // UI_SET_KEYBIT
	uiSetAbsBit  uintptr = 0x40045567 // UI_SET_ABSBIT
	uiDevSetup   uintptr = 0x405c5503 // UI_DEV_SETUP
	uiAbsSetup   uintptr = 0x401c5504 // UI_ABS_SETUP
	uiDevCreate  uintptr = 0x5501     // UI_DEV_CREATE
	uiDevDestroy uintptr = 0x5502     // UI_DEV_DESTROY
	eviocgrab    uintptr = 0x40044590 // EVIOCGRAB — exclusive device access
)

// --- Event type constants ---

const (
	evSyn uint16 = 0x00 // EV_SYN
	evKey uint16 = 0x01 // EV_KEY
	evAbs uint16 = 0x03 // EV_ABS
)

// --- Xbox 360 identity ---

const (
	xbox360VID uint16 = 0x045e // Microsoft
	xbox360PID uint16 = 0x028e // Xbox 360 Controller
	busUSB     uint16 = 0x03   // BUS_USB
)

// --- Xbox 360 button codes (BTN_* from linux/input-event-codes.h) ---

const (
	btnA      uint16 = 0x130 // BTN_A
	btnB      uint16 = 0x131 // BTN_B
	btnX      uint16 = 0x133 // BTN_X
	btnY      uint16 = 0x134 // BTN_Y
	btnTL     uint16 = 0x136 // BTN_TL
	btnTR     uint16 = 0x137 // BTN_TR
	btnSelect uint16 = 0x13a // BTN_SELECT
	btnStart  uint16 = 0x13b // BTN_START
	btnMode   uint16 = 0x13c // BTN_MODE
	btnThumbL uint16 = 0x13d // BTN_THUMBL
	btnThumbR uint16 = 0x13e // BTN_THUMBR
)

// xbox360Buttons lists all Xbox 360 button codes for UI_SET_KEYBIT registration.
var xbox360Buttons = []uint16{
	btnA, btnB, btnX, btnY,
	btnTL, btnTR,
	btnSelect, btnStart, btnMode,
	btnThumbL, btnThumbR,
}

// --- Xbox 360 absolute axis codes (ABS_* from linux/input-event-codes.h) ---

const (
	absX     uint16 = 0x00 // ABS_X (left stick X)
	absY     uint16 = 0x01 // ABS_Y (left stick Y)
	absZ     uint16 = 0x02 // ABS_Z (left trigger)
	absRX    uint16 = 0x03 // ABS_RX (right stick X)
	absRY    uint16 = 0x04 // ABS_RY (right stick Y)
	absRZ    uint16 = 0x05 // ABS_RZ (right trigger)
	absHat0X uint16 = 0x10 // ABS_HAT0X (dpad X)
	absHat0Y uint16 = 0x11 // ABS_HAT0Y (dpad Y)
)

// axisSpec defines an absolute axis with its value range.
type axisSpec struct {
	Code uint16
	Min  int32
	Max  int32
	Fuzz int32
	Flat int32
}

// xbox360Axes lists all Xbox 360 axes with ranges from kernel xpad.c.
var xbox360Axes = []axisSpec{
	{absX, -32768, 32767, 16, 128},  // Left stick X
	{absY, -32768, 32767, 16, 128},  // Left stick Y
	{absZ, 0, 255, 0, 0},            // Left trigger
	{absRX, -32768, 32767, 16, 128}, // Right stick X
	{absRY, -32768, 32767, 16, 128}, // Right stick Y
	{absRZ, 0, 255, 0, 0},           // Right trigger
	{absHat0X, -1, 1, 0, 0},         // D-pad X
	{absHat0Y, -1, 1, 0, 0},         // D-pad Y
}

// openUinput opens /dev/uinput for writing.
func openUinput() (*os.File, error) {
	fd, err := os.OpenFile("/dev/uinput", os.O_WRONLY|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("opening /dev/uinput: %w (is the input group or uaccess configured?)", err)
	}
	return fd, nil
}

// createVirtualXbox360 configures a uinput file descriptor as a virtual Xbox 360 controller.
// The caller must have opened /dev/uinput via openUinput() first.
func createVirtualXbox360(fd *os.File) error {
	ufd := int(fd.Fd())

	// 1. Register event types: EV_KEY (buttons), EV_ABS (axes), EV_SYN (sync).
	for _, evType := range []uint16{evKey, evAbs, evSyn} {
		if err := unix.IoctlSetInt(ufd, uint(uiSetEvBit), int(evType)); err != nil {
			return fmt.Errorf("UI_SET_EVBIT(0x%x): %w", evType, err)
		}
	}

	// 2. Register buttons.
	for _, btn := range xbox360Buttons {
		if err := unix.IoctlSetInt(ufd, uint(uiSetKeyBit), int(btn)); err != nil {
			return fmt.Errorf("UI_SET_KEYBIT(0x%x): %w", btn, err)
		}
	}

	// 3. Register axes with ranges.
	for _, axis := range xbox360Axes {
		// Register the axis code.
		if err := unix.IoctlSetInt(ufd, uint(uiSetAbsBit), int(axis.Code)); err != nil {
			return fmt.Errorf("UI_SET_ABSBIT(0x%x): %w", axis.Code, err)
		}

		// Configure axis range via UI_ABS_SETUP.
		absSetup := uinputAbsSetup{
			Code: axis.Code,
			Info: inputAbsinfo{
				Minimum: axis.Min,
				Maximum: axis.Max,
				Fuzz:    axis.Fuzz,
				Flat:    axis.Flat,
			},
		}
		if _, _, errno := unix.Syscall(
			unix.SYS_IOCTL,
			uintptr(ufd),
			uiAbsSetup,
			uintptr(unsafe.Pointer(&absSetup)),
		); errno != 0 {
			return fmt.Errorf("UI_ABS_SETUP(0x%x): %w", axis.Code, errno)
		}
	}

	// 4. Set device identity (Xbox 360 controller).
	setup := uinputSetup{
		ID: inputID{
			Bustype: busUSB,
			Vendor:  xbox360VID,
			Product: xbox360PID,
			Version: 0x0110,
		},
	}
	copy(setup.Name[:], "Microsoft X-Box 360 pad")

	if _, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(ufd),
		uiDevSetup,
		uintptr(unsafe.Pointer(&setup)),
	); errno != 0 {
		return fmt.Errorf("UI_DEV_SETUP: %w", errno)
	}

	// 5. Create the virtual device.
	if _, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(ufd),
		uiDevCreate,
		0,
	); errno != 0 {
		return fmt.Errorf("UI_DEV_CREATE: %w", errno)
	}

	return nil
}

// destroyVirtualDevice sends UI_DEV_DESTROY to clean up the virtual device.
// Note: closing the fd also destroys the device automatically.
func destroyVirtualDevice(fd *os.File) error {
	if _, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fd.Fd()),
		uiDevDestroy,
		0,
	); errno != 0 {
		return fmt.Errorf("UI_DEV_DESTROY: %w", errno)
	}
	return nil
}

// writeEvent writes a single input event to the uinput device.
func writeEvent(fd *os.File, typ uint16, code uint16, value int32) error {
	ev := linuxInputEvent{
		Type:  typ,
		Code:  code,
		Value: value,
	}
	buf := (*[unsafe.Sizeof(ev)]byte)(unsafe.Pointer(&ev))[:]
	_, err := fd.Write(buf)
	return err
}

// invertY negates a Y-axis value for evdev-to-XInput conversion.
// Evdev Y+ = down, XInput Y+ = up. Clamps -32768 to 32767.
func invertY(v int32) int32 {
	if v == -32768 {
		return 32767 // Prevent overflow: -(-32768) = 32768 > int16 max
	}
	return -v
}
