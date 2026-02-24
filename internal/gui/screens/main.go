//go:build gui

package screens

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/0xc0re/cluckers/internal/auth"
	"github.com/0xc0re/cluckers/internal/config"
)

// MakeMainView builds the main application view with welcome message,
// launch button, bot name field, and navigation to settings and logout.
func MakeMainView(w fyne.Window, cfg *config.Config, username string, onLogout func(), onSettings func()) fyne.CanvasObject {
	_ = cfg // Will be used for launch pipeline integration in plan 03.

	// Welcome label.
	welcomeLabel := widget.NewLabelWithStyle(
		"Welcome, "+username,
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	// Launch button.
	launchBtn := widget.NewButton("Launch", nil)
	launchBtn.Importance = widget.HighImportance
	launchBtn.OnTapped = func() {
		// Stub: plan 03 will implement the full launch flow with ProgressReporter.
		launchBtn.SetText("Launching...")
		launchBtn.Disable()
	}

	// --- Supporter Features: Bot Name ---
	botNameEntry := widget.NewEntry()
	botNameEntry.PlaceHolder = "Set bot name (supporters only)"

	botSetBtn := widget.NewButton("Set", nil)
	botSetBtn.OnTapped = func() {
		// TODO: Implement bot name API call when gateway endpoint is documented
		dialog.ShowInformation("Bot Name", "Bot name feature coming soon.", w)
	}

	botNameRow := container.NewBorder(nil, nil, nil, botSetBtn, botNameEntry)

	botNameSection := container.NewVBox(
		widget.NewLabelWithStyle("Supporter Features", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
		container.NewCenter(
			container.NewGridWrap(fyne.NewSize(300, 36), botNameRow),
		),
	)

	// Settings button.
	settingsBtn := widget.NewButton("Settings", nil)
	settingsBtn.OnTapped = func() {
		onSettings()
	}

	// Logout button.
	logoutBtn := widget.NewButton("Logout", nil)
	logoutBtn.OnTapped = func() {
		_ = auth.DeleteCredentials()
		_ = auth.ClearTokenCache()
		onLogout()
	}

	// Fixed-width button rows.
	btnWidth := fyne.NewSize(300, 40)

	content := container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(welcomeLabel),
		container.NewCenter(
			container.NewGridWrap(btnWidth, launchBtn),
		),
		widget.NewSeparator(),
		botNameSection,
		widget.NewSeparator(),
		container.NewCenter(
			container.NewGridWrap(btnWidth, settingsBtn),
		),
		container.NewCenter(
			container.NewGridWrap(btnWidth, logoutBtn),
		),
		layout.NewSpacer(),
	)

	return content
}
