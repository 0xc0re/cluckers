//go:build gui

package screens

import (
	"context"
	"fmt"
	"image/color"
	"time"

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
	"github.com/0xc0re/cluckers/internal/ui"
)

// MakeRegisterScreen builds the registration screen with logo, username/password/email
// fields, register button, back-to-login link, and inline error display. On successful
// registration, credentials are saved, a Discord link code is requested, and the user
// transitions to the Discord linking view or directly to the main view.
func MakeRegisterScreen(w fyne.Window, cfg *config.Config, onSuccess func(username, password string), onBackToLogin func()) fyne.CanvasObject {
	// Logo.
	logo := canvas.NewImageFromResource(guiassets.LogoResource())
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(128, 128))

	// Title.
	title := widget.NewRichTextFromMarkdown("# Create Account")
	title.Wrapping = fyne.TextWrapOff

	// Subtitle.
	subtitle := widget.NewLabelWithStyle(
		"Register for Project Crown",
		fyne.TextAlignCenter,
		fyne.TextStyle{Italic: true},
	)

	// Username entry.
	usernameEntry := widget.NewEntry()
	usernameEntry.PlaceHolder = "Username"

	// Password entry.
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.PlaceHolder = "Password"

	// Email entry.
	emailEntry := widget.NewEntry()
	emailEntry.PlaceHolder = "Email"

	// Error label (initially empty/hidden).
	errorLabel := canvas.NewText("", color.NRGBA{R: 255, G: 80, B: 80, A: 255})
	errorLabel.TextSize = 13
	errorLabel.Alignment = fyne.TextAlignCenter

	// Register button.
	registerBtn := widget.NewButton("Register", nil)
	registerBtn.Importance = widget.HighImportance

	// Back to login button.
	backBtn := widget.NewButton("Back to Login", onBackToLogin)

	// Register handler.
	doRegister := func() {
		username := usernameEntry.Text
		password := passwordEntry.Text
		email := emailEntry.Text

		if username == "" || password == "" || email == "" {
			errorLabel.Text = "Please enter username, password, and email"
			errorLabel.Refresh()
			return
		}

		// Disable button and clear previous error.
		registerBtn.Disable()
		errorLabel.Text = ""
		errorLabel.Refresh()

		go func() {
			client := gateway.NewClient(cfg.Gateway, cfg.Verbose)
			result, err := auth.Register(context.Background(), client, username, password, email)
			if err != nil {
				fyne.Do(func() {
					errorLabel.Text = formatGUIError(err)
					errorLabel.Refresh()
					registerBtn.Enable()
				})
				return
			}

			// Save credentials for future launches (non-fatal on failure).
			if err := auth.SaveCredentials(username, password); err != nil {
				ui.Warn(fmt.Sprintf("could not save credentials: %s", err))
			}

			// Cache the access token from registration (acts as auto-login).
			if err := auth.SaveTokenCache(&auth.TokenCache{
				Username:       result.Username,
				AccessToken:    result.AccessToken,
				AccessCachedAt: time.Now(),
			}); err != nil {
				ui.Warn(fmt.Sprintf("could not save token cache: %s", err))
			}

			// Request Discord link code (uses password auth).
			code, err := auth.RequestLinkCode(context.Background(), client, username, password)
			if err != nil {
				// Registration succeeded but link code failed -- warn and continue to main view.
				ui.Warn(fmt.Sprintf("could not get Discord link code: %s", err))
				fyne.Do(func() {
					d := dialog.NewInformation("Discord Linking",
						fmt.Sprintf("Could not get Discord link code: %s\nYou can request a link code later by logging in.", err),
						w)
					d.SetOnClosed(func() { onSuccess(username, password) })
					d.Show()
				})
				return
			}

			// Show Discord linking screen.
			fyne.Do(func() {
				showDiscordLinking(w, cfg, code, result.Username, result.AccessToken, username, password, onSuccess)
			})
		}()
	}

	registerBtn.OnTapped = doRegister

	// Allow Enter key to submit from email field.
	emailEntry.OnSubmitted = func(_ string) {
		doRegister()
	}

	// Form layout: fixed-width entries centered horizontally.
	formWidth := float32(300)
	formHeight := float32(40)

	usernameRow := container.NewGridWrap(fyne.NewSize(formWidth, formHeight), usernameEntry)
	passwordRow := container.NewGridWrap(fyne.NewSize(formWidth, formHeight), passwordEntry)
	emailRow := container.NewGridWrap(fyne.NewSize(formWidth, formHeight), emailEntry)
	buttonRow := container.NewGridWrap(fyne.NewSize(formWidth, formHeight), registerBtn)
	backRow := container.NewGridWrap(fyne.NewSize(formWidth, formHeight), backBtn)

	// Vertical form stack.
	form := container.NewVBox(
		container.NewCenter(logo),
		container.NewCenter(title),
		container.NewCenter(subtitle),
		widget.NewSeparator(),
		container.NewCenter(usernameRow),
		container.NewCenter(passwordRow),
		container.NewCenter(emailRow),
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

// showDiscordLinking replaces the window content with a Discord linking view
// that displays the link code and polls for linking status.
func showDiscordLinking(w fyne.Window, cfg *config.Config, code, regUsername, accessToken, username, password string, onSuccess func(username, password string)) {
	// Logo.
	logo := canvas.NewImageFromResource(guiassets.LogoResource())
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(128, 128))

	// Title.
	title := widget.NewRichTextFromMarkdown("# Discord Linking")
	title.Wrapping = fyne.TextWrapOff

	// Instruction text.
	instruction := widget.NewLabel("DM the following code to the Project Crown Discord bot:")
	instruction.Alignment = fyne.TextAlignCenter
	instruction.Wrapping = fyne.TextWrapWord

	// Code display — bold label, readable against dark theme.
	codeLabel := widget.NewLabelWithStyle(code, fyne.TextAlignCenter, fyne.TextStyle{Bold: true, Monospace: true})

	// Copy button.
	copyBtn := widget.NewButton("Copy Code", func() {
		w.Clipboard().SetContent(code)
	})
	copyBtn.Importance = widget.MediumImportance

	// Status label.
	statusLabel := widget.NewLabel("Waiting for Discord linking...")
	statusLabel.Alignment = fyne.TextAlignCenter

	// Cancellable context for the polling goroutine.
	ctx, cancelFunc := context.WithCancel(context.Background())

	// Continue without linking button.
	continueBtn := widget.NewButton("Continue Without Linking", func() {
		cancelFunc()
		onSuccess(username, password)
	})

	// Form layout: fixed-width rows so text doesn't collapse.
	formWidth := float32(300)
	formHeight := float32(40)

	instructionRow := container.NewGridWrap(fyne.NewSize(formWidth, formHeight*2), instruction)
	codeRow := container.NewGridWrap(fyne.NewSize(formWidth, formHeight), codeLabel)
	copyRow := container.NewGridWrap(fyne.NewSize(formWidth, formHeight), copyBtn)
	statusRow := container.NewGridWrap(fyne.NewSize(formWidth, formHeight), statusLabel)
	buttonRow := container.NewGridWrap(fyne.NewSize(formWidth, formHeight), continueBtn)

	form := container.NewVBox(
		container.NewCenter(logo),
		container.NewCenter(title),
		widget.NewSeparator(),
		container.NewCenter(instructionRow),
		container.NewCenter(codeRow),
		container.NewCenter(copyRow),
		container.NewCenter(statusRow),
		widget.NewSeparator(),
		container.NewCenter(buttonRow),
	)

	content := container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(form),
		layout.NewSpacer(),
	)

	w.SetContent(content)

	// Start polling goroutine.
	go func() {
		client := gateway.NewClient(cfg.Gateway, cfg.Verbose)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		timeout := time.After(5 * time.Minute)

		for {
			select {
			case <-ctx.Done():
				return
			case <-timeout:
				fyne.Do(func() {
					statusLabel.Importance = widget.WarningImportance
					statusLabel.SetText("Linking timed out - you can link later")
				})
				time.Sleep(2 * time.Second)
				fyne.Do(func() {
					onSuccess(username, password)
				})
				return
			case <-ticker.C:
				linked, err := auth.CheckDiscordStatus(ctx, client, regUsername, accessToken)
				if err != nil {
					ui.Warn(fmt.Sprintf("Discord status poll error: %s", err))
					continue
				}
				if linked {
					fyne.Do(func() {
						statusLabel.Importance = widget.SuccessImportance
						statusLabel.SetText("Discord linked!")
					})
					time.Sleep(1500 * time.Millisecond)
					fyne.Do(func() {
						onSuccess(username, password)
					})
					return
				}
			}
		}
	}()
}
