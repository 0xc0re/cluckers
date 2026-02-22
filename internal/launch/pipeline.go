package launch

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/cstory/cluckers/internal/auth"
	"github.com/cstory/cluckers/internal/config"
	"github.com/cstory/cluckers/internal/game"
	"github.com/cstory/cluckers/internal/gateway"
	"github.com/cstory/cluckers/internal/ui"
	"github.com/cstory/cluckers/internal/wine"
)

// LaunchState holds accumulated state across pipeline steps.
type LaunchState struct {
	Config       *config.Config
	Client       *gateway.Client
	Username     string
	Password     string
	AccessToken  string
	OIDCToken    string
	Bootstrap    []byte
	WinePath     string
	PrefixPath   string
	GameDir      string
	VersionInfo  *game.VersionInfo
	NeedsDownload bool
}

// Step represents a single step in the launch pipeline.
type Step struct {
	Name string
	Fn   func(ctx context.Context, state *LaunchState, spinner *ui.StepSpinner) error
}

// Run orchestrates the full launch pipeline: health check, auth, OIDC, bootstrap, game launch.
// Each step shows a spinner while active and a checkmark on completion.
func Run(ctx context.Context, cfg *config.Config) error {
	// Set up signal handling for clean shutdown.
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Force exit on Ctrl+C — stdin reads block and don't check context.
	go func() {
		<-ctx.Done()
		fmt.Println("\nInterrupted.")
		os.Exit(130)
	}()

	// Create gateway client.
	client := gateway.NewClient(cfg.Gateway, cfg.Verbose)

	state := &LaunchState{
		Config: cfg,
		Client: client,
	}

	steps := []Step{
		{Name: "Checking gateway", Fn: stepHealthCheck},
		{Name: "Authenticating", Fn: stepAuthenticate},
		{Name: "Requesting OIDC token", Fn: stepOIDCToken},
		{Name: "Requesting content bootstrap", Fn: stepBootstrap},
		{Name: "Detecting Wine", Fn: stepDetectWine},
		{Name: "Ensuring Wine prefix", Fn: stepEnsurePrefix},
		{Name: "Verifying Wine prefix", Fn: stepVerifyPrefix},
		{Name: "Checking game version", Fn: stepCheckVersion},
		{Name: "Downloading game update", Fn: stepDownloadGame},
		{Name: "Launching game", Fn: stepLaunchGame},
	}

	for _, step := range steps {
		spinner := ui.StartStep(step.Name)

		if err := step.Fn(ctx, state, spinner); err != nil {
			spinner.Fail()
			ui.Error(ui.FormatError(err, cfg.Verbose))
			return err
		}

		spinner.Success()
	}

	return nil
}

// stepHealthCheck verifies the gateway is reachable. Warns but continues on failure
// (matching POC behavior -- gateway might be flaky but login still works).
func stepHealthCheck(ctx context.Context, state *LaunchState, _ *ui.StepSpinner) error {
	if err := state.Client.HealthCheck(ctx); err != nil {
		// Warn but continue -- gateway might be flaky.
		ui.Warn("Gateway health check failed, continuing anyway...")
		ui.Verbose(fmt.Sprintf("Health check error: %s", err), state.Config.Verbose)
	}
	return nil
}

// stepAuthenticate loads saved credentials or prompts for new ones, then logs in.
// On saved credential failure, re-prompts once before returning an error.
func stepAuthenticate(ctx context.Context, state *LaunchState, spinner *ui.StepSpinner) error {
	// Try saved credentials first.
	creds, err := auth.LoadCredentials()
	if err != nil {
		ui.Verbose(fmt.Sprintf("Could not load saved credentials: %s", err), state.Config.Verbose)
	}

	if creds != nil {
		// Try login with saved credentials.
		result, err := auth.Login(ctx, state.Client, creds.Username, creds.Password)
		if err == nil {
			state.Username = result.Username
			state.AccessToken = result.AccessToken
			state.Password = creds.Password
			ui.Verbose("Logged in with saved credentials", state.Config.Verbose)
			return nil
		}
		// Saved credentials failed -- re-prompt once.
		spinner.Stop()
		ui.Warn("Saved credentials failed, please re-enter.")
		ui.Verbose(fmt.Sprintf("Saved login error: %s", err), state.Config.Verbose)
	} else {
		// No saved creds -- stop spinner so prompt is visible.
		spinner.Stop()
	}

	// Prompt for credentials.
	username, err := ui.PromptUsername()
	if err != nil {
		return err
	}

	password, err := ui.PromptPassword()
	if err != nil {
		return err
	}

	result, err := auth.Login(ctx, state.Client, username, password)
	if err != nil {
		return err
	}

	state.Username = result.Username
	state.AccessToken = result.AccessToken
	state.Password = password

	// Save credentials for future launches.
	if saveErr := auth.SaveCredentials(username, password); saveErr != nil {
		ui.Warn(fmt.Sprintf("Could not save credentials: %s", saveErr))
	}

	return nil
}

// stepOIDCToken retrieves an EAC OIDC JWT token from the gateway.
func stepOIDCToken(ctx context.Context, state *LaunchState, _ *ui.StepSpinner) error {
	token, err := auth.GetOIDCToken(ctx, state.Client, state.Username, state.AccessToken)
	if err != nil {
		return err
	}
	state.OIDCToken = token
	return nil
}

