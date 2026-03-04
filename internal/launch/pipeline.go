package launch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/0xc0re/cluckers/internal/auth"
	"github.com/0xc0re/cluckers/internal/config"
	"github.com/0xc0re/cluckers/internal/game"
	"github.com/0xc0re/cluckers/internal/gateway"
	"github.com/0xc0re/cluckers/internal/ui"
)

// LaunchState holds accumulated state across pipeline steps.
type LaunchState struct {
	Config              *config.Config
	Client              *gateway.Client
	Username            string
	Password            string
	AccessToken         string
	OIDCToken           string
	Bootstrap           []byte
	ProtonScript        string // Path to the proton Python script (Linux only).
	ProtonDir           string // Root of the Proton-GE installation (Linux only).
	ProtonDisplayVersion string // Human-readable version like "GE-Proton10-1" (Linux only).
	CompatDataPath      string // Path to Proton compatdata directory (Linux only).
	SteamInstallPath    string // Detected Steam root directory (Linux only). Empty if not found.
	SteamGameId         string // Non-Steam shortcut app ID for Gamescope tracking (Linux only). "0" if not found.
	SteamShortcutAppID  uint32 // Non-Steam shortcut appid (parsed from shortcuts.vdf). 0 if not found.
	GameDir             string
	VersionInfo         *game.VersionInfo // Used by prep pipeline only.
	NeedsDownload       bool              // Used by prep pipeline only.
	TokenCache          *auth.TokenCache
	Reporter            ProgressReporter
	OIDCTempFile        string // Path to OIDC JWT temp file for cleanup on interrupt.
}

// Step represents a single step in the launch pipeline.
type Step struct {
	Name string
	Fn   func(ctx context.Context, state *LaunchState) error
}

// Run orchestrates the full launch pipeline: health check, auth, OIDC, bootstrap, game launch.
// Each step shows a spinner while active and a checkmark on completion.
// This is a convenience wrapper that uses CLIReporter for terminal output.
func Run(ctx context.Context, cfg *config.Config) error {
	return RunWithReporter(ctx, cfg, NewCLIReporter())
}

// RunWithReporter orchestrates the full launch pipeline using the provided ProgressReporter
// for step progress callbacks. This allows both CLI (spinners) and GUI (step list) to
// receive pipeline progress updates.
func RunWithReporter(ctx context.Context, cfg *config.Config, reporter ProgressReporter) error {
	// Set up signal handling for clean shutdown.
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Create gateway client and state before signal handler (goroutine captures state).
	client := gateway.NewClient(cfg.Gateway, cfg.Verbose)

	state := &LaunchState{
		Config:   cfg,
		Client:   client,
		Reporter: reporter,
	}

	// Force exit on Ctrl+C — stdin reads block and don't check context.
	// Clean up sensitive OIDC temp files before exiting.
	go func() {
		<-ctx.Done()
		if state.OIDCTempFile != "" {
			os.Remove(state.OIDCTempFile)
		}
		fmt.Println("\nInterrupted.")
		os.Exit(130)
	}()

	steps := buildSteps(state)

	for _, step := range steps {
		reporter.StepStarted(step.Name)

		if err := step.Fn(ctx, state); err != nil {
			reporter.StepFailed(step.Name, err)
			return err
		}

		reporter.StepCompleted(step.Name)
	}

	return nil
}

// RunWithReporterAndCreds orchestrates the full launch pipeline with pre-populated
// credentials. This is used by the GUI where the user has already authenticated via
// the login screen. The credentials are set on the launch state so that stepAuthenticate
// can use them directly without prompting.
func RunWithReporterAndCreds(ctx context.Context, cfg *config.Config, reporter ProgressReporter, username, password string) error {
	// Create gateway client.
	client := gateway.NewClient(cfg.Gateway, cfg.Verbose)

	state := &LaunchState{
		Config:   cfg,
		Client:   client,
		Reporter: reporter,
		Username: username,
		Password: password,
	}

	steps := buildSteps(state)

	for _, step := range steps {
		// Check for context cancellation before starting each step.
		if ctx.Err() != nil {
			return ctx.Err()
		}

		reporter.StepStarted(step.Name)

		if err := step.Fn(ctx, state); err != nil {
			reporter.StepFailed(step.Name, err)
			return err
		}

		reporter.StepCompleted(step.Name)
	}

	return nil
}

