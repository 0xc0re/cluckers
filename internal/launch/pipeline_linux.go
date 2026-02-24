//go:build linux

package launch

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/0xc0re/cluckers/internal/wine"
)

// platformSteps returns Linux-specific pipeline steps: Wine detection, prefix
// creation, and prefix verification.
func platformSteps(_ *LaunchState) []Step {
	return []Step{
		{Name: "Detecting Wine", Fn: stepDetectWine},
		{Name: "Ensuring Wine prefix", Fn: stepEnsurePrefix},
		{Name: "Verifying Wine prefix", Fn: stepVerifyPrefix},
	}
}

// platformPostSteps returns Linux-specific post-download steps: Steam Deck config.
func platformPostSteps(_ *LaunchState) []Step {
	return []Step{
		{Name: "Configuring for Steam Deck", Fn: stepDeckConfig},
	}
}

// stepDetectWine finds a suitable Wine binary.
func stepDetectWine(_ context.Context, state *LaunchState, _ *ui.StepSpinner) error {
	winePath, err := wine.FindWine(state.Config.WinePath)
	if err != nil {
		return err
	}
	state.WinePath = winePath
	ui.Verbose(fmt.Sprintf("Wine: %s", winePath), state.Config.Verbose)
	return nil
}

// stepEnsurePrefix ensures the Wine prefix exists, creating it if needed.
func stepEnsurePrefix(_ context.Context, state *LaunchState, _ *ui.StepSpinner) error {
	// Determine prefix path: config override or default.
	prefixPath := state.Config.WinePrefix
	if prefixPath == "" {
		prefixPath = wine.PrefixPath()
	}

	// Check if prefix already exists.
	if _, err := os.Stat(prefixPath); err == nil {
		ui.Verbose(fmt.Sprintf("Wine prefix exists: %s", prefixPath), state.Config.Verbose)
		state.PrefixPath = prefixPath
		return nil
	}

	// Prefix doesn't exist -- create it.
	if err := wine.CreatePrefix(prefixPath, state.WinePath, state.Config.Verbose); err != nil {
		return err
	}

	state.PrefixPath = prefixPath
	return nil
}

// stepVerifyPrefix checks that all required DLLs exist in the Wine prefix.
func stepVerifyPrefix(_ context.Context, state *LaunchState, _ *ui.StepSpinner) error {
	healthy, missing := wine.VerifyPrefix(state.PrefixPath)
	if !healthy {
		return &ui.UserError{
			Message:    "Wine prefix missing required DLLs",
			Detail:     fmt.Sprintf("Missing: %s", strings.Join(missing, ", ")),
			Suggestion: wine.RepairInstructions(state.WinePath, missing),
		}
	}
	ui.Verbose(fmt.Sprintf("Prefix verified: %d DLLs present", len(wine.RequiredDLLs)), state.Config.Verbose)
	return nil
}

// stepDeckConfig patches game settings for Steam Deck (fullscreen, resolution).
// Skips silently on non-Deck systems or if already configured.
func stepDeckConfig(_ context.Context, state *LaunchState, _ *ui.StepSpinner) error {
	return PatchDeckConfig(state.GameDir)
}
