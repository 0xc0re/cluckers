package cli

import (
	"strings"

	"github.com/0xc0re/cluckers/internal/selfupdate"
	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/spf13/cobra"
)

var selfUpdateCmd = &cobra.Command{
	Use:   "self-update",
	Short: "Update the cluckers launcher to the latest version",
	Long:  "Checks GitHub releases for a newer version of cluckers and downloads it, replacing the current binary.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Clean up any leftover .old binary from a previous self-update.
		// On Windows, the .old file cannot be deleted during the update
		// because the process is still running. Non-fatal if it fails.
		_ = selfupdate.CleanupOldBinary()

		// Extract the version part before the first space.
		// versionStr format: "0.4.1 (commit abc, built 2026-02-24)" or "dev (commit none, built unknown)"
		currentVersion := versionStr
		if idx := strings.Index(currentVersion, " "); idx != -1 {
			currentVersion = currentVersion[:idx]
		}

		// Dev builds cannot self-update.
		if strings.HasPrefix(currentVersion, "dev") {
			ui.Warn("Development build detected. Self-update only works with release builds.")
			return nil
		}

		ui.Info("Checking for launcher updates...")

		info, err := selfupdate.CheckLatestVersion(cmd.Context())
		if err != nil {
			return err
		}

		if !selfupdate.IsNewer(currentVersion, info.TagName) {
			ui.Success("Launcher is up to date (version " + currentVersion + ")")
			return nil
		}

		ui.Info("Update available: " + info.TagName)

		archive, checksums, err := selfupdate.FindAsset(info)
		if err != nil {
			return err
		}

		if err := selfupdate.DownloadAndReplace(cmd.Context(), archive, checksums); err != nil {
			return err
		}

		ui.Success("Launcher updated to " + info.TagName + ". Restart cluckers to use the new version.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(selfUpdateCmd)
}
