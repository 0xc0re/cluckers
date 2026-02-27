//go:build linux

package cli

import (
	"github.com/0xc0re/cluckers/internal/launch"
	"github.com/spf13/cobra"
)

var prepCmd = &cobra.Command{
	Use:   "prep",
	Short: "Authenticate, update, and write launch config for Steam-managed launch",
	Long: `Runs the full launch pipeline (auth, tokens, bootstrap, platform setup, version
check, download) then writes persistent config files instead of launching the game.

Use this with Steam's %command% mechanism so Steam manages the Proton lifecycle,
keeping Steam Input controller bindings stable through map transitions.

Launch Options in Steam:
  /path/to/cluckers prep && WINEDLLOVERRIDES=dxgi=n %command%`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return launch.RunPrep(cmd.Context(), Cfg)
	},
}

func init() {
	rootCmd.AddCommand(prepCmd)
}
