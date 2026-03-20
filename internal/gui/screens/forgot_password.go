//go:build gui

package screens

import (
	"context"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/0xc0re/cluckers/internal/auth"
	"github.com/0xc0re/cluckers/internal/config"
	"github.com/0xc0re/cluckers/internal/gateway"
	guiassets "github.com/0xc0re/cluckers/internal/gui/assets"
)

// MakeForgotPasswordScreen builds the password reset screen with a username
// entry and a request button. On success, a dialog confirms the reset was
// requested and the user returns to the login screen via onBackToLogin.
func MakeForgotPasswordScreen(w fyne.Window, cfg *config.Config, onBackToLogin func()) fyne.CanvasObject {
	// Logo.
	logo := canvas.NewImageFromResource(guiassets.LogoResource())
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(128, 128))

	// Title.
	title := widget.NewRichTextFromMarkdown("# Reset Password")
	title.Wrapping = fyne.TextWrapOff

	// Subtitle.
	subtitle := widget.NewLabelWithStyle(
		"Enter your username to request a password reset",
		fyne.TextAlignCenter,
		fyne.TextStyle{Italic: true},
	)

	// Username entry.
	usernameEntry := widget.NewEntry()
	usernameEntry.PlaceHolder = "Username"

	// Error label (initially empty/hidden).
	errorLabel := canvas.NewText("", color.NRGBA{R: 255, G: 80, B: 80, A: 255})
	errorLabel.TextSize = 13
	errorLabel.Alignment = fyne.TextAlignCenter

	// Request reset button.
	resetBtn := widget.NewButton("Request Reset", nil)
	resetBtn.Importance = widget.HighImportance

	// Back to login button.
	backBtn := widget.NewButton("Back to Login", onBackToLogin)

	// Reset handler.
	doReset := func() {
		username := usernameEntry.Text

		if username == "" {
			errorLabel.Text = "Please enter your username"
			errorLabel.Refresh()
			return
		}

		// Disable button and clear previous error.
		resetBtn.Disable()
		errorLabel.Text = ""
		errorLabel.Refresh()

		go func() {
			client := gateway.NewClient(cfg.Gateway, cfg.Verbose)
			err := auth.RequestPasswordReset(context.Background(), client, username)
			if err != nil {
				fyne.Do(func() {
					errorLabel.Text = formatGUIError(err)
					errorLabel.Refresh()
					resetBtn.Enable()
				})
				return
			}

			fyne.Do(func() {
				d := dialog.NewInformation("Password Reset Requested",
					"Check your email or Discord for reset instructions.",
					w)
				d.SetOnClosed(onBackToLogin)
				d.Show()
			})
		}()
	}

	resetBtn.OnTapped = doReset

	// Allow Enter key to submit from username field.
	usernameEntry.OnSubmitted = func(_ string) {
		doReset()
	}

	// Form layout: fixed-width entries centered horizontally.
	formWidth := float32(300)
	formHeight := float32(40)

	usernameRow := container.NewGridWrap(fyne.NewSize(formWidth, formHeight), usernameEntry)
	buttonRow := container.NewGridWrap(fyne.NewSize(formWidth, formHeight), resetBtn)
	backRow := container.NewGridWrap(fyne.NewSize(formWidth, formHeight), backBtn)

	// Vertical form stack.
	form := container.NewVBox(
		container.NewCenter(logo),
		container.NewCenter(title),
		container.NewCenter(subtitle),
		widget.NewSeparator(),
		container.NewCenter(usernameRow),
		container.NewCenter(errorLabel),
		container.NewCenter(buttonRow),
		container.NewCenter(backRow),
	)

	// Center the form vertically within the window.
	return container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(form),
		layout.NewSpacer(),
	)
}
