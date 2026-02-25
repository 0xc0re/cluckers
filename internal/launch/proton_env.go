//go:build linux

package launch

import (
	"fmt"
	"os"
	"strings"

	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/0xc0re/cluckers/internal/wine"
)

// strippedEnvKeys are environment variables that must be removed before
// launching via proton run. These conflict with Proton's own environment
// setup and can cause crashes or incorrect Wine prefix usage.
var strippedEnvKeys = []string{
	"LD_LIBRARY_PATH",
	"WINEPREFIX",
	"WINE",
	"WINEDLLOVERRIDES",
	"WINEFSYNC",
	"WINEESYNC",
}

// filterEnv removes environment variables matching any of the given key names
// from the env slice. Matching is done on the part before the first "=".
func filterEnv(env []string, keys ...string) []string {
	keySet := make(map[string]bool, len(keys))
	for _, k := range keys {
		keySet[k] = true
	}

	filtered := make([]string, 0, len(env))
	for _, entry := range env {
		k, _, _ := strings.Cut(entry, "=")
		if keySet[k] {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

// buildProtonEnv constructs the environment variable slice for proton run
// by filtering the current process environment and appending required Proton
// variables. This is the public entry point that uses os.Environ().
func buildProtonEnv(compatDataPath, steamInstallPath, steamGameId string, verbose bool) []string {
	return buildProtonEnvFrom(os.Environ(), compatDataPath, steamInstallPath, steamGameId, verbose)
}

// buildProtonEnvFrom constructs the environment variable slice for proton run
// from a provided base environment. Exported for testability with deterministic
// input. Strips conflicting Wine/AppImage variables and adds required Proton vars.
func buildProtonEnvFrom(baseEnv []string, compatDataPath, steamInstallPath, steamGameId string, verbose bool) []string {
	env := filterEnv(baseEnv, strippedEnvKeys...)

	// Default steamGameId to "0" when not resolved (detection failed or not in Steam).
	if steamGameId == "" {
		steamGameId = "0"
	}

	// Required Proton environment variables.
	// SteamAppId is set to match SteamGameId — Proton Wine reads SteamAppId
	// for X11 class hints (steam_app_{id}), which Gamescope uses for window tracking.
	env = append(env,
		"STEAM_COMPAT_DATA_PATH="+compatDataPath,
		"STEAM_COMPAT_CLIENT_INSTALL_PATH="+steamInstallPath,
		"SteamGameId="+steamGameId,
		"SteamAppId="+steamGameId,
		"WINEDLLOVERRIDES=dxgi=n",
	)

	if verbose {
		env = append(env, "PROTON_LOG=1")
	}

	return env
}

// buildProtonCommand constructs the python3 proton run command with correct
// argument ordering for both SHM (bootstrap present) and non-SHM modes.
//
// With SHM: python3 <protonScript> run <shmPath> <Z:\bootstrapPath> <shmName> <Z:\gameExe> <gameArgs...>
// Without SHM: python3 <protonScript> run <gameExe> <gameArgs...>
//
// shmPath is a Linux path (proton converts it internally). bootstrapPath and
// gameExe (in SHM mode) are Wine Z: drive paths because they are consumed by
// Windows processes running under Wine.
func buildProtonCommand(protonScript, shmPath, bootstrapPath, shmName, gameExe string, gameArgs []string) (string, []string) {
	args := []string{protonScript, "run"}

	if shmPath != "" {
		// SHM mode: launch shm_launcher.exe which creates shared memory
		// and spawns the game as a child process.
		args = append(args,
			shmPath,
			wine.LinuxToWinePath(bootstrapPath),
			shmName,
			wine.LinuxToWinePath(gameExe),
		)
	} else {
		// No SHM: launch game directly. gameExe is a Linux path here
		// since proton converts it internally.
		args = append(args, gameExe)
	}

	args = append(args, gameArgs...)
	return "python3", args
}

// protonErrorSuggestion returns actionable error suggestion text for Proton
// launch failures. The three steps match the locked user decision for
// Proton error recovery.
func protonErrorSuggestion(compatDataPath string) string {
	return fmt.Sprintf(
		"1. Delete %s/ and relaunch\n2. Update Proton-GE to latest version\n3. Run `cluckers update` to verify game files",
		compatDataPath,
	)
}

// shmBridgeError inspects a process exit error and stderr output for
// shm_launcher-specific failure patterns. Returns a distinct UserError for
// SHM bridge failures (separate from general Proton crashes), or nil if
// the error is not SHM-related.
func shmBridgeError(exitErr error, stderr string, compatDataPath string) *ui.UserError {
	if exitErr == nil {
		return nil
	}

	lowerStderr := strings.ToLower(stderr)
	patterns := []string{
		"createfilemapping",
		"openfilemapping",
		"shm_launcher",
		"shared memory",
	}

	for _, p := range patterns {
		if strings.Contains(lowerStderr, p) {
			return &ui.UserError{
				Message: "Shared memory bridge failed",
				Detail:  lastNLines(stderr, 10),
				Suggestion: fmt.Sprintf(
					"Try deleting %s and relaunching. If the problem persists, run 'cluckers update' to verify game files.",
					compatDataPath,
				),
			}
		}
	}

	return nil
}

// lastNLines returns the last n lines from a string. If the string has
// fewer than n lines, it is returned as-is. An empty string returns empty.
func lastNLines(s string, n int) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}
