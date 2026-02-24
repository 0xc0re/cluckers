//go:build !gui

package cli

// Without the gui build tag, the root command uses Cobra's default behavior:
// when no subcommand is given, it displays the help output. No changes needed.
