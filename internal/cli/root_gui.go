//go:build gui

package cli

import (
	"fmt"
	"os"
	"os/exec"

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

		// If already the detached child, run the GUI directly.
		if os.Getenv("CLUCKERS_GUI_FG") == "1" {
			return gui.Run(Cfg)
		}

		// Re-exec ourselves detached from the terminal so the shell
		// gets its prompt back immediately.
		exe, err := os.Executable()
		if err != nil {
			return gui.Run(Cfg) // fallback: run in foreground
		}

		child := exec.Command(exe, os.Args[1:]...)
		child.Env = append(os.Environ(), "CLUCKERS_GUI_FG=1")
		child.SysProcAttr = detachSysProcAttr()
		child.Stdout = nil
		child.Stderr = nil
		child.Stdin = nil

		if err := child.Start(); err != nil {
			return gui.Run(Cfg) // fallback: run in foreground
		}

		fmt.Println("Cluckers launched in background.")
		os.Exit(0)
		return nil
	}
}
