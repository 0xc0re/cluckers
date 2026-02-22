package ui

import (
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"golang.org/x/term"
)

// StepSpinner wraps a terminal spinner for pipeline step feedback.
type StepSpinner struct {
	name    string
	spinner *spinner.Spinner
	isTTY   bool
}

// StartStep creates and starts a spinner with the given step name.
// When stdout is not a TTY, the spinner is skipped and only the step name is printed.
func StartStep(name string) *StepSpinner {
	isTTY := term.IsTerminal(int(os.Stdout.Fd()))

	ss := &StepSpinner{
		name:  name,
		isTTY: isTTY,
	}

	if isTTY {
		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Suffix = "  " + name
		s.Color("cyan")
		s.Start()
		ss.spinner = s
	} else {
		// Non-TTY: just print the step name for CI/logging.
		fmt.Printf("  %s...\n", name)
	}

	return ss
}

// Stop stops the spinner animation without printing any status.
// Use this before interactive prompts that need clean stdout.
func (ss *StepSpinner) Stop() {
	if ss.spinner != nil {
		ss.spinner.Stop()
		ss.spinner = nil
	}
}

// Success stops the spinner and prints a green checkmark with the step name.
func (ss *StepSpinner) Success() {
	ss.Stop()
	if ss.isTTY {
		Success(ss.name)
	}
}

// Fail stops the spinner and prints a red cross with the step name.
func (ss *StepSpinner) Fail() {
	ss.Stop()
	if ss.isTTY {
		Error(ss.name)
	}
}
