package cli

import (
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

		info, err := game.ResolveVersionInfo(cmd.Context(), Cfg.PinnedVersion)
		if err != nil {
			return err
		}

		if Cfg.PinnedVersion != "" {
			ui.Verbose("Pinned to version: "+info.LatestVersion, Cfg.Verbose)
		} else {
			ui.Verbose("Server version: "+info.LatestVersion, Cfg.Verbose)
		}

		needsUpdate, manifest, err := game.ResolveNeedsUpdate(cmd.Context(), gameDir, info)
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

		// ResolveNeedsUpdate only fetches the manifest on the pinned path; fetch
		// it here for the latest path.
		if manifest == nil {
			manifest, err = game.FetchManifest(cmd.Context(), info)
			if err != nil {
				return err
			}
		}

		if err := game.SyncManifest(cmd.Context(), info, manifest, gameDir, nil); err != nil {
			return err
		}

		// Verify the sync actually produced matching game files.
		stillNeedsUpdate, verifyErr := game.NeedsUpdateFromManifest(gameDir, manifest)
		if verifyErr != nil {
			return &ui.UserError{
				Message:    "Could not verify game files after sync.",
				Detail:     verifyErr.Error(),
				Suggestion: "Try running `cluckers update` again.",
			}
		}
		if stillNeedsUpdate {
			return &ui.UserError{
				Message:    "Game files were downloaded but verification failed.",
				Detail:     "GameVersion.dat hash does not match expected value after sync.",
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
