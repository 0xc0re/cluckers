package launch

// StepStatus represents the status of a pipeline step.
type StepStatus int

const (
	// StepPending indicates the step has not yet started.
	StepPending StepStatus = iota
	// StepRunning indicates the step is currently executing.
	StepRunning
	// StepDone indicates the step completed successfully.
	StepDone
	// StepFailed indicates the step failed with an error.
	StepFailed
	// StepSkipped indicates the step was skipped.
	StepSkipped
)

// ProgressReporter receives callbacks about pipeline step progress.
// Implementations can render step progress in different UIs (terminal spinner, GUI, etc.).
type ProgressReporter interface {
	StepStarted(name string)
	StepCompleted(name string)
	StepFailed(name string, err error)
	StepSkipped(name string)
	StepPaused(name string)
}
