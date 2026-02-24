//go:build gui

package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/0xc0re/cluckers/internal/config"
	guiassets "github.com/0xc0re/cluckers/internal/gui/assets"
)

// Run starts the Fyne GUI application. This is the main entry point for GUI mode.
// It creates a window with a placeholder layout (logo + label) and applies the
// custom Cluckers dark theme. The placeholder content will be replaced by login
// and main screens in subsequent plans.
func Run(_ *config.Config) error {
	a := app.New()
	a.Settings().SetTheme(NewCluckersTheme())

	w := a.NewWindow("Cluckers")

	// Placeholder content: centered logo with title label.
	logo := canvas.NewImageFromResource(guiassets.LogoResource())
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(128, 128))

	title := widget.NewLabelWithStyle(
		"Cluckers Central",
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	subtitle := widget.NewLabelWithStyle(
		"Realm Royale on Project Crown",
		fyne.TextAlignCenter,
		fyne.TextStyle{Italic: true},
	)

	content := container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(logo),
		container.NewCenter(title),
		container.NewCenter(subtitle),
		layout.NewSpacer(),
	)

	w.SetContent(content)

	// Steam Deck: fullscreen. Desktop: windowed.
	if isSteamDeck() {
		w.SetFullScreen(true)
	} else {
		w.Resize(fyne.NewSize(480, 640))
	}

	w.ShowAndRun()
	return nil
}
