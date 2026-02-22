package cli

import (
	"github.com/cstory/cluckers/internal/auth"
	"github.com/cstory/cluckers/internal/ui"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove saved credentials",
	Long:  "Deletes saved login credentials from disk. You will be prompted to log in again on next launch.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := auth.DeleteCredentials(); err != nil {
			return err
		}
		ui.Success("Credentials removed")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