// stepBootstrap retrieves the content bootstrap from the gateway.
// Nil bootstrap is OK -- the game can launch without it.
func stepBootstrap(ctx context.Context, state *LaunchState, _ *ui.StepSpinner) error {
	data, err := auth.GetContentBootstrap(ctx, state.Client, state.Username, state.AccessToken)
	if err != nil {
		return err
	}
	state.Bootstrap = data
	if data == nil {
		ui.Warn("No content bootstrap received (game may still work)")
	} else {
		ui.Verbose(fmt.Sprintf("Content bootstrap: %d bytes", len(data)), state.Config.Verbose)
	}
	return nil
}

// stepDetectWine finds a suitable Wine binary.
func stepDetectWine(ctx context.Context, state *LaunchState, _ *ui.StepSpinner) error {
	winePath, err := wine.FindWine(state.Config.WinePath)
	if err != nil {
		return err
	}
	state.WinePath = winePath
	ui.Verbose(fmt.Sprintf("Wine: %s", winePath), state.Config.Verbose)
	return nil
}

// stepEnsurePrefix ensures the Wine prefix exists, creating it if needed.
func stepEnsurePrefix(ctx context.Context, state *LaunchState, _ *ui.StepSpinner) error {
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
func stepVerifyPrefix(ctx context.Context, state *LaunchState, _ *ui.StepSpinner) error {
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

// stepCheckVersion checks the remote game version and determines if a download is needed.
func stepCheckVersion(ctx context.Context, state *LaunchState, _ *ui.StepSpinner) error {
	// Resolve game directory.
	gameDir := state.Config.GameDir
	if gameDir == "" {
		gameDir = game.GameDir()
	}
	state.GameDir = gameDir

	info, err := game.FetchVersionInfo(ctx)
	if err != nil {
		return &ui.UserError{
			Message:    "Could not check game version.",
			Detail:     fmt.Sprintf("%s", err),
			Suggestion: "Check your internet connection and try again.",
		}
	}
	state.VersionInfo = info

	needsUpdate, err := game.NeedsUpdate(gameDir, info)
	if err != nil {
		return fmt.Errorf("checking game version: %w", err)
	}

	if needsUpdate {
		state.NeedsDownload = true
		ui.Verbose(fmt.Sprintf("Game update available: %s", info.LatestVersion), state.Config.Verbose)
	} else {
		ui.Verbose(fmt.Sprintf("Game is up to date (version %s)", info.LatestVersion), state.Config.Verbose)
	}

	return nil
}

// stepDownloadGame downloads and extracts the game update if needed.
// This step is a no-op when the game is already up to date.
func stepDownloadGame(ctx context.Context, state *LaunchState, spinner *ui.StepSpinner) error {
	if !state.NeedsDownload {
		ui.Verbose("Game files up to date, skipping download", state.Config.Verbose)
		return nil
	}

	// Stop the spinner -- the progress bar handles visual feedback during download.
	spinner.Stop()

	if err := config.EnsureDir(state.GameDir); err != nil {
		return fmt.Errorf("creating game directory: %w", err)
	}

	if err := game.DownloadAndVerify(ctx, state.VersionInfo, state.GameDir); err != nil {
		return &ui.UserError{
			Message:    "Failed to download game update.",
			Detail:     fmt.Sprintf("%s", err),
			Suggestion: "Check your internet connection and try again. Partial downloads will be resumed.",
		}
	}

	zipPath := filepath.Join(state.GameDir, "game.zip")
	if err := game.ExtractZip(zipPath, state.GameDir); err != nil {
		return &ui.UserError{
			Message:    "Failed to extract game files.",
			Detail:     fmt.Sprintf("%s", err),
			Suggestion: "Run `cluckers update` to retry.",
		}
	}

	ui.Success("Game files updated to version " + state.VersionInfo.LatestVersion)
	return nil
}

// stepLaunchGame writes temp files and launches the game under Wine.
func stepLaunchGame(ctx context.Context, state *LaunchState, _ *ui.StepSpinner) error {
	// Write OIDC token to temp file.
	oidcPath, oidcCleanup, err := writeOIDCTokenFile(state.OIDCToken)
	if err != nil {
		return err
	}
	defer oidcCleanup()

	return LaunchGame(ctx, &LaunchConfig{
		WinePath:         state.WinePath,
		WinePrefix:       state.PrefixPath,
		GameDir:          state.Config.GameDir,
		Username:         state.Username,
		AccessToken:      state.AccessToken,
		OIDCTokenPath:    oidcPath,
		ContentBootstrap: state.Bootstrap,
		HostX:            state.Config.HostX,
		Verbose:          state.Config.Verbose,
	})
}

// writeOIDCTokenFile writes the OIDC token string to a temp file.
func writeOIDCTokenFile(token string) (path string, cleanup func(), err error) {
	f, err := os.CreateTemp("", "realm_eac_oidc_*.txt")
	if err != nil {
		return "", nil, fmt.Errorf("create temp file for OIDC token: %w", err)
	}

	if _, err := f.WriteString(token); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", nil, fmt.Errorf("write OIDC token: %w", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", nil, fmt.Errorf("close OIDC token temp file: %w", err)
	}

	cleanup = func() {
		os.Remove(f.Name())
	}

	return f.Name(), cleanup, nil
}
