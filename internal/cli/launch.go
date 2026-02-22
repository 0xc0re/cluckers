package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var launchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Authenticate and launch Realm Royale",
	Long:  "Authenticates with the Project Crown gateway, retrieves tokens and bootstrap data, then launches Realm Royale under Wine.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Launch pipeline not yet implemented.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(launchCmd)
}
