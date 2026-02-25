//go:build linux

package inputproxy

import (
	"os"
	"time"
)

// DefaultHoldTimeout is the duration to hold button state during ServerTravel.
const DefaultHoldTimeout = 500 * time.Millisecond

// buttonState tracks the shadow state of buttons and triggers for dead reckoning.
type buttonState struct {
	buttons     uint32
	lTrig       uint8
	rTrig       uint8
	lastNonZero time.Time
}

// Stub methods -- to be implemented in Plan 07.1-02 GREEN phase.

func (s *buttonState) updateButton(code uint16, value int32) {}
func (s *buttonState) updateTrigger(code uint16, value int32) {}
func (s *buttonState) isAllButtonsZero() bool                { return true }
func (s *buttonState) shouldHold(timeout time.Duration) bool { return false }

// InputProxy is the core proxy struct for Plan 07.1-03 to flesh out.
type InputProxy struct {
	source   *os.File
	sink     *os.File
	state    buttonState
	holdTime time.Duration
	stopCh   chan struct{}
	done     chan struct{}
}
