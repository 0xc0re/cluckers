package launch

import "github.com/0xc0re/cluckers/internal/ui"

// CLIReporter implements ProgressReporter for terminal output using spinners.
// It wraps the existing ui.StepSpinner behavior so that CLI pipeline output
// remains identical to the pre-refactor behavior.
type CLIReporter struct {
	activeSpinner *ui.StepSpinner
}

// NewCLIReporter creates a new CLIReporter.
func NewCLIReporter() *CLIReporter {
	return &CLIReporter{}
}

// StepStarted creates and starts a new terminal spinner for the named step.
func (c *CLIReporter) StepStarted(name string) {
	c.activeSpinner = ui.StartStep(name)
}

// StepCompleted marks the current step as successful with a green checkmark.
func (c *CLIReporter) StepCompleted(name string) {
	if c.activeSpinner != nil {
		c.activeSpinner.Success()
	}
}

// StepFailed marks the current step as failed with a red cross.
func (c *CLIReporter) StepFailed(name string, err error) {
	if c.activeSpinner != nil {
		c.activeSpinner.Fail()
	}
}

// StepSkipped marks the current step as completed (same visual as success in CLI).
func (c *CLIReporter) StepSkipped(name string) {
	if c.activeSpinner != nil {
		c.activeSpinner.Success()
	}
}

// StepPaused stops the spinner animation without printing any status indicator.
// Used when a step needs to display its own output (e.g., progress bar during download,
// or interactive prompts during authentication).
func (c *CLIReporter) StepPaused(name string) {
	if c.activeSpinner != nil {
		c.activeSpinner.Stop()
	}
}
