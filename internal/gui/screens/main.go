//go:build gui

package screens

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/0xc0re/cluckers/internal/auth"
	"github.com/0xc0re/cluckers/internal/config"
	"github.com/0xc0re/cluckers/internal/game"
	"github.com/0xc0re/cluckers/internal/gateway"
	guiassets "github.com/0xc0re/cluckers/internal/gui/assets"
)

// MakeMainView builds the main application view with launch button, game management,
// community links, and navigation. This is the primary hub of the GUI launcher.
//
// Parameters:
//   - w: the application window
//   - cfg: application configuration
//   - username, password: authenticated credentials for pipeline use
//   - onLaunch: called when the user clicks the Launch button
//   - onLogout: called when the user clicks Logout
func MakeMainView(w fyne.Window, cfg *config.Config, username, password string, onLaunch func(), onLogout func(), onSettings func()) fyne.CanvasObject {
	// Logo (smaller for main view).
	logo := canvas.NewImageFromResource(guiassets.LogoResource())
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(64, 64))

	// Title row: logo + app name.
	titleLabel := widget.NewRichTextFromMarkdown("## Cluckers Central")
	titleLabel.Wrapping = fyne.TextWrapOff
	titleRow := container.NewHBox(logo, titleLabel)

	// Welcome text.
	welcomeLabel := widget.NewLabelWithStyle(
		"Logged in as "+username,
		fyne.TextAlignCenter,
		fyne.TextStyle{Italic: true},
	)

	// ---- Launch button (large, prominent, green) ----
	launchBtn := widget.NewButton("  LAUNCH  ", nil)
	launchBtn.Importance = widget.HighImportance
	launchBtn.OnTapped = func() {
		onLaunch()
	}
	launchBtnRow := container.NewCenter(
		container.NewGridWrap(fyne.NewSize(320, 48), launchBtn),
	)

	// ---- Game management section ----
	verifyBtn := widget.NewButtonWithIcon("Verify Files", theme.SearchIcon(), nil)
	verifyBtn.OnTapped = func() {
		verifyBtn.Disable()
		go func() {
			gameDir := cfg.GameDir
			if gameDir == "" {
				gameDir = game.GameDir()
			}
			info, err := game.FetchVersionInfo(context.Background())
			if err != nil {
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf("could not check version: %s", err), w)
					verifyBtn.Enable()
				})
				return
			}
			needsUpdate, err := game.NeedsUpdate(gameDir, info)
			if err != nil {
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf("verification error: %s", err), w)
					verifyBtn.Enable()
				})
				return
			}
			fyne.Do(func() {
				if needsUpdate {
					dialog.ShowInformation("Verify Game Files",
						"Game files are out of date or missing.\nUse Update to download the latest version.", w)
				} else {
					dialog.ShowInformation("Verify Game Files",
						"Game files are up to date and verified.", w)
				}
				verifyBtn.Enable()
			})
		}()
	}

	updateBtn := widget.NewButtonWithIcon("Update Game", theme.DownloadIcon(), nil)
	updateBtn.OnTapped = func() {
		updateBtn.Disable()
		go func() {
			gameDir := cfg.GameDir
			if gameDir == "" {
				gameDir = game.GameDir()
			}
			info, err := game.FetchVersionInfo(context.Background())
			if err != nil {
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf("could not check version: %s", err), w)
					updateBtn.Enable()
				})
				return
			}
			needsUpdate, err := game.NeedsUpdate(gameDir, info)
			if err != nil {
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf("version check error: %s", err), w)
					updateBtn.Enable()
				})
				return
			}
			if !needsUpdate {
				fyne.Do(func() {
					dialog.ShowInformation("Update Game", "Game is already up to date.", w)
					updateBtn.Enable()
				})
				return
			}
			if err := config.EnsureDir(gameDir); err != nil {
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf("could not create game directory: %s", err), w)
					updateBtn.Enable()
				})
				return
			}
			if err := game.DownloadAndVerify(context.Background(), info, gameDir); err != nil {
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf("download failed: %s", err), w)
					updateBtn.Enable()
				})
				return
			}
			fyne.Do(func() {
				dialog.ShowInformation("Update Game",
					"Game updated to version "+info.LatestVersion, w)
				updateBtn.Enable()
			})
		}()
	}

	repairBtn := widget.NewButtonWithIcon("Repair Game", theme.ViewRefreshIcon(), nil)
	repairBtn.OnTapped = func() {
		dialog.ShowConfirm("Repair Game",
			"This will delete all game files and re-download them.\nContinue?",
			func(confirmed bool) {
				if !confirmed {
					return
				}
				repairBtn.Disable()
				go func() {
					gameDir := cfg.GameDir
					if gameDir == "" {
						gameDir = game.GameDir()
					}
					// Delete game directory contents.
					if err := removeGameFiles(gameDir); err != nil {
						fyne.Do(func() {
							dialog.ShowError(fmt.Errorf("could not delete game files: %s", err), w)
							repairBtn.Enable()
						})
						return
					}
					info, err := game.FetchVersionInfo(context.Background())
					if err != nil {
						fyne.Do(func() {
							dialog.ShowError(fmt.Errorf("could not check version: %s", err), w)
							repairBtn.Enable()
						})
						return
					}
					if err := config.EnsureDir(gameDir); err != nil {
						fyne.Do(func() {
							dialog.ShowError(fmt.Errorf("could not create game directory: %s", err), w)
							repairBtn.Enable()
						})
						return
					}
					if err := game.DownloadAndVerify(context.Background(), info, gameDir); err != nil {
						fyne.Do(func() {
							dialog.ShowError(fmt.Errorf("download failed: %s", err), w)
							repairBtn.Enable()
						})
						return
					}
					fyne.Do(func() {
						dialog.ShowInformation("Repair Game",
							"Game files repaired and updated to version "+info.LatestVersion, w)
						repairBtn.Enable()
					})
				}()
			}, w)
	}

	gameManagementGrid := container.NewGridWithColumns(3, verifyBtn, updateBtn, repairBtn)

	// ---- Supporter Features: Bot Names ----
	botName1Entry := widget.NewEntry()
	botName1Entry.PlaceHolder = "Bot name 1 (supporters only)"
	botName2Entry := widget.NewEntry()
	botName2Entry.PlaceHolder = "Bot name 2 (supporters only)"

	botSetBtn := widget.NewButton("Set Bot Names", nil)
	botSetBtn.OnTapped = func() {
		name1 := botName1Entry.Text
		name2 := botName2Entry.Text
		if name1 == "" && name2 == "" {
			dialog.ShowInformation("Bot Names", "Enter at least one bot name.", w)
			return
		}
		botSetBtn.Disable()
		go func() {
			var accessToken string

			// Fast path: try cached access token first.
			cache, err := auth.LoadTokenCache()
			if err == nil && cache != nil && cache.AccessTokenValid() {
				accessToken = cache.AccessToken
				fmt.Println("[bot-names] Using cached access token for", username)
			}

			// Fallback: authenticate inline using available credentials.
			if accessToken == "" {
				fmt.Println("[bot-names] No cached token, authenticating inline for", username)
				client := gateway.NewClient(cfg.Gateway, cfg.Verbose)
				result, loginErr := auth.Login(context.Background(), client, username, password)
				if loginErr != nil {
					fyne.Do(func() {
						dialog.ShowError(fmt.Errorf("could not authenticate: %s", loginErr), w)
						botSetBtn.Enable()
					})
					return
				}
				accessToken = result.AccessToken

				// Cache the fresh token so subsequent calls are fast.
				now := time.Now()
				newCache := &auth.TokenCache{
					AccessToken:    accessToken,
					Username:       username,
					AccessCachedAt: now,
				}
				// Preserve existing OIDC token if present.
				if cache != nil {
					newCache.OIDCToken = cache.OIDCToken
					newCache.OIDCCachedAt = cache.OIDCCachedAt
				}
				if saveErr := auth.SaveTokenCache(newCache); saveErr != nil {
					fmt.Println("[bot-names] Warning: could not save token cache:", saveErr)
				}
			}

			client := gateway.NewClient(cfg.Gateway, cfg.Verbose)
			req := gateway.BotNameRequest{
				UserName:    username,
				AccessToken: accessToken,
				BotName1:    name1,
				BotName2:    name2,
			}
			var resp gateway.BotNameResponse
			if err := client.Post(context.Background(), "LAUNCHER_SET_BOT_NAME", req, &resp); err != nil {
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf("failed to set bot names: %s", err), w)
					botSetBtn.Enable()
				})
				return
			}
			if !bool(resp.Success) {
				msg := resp.TextValue
				if msg == "" {
					msg = "Server rejected request (no details provided). Try launching the game first to verify your account works."
				} else {
					msg = "Server rejected request: " + msg
				}
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf("failed to set bot names: %s\n\n(SUCCESS=%v)", msg, resp.Success), w)
					botSetBtn.Enable()
				})
				return
			}
			fyne.Do(func() {
				dialog.ShowInformation("Bot Names", "Bot names updated successfully!", w)
				botSetBtn.Enable()
			})
		}()
	}

	botNameSection := container.NewVBox(
		widget.NewLabelWithStyle("Supporter Features", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
		container.NewCenter(
			container.NewGridWrap(fyne.NewSize(340, 36), botName1Entry),
		),
		container.NewCenter(
			container.NewGridWrap(fyne.NewSize(340, 36), botName2Entry),
		),
		container.NewCenter(
			container.NewGridWrap(fyne.NewSize(200, 36), botSetBtn),
		),
	)

	// ---- Links section ----
	discordURL, _ := url.Parse("https://discord.gg/RealmRoyale")
	discordLink := widget.NewHyperlink("Discord", discordURL)

	supportURL, _ := url.Parse("https://ko-fi.com/projectcrown/tiers")
	supportLink := widget.NewHyperlink("Support", supportURL)

	linksRow := container.NewHBox(
		layout.NewSpacer(),
		discordLink,
		widget.NewLabel("  |  "),
		supportLink,
		layout.NewSpacer(),
	)

	// ---- Bottom row: Settings + Logout ----
	settingsBtn := widget.NewButtonWithIcon("Settings", theme.SettingsIcon(), nil)
	settingsBtn.OnTapped = func() {
		onSettings()
	}

	logoutBtn := widget.NewButtonWithIcon("Logout", theme.LogoutIcon(), nil)
	logoutBtn.OnTapped = func() {
		onLogout()
	}

	bottomRow := container.NewHBox(
		layout.NewSpacer(),
		settingsBtn,
		logoutBtn,
		layout.NewSpacer(),
	)

	// ---- Assemble main layout ----
	content := container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(titleRow),
		container.NewCenter(welcomeLabel),
		widget.NewSeparator(),
		launchBtnRow,
		widget.NewSeparator(),
		container.NewCenter(gameManagementGrid),
		widget.NewSeparator(),
		botNameSection,
		widget.NewSeparator(),
		linksRow,
		bottomRow,
		layout.NewSpacer(),
	)

	return content
}

// removeGameFiles removes all files in the game directory for repair.
// The parent directory itself is preserved.
func removeGameFiles(gameDir string) error {
	entries, err := os.ReadDir(gameDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Nothing to remove.
		}
		return err
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(gameDir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}
