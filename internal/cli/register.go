package cli

import (
	"fmt"
	"time"

	"github.com/0xc0re/cluckers/internal/auth"
	"github.com/0xc0re/cluckers/internal/gateway"
	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/spf13/cobra"
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Create a new Project Crown account",
	Long:  "Creates a new account on the Project Crown server, saves credentials, and provides a Discord verification code to complete account linking.",
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

		// Register account.
		result, err := auth.Register(cmd.Context(), client, username, password, email)
		if err != nil {
			return err
		}

		ui.Success("Account created for " + result.Username)

		// Save credentials so login/launch work immediately.
		if err := auth.SaveCredentials(username, password); err != nil {
			ui.Warn(fmt.Sprintf("Could not save credentials: %s", err))
		}

		// Cache the access token from registration (acts as auto-login).
		cache := &auth.TokenCache{
			Username:       result.Username,
			AccessToken:    result.AccessToken,
			AccessCachedAt: time.Now(),
		}
		if err := auth.SaveTokenCache(cache); err != nil {
			ui.Warn(fmt.Sprintf("Could not save token cache: %s", err))
		}

		// Request Discord link code (uses password auth).
		code, err := auth.RequestLinkCode(cmd.Context(), client, result.Username, password)
		if err != nil {
			ui.Warn(fmt.Sprintf("Could not get Discord link code: %s", err))
			ui.Info("You can request a link code later by logging in.")
			return nil
		}

		// Display the link code with instructions.
		fmt.Println()
		ui.Info("To complete your account, DM the following code to the Project Crown Discord bot:")
		fmt.Println()
		fmt.Printf("  Your verification code: %s\n", code)
		fmt.Println()

		// Poll for Discord linking status.
		sp := ui.StartStep("Waiting for Discord linking...")
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		timeout := time.After(5 * time.Minute)

		for {
			select {
			case <-cmd.Context().Done():
				sp.Stop()
				return cmd.Context().Err()
			case <-timeout:
				sp.Stop()
				ui.Info("Linking timed out. You can check your status later with: cluckers login")
				return nil
			case <-ticker.C:
				linked, err := auth.CheckDiscordStatus(cmd.Context(), client, result.Username, result.AccessToken)
				if err != nil {
					ui.Verbose(fmt.Sprintf("Discord status check failed: %s", err), Cfg.Verbose)
					continue
				}
				if linked {
					sp.Success()
					ui.Success("Discord account linked! You can now launch the game with: cluckers launch")
					return nil
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(registerCmd)
}
