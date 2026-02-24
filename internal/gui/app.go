//go:build gui

package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

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

// showMainView sets the window content to the main application view with
// launch button, bot name field, settings navigation, and logout.
func showMainView(w fyne.Window, cfg *config.Config, username, password string) {
	content := screens.MakeMainView(w, cfg, username,
		func() {
			// onLogout: return to login screen.
			showLoginScreen(w, cfg)
		},
		func() {
			// onSettings: navigate to settings view.
			showSettingsView(w, cfg, username, password)
		},
	)
	w.SetContent(content)
}

// showSettingsView sets the window content to the settings screen.
// Back button returns to the main view.
func showSettingsView(w fyne.Window, cfg *config.Config, username, password string) {
	content := screens.MakeSettingsView(w, cfg, func() {
		// onBack: return to main view.
		showMainView(w, cfg, username, password)
	})
	w.SetContent(content)
}
