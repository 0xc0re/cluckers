//go:build linux

package inputproxy

import (
	"testing"
	"time"
)

func TestDeadReckoningHold(t *testing.T) {
	// When all buttons AND triggers are non-zero, then ALL go to zero
	// simultaneously, shouldHold(timeout) returns true if within timeout window.
	s := &buttonState{}

	// Set buttons=BTN_A pressed, triggers non-zero
	s.updateButton(btnA, 1)
	s.updateTrigger(absZ, 100)
	s.updateTrigger(absRZ, 50)

	// Now zero everything simultaneously
	s.updateButton(btnA, 0)
	s.updateTrigger(absZ, 0)
	s.updateTrigger(absRZ, 0)

	// Should hold because we recently had non-zero state and now all are zero
	if !s.shouldHold(500 * time.Millisecond) {
		t.Error("expected shouldHold to return true immediately after zeroing all buttons/triggers")
	}
}

func TestDeadReckoningTimeout(t *testing.T) {
	// After hold window expires, shouldHold returns false.
	s := &buttonState{}

	// Set non-zero state
	s.updateButton(btnA, 1)
	s.updateTrigger(absZ, 100)

	// Zero everything
	s.updateButton(btnA, 0)
	s.updateTrigger(absZ, 0)

	// Wait past the timeout
	time.Sleep(60 * time.Millisecond)

	// Should NOT hold because timeout expired
	if s.shouldHold(50 * time.Millisecond) {
		t.Error("expected shouldHold to return false after timeout expired")
	}
}

func TestDeadReckoningNormalRelease(t *testing.T) {
	// Single button release does NOT trigger hold.
	// Case 1: Only one button was pressed (no triggers), release it.
	s := &buttonState{}

	// Set only BTN_A pressed, triggers at zero
	s.updateButton(btnA, 1)

	// Release BTN_A -- triggers were never non-zero, so this is a normal release
	s.updateButton(btnA, 0)

	// Should NOT hold because triggers were never non-zero before zeroing
	if s.shouldHold(500 * time.Millisecond) {
		t.Error("expected shouldHold to return false for normal single-button release without prior trigger activity")
	}

	// Case 2: Multiple buttons pressed, release one -- not all zero
	s2 := &buttonState{}
	s2.updateButton(btnA, 1)
	s2.updateButton(btnB, 1)
	s2.updateTrigger(absZ, 100)

	// Release only A -- B and trigger still held
	s2.updateButton(btnA, 0)

	// Should NOT hold because not all buttons/triggers are zero
	if s2.shouldHold(500 * time.Millisecond) {
		t.Error("expected shouldHold to return false when not all buttons/triggers are zero")
	}
}

func TestDeadReckoningAxesNotConsidered(t *testing.T) {
	// Axes (sticks) don't affect hold logic.
	// If buttons and triggers are zero, hold state should not depend on axis values.
	s := &buttonState{}

	// Set buttons and triggers non-zero
	s.updateButton(btnA, 1)
	s.updateTrigger(absZ, 200)

	// Zero buttons and triggers (axes would still have values in a real scenario,
	// but axes are not tracked in buttonState)
	s.updateButton(btnA, 0)
	s.updateTrigger(absZ, 0)

	// Should hold -- axis values are irrelevant
	if !s.shouldHold(500 * time.Millisecond) {
		t.Error("expected shouldHold to return true -- axes should not affect hold logic")
	}

	// Verify that buttonState has no axis tracking
	// (sticks are not part of isAllButtonsZero check)
	s2 := &buttonState{}
	if !s2.isAllButtonsZero() {
		t.Error("fresh buttonState should have all buttons zero")
	}
}

func TestButtonStateUpdate(t *testing.T) {
	// Verify button shadow state tracking via bitmask.
	s := &buttonState{}

	// Initially all zero
	if !s.isAllButtonsZero() {
		t.Error("initial state should have all buttons zero")
	}

	// Press BTN_A
	s.updateButton(btnA, 1)
	if s.isAllButtonsZero() {
		t.Error("should not be all zero after pressing BTN_A")
	}

	// Press BTN_B independently
	s.updateButton(btnB, 1)
	if s.isAllButtonsZero() {
		t.Error("should not be all zero after pressing BTN_A and BTN_B")
	}

	// Release BTN_A -- BTN_B still pressed
	s.updateButton(btnA, 0)
	if s.isAllButtonsZero() {
		t.Error("should not be all zero with BTN_B still pressed")
	}

	// Release BTN_B -- now all zero
	s.updateButton(btnB, 0)
	if s.buttons != 0 {
		t.Errorf("buttons should be 0 after releasing all, got 0x%x", s.buttons)
	}

	// Trigger state tracking
	s.updateTrigger(absZ, 128)
	if s.isAllButtonsZero() {
		t.Error("should not be all zero with left trigger active")
	}
	s.updateTrigger(absZ, 0)
	s.updateTrigger(absRZ, 64)
	if s.isAllButtonsZero() {
		t.Error("should not be all zero with right trigger active")
	}
	s.updateTrigger(absRZ, 0)
	if !s.isAllButtonsZero() {
		t.Error("should be all zero after clearing all triggers")
	}
}

func TestAxisYInversion(t *testing.T) {
	// Verify Y-axis negation with overflow clamping.
	tests := []struct {
		name  string
		input int32
		want  int32
	}{
		{"positive to negative", 16384, -16384},
		{"negative to positive", -16384, 16384},
		{"zero stays zero", 0, 0},
		{"max positive", 32767, -32767},
		{"min negative clamped", -32768, 32767}, // -(-32768) = 32768 > int16 max, clamp to 32767
		{"one", 1, -1},
		{"minus one", -1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := invertY(tt.input)
			if got != tt.want {
				t.Errorf("invertY(%d) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestAxisXNotInverted(t *testing.T) {
	// X-axis values should pass through unchanged.
	// This is a documentation test -- invertY is only called for Y axes.
	// Verify that invertY correctly negates (and that X values should NOT go through it).
	got := invertY(16384)
	if got == 16384 {
		t.Error("invertY should negate the value, not pass through")
	}
}
