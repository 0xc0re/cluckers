//go:build gui

package cli

import (
	"github.com/0xc0re/cluckers/internal/gui"
	"github.com/spf13/cobra"
)

func init() {
	// When built with the gui tag, the root command launches the GUI
	// if a display server is available. Otherwise it falls back to
	// showing the standard CLI help output.
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if !gui.CanShowGUI() {
			return cmd.Help()
		}
		return gui.Run(Cfg)
	}
}
