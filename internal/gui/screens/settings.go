//go:build gui

package screens

import (
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/0xc0re/cluckers/internal/config"
	"github.com/spf13/viper"
)

// MakeSettingsView builds the Settings screen with config options and
// persistence. On save, values are written to the TOML config file.
// onBack returns to the main view without saving.
func MakeSettingsView(w fyne.Window, cfg *config.Config, onBack func()) fyne.CanvasObject {
	// Title.
	title := widget.NewRichTextFromMarkdown("# Settings")

	// Gateway URL.
	gatewayEntry := widget.NewEntry()
	gatewayEntry.SetText(cfg.Gateway)
	gatewayEntry.PlaceHolder = "https://gateway-dev.project-crown.com"

	// Verbose mode.
	verboseCheck := widget.NewCheck("Enable verbose output", nil)
	verboseCheck.SetChecked(cfg.Verbose)

	// Game directory.
	gameDirEntry := widget.NewEntry()
	gameDirEntry.SetText(cfg.GameDir)
	if runtime.GOOS == "linux" {
		gameDirEntry.PlaceHolder = "~/.cluckers/game (default)"
	} else {
		gameDirEntry.PlaceHolder = "%LOCALAPPDATA%\\cluckers\\game (default)"
	}

	// Build form items. Start with common fields.
	formItems := []*widget.FormItem{
		widget.NewFormItem("Gateway URL", gatewayEntry),
		widget.NewFormItem("Verbose", verboseCheck),
		widget.NewFormItem("Game Directory", gameDirEntry),
	}

	// Proton path entry (Linux only).
	var winePathEntry *widget.Entry
	if runtime.GOOS == "linux" {
		winePathEntry = widget.NewEntry()
		winePathEntry.SetText(cfg.WinePath)
		winePathEntry.PlaceHolder = "auto-detect"

		formItems = append(formItems,
			widget.NewFormItem("Proton Path", winePathEntry),
		)
	}

	form := widget.NewForm(formItems...)

	// Save button.
	saveBtn := widget.NewButton("Save", nil)
	saveBtn.Importance = widget.HighImportance
	saveBtn.OnTapped = func() {
		// Update viper values.
		viper.Set("gateway", gatewayEntry.Text)
		viper.Set("verbose", verboseCheck.Checked)
		viper.Set("game_dir", gameDirEntry.Text)

		if runtime.GOOS == "linux" && winePathEntry != nil {
			viper.Set("wine_path", winePathEntry.Text)
		}

		// Ensure config directory exists.
		if err := config.EnsureDir(config.ConfigDir()); err != nil {
			dialog.ShowError(err, w)
			return
		}

		// Write config file.
		if err := viper.WriteConfigAs(config.ConfigFile()); err != nil {
			dialog.ShowError(err, w)
			return
		}

		// Update in-memory config.
		cfg.Gateway = gatewayEntry.Text
		cfg.Verbose = verboseCheck.Checked
		cfg.GameDir = gameDirEntry.Text
		if runtime.GOOS == "linux" && winePathEntry != nil {
			cfg.WinePath = winePathEntry.Text
		}

		dialog.ShowInformation("Settings Saved", "Configuration updated.", w)
	}

	// Back button.
	backBtn := widget.NewButton("Back", nil)
	backBtn.OnTapped = func() {
		onBack()
	}

	// Button row.
	btnWidth := float32(140)
	btnHeight := float32(40)
	buttons := container.NewHBox(
		container.NewGridWrap(fyne.NewSize(btnWidth, btnHeight), saveBtn),
		container.NewGridWrap(fyne.NewSize(btnWidth, btnHeight), backBtn),
	)

	// Full layout with wider form area for comfortable label + input display.
	content := container.NewVBox(
		container.NewCenter(title),
		widget.NewSeparator(),
		form,
		widget.NewSeparator(),
		container.NewCenter(buttons),
	)

	return container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(
			container.NewGridWrap(fyne.NewSize(440, 0), content),
		),
		layout.NewSpacer(),
	)
}
