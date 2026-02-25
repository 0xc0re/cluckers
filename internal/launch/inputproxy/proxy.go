//go:build linux

package inputproxy

import (
	"os"
	"time"
)

// DefaultHoldTimeout is the duration to hold button state during ServerTravel.
// ServerTravel transitions take >1 second; 500ms covers the gap without
// noticeable phantom input in normal gameplay.
const DefaultHoldTimeout = 500 * time.Millisecond

// buttonCodeToMask maps evdev BTN_* codes to bitmask positions in buttonState.buttons.
var buttonCodeToMask = map[uint16]uint32{
	btnA:      0x0001,
	btnB:      0x0002,
	btnX:      0x0004,
	btnY:      0x0008,
	btnTL:     0x0010,
	btnTR:     0x0020,
	btnSelect: 0x0040,
	btnStart:  0x0080,
	btnMode:   0x0100,
	btnThumbL: 0x0200,
	btnThumbR: 0x0400,
}

// buttonState tracks the shadow state of buttons and triggers for dead reckoning.
// Dead reckoning holds the last-known button state when ALL buttons AND triggers
// go to zero simultaneously (the ServerTravel signature from Steam Input firmware).
type buttonState struct {
	buttons          uint32    // Bitmask of currently pressed buttons
	lTrig            uint8     // Left trigger value (ABS_Z)
	rTrig            uint8     // Right trigger value (ABS_RZ)
	lastNonZero      time.Time // Last time any button/trigger was non-zero
	hadTrigActivity  bool      // Whether triggers have ever been non-zero
}

// updateButton sets or clears a button in the bitmask and updates lastNonZero.
func (s *buttonState) updateButton(code uint16, value int32) {
	mask, ok := buttonCodeToMask[code]
	if !ok {
		return // Unknown button code -- ignore
	}
	if value != 0 {
		s.buttons |= mask
		s.lastNonZero = time.Now()
	} else {
		s.buttons &^= mask
	}
}

// updateTrigger updates trigger state and lastNonZero tracking.
func (s *buttonState) updateTrigger(code uint16, value int32) {
	v := uint8(value)
	switch code {
	case absZ:
		s.lTrig = v
	case absRZ:
		s.rTrig = v
	default:
		return // Not a trigger
	}
	if v != 0 {
		s.lastNonZero = time.Now()
		s.hadTrigActivity = true
	}
}

// isAllButtonsZero returns true when no buttons are pressed and both triggers are at zero.
func (s *buttonState) isAllButtonsZero() bool {
	return s.buttons == 0 && s.lTrig == 0 && s.rTrig == 0
}

// shouldHold returns true when the dead reckoning hold should be active.
// Hold is active when:
//  1. All buttons and triggers are zero (potential ServerTravel zeroing)
//  2. We recently had non-zero button/trigger state (within timeout)
//  3. We have ever had non-zero state (lastNonZero is not zero time)
//  4. Triggers were previously non-zero (distinguishes ServerTravel from normal release)
//
// This prevents false positives: a single button release (normal gameplay)
// does not trigger hold because the triggers were never non-zero prior to zeroing.
// The ServerTravel signature is ALL buttons AND triggers going to zero simultaneously,
// which requires that triggers were previously active.
func (s *buttonState) shouldHold(timeout time.Duration) bool {
	if !s.isAllButtonsZero() {
		return false
	}
	if s.lastNonZero.IsZero() {
		return false // Never had non-zero state
	}
	if !s.hadTrigActivity {
		return false // No prior trigger activity -- normal button release
	}
	return time.Since(s.lastNonZero) < timeout
}

// InputProxy is the core proxy struct that reads from the Steam Input evdev
// device and writes to the virtual uinput gamepad. Plan 07.1-03 will implement
// the Run loop and full event forwarding.
type InputProxy struct {
	source   *os.File      // Steam Input evdev device (read)
	sink     *os.File      // /dev/uinput virtual gamepad (write)
	state    buttonState   // Dead reckoning shadow state
	holdTime time.Duration // How long to hold state on zero detection
	stopCh   chan struct{} // Signal to stop the proxy
	done     chan struct{} // Closed when proxy goroutine exits
}
