package cli

import (
	"github.com/spf13/cobra"
)

var steamCmd = &cobra.Command{
	Use:   "steam",
	Short: "Steam integration commands",
}

var steamAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add Realm Royale to Steam library as a non-Steam game",
	Long:  "Creates a shortcut for Cluckers so it can be added to Steam as a non-Steam game.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSteamAdd()
	},
}

func init() {
	steamCmd.AddCommand(steamAddCmd)
	rootCmd.AddCommand(steamCmd)
}
