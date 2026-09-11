//go:build gui

package screens

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"net/url"
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
	"github.com/0xc0re/cluckers/internal/ui"
)

// MakeRegisterScreen builds the registration screen with logo, username/password/email
// fields, register button, back-to-login link, and inline error display. On successful
// registration, credentials are saved and the user transitions to the Discord linking
// view (the gateway hands out the link code with the registration reply) or, if the
// account is already linked, directly to the main view.
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
			var nl *auth.NotLinkedError
			if err != nil && !errors.As(err, &nl) {
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

			if nl != nil {
				// A new account must be linked to Discord before it gets a session.
				fyne.Do(func() {
					ShowDiscordLinking(w, cfg, username, password, nl.LinkCode, onSuccess)
				})
				return
			}

			// Cache the session from registration (acts as auto-login).
			if err := auth.SaveTokenCache(auth.NewTokenCache(result)); err != nil {
				ui.Warn(fmt.Sprintf("could not save token cache: %s", err))
			}
			fyne.Do(func() { onSuccess(username, password) })
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

// ShowDiscordLinking replaces the window content with a Discord linking view
// that displays the link code and polls the gateway (via session-or-link)
// until the account is linked. The code is refreshed on screen if the server
// rotates it. On success the fresh session is cached and onLinked is called.
func ShowDiscordLinking(w fyne.Window, cfg *config.Config, username, password, code string, onLinked func(username, password string)) {
	// Logo.
	logo := canvas.NewImageFromResource(guiassets.LogoResource())
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(128, 128))

	// Title.
	title := widget.NewRichTextFromMarkdown("# Discord Linking")
	title.Wrapping = fyne.TextWrapOff

	// Instruction text.
	instruction := widget.NewLabel("Your account must be linked to Discord before you can play. DM the following code to the Project Crown bot:")
	instruction.Alignment = fyne.TextAlignCenter
	instruction.Wrapping = fyne.TextWrapWord

	// Code display — bold label, readable against dark theme.
	currentCode := code
	codeLabel := widget.NewLabelWithStyle(code, fyne.TextAlignCenter, fyne.TextStyle{Bold: true, Monospace: true})

	// Copy button.
	copyBtn := widget.NewButton("Copy Code", func() {
		w.Clipboard().SetContent(currentCode)
	})
	copyBtn.Importance = widget.MediumImportance

	// Discord invite link.
	discordURL, _ := url.Parse(auth.DiscordInviteURL)
	discordLink := widget.NewHyperlink("Open the Project Crown Discord", discordURL)
	discordLink.Alignment = fyne.TextAlignCenter

	// Status label.
	statusLabel := widget.NewLabel("Waiting for Discord linking (checking every 3 seconds)...")
	statusLabel.Alignment = fyne.TextAlignCenter
	statusLabel.Wrapping = fyne.TextWrapWord

	// Cancellable context for the polling goroutine.
	ctx, cancelFunc := context.WithCancel(context.Background())

	// Continue without linking button (launching will prompt again).
	continueBtn := widget.NewButton("Continue Without Linking", func() {
		cancelFunc()
		onLinked(username, password)
	})

	// Form layout: fixed-width rows so text doesn't collapse.
	formWidth := float32(320)
	formHeight := float32(40)

	instructionRow := container.NewGridWrap(fyne.NewSize(formWidth, formHeight*2), instruction)
	codeRow := container.NewGridWrap(fyne.NewSize(formWidth, formHeight), codeLabel)
	copyRow := container.NewGridWrap(fyne.NewSize(formWidth, formHeight), copyBtn)
	linkRow := container.NewGridWrap(fyne.NewSize(formWidth, formHeight), discordLink)
	statusRow := container.NewGridWrap(fyne.NewSize(formWidth, formHeight*2), statusLabel)
	buttonRow := container.NewGridWrap(fyne.NewSize(formWidth, formHeight), continueBtn)

	form := container.NewVBox(
		container.NewCenter(logo),
		container.NewCenter(title),
		widget.NewSeparator(),
		container.NewCenter(instructionRow),
		container.NewCenter(codeRow),
		container.NewCenter(copyRow),
		container.NewCenter(linkRow),
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

	// Poll until linked.
	go func() {
		client := gateway.NewClient(cfg.Gateway, cfg.Verbose)
		result, err := auth.WaitForLink(ctx, client, username, password, func(newCode string) {
			if newCode == currentCode {
				return
			}
			currentCode = newCode
			fyne.Do(func() {
				codeLabel.SetText(newCode)
				statusLabel.SetText("The server issued a new code. DM the code above to the bot.")
			})
		})
		if ctx.Err() != nil {
			return // User continued without linking.
		}
		if err != nil {
			fyne.Do(func() {
				statusLabel.Importance = widget.WarningImportance
				statusLabel.SetText(formatGUIError(err))
			})
			return
		}

		if err := auth.SaveTokenCache(auth.NewTokenCache(result)); err != nil {
			ui.Warn(fmt.Sprintf("could not save token cache: %s", err))
		}
		fyne.Do(func() {
			statusLabel.Importance = widget.SuccessImportance
			statusLabel.SetText("Discord linked!")
		})
		time.Sleep(1500 * time.Millisecond)
		fyne.Do(func() {
			onLinked(username, password)
		})
	}()
}