// StepNames returns the ordered list of pipeline step names for display.
// This is used by the GUI to create the step list widget before the pipeline runs.
func StepNames(cfg *config.Config) []string {
	// Build a temporary state to get platform steps.
	state := &LaunchState{Config: cfg}
	steps := buildSteps(state)
	names := make([]string, len(steps))
	for i, s := range steps {
		names[i] = s.Name
	}
	return names
}

// buildSteps constructs the ordered list of pipeline steps including platform-specific steps.
func buildSteps(state *LaunchState) []Step {
	steps := []Step{
		{Name: "Checking gateway", Fn: stepHealthCheck},
		{Name: "Authenticating", Fn: stepAuthenticate},
		{Name: "Requesting OIDC token", Fn: stepOIDCToken},
		{Name: "Requesting content bootstrap", Fn: stepBootstrap},
	}
	steps = append(steps, platformSteps(state)...)
	steps = append(steps,
		Step{Name: "Verifying game installation", Fn: stepVerifyGameInstalled},
	)
	steps = append(steps, platformPostSteps(state)...)
	steps = append(steps, platformLaunchStep())
	return steps
}

// stepHealthCheck verifies the gateway is reachable. Warns but continues on failure
// (matching POC behavior -- gateway might be flaky but login still works).
func stepHealthCheck(ctx context.Context, state *LaunchState) error {
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
// When Username and Password are pre-populated on state (GUI mode), uses them directly.
func stepAuthenticate(ctx context.Context, state *LaunchState) error {
	// Load token cache for potential reuse.
	cache, err := auth.LoadTokenCache()
	if err != nil {
		ui.Verbose(fmt.Sprintf("Could not load token cache: %s", err), state.Config.Verbose)
	}

	// Determine credentials source: pre-populated (GUI) or saved/prompted (CLI).
	username := state.Username
	password := state.Password

	if username == "" || password == "" {
		// CLI path: try saved credentials.
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
			username = creds.Username
			password = creds.Password
		}
	} else {
		// GUI path: credentials pre-populated. Check cache for same user.
		if cache != nil && cache.AccessTokenValid() && cache.Username == username {
			state.AccessToken = cache.AccessToken
			state.TokenCache = cache
			ui.Verbose("Using cached access token (still valid)", state.Config.Verbose)
			return nil
		}
	}

	if username != "" && password != "" {
		// Try login with available credentials.
		result, err := auth.Login(ctx, state.Client, username, password)
		if err == nil {
			state.Username = result.Username
			state.AccessToken = result.AccessToken
			state.Password = password

			// Cache the access token for future launches.
			state.TokenCache = &auth.TokenCache{
				Username:       result.Username,
				AccessToken:    result.AccessToken,
				AccessCachedAt: time.Now(),
			}
			if saveErr := auth.SaveTokenCache(state.TokenCache); saveErr != nil {
				ui.Verbose(fmt.Sprintf("Could not save token cache: %s", saveErr), state.Config.Verbose)
			}

			ui.Verbose("Logged in with credentials", state.Config.Verbose)
			return nil
		}

		// If credentials were pre-populated (GUI), don't fall through to prompts.
		if state.Username != "" {
			return err
		}

		// Saved credentials failed -- pause spinner so prompt is visible.
		state.Reporter.StepPaused("Authenticating")
		ui.Warn("Saved credentials failed, please re-enter.")
		ui.Verbose(fmt.Sprintf("Saved login error: %s", err), state.Config.Verbose)
	} else {
		// No saved creds -- pause spinner so prompt is visible.
		state.Reporter.StepPaused("Authenticating")
	}

	// Prompt for credentials (CLI only -- GUI never reaches here).
	promptedUsername, err := ui.PromptUsername()
	if err != nil {
		return err
	}

	promptedPassword, err := ui.PromptPassword()
	if err != nil {
		return err
	}

	result, err := auth.Login(ctx, state.Client, promptedUsername, promptedPassword)
	if err != nil {
		return err
	}

	state.Username = result.Username
	state.AccessToken = result.AccessToken
	state.Password = promptedPassword

	// Save credentials for future launches.
	if saveErr := auth.SaveCredentials(promptedUsername, promptedPassword); saveErr != nil {
		ui.Warn(fmt.Sprintf("Could not save credentials: %s", saveErr))
	}

	// Cache the access token for future launches.
	state.TokenCache = &auth.TokenCache{
		Username:       result.Username,
		AccessToken:    result.AccessToken,
		AccessCachedAt: time.Now(),
	}
	if saveErr := auth.SaveTokenCache(state.TokenCache); saveErr != nil {
		ui.Verbose(fmt.Sprintf("Could not save token cache: %s", saveErr), state.Config.Verbose)
	}

	return nil
}

