package cli

import (
	"errors"
	"fmt"

	"github.com/0xc0re/cluckers/internal/auth"
	"github.com/0xc0re/cluckers/internal/gateway"
	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/spf13/cobra"
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Create a new Project Crown account",
	Long:  "Creates a new account on the Project Crown server, saves credentials, and walks you through linking the account to Discord with a code you DM to the Project Crown bot.",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := gateway.NewClient(Cfg.Gateway, Cfg.Verbose)

		// Prompt for registration details.
		username, err := ui.PromptUsername()
		if err != nil {
			return err
		}
		password, err := ui.PromptPassword()
		if err != nil {
			return err
		}
		email, err := ui.PromptEmail()
		if err != nil {
			return err
		}

		// Register account. A new account is normally not linked to Discord
		// yet, in which case the reply carries the link code instead of a token.
		result, err := auth.Register(cmd.Context(), client, username, password, email)
		var nl *auth.NotLinkedError
		switch {
		case err == nil:
		case errors.As(err, &nl):
		default:
			return err
		}

		ui.Success("Account created for " + username)

		// Save credentials so login/launch work immediately.
		if err := auth.SaveCredentials(username, password); err != nil {
			ui.Warn(fmt.Sprintf("Could not save credentials: %s", err))
		}

		if nl != nil {
			printLinkCode(nl.LinkCode)
			sp := ui.StartStep("Waiting for Discord linking (checking every 3 seconds)...")
			result, err = auth.WaitForLink(cmd.Context(), client, username, password, func(code string) {
				if code == nl.LinkCode {
					return
				}
				sp.Stop()
				ui.Warn("The server issued a new link code.")
				printLinkCode(code)
				sp = ui.StartStep("Waiting for Discord linking (checking every 3 seconds)...")
			})
			if err != nil {
				sp.Fail()
				if errors.Is(err, auth.ErrPinRequired) {
					return err
				}
				ui.Warn(fmt.Sprintf("Discord linking did not complete: %s", err))
				ui.Info("DM the code to the bot, then run: cluckers login")
				return nil
			}
			sp.Success()
			ui.Success("Discord account linked!")
		}

		// Cache the session from registration/linking (acts as auto-login).
		if err := auth.SaveTokenCache(auth.NewTokenCache(result)); err != nil {
			ui.Warn(fmt.Sprintf("Could not save token cache: %s", err))
		}

		ui.Success("You can now launch the game with: cluckers launch")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(registerCmd)
}
