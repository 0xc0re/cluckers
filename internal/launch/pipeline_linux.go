//go:build linux

package launch

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/0xc0re/cluckers/internal/wine"
)

// protonMajorVersionRe extracts the major version number from GE-Proton directory names.
var protonMajorVersionRe = regexp.MustCompile(`GE-Proton(\d+)-(\d+)`)

// platformSteps returns Linux-specific pipeline steps: Proton detection,
// compatdata environment preparation, and Steam integration.
func platformSteps(_ *LaunchState) []Step {
	return []Step{
		{Name: "Detecting Proton", Fn: stepDetectProton},
		{Name: "Preparing Proton environment", Fn: stepEnsureCompatdata},
		{Name: "Resolving Steam integration", Fn: stepResolveSteamIntegration},
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

// stepResolveSteamIntegration detects the Steam installation path and resolves
// the non-Steam shortcut app ID for Gamescope window tracking. All failures
// are non-fatal -- the game still launches with fallback values.
func stepResolveSteamIntegration(_ context.Context, state *LaunchState) error {
	steamDir := wine.FindSteamInstall()
	if steamDir == "" {
		ui.Verbose("Steam installation not found, controller tracking may be limited", state.Config.Verbose)
		return nil // Non-fatal
	}
	state.SteamInstallPath = steamDir
	ui.Verbose(fmt.Sprintf("Steam: %s", steamDir), state.Config.Verbose)

	// Resolve non-Steam game app ID from shortcuts.vdf.
	pattern := filepath.Join(steamDir, "userdata", "*", "config", "shortcuts.vdf")
	matches, _ := filepath.Glob(pattern)
	for _, shortcutsPath := range matches {
		data, err := os.ReadFile(shortcutsPath)
		if err != nil {
			continue
		}
		if appID := FindCluckersAppID(data); appID != 0 {
			state.SteamGameId = fmt.Sprintf("%d", appID)
			state.SteamShortcutAppID = appID
			ui.Verbose(fmt.Sprintf("Steam shortcut app ID: %s", state.SteamGameId), state.Config.Verbose)
			return nil
		}
	}

	ui.Verbose("Cluckers shortcut not found in Steam, using default game ID", state.Config.Verbose)
	return nil
}

// platformLaunchStep returns the Linux launch step, which dispatches between
// Steam-managed launch (on Deck with shortcut) and direct proton run (desktop).
func platformLaunchStep() Step {
	return Step{Name: "Launching game", Fn: stepLaunchGameLinux}
}

// stepLaunchGameLinux dispatches between Steam-managed launch and direct proton run.
// On Steam Deck with a configured shortcut, it writes prep config then launches
// via steam://rungameid/. On desktop Linux (or Deck without shortcut), it falls
// back to the standard direct proton run.
func stepLaunchGameLinux(ctx context.Context, state *LaunchState) error {
	if wine.IsSteamDeck() && state.SteamShortcutAppID != 0 {
		// Steam-managed mode: write prep config then launch via Steam.
		if err := stepWriteLaunchConfig(ctx, state); err != nil {
			return err
		}
		ui.Success("Launching via Steam...")
		return launchViaSteam(state.SteamShortcutAppID)
	}

	if wine.IsSteamDeck() {
		ui.Warn("No Steam shortcut found. Run 'cluckers steam add' for better controller support.")
	}

	// Desktop Linux or Deck without shortcut: direct proton run.
	return stepLaunchGame(ctx, state)
}

// launchViaSteam launches the game through Steam using the steam://rungameid/ protocol.
// This lets Steam manage the Proton lifecycle, keeping Steam Input controller
// bindings stable through game map transitions (ServerTravel).
func launchViaSteam(appID uint32) error {
	bpid := CalculateBPID(appID)
	url := fmt.Sprintf("steam://rungameid/%d", bpid)
	cmd := exec.Command("steam", url)
	if err := cmd.Start(); err != nil {
		return &ui.UserError{
			Message:    "Could not launch via Steam",
			Detail:     err.Error(),
			Suggestion: "Make sure Steam is running. On Desktop Mode, start Steam first.",
		}
	}
	return nil
}

// stepDeckConfig patches game settings for Steam Deck (fullscreen, resolution).
// Skips silently on non-Deck systems or if already configured.
func stepDeckConfig(_ context.Context, state *LaunchState) error {
	return PatchDeckConfig(state.GameDir)
}
