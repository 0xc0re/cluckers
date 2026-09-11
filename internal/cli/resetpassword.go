package cli

import (
	"fmt"

	"github.com/0xc0re/cluckers/internal/auth"
	"github.com/0xc0re/cluckers/internal/gateway"
	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/spf13/cobra"
)

var resetPasswordCmd = &cobra.Command{
	Use:   "reset-password",
	Short: "Request a password reset for your account",
	Long:  "Requests a password reset from the Project Crown server. The server replies with a reset code that you DM to the Project Crown Discord bot, then reply with your new password.",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := gateway.NewClient(Cfg.Gateway, Cfg.Verbose)

		// Health check: warn but continue (same pattern as launch pipeline).
		if err := client.HealthCheck(cmd.Context()); err != nil {
			ui.Warn(fmt.Sprintf("Gateway health check failed: %s", err))
		}

		username, err := ui.PromptUsername()
		if err != nil {
			return err
		}

		sp := ui.StartStep("Requesting password reset...")
		result, err := auth.RequestPasswordReset(cmd.Context(), client, username)
		if err != nil {
			sp.Fail()
			return err
		}
		sp.Success()

		ui.Success("Password reset requested for " + username)
		if result.Message != "" {
			ui.Info(result.Message)
		} else {
			ui.Info("DM the reset code to the Project Crown bot on Discord, then reply with your new password.")
		}
		if result.Code != "" {
			fmt.Println()
			fmt.Printf("  Your reset code: %s\n", result.Code)
			fmt.Println()
			ui.Info("Project Crown Discord: " + auth.DiscordInviteURL)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resetPasswordCmd)
}
