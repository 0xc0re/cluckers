package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xc0re/cluckers/internal/config"
	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/spf13/cobra"
)

var logsTail bool

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show log file path or recent log entries",
	Long:  "Prints the path to the cluckers log file. Use --tail to display the last 50 lines.",
	RunE: func(cmd *cobra.Command, args []string) error {
		logPath := filepath.Join(config.LogDir(), "cluckers.log")

		if !logsTail {
			fmt.Println("Log file:", logPath)
			return nil
		}

		f, err := os.Open(logPath)
		if err != nil {
			return &ui.UserError{
				Message:    "Could not open log file",
				Detail:     err.Error(),
				Suggestion: "Run a command first to create the log file.",
			}
		}
		defer f.Close()

		// Read all lines, keep last 50.
		var lines []string
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("reading log: %w", err)
		}

		start := 0
		if len(lines) > 50 {
			start = len(lines) - 50
		}
		for _, line := range lines[start:] {
			fmt.Println(line)
		}
		return nil
	},
}

func init() {
	logsCmd.Flags().BoolVar(&logsTail, "tail", false, "Show last 50 lines of the log file")
	rootCmd.AddCommand(logsCmd)
}
