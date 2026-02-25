//go:build linux

package launch

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/0xc0re/cluckers/internal/wine"
)

// protonMajorVersionRe extracts the major version number from GE-Proton directory names.
var protonMajorVersionRe = regexp.MustCompile(`GE-Proton(\d+)-(\d+)`)

// platformSteps returns Linux-specific pipeline steps: Proton detection and
// compatdata environment preparation.
func platformSteps(_ *LaunchState) []Step {
	return []Step{
		{Name: "Detecting Proton", Fn: stepDetectProton},
		{Name: "Preparing Proton environment", Fn: stepEnsureCompatdata},
	}
}

// platformPostSteps returns Linux-specific post-download steps: Steam Deck config.
func platformPostSteps(_ *LaunchState) []Step {
	return []Step{
		{Name: "Configuring for Steam Deck", Fn: stepDeckConfig},
	}
}

// stepDetectProton finds a suitable Proton-GE installation.
func stepDetectProton(_ context.Context, state *LaunchState) error {
	install, err := wine.FindProton(state.Config.WinePath)
	if err != nil {
		return err
	}

	state.ProtonScript = install.ProtonScript()
	state.ProtonDir = install.ProtonDir
	state.ProtonDisplayVersion = install.DisplayVersion()
	// Keep WinePath populated for backward compatibility.
	state.WinePath = install.WinePath

	ui.Verbose(fmt.Sprintf("Proton: %s (%s)", install.DisplayVersion(), install.ProtonDir), state.Config.Verbose)

	// Warn if Proton-GE version is older than 9 (recommended minimum).
	if m := protonMajorVersionRe.FindStringSubmatch(install.DisplayVersion()); m != nil {
		major, _ := strconv.Atoi(m[1])
		if major < 9 {
			ui.Warn(fmt.Sprintf("%s detected, version 9+ recommended", install.DisplayVersion()))
		}
	}

	return nil
}

// stepEnsureCompatdata ensures the Proton compatdata directory exists and is healthy.
// Corrupted compatdata is auto-deleted and recreated with a warning.
func stepEnsureCompatdata(_ context.Context, state *LaunchState) error {
	compatdata := wine.CompatdataPath()

	if wine.CompatdataHealthy(compatdata) {
		state.CompatDataPath = compatdata
		ui.Verbose(fmt.Sprintf("Proton environment healthy: %s", compatdata), state.Config.Verbose)
		return nil
	}

	// Check if directory exists but is damaged.
	if _, err := os.Stat(compatdata); err == nil {
		ui.Warn("Proton environment damaged, recreating...")
		if err := os.RemoveAll(compatdata); err != nil {
			return fmt.Errorf("removing damaged compatdata: %w", err)
		}
	}

	// Create the compatdata directory. Proton will populate it on first run.
	ui.Info("Preparing Proton environment (first launch only)...")
	if err := os.MkdirAll(compatdata, 0755); err != nil {
		return fmt.Errorf("creating compatdata directory: %w", err)
	}

	state.CompatDataPath = compatdata
	return nil
}

// stepDeckConfig patches game settings for Steam Deck (fullscreen, resolution).
// Skips silently on non-Deck systems or if already configured.
func stepDeckConfig(_ context.Context, state *LaunchState) error {
	return PatchDeckConfig(state.GameDir)
}