// stepOIDCToken retrieves an EAC OIDC JWT token from the gateway.
// Checks the token cache first to skip the API call when a valid cached OIDC token exists.
func stepOIDCToken(ctx context.Context, state *LaunchState) error {
	// Check cache for a valid OIDC token.
	if state.TokenCache != nil && state.TokenCache.OIDCTokenValid() {
		state.OIDCToken = state.TokenCache.OIDCToken
		ui.Verbose("Using cached OIDC token (still valid)", state.Config.Verbose)
		return nil
	}

	token, err := auth.GetOIDCToken(ctx, state.Client, state.Username, state.AccessToken)
	if err != nil {
		if !errors.Is(err, auth.ErrTokenRejected) {
			return err
		}

		// Stale access token — re-authenticate and retry once.
		ui.Verbose("Access token rejected, re-authenticating...", state.Config.Verbose)

		if clearErr := auth.ClearTokenCache(); clearErr != nil {
			ui.Verbose(fmt.Sprintf("Could not clear token cache: %s", clearErr), state.Config.Verbose)
		}

		result, loginErr := auth.Login(ctx, state.Client, state.Username, state.Password)
		if loginErr != nil {
			return loginErr
		}
		state.AccessToken = result.AccessToken

		// Save fresh token cache.
		state.TokenCache = &auth.TokenCache{
			Username:       result.Username,
			AccessToken:    result.AccessToken,
			AccessCachedAt: time.Now(),
		}
		if saveErr := auth.SaveTokenCache(state.TokenCache); saveErr != nil {
			ui.Verbose(fmt.Sprintf("Could not save token cache: %s", saveErr), state.Config.Verbose)
		}

		token, err = auth.GetOIDCToken(ctx, state.Client, state.Username, state.AccessToken)
		if err != nil {
			return err
		}
	}
	state.OIDCToken = token

	// Update the token cache with the fresh OIDC token.
	if state.TokenCache == nil {
		state.TokenCache = &auth.TokenCache{
			Username:       state.Username,
			AccessToken:    state.AccessToken,
			AccessCachedAt: time.Now(),
		}
	}
	state.TokenCache.OIDCToken = token
	state.TokenCache.OIDCCachedAt = time.Now()
	if saveErr := auth.SaveTokenCache(state.TokenCache); saveErr != nil {
		ui.Verbose(fmt.Sprintf("Could not save token cache: %s", saveErr), state.Config.Verbose)
	}

	return nil
}

