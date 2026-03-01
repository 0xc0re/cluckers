package cli

import (
	"github.com/0xc0re/cluckers/internal/launch"
	"github.com/spf13/cobra"
)

var launchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Authenticate and launch Realm Royale",
	Long:  "Authenticates with the Project Crown gateway, retrieves tokens and bootstrap data, then launches Realm Royale.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return launch.Run(cmd.Context(), Cfg)
	},
}

func init() {
	rootCmd.AddCommand(launchCmd)
}
