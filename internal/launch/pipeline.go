package launch

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/0xc0re/cluckers/internal/auth"
	"github.com/0xc0re/cluckers/internal/config"
	"github.com/0xc0re/cluckers/internal/game"
	"github.com/0xc0re/cluckers/internal/gateway"
	"github.com/0xc0re/cluckers/internal/ui"
)

// LaunchState holds accumulated state across pipeline steps.
type LaunchState struct {
	Config        *config.Config
	Client        *gateway.Client
	Username      string
	Password      string
	AccessToken   string
	OIDCToken     string
	Bootstrap     []byte
	WinePath      string
	PrefixPath    string
	GameDir       string
	VersionInfo   *game.VersionInfo
	NeedsDownload bool
	TokenCache    *auth.TokenCache
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
	}
	steps = append(steps, platformSteps(state)...)
	steps = append(steps,
		Step{Name: "Checking game version", Fn: stepCheckVersion},
		Step{Name: "Downloading game update", Fn: stepDownloadGame},
	)
	steps = append(steps, platformPostSteps(state)...)
	steps = append(steps, Step{Name: "Launching game", Fn: stepLaunchGame})

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
// Checks the token cache first to skip the API call when a valid cached token exists.
func stepAuthenticate(ctx context.Context, state *LaunchState, spinner *ui.StepSpinner) error {
	// Load token cache for potential reuse.
	cache, err := auth.LoadTokenCache()
	if err != nil {
		ui.Verbose(fmt.Sprintf("Could not load token cache: %s", err), state.Config.Verbose)
	}

	// Try saved credentials first.
	creds, err := auth.LoadCredentials()
	if err != nil {
		ui.Verbose(fmt.Sprintf("Could not load saved credentials: %s", err), state.Config.Verbose)
	}

	// If cache has a valid access token for the same user, use it directly.
	if cache != nil && cache.AccessTokenValid() && creds != nil && cache.Username == creds.Username {
		state.Username = cache.Username
		state.AccessToken = cache.AccessToken
		state.Password = creds.Password
		state.TokenCache = cache
		ui.Verbose("Using cached access token (still valid)", state.Config.Verbose)
		return nil
	}

	if creds != nil {
		// Try login with saved credentials.
		result, err := auth.Login(ctx, state.Client, creds.Username, creds.Password)
		if err == nil {
			state.Username = result.Username
			state.AccessToken = result.AccessToken
			state.Password = creds.Password

			// Cache the access token for future launches.
			state.TokenCache = &auth.TokenCache{
				Username:    result.Username,
				AccessToken: result.AccessToken,
			}
			if saveErr := auth.SaveTokenCache(state.TokenCache); saveErr != nil {
				ui.Verbose(fmt.Sprintf("Could not save token cache: %s", saveErr), state.Config.Verbose)
			}

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

	// Cache the access token for future launches.
	state.TokenCache = &auth.TokenCache{
		Username:    result.Username,
		AccessToken: result.AccessToken,
	}
	if saveErr := auth.SaveTokenCache(state.TokenCache); saveErr != nil {
		ui.Verbose(fmt.Sprintf("Could not save token cache: %s", saveErr), state.Config.Verbose)
	}

	return nil
}

// stepOIDCToken retrieves an EAC OIDC JWT token from the gateway.
// Checks the token cache first to skip the API call when a valid cached OIDC token exists.
func stepOIDCToken(ctx context.Context, state *LaunchState, _ *ui.StepSpinner) error {
	// Check cache for a valid OIDC token.
	if state.TokenCache != nil && state.TokenCache.OIDCTokenValid() {
		state.OIDCToken = state.TokenCache.OIDCToken
		ui.Verbose("Using cached OIDC token (still valid)", state.Config.Verbose)
		return nil
	}

	token, err := auth.GetOIDCToken(ctx, state.Client, state.Username, state.AccessToken)
	if err != nil {
		return err
	}
	state.OIDCToken = token

	// Update the token cache with the fresh OIDC token.
	if state.TokenCache == nil {
		state.TokenCache = &auth.TokenCache{
			Username:    state.Username,
			AccessToken: state.AccessToken,
		}
	}
	state.TokenCache.OIDCToken = token
	if saveErr := auth.SaveTokenCache(state.TokenCache); saveErr != nil {
		ui.Verbose(fmt.Sprintf("Could not save token cache: %s", saveErr), state.Config.Verbose)
	}

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

// stepLaunchGame writes temp files and launches the game.
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
		GameDir:          state.GameDir,
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
