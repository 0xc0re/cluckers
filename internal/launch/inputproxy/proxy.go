//go:build linux

package inputproxy

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kenshaw/evdev"
	"golang.org/x/sys/unix"
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
	buttons         uint32    // Bitmask of currently pressed buttons
	lTrig           uint8     // Left trigger value (ABS_Z)
	rTrig           uint8     // Right trigger value (ABS_RZ)
	lastNonZero     time.Time // Last time any button/trigger was non-zero
	hadTrigActivity bool      // Whether triggers have ever been non-zero
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

// InputProxy reads evdev events from the Steam Input virtual pad and forwards
// them to a virtual Xbox 360 gamepad created via /dev/uinput. During
// ServerTravel transitions (when Steam Input firmware zeros all button data),
// the proxy holds the last-known button state instead of forwarding zeros.
type InputProxy struct {
	source    *os.File      // Steam Input evdev device (read)
	sourceDev *evdev.Evdev  // kenshaw/evdev wrapper for Poll()
	sink      *os.File      // /dev/uinput virtual gamepad (write)
	grabbed   []*os.File    // All Steam Input pads with EVIOCGRAB (for cleanup)
	state     buttonState   // Dead reckoning shadow state
	holdTime  time.Duration // How long to hold state on zero detection
	stopCh    chan struct{}  // Signal to stop the proxy
	done      chan struct{}  // Closed when proxy goroutine exits
}

// NewInputProxy creates a new InputProxy with the given hold timeout.
// If holdTimeout is zero, DefaultHoldTimeout is used.
// The proxy is not started until Start() is called.
func NewInputProxy(holdTimeout time.Duration) *InputProxy {
	if holdTimeout == 0 {
		holdTimeout = DefaultHoldTimeout
	}
	return &InputProxy{
		holdTime: holdTimeout,
		stopCh:   make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Start detects the Steam Input virtual pad, creates a virtual Xbox 360
// gamepad, and begins forwarding events in a background goroutine.
// Returns an error if the source device or uinput cannot be opened.
// The caller decides whether to treat errors as fatal.
func (p *InputProxy) Start(ctx context.Context) error {
	// Find all Steam Input virtual pads.
	allPads := FindAllSteamInputPads()
	if len(allPads) == 0 {
		return fmt.Errorf("detecting Steam Input pad: Steam Input virtual pad not found -- is Steam running?")
	}

	// Open the primary source device with a raw fd for EVIOCGRAB, then wrap
	// with kenshaw/evdev for event polling.
	sourcePath := allPads[0]
	sourceFile, err := os.OpenFile(sourcePath, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("opening source device %s: %w", sourcePath, err)
	}
	p.source = sourceFile

	// EVIOCGRAB gives exclusive access — Wine/SDL can no longer read this device.
	if err := unix.IoctlSetInt(int(sourceFile.Fd()), uint(eviocgrab), 1); err != nil {
		sourceFile.Close()
		p.source = nil
		return fmt.Errorf("grabbing source device %s: %w", sourcePath, err)
	}

	// Wrap the grabbed fd with kenshaw/evdev for Poll().
	dev := evdev.Open(sourceFile)
	p.sourceDev = dev

	// Grab any additional Steam Input pads to prevent Wine from seeing them.
	for _, padPath := range allPads[1:] {
		f, err := os.OpenFile(padPath, os.O_RDONLY, 0)
		if err != nil {
			continue // Best effort
		}
		if err := unix.IoctlSetInt(int(f.Fd()), uint(eviocgrab), 1); err != nil {
			f.Close()
			continue
		}
		p.grabbed = append(p.grabbed, f)
	}

	// Open /dev/uinput and create virtual Xbox 360 gamepad.
	sink, err := openUinput()
	if err != nil {
		p.releaseGrabs()
		p.sourceDev = nil
		sourceFile.Close()
		p.source = nil
		return fmt.Errorf("opening uinput: %w", err)
	}
	p.sink = sink

	if err := createVirtualXbox360(sink); err != nil {
		p.releaseGrabs()
		p.sourceDev = nil
		sourceFile.Close()
		p.source = nil
		sink.Close()
		p.sink = nil
		return fmt.Errorf("creating virtual gamepad: %w", err)
	}

	// Small delay for udev to register the new device before the game opens it.
	time.Sleep(100 * time.Millisecond)

	// Start event forwarding goroutine.
	go p.run(ctx)

	return nil
}

// releaseGrabs releases EVIOCGRAB on all grabbed secondary pads.
func (p *InputProxy) releaseGrabs() {
	for _, f := range p.grabbed {
		unix.IoctlSetInt(int(f.Fd()), uint(eviocgrab), 0)
		f.Close()
	}
	p.grabbed = nil
}

// run is the main event forwarding loop. It reads events from the Steam Input
// virtual pad via kenshaw/evdev Poll and forwards them to the uinput sink,
// applying dead reckoning for button events and Y-axis inversion for sticks.
func (p *InputProxy) run(ctx context.Context) {
	defer close(p.done)

	// Create a child context that we cancel when stopCh is closed.
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		select {
		case <-p.stopCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	ch := p.sourceDev.Poll(ctx)
	for envelope := range ch {
		// Check for stop signal.
		select {
		case <-p.stopCh:
			cancel()
			return
		default:
		}

		ev := envelope.Event
		evType := uint16(ev.Type)
		code := ev.Code
		value := ev.Value

		switch evType {
		case evKey:
			// Button event: update shadow state, apply dead reckoning.
			p.state.updateButton(code, value)
			if p.state.shouldHold(p.holdTime) {
				// ServerTravel detected -- suppress zero forwarding.
				continue
			}
			writeEvent(p.sink, evType, code, value)

		case evAbs:
			// Axis event: invert Y axes, track triggers.
			switch code {
			case absY, absRY:
				value = invertY(value)
			case absZ, absRZ:
				p.state.updateTrigger(code, value)
			}
			writeEvent(p.sink, evType, code, value)

		case evSyn:
			// Sync event: always forward.
			writeEvent(p.sink, evType, code, value)
		}
	}
}

// Stop signals the proxy to stop and waits for the goroutine to exit.
// It cleans up the virtual device and closes both source and sink file
// descriptors. Safe to call multiple times.
func (p *InputProxy) Stop() {
	// Signal the run goroutine to stop.
	select {
	case <-p.stopCh:
		// Already closed.
		return
	default:
		close(p.stopCh)
	}

	// Wait for goroutine exit with timeout to prevent hanging.
	select {
	case <-p.done:
	case <-time.After(2 * time.Second):
	}

	// Clean up virtual device and file descriptors.
	if p.sink != nil {
		destroyVirtualDevice(p.sink)
		p.sink.Close()
		p.sink = nil
	}
	if p.source != nil {
		// Release EVIOCGRAB on primary source.
		unix.IoctlSetInt(int(p.source.Fd()), uint(eviocgrab), 0)
	}
	if p.sourceDev != nil {
		p.sourceDev.Close()
		p.sourceDev = nil
	}
	if p.source != nil {
		p.source.Close()
		p.source = nil
	}
	// Release grabs on secondary pads.
	p.releaseGrabs()
}

// Cleanup returns a function suitable for deferred cleanup calls.
// The returned function calls Stop() if the proxy was successfully started.
// Safe to call on a nil InputProxy (returns a no-op).
func (p *InputProxy) Cleanup() func() {
	if p == nil {
		return func() {}
	}
	return func() {
		p.Stop()
	}
}
