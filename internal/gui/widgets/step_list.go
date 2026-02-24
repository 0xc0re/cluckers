//go:build gui

package widgets

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/0xc0re/cluckers/internal/launch"
)

// StepItem represents a single pipeline step and its current status.
type StepItem struct {
	Name   string
	Status launch.StepStatus
}

// StepListWidget displays a vertical list of pipeline steps with status icons.
// Each step shows an icon indicating its state (pending, running, done, failed, skipped)
// and a text label with the step name. It is not a true Fyne widget but a container-based
// composite that exposes its layout via GetContainer().
type StepListWidget struct {
	items     []StepItem
	icons     []*widget.Icon
	labels    []*canvas.Text
	container *fyne.Container
}

// NewStepListWidget creates a step list widget with the given step names.
// All steps start in the pending state.
func NewStepListWidget(stepNames []string) *StepListWidget {
	s := &StepListWidget{
		items:  make([]StepItem, len(stepNames)),
		icons:  make([]*widget.Icon, len(stepNames)),
		labels: make([]*canvas.Text, len(stepNames)),
	}

	rows := make([]fyne.CanvasObject, 0, len(stepNames))
	for i, name := range stepNames {
		s.items[i] = StepItem{Name: name, Status: launch.StepPending}

		icon := widget.NewIcon(theme.MediaRecordIcon())
		s.icons[i] = icon

		label := canvas.NewText(name, color.NRGBA{R: 160, G: 160, B: 170, A: 255})
		label.TextSize = 14
		s.labels[i] = label

		row := container.NewHBox(icon, label)
		rows = append(rows, row)
	}

	s.container = container.NewVBox(rows...)
	return s
}

// UpdateStep updates the status of the named step and refreshes its icon and label.
func (s *StepListWidget) UpdateStep(name string, status launch.StepStatus) {
	for i, item := range s.items {
		if item.Name != name {
			continue
		}

		s.items[i].Status = status

		// Update icon based on status.
		switch status {
		case launch.StepPending:
			s.icons[i].SetResource(theme.MediaRecordIcon())
			s.labels[i].Color = color.NRGBA{R: 160, G: 160, B: 170, A: 255} // Grey
		case launch.StepRunning:
			s.icons[i].SetResource(theme.MediaPlayIcon())
			s.labels[i].Color = color.NRGBA{R: 76, G: 175, B: 80, A: 255} // Green - active
			s.labels[i].TextStyle = fyne.TextStyle{Bold: true}
		case launch.StepDone:
			s.icons[i].SetResource(theme.ConfirmIcon())
			s.labels[i].Color = color.NRGBA{R: 76, G: 175, B: 80, A: 255} // Green
			s.labels[i].TextStyle = fyne.TextStyle{}
		case launch.StepFailed:
			s.icons[i].SetResource(theme.ErrorIcon())
			s.labels[i].Color = color.NRGBA{R: 255, G: 80, B: 80, A: 255} // Red
			s.labels[i].TextStyle = fyne.TextStyle{Bold: true}
		case launch.StepSkipped:
			s.icons[i].SetResource(theme.ConfirmIcon())
			s.labels[i].Color = color.NRGBA{R: 120, G: 120, B: 130, A: 255} // Dim grey
			s.labels[i].TextStyle = fyne.TextStyle{Italic: true}
		}

		s.labels[i].Refresh()
		s.container.Refresh()
		return
	}
}

// GetContainer returns the internal container for embedding in layouts.
func (s *StepListWidget) GetContainer() *fyne.Container {
	return s.container
}
