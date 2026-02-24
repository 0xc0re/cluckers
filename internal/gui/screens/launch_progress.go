//go:build gui

package screens

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/0xc0re/cluckers/internal/config"
	guiassets "github.com/0xc0re/cluckers/internal/gui/assets"
	"github.com/0xc0re/cluckers/internal/gui/widgets"
	"github.com/0xc0re/cluckers/internal/launch"
)

// MakeLaunchProgressView builds the launch progress screen that shows pipeline
// steps with live status updates (pending -> running -> done/failed).
// The pipeline runs in a background goroutine. All UI updates are thread-safe
// via fyne.Do() in the GUIReporter.
//
// Parameters:
//   - w: the application window
//   - cfg: application configuration
//   - username, password: authenticated credentials for pipeline
//   - onComplete: called when the pipeline finishes successfully (game launched)
//   - onError: called when the pipeline fails with the error
func MakeLaunchProgressView(w fyne.Window, cfg *config.Config, username, password string, onComplete func(), onError func(error)) fyne.CanvasObject {
	// Logo (small, at top).
	logo := canvas.NewImageFromResource(guiassets.LogoResource())
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(48, 48))

	// Title.
	title := widget.NewLabelWithStyle(
		"Launching...",
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	// Build step names from the pipeline.
	stepNames := launch.StepNames(cfg)

	// Create step list widget.
	stepList := widgets.NewStepListWidget(stepNames)

	// Cancel context for the pipeline.
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel button.
	cancelBtn := widget.NewButton("Cancel", nil)
	cancelBtn.OnTapped = func() {
		cancel()
	}

	// Start pipeline in background goroutine.
	go func() {
		reporter := launch.NewGUIReporter(func(name string, status launch.StepStatus) {
			stepList.UpdateStep(name, status)
		})
		err := launch.RunWithReporterAndCreds(ctx, cfg, reporter, username, password)
		if err != nil {
			fyne.Do(func() { onError(err) })
			return
		}
		fyne.Do(func() { onComplete() })
	}()

	// Assemble layout.
	content := container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(logo),
		container.NewCenter(title),
		widget.NewSeparator(),
		container.NewCenter(stepList.GetContainer()),
		widget.NewSeparator(),
		container.NewCenter(
			container.NewGridWrap(fyne.NewSize(200, 40), cancelBtn),
		),
		layout.NewSpacer(),
	)

	return content
}
