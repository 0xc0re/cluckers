//go:build gui

package launch

import "fyne.io/fyne/v2"

// GUIReporter implements ProgressReporter for the Fyne GUI.
// It bridges pipeline step progress callbacks to the GUI step list widget
// by calling updateFn wrapped in fyne.Do() for thread safety.
type GUIReporter struct {
	updateFn func(name string, status StepStatus)
}

// NewGUIReporter creates a GUIReporter that calls updateFn on each step status change.
// The updateFn is automatically wrapped in fyne.Do() to ensure thread-safe GUI updates
// since the pipeline runs in a background goroutine.
func NewGUIReporter(updateFn func(name string, status StepStatus)) *GUIReporter {
	return &GUIReporter{updateFn: updateFn}
}

// StepStarted marks the step as running in the GUI.
func (g *GUIReporter) StepStarted(name string) {
	fyne.Do(func() { g.updateFn(name, StepRunning) })
}

// StepCompleted marks the step as done in the GUI.
func (g *GUIReporter) StepCompleted(name string) {
	fyne.Do(func() { g.updateFn(name, StepDone) })
}

// StepFailed marks the step as failed in the GUI.
func (g *GUIReporter) StepFailed(name string, err error) {
	fyne.Do(func() { g.updateFn(name, StepFailed) })
}

// StepSkipped marks the step as skipped in the GUI.
func (g *GUIReporter) StepSkipped(name string) {
	fyne.Do(func() { g.updateFn(name, StepSkipped) })
}

// StepPaused is a no-op in the GUI. Download progress is shown via step status
// rather than a terminal progress bar.
func (g *GUIReporter) StepPaused(name string) {
	// No-op: GUI does not use terminal progress bars.
}
