//go:build gui

package screens

import (
	"context"
	"image/color"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/0xc0re/cluckers/internal/auth"
	"github.com/0xc0re/cluckers/internal/config"
	"github.com/0xc0re/cluckers/internal/gateway"
	guiassets "github.com/0xc0re/cluckers/internal/gui/assets"
)

// MakeLoginScreen builds the login screen with logo, username/password fields,
// login button, create account button, and inline error display. On successful
// login, credentials are saved and onSuccess is called with the username and
// password. The onRegister callback navigates to the registration screen.
func MakeLoginScreen(w fyne.Window, cfg *config.Config, onSuccess func(username, password string), onRegister func()) fyne.CanvasObject {
	// Logo.
	logo := canvas.NewImageFromResource(guiassets.LogoResource())
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(128, 128))

	// Title.
	title := widget.NewRichTextFromMarkdown("# Cluckers Central")
	title.Wrapping = fyne.TextWrapOff

	// Subtitle.
	subtitle := widget.NewLabelWithStyle(
		"Realm Royale on Project Crown",
		fyne.TextAlignCenter,
		fyne.TextStyle{Italic: true},
	)

	// Username entry.
	usernameEntry := widget.NewEntry()
	usernameEntry.PlaceHolder = "Username"

	// Password entry.
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.PlaceHolder = "Password"

	// Error label (initially empty/hidden).
	errorLabel := canvas.NewText("", color.NRGBA{R: 255, G: 80, B: 80, A: 255})
	errorLabel.TextSize = 13
	errorLabel.Alignment = fyne.TextAlignCenter

	// Login button.
	loginBtn := widget.NewButton("Login", nil)
	loginBtn.Importance = widget.HighImportance

	// Login handler.
	doLogin := func() {
		username := usernameEntry.Text
		password := passwordEntry.Text

		if username == "" || password == "" {
			errorLabel.Text = "Please enter both username and password"
			errorLabel.Refresh()
			return
		}

		// Disable button and clear previous error.
		loginBtn.Disable()
		errorLabel.Text = ""
		errorLabel.Refresh()

		go func() {
			client := gateway.NewClient(cfg.Gateway, cfg.Verbose)
			result, err := auth.Login(context.Background(), client, username, password)
			if err != nil {
				fyne.Do(func() {
					// Extract user-friendly message.
					errorLabel.Text = err.Error()
					errorLabel.Refresh()
					loginBtn.Enable()
				})
				return
			}

			// Save credentials for future launches (non-fatal on failure).
			if err := auth.SaveCredentials(username, password); err != nil {
				log.Printf("WARNING: could not save credentials: %s", err)
			}

			// Cache the access token so downstream features (bot names) work without re-auth.
			if err := auth.SaveTokenCache(&auth.TokenCache{
				Username:       username,
				AccessToken:    result.AccessToken,
				AccessCachedAt: time.Now(),
			}); err != nil {
				log.Printf("WARNING: could not save token cache: %s", err)
			}

			fyne.Do(func() {
				onSuccess(username, password)
			})
		}()
	}

	loginBtn.OnTapped = doLogin

	// Allow Enter key to submit from password field.
	passwordEntry.OnSubmitted = func(_ string) {
		doLogin()
	}

	// Form layout: fixed-width entries centered horizontally.
	formWidth := float32(300)
	formHeight := float32(40)

	// Create Account button.
	registerBtn := widget.NewButton("Create Account", onRegister)

	usernameRow := container.NewGridWrap(fyne.NewSize(formWidth, formHeight), usernameEntry)
	passwordRow := container.NewGridWrap(fyne.NewSize(formWidth, formHeight), passwordEntry)
	buttonRow := container.NewGridWrap(fyne.NewSize(formWidth, formHeight), loginBtn)
	registerRow := container.NewGridWrap(fyne.NewSize(formWidth, formHeight), registerBtn)

	// Vertical form stack.
	form := container.NewVBox(
		container.NewCenter(logo),
		container.NewCenter(title),
		container.NewCenter(subtitle),
		widget.NewSeparator(),
		container.NewCenter(usernameRow),
		container.NewCenter(passwordRow),
		container.NewCenter(errorLabel),
		container.NewCenter(buttonRow),
		container.NewCenter(registerRow),
	)

	// Center the form vertically within the window.
	return container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(form),
		layout.NewSpacer(),
	)
}
