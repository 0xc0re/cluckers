package cli

import (
	"github.com/0xc0re/cluckers/internal/launch"
	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/spf13/cobra"
)

var launchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Authenticate and launch Realm Royale",
	Long:  "Authenticates with the Project Crown gateway, retrieves tokens and bootstrap data, then launches Realm Royale under Wine.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := launch.Run(cmd.Context(), Cfg); err != nil {
			// Error is already printed by the pipeline with spinner feedback.
			// Return formatted error for Cobra's error handling.
			var ue *ui.UserError
			if ok := isUserError(err, &ue); ok {
				return err
			}
			return err
		}
		return nil
	},
}

// isUserError checks if err is a *ui.UserError and assigns it to target.
func isUserError(err error, target **ui.UserError) bool {
	ue, ok := err.(*ui.UserError)
	if ok {
		*target = ue
	}
	return ok
}

func init() {
	rootCmd.AddCommand(launchCmd)
}
