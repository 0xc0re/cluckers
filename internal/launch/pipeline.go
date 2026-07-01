package launch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
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
	Config               *config.Config
	Client               *gateway.Client
	Username             string
	Password             string
	AccessToken          string
	Bootstrap            []byte
	ProtonScript         string // Path to the proton Python script (Linux only).
	ProtonDir            string // Root of the Proton-GE installation (Linux only).
	ProtonDisplayVersion string // Human-readable version like "GE-Proton10-1" (Linux only).
	CompatDataPath       string // Path to Proton compatdata directory (Linux only).
	SteamInstallPath     string // Detected Steam root directory (Linux only). Empty if not found.
	SteamGameId          string // Non-Steam shortcut app ID for Gamescope tracking (Linux only). "0" if not found.
	SteamShortcutAppID   uint32 // Non-Steam shortcut appid (parsed from shortcuts.vdf). 0 if not found.
	GameDir              string
	VersionInfo          *game.VersionInfo // Used by prep pipeline only.
	Manifest             *game.Manifest    // Reused between check/download when pinned. Prep only.
	NeedsDownload        bool              // Used by prep pipeline only.
	TokenCache           *auth.TokenCache
	Reporter             ProgressReporter
	TokenTempFile        string // Path to the access-token temp file for cleanup on interrupt.
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
	// Clean up sensitive OIDC temp files before exiting. Listens on a dedicated
	// signal channel (not ctx.Done()) so it never fires on normal return, where
	// defer cancel() closes the context.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		if state.TokenTempFile != "" {
			os.Remove(state.TokenTempFile)
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

// stepBootstrap retrieves the content bootstrap from the gateway using the
// access token as a Bearer credential. Nil bootstrap is OK -- the game can
// launch without it. If the server rejects the token (HTTP 401),
// re-authenticates once and retries.
func stepBootstrap(ctx context.Context, state *LaunchState) error {
	data, err := auth.GetContentBootstrap(ctx, state.Client, state.AccessToken)
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

		data, err = auth.GetContentBootstrap(ctx, state.Client, state.AccessToken)
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

	info, err := game.ResolveVersionInfo(ctx, state.Config.PinnedVersion)
	if err != nil {
		return &ui.UserError{
			Message:    "Could not check game version.",
			Detail:     fmt.Sprintf("%s", err),
			Suggestion: "Check your internet connection and try again.",
		}
	}
	state.VersionInfo = info

	needsUpdate, manifest, err := game.ResolveNeedsUpdate(ctx, gameDir, info)
	if err != nil {
		return fmt.Errorf("checking game version: %w", err)
	}
	state.Manifest = manifest

	if needsUpdate {
		state.NeedsDownload = true
		ui.Verbose(fmt.Sprintf("Game update available: %s", info.LatestVersion), state.Config.Verbose)
	} else {
		ui.Verbose(fmt.Sprintf("Game is up to date (version %s)", info.LatestVersion), state.Config.Verbose)
	}

	return nil
}

// stepDownloadGame syncs the game files to the manifest if needed.
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

	// ResolveNeedsUpdate only fetches the manifest on the pinned path; fetch it
	// here for the latest path.
	manifest := state.Manifest
	if manifest == nil {
		var err error
		manifest, err = game.FetchManifest(ctx, state.VersionInfo)
		if err != nil {
			return &ui.UserError{
				Message:    "Failed to fetch game manifest.",
				Detail:     fmt.Sprintf("%s", err),
				Suggestion: "Check your internet connection and try again.",
			}
		}
	}

	if err := game.SyncManifest(ctx, state.VersionInfo, manifest, state.GameDir, nil); err != nil {
		return &ui.UserError{
			Message:    "Failed to download game update.",
			Detail:     fmt.Sprintf("%s", err),
			Suggestion: "Check your internet connection and try again. Interrupted downloads resume on the next run.",
		}
	}

	ui.Success("Game files updated to version " + state.VersionInfo.LatestVersion)
	return nil
}

// stepVerifyGameInstalled checks that the game executable exists on disk and
// that no previous sync was interrupted.
// The launch pipeline does not download or update game files -- users must
// run `cluckers update` separately before launching.
func stepVerifyGameInstalled(_ context.Context, state *LaunchState) error {
	// Resolve game directory.
	gameDir := state.Config.GameDir
	if gameDir == "" {
		gameDir = game.GameDir()
	}
	state.GameDir = gameDir

	// Check for an interrupted sync before checking the exe.
	if game.IsSyncIncomplete(gameDir) {
		return &ui.UserError{
			Message:    "Game update was interrupted.",
			Suggestion: "Run `cluckers update` to finish downloading the game files.",
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
	// Write the access token to a temp file passed to the game via -token_file.
	tokenPath, tokenCleanup, err := writeTokenFile(state.AccessToken)
	if err != nil {
		return err
	}
	defer tokenCleanup()

	// Store path for signal handler cleanup (os.Exit bypasses defers).
	state.TokenTempFile = tokenPath

	return LaunchGame(ctx, &LaunchConfig{
		ProtonScript:     state.ProtonScript,
		ProtonDir:        state.ProtonDir,
		CompatDataPath:   state.CompatDataPath,
		SteamInstallPath: state.SteamInstallPath,
		SteamGameId:      state.SteamGameId,
		GameDir:          state.GameDir,
		Username:         state.Username,
		AccessToken:      state.AccessToken,
		TokenPath:        tokenPath,
		ContentBootstrap: state.Bootstrap,
		Verbose:          state.Config.Verbose,
	})
}

// writeTokenFile writes the launcher access token to a temp file. The path is
// passed to the game via -token_file (the v1 game reads the token from disk).
func writeTokenFile(token string) (path string, cleanup func(), err error) {
	tmpDir := config.TmpDir()
	if err := config.EnsureDir(tmpDir); err != nil {
		return "", nil, fmt.Errorf("create temp dir for token: %w", err)
	}

	f, err := os.CreateTemp(tmpDir, "realm_token_*.txt")
	if err != nil {
		return "", nil, fmt.Errorf("create temp file for token: %w", err)
	}

	if _, err := f.WriteString(token); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", nil, fmt.Errorf("write token: %w", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", nil, fmt.Errorf("close token temp file: %w", err)
	}

	if err := os.Chmod(f.Name(), 0600); err != nil {
		os.Remove(f.Name())
		return "", nil, fmt.Errorf("chmod token temp file: %w", err)
	}

	cleanup = func() {
		os.Remove(f.Name())
	}

	return f.Name(), cleanup, nil
}
