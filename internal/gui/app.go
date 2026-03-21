//go:build gui

package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/0xc0re/cluckers/internal/auth"
	"github.com/0xc0re/cluckers/internal/config"
	guiassets "github.com/0xc0re/cluckers/internal/gui/assets"
	"github.com/0xc0re/cluckers/internal/gui/screens"
	"github.com/0xc0re/cluckers/internal/ui"
)

// gameRunning tracks whether the game process has been launched. When true,
// the window close button hides to system tray instead of quitting.
var gameRunning bool

// Run starts the Fyne GUI application. This is the main entry point for GUI mode.
// It checks for saved credentials and either shows the login screen or skips
// directly to the main view.
func Run(cfg *config.Config) error {
	unlock, err := tryLock()
	if err != nil {
		ui.Warn("Cluckers is already running.")
		return nil
	}
	defer unlock()

	a := app.NewWithID("com.projectcrown.cluckers")
	a.SetIcon(guiassets.LogoResource())
	a.Settings().SetTheme(NewCluckersTheme())

	w := a.NewWindow("Cluckers")

	// System tray setup (desktop only, not Steam Deck).
	if !isSteamDeck() {
		if deskApp, ok := a.(desktop.App); ok {
			deskApp.SetSystemTrayIcon(guiassets.LogoResource())
			showItem := fyne.NewMenuItem("Show Cluckers", func() {
				w.Show()
				w.RequestFocus()
			})
			quitItem := fyne.NewMenuItem("Quit", func() {
				a.Quit()
			})
			deskApp.SetSystemTrayMenu(fyne.NewMenu("Cluckers", showItem, quitItem))
		}

		// Close intercept: hide to tray when game is running, quit otherwise.
		w.SetCloseIntercept(func() {
			if gameRunning {
				w.Hide()
			} else {
				a.Quit()
			}
		})
	}

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
		w.Resize(fyne.NewSize(560, 640))
	}

	w.ShowAndRun()
	return nil
}

// showLoginScreen sets the window content to the login screen. On successful
// login, it transitions to the main view. The "Create Account" button navigates
// to the registration screen.
func showLoginScreen(w fyne.Window, cfg *config.Config) {
	loginContent := screens.MakeLoginScreen(w, cfg, func(username, password string) {
		showMainView(w, cfg, username, password)
	}, func() {
		showRegisterScreen(w, cfg)
	}, func() {
		showForgotPasswordScreen(w, cfg)
	})
	w.SetContent(loginContent)
}

// showForgotPasswordScreen sets the window content to the password reset screen.
// The "Back to Login" button returns to the login screen.
func showForgotPasswordScreen(w fyne.Window, cfg *config.Config) {
	content := screens.MakeForgotPasswordScreen(w, cfg, func() {
		showLoginScreen(w, cfg)
	})
	w.SetContent(content)
}

// showRegisterScreen sets the window content to the registration screen. On
// successful registration (with or without Discord linking), it transitions to
// the main view. The "Back to Login" button returns to the login screen.
func showRegisterScreen(w fyne.Window, cfg *config.Config) {
	content := screens.MakeRegisterScreen(w, cfg, func(username, password string) {
		showMainView(w, cfg, username, password)
	}, func() {
		showLoginScreen(w, cfg)
	})
	w.SetContent(content)
}

// showMainView sets the window content to the main application view with
// launch button, game management, links, and navigation.
func showMainView(w fyne.Window, cfg *config.Config, username, password string) {
	content := screens.MakeMainView(w, cfg, username, password,
		func() {
			// onLaunch: transition to launch progress view.
			showLaunchProgress(w, cfg, username, password)
		},
		func() {
			// onLogout: clear credentials and return to login screen.
			if err := auth.DeleteCredentials(); err != nil {
				ui.Warn(fmt.Sprintf("could not delete credentials: %s", err))
			}
			if err := auth.ClearTokenCache(); err != nil {
				ui.Warn(fmt.Sprintf("could not clear token cache: %s", err))
			}
			showLoginScreen(w, cfg)
		},
		func() {
			// onSettings: navigate to settings screen.
			showSettingsView(w, cfg, username, password)
		},
	)
	w.SetContent(content)
}

// showLaunchProgress sets the window content to the launch progress view.
// On successful pipeline completion, the window hides to system tray (desktop)
// or closes (Steam Deck). The main view content is restored so the window is
// ready if the user returns from the tray.
// On error, an error dialog is shown and the user returns to the main view.
func showLaunchProgress(w fyne.Window, cfg *config.Config, username, password string) {
	content := screens.MakeLaunchProgressView(w, cfg, username, password,
		func() {
			// onComplete: game has launched.
			gameRunning = true
			// Restore main view content so tray restore shows the main view.
			showMainView(w, cfg, username, password)
			if isSteamDeck() {
				// Steam Deck: no tray, just close.
				w.Close()
			} else {
				// Desktop: hide to system tray.
				w.Hide()
			}
		},
		func(err error) {
			// onError: show error dialog, then return to main view.
			dialog.ShowError(err, w)
			showMainView(w, cfg, username, password)
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
