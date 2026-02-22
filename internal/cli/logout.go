package cli

import (
	"github.com/0xc0re/cluckers/internal/auth"
	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove saved credentials and cached tokens",
	Long:  "Deletes saved login credentials and cached tokens from disk. You will be prompted to log in again on next launch.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := auth.DeleteCredentials(); err != nil {
			return err
		}
		if err := auth.ClearTokenCache(); err != nil {
			ui.Warn("Could not clear token cache: " + err.Error())
		}
		ui.Success("Credentials and cached tokens removed")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