// stepBootstrap retrieves the content bootstrap from the gateway.
// Nil bootstrap is OK -- the game can launch without it.
// If the server rejects the token, re-authenticates once and retries
// (same pattern as stepOIDCToken stale-token handling).
func stepBootstrap(ctx context.Context, state *LaunchState) error {
	data, err := auth.GetContentBootstrap(ctx, state.Client, state.Username, state.AccessToken)
	if err != nil {
		if !errors.Is(err, auth.ErrTokenRejected) {
			return err
		}

		// Stale access token — re-authenticate and retry once.
		ui.Verbose("Access token rejected during bootstrap, re-authenticating...", state.Config.Verbose)

		if clearErr := auth.ClearTokenCache(); clearErr != nil {
			ui.Verbose(fmt.Sprintf("Could not clear token cache: %s", clearErr), state.Config.Verbose)
		}

		result, loginErr := auth.Login(ctx, state.Client, state.Username, state.Password)
		if loginErr != nil {
			return loginErr
		}
		state.AccessToken = result.AccessToken

		// Save fresh token cache.
		state.TokenCache = &auth.TokenCache{
			Username:       result.Username,
			AccessToken:    result.AccessToken,
			AccessCachedAt: time.Now(),
		}
		if saveErr := auth.SaveTokenCache(state.TokenCache); saveErr != nil {
			ui.Verbose(fmt.Sprintf("Could not save token cache: %s", saveErr), state.Config.Verbose)
		}

		data, err = auth.GetContentBootstrap(ctx, state.Client, state.Username, state.AccessToken)
		if err != nil {
			return err
		}
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
// Used by the prep pipeline (prep.go) -- not used in the launch pipeline.
func stepCheckVersion(ctx context.Context, state *LaunchState) error {
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
// Used by the prep pipeline (prep.go) -- not used in the launch pipeline.
func stepDownloadGame(ctx context.Context, state *LaunchState) error {
	if !state.NeedsDownload {
		ui.Verbose("Game files up to date, skipping download", state.Config.Verbose)
		return nil
	}

	// Pause the reporter -- the progress bar handles visual feedback during download.
	state.Reporter.StepPaused("Downloading game update")

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

// stepVerifyGameInstalled checks that the game executable exists on disk and
// that no previous extraction was interrupted.
// The launch pipeline does not download or update game files -- users must
// run `cluckers update` separately before launching.
func stepVerifyGameInstalled(_ context.Context, state *LaunchState) error {
	// Resolve game directory.
	gameDir := state.Config.GameDir
	if gameDir == "" {
		gameDir = game.GameDir()
	}
	state.GameDir = gameDir

	// Check for interrupted extraction before checking exe.
	if game.IsExtractionIncomplete(gameDir) {
		return &ui.UserError{
			Message:    "Game extraction was interrupted.",
			Suggestion: "Run `cluckers update` to re-extract the game files.",
		}
	}

	exePath := game.GameExePath(gameDir)
	if _, err := os.Stat(exePath); err != nil {
		return &ui.UserError{
			Message:    "Game not installed.",
			Suggestion: "Run `cluckers update` to download game files before launching.",
		}
	}

	return nil
}

// stepLaunchGame writes temp files and launches the game.
func stepLaunchGame(ctx context.Context, state *LaunchState) error {
	// Write OIDC token to temp file.
	oidcPath, oidcCleanup, err := writeOIDCTokenFile(state.OIDCToken)
	if err != nil {
		return err
	}
	defer oidcCleanup()

	// Store path for signal handler cleanup (os.Exit bypasses defers).
	state.OIDCTempFile = oidcPath

	return LaunchGame(ctx, &LaunchConfig{
		ProtonScript:       state.ProtonScript,
		ProtonDir:          state.ProtonDir,
		CompatDataPath:     state.CompatDataPath,
		SteamInstallPath:   state.SteamInstallPath,
		SteamGameId:        state.SteamGameId,
		GameDir:            state.GameDir,
		Username:           state.Username,
		AccessToken:        state.AccessToken,
		OIDCTokenPath:      oidcPath,
		ContentBootstrap:   state.Bootstrap,
		HostX:              state.Config.HostX,
		Verbose:            state.Config.Verbose,
	})
}

// writeOIDCTokenFile writes the OIDC token string to a temp file.
func writeOIDCTokenFile(token string) (path string, cleanup func(), err error) {
	tmpDir := config.TmpDir()
	if err := config.EnsureDir(tmpDir); err != nil {
		return "", nil, fmt.Errorf("create temp dir for OIDC token: %w", err)
	}

	f, err := os.CreateTemp(tmpDir, "realm_eac_oidc_*.txt")
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
