//go:build gui

package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/0xc0re/cluckers/internal/auth"
	"github.com/0xc0re/cluckers/internal/config"
	"github.com/0xc0re/cluckers/internal/gui/screens"
	"github.com/0xc0re/cluckers/internal/ui"
)

// Run starts the Fyne GUI application. This is the main entry point for GUI mode.
// It checks for saved credentials and either shows the login screen or skips
// directly to the main view.
func Run(cfg *config.Config) error {
	a := app.New()
	a.Settings().SetTheme(NewCluckersTheme())

	w := a.NewWindow("Cluckers")

	// Check for saved credentials.
	creds, err := auth.LoadCredentials()
	if err != nil {
		ui.Verbose("Could not load saved credentials: "+err.Error(), cfg.Verbose)
	}

	if creds != nil && creds.Username != "" && creds.Password != "" {
		// Saved credentials exist -- skip login and go to main view.
		showMainView(w, cfg, creds.Username, creds.Password)
	} else {
		// No saved credentials -- show login screen.
		showLoginScreen(w, cfg)
	}

	// Steam Deck: fullscreen. Desktop: windowed.
	if isSteamDeck() {
		w.SetFullScreen(true)
	} else {
		w.Resize(fyne.NewSize(480, 640))
	}

	w.ShowAndRun()
	return nil
}

// showLoginScreen sets the window content to the login screen. On successful
// login, it transitions to the main view.
func showLoginScreen(w fyne.Window, cfg *config.Config) {
	loginContent := screens.MakeLoginScreen(w, cfg, func(username, password string) {
		showMainView(w, cfg, username, password)
	})
	w.SetContent(loginContent)
}

// showMainView sets the window content to the main application view.
// This is a stub that will be fully implemented in plan 03.
func showMainView(w fyne.Window, cfg *config.Config, username, password string) {
	_ = cfg // Will be used in plan 03 for launch pipeline integration.

	welcomeLabel := widget.NewLabelWithStyle(
		"Welcome, "+username,
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	launchBtn := widget.NewButton("Launch", nil)
	launchBtn.Importance = widget.HighImportance
	launchBtn.OnTapped = func() {
		// Stub: plan 03 will implement the full launch flow with ProgressReporter.
		launchBtn.SetText("Launching...")
		launchBtn.Disable()
	}

	logoutBtn := widget.NewButton("Logout", nil)
	logoutBtn.OnTapped = func() {
		_ = auth.DeleteCredentials()
		_ = auth.ClearTokenCache()
		showLoginScreen(w, cfg)
	}

	content := container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(welcomeLabel),
		container.NewCenter(
			container.NewGridWrap(fyne.NewSize(300, 40), launchBtn),
		),
		container.NewCenter(
			container.NewGridWrap(fyne.NewSize(300, 40), logoutBtn),
		),
		layout.NewSpacer(),
	)

	w.SetContent(content)
}
