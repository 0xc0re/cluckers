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
	Long:  "Sends a password reset request to the Project Crown server. Reset instructions will be sent to your registered email or Discord.",
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
		err = auth.RequestPasswordReset(cmd.Context(), client, username)
		if err != nil {
			sp.Fail()
			return err
		}
		sp.Success()

		ui.Success("Password reset requested for " + username)
		ui.Info("Check your email or Discord for reset instructions.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resetPasswordCmd)
}
