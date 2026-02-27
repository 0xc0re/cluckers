package cli

import (
	"path/filepath"

	"github.com/0xc0re/cluckers/internal/config"
	"github.com/0xc0re/cluckers/internal/game"
	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for game updates and download if needed",
	Long:  "Checks the latest game version from the server. If the local game files are outdated or missing, downloads and installs the update.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Resolve game directory.
		gameDir := Cfg.GameDir
		if gameDir == "" {
			gameDir = game.GameDir()
		}

		ui.Info("Checking for updates...")

		info, err := game.FetchVersionInfo(cmd.Context())
		if err != nil {
			return err
		}

		ui.Verbose("Server version: "+info.LatestVersion, Cfg.Verbose)

		needsUpdate, err := game.NeedsUpdate(gameDir, info)
		if err != nil {
			return err
		}

		if !needsUpdate {
			ui.Success("Game is up to date (version " + info.LatestVersion + ")")
			return nil
		}

		ui.Info("Update available: " + info.LatestVersion)

		if err := config.EnsureDir(gameDir); err != nil {
			return err
		}

		if err := game.DownloadAndVerify(cmd.Context(), info, gameDir); err != nil {
			return err
		}

		if err := game.ExtractZip(filepath.Join(gameDir, "game.zip"), gameDir); err != nil {
			return err
		}

		// Verify the extraction actually produced matching game files.
		stillNeedsUpdate, verifyErr := game.NeedsUpdate(gameDir, info)
		if verifyErr != nil {
			return &ui.UserError{
				Message:    "Could not verify game files after extraction.",
				Detail:     verifyErr.Error(),
				Suggestion: "Try running `cluckers update` again.",
			}
		}
		if stillNeedsUpdate {
			return &ui.UserError{
				Message:    "Game files were extracted but verification failed.",
				Detail:     "GameVersion.dat hash does not match expected value after extraction.",
				Suggestion: "The download may be corrupted. Delete the game directory and run `cluckers update` again.",
			}
		}

		ui.Success("Game updated to version " + info.LatestVersion)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
