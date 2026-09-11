package launch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
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
	LaunchToken          string    // Per-launch game token from launch-auth (written to -token_file).
	LaunchExpiresAt      time.Time // Zero if the gateway did not report it.
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

	// tokenTempFile holds the launch-token temp file path for cleanup on
	// interrupt. Atomic because the signal-handler goroutine reads it while
	// the pipeline goroutine writes it.
	tokenTempFile atomic.Pointer[string]
}

// SetTokenTempFile records the launch-token temp file path for interrupt cleanup.
func (s *LaunchState) SetTokenTempFile(path string) {
	s.tokenTempFile.Store(&path)
}

// TokenTempFile returns the recorded launch-token temp file path, or "" if unset.
func (s *LaunchState) TokenTempFile() string {
	if p := s.tokenTempFile.Load(); p != nil {
		return *p
	}
	return ""
}

// Step represents a single step in the launch pipeline.
type Step struct {
	Name string
	Fn   func(ctx context.Context, state *LaunchState) error
}

// Run orchestrates the full launch pipeline: health check, auth, install check, launch-auth, game launch.
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
		if path := state.TokenTempFile(); path != "" {
			os.Remove(path)
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
	// Launch authorization runs after the install check because the request
	// carries the installed game build version, and as late as possible
	// because the launch token is short-lived.
	steps := []Step{
		{Name: "Checking gateway", Fn: stepHealthCheck},
		{Name: "Authenticating", Fn: stepAuthenticate},
		{Name: "Verifying game installation", Fn: stepVerifyGameInstalled},
		{Name: "Requesting launch authorization", Fn: stepLaunchAuth},
	}
	steps = append(steps, platformSteps(state)...)
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

// stepAuthenticate obtains a session with the least intrusive method (cached
// access token, refresh token, saved or pre-populated password), prompting on
// the terminal only when the CLI has nothing usable. It also runs the Discord
// link flow on the terminal when the account is not linked yet. In GUI mode
// (Username and Password pre-populated on state) it never prompts: link/PIN
// outcomes are returned as errors for the GUI to handle.
func stepAuthenticate(ctx context.Context, state *LaunchState) error {
	cache, err := auth.LoadTokenCache()
	if err != nil {
		ui.Verbose(fmt.Sprintf("Could not load token cache: %s", err), state.Config.Verbose)
	}

	guiMode := state.Username != "" && state.Password != ""
	username, password := state.Username, state.Password
	var creds *auth.Credentials
	if !guiMode {
		creds, err = auth.LoadCredentials()
		if err != nil {
			ui.Verbose(fmt.Sprintf("Could not load saved credentials: %s", err), state.Config.Verbose)
		}
		if creds != nil {
			username, password = creds.Username, creds.Password
		}
	}

	sess, err := auth.EnsureSession(ctx, state.Client, auth.SessionRequest{
		Username: username, Password: password, Cache: cache, Verbose: state.Config.Verbose,
	})
	if err == nil {
		state.applySession(sess, password)
		return nil
	}
	if guiMode {
		return err
	}

	var nl *auth.NotLinkedError
	switch {
	case errors.Is(err, auth.ErrPinRequired):
		return err
	case errors.As(err, &nl):
		// Saved credentials are fine; the account just needs linking.
		state.Reporter.StepPaused("Authenticating")
		result, linkErr := WaitForLinkInteractive(ctx, state.Client, username, password, nl.LinkCode)
		if linkErr != nil {
			return linkErr
		}
		return state.applyLogin(result, password)
	case errors.Is(err, auth.ErrNoCredentials):
		state.Reporter.StepPaused("Authenticating")
	default:
		state.Reporter.StepPaused("Authenticating")
		ui.Warn("Saved credentials failed, please re-enter.")
		ui.Verbose(fmt.Sprintf("Saved login error: %s", err), state.Config.Verbose)
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

	result, err := LoginInteractive(ctx, state.Client, promptedUsername, promptedPassword)
	if err != nil {
		return err
	}

	// Save credentials for future launches.
	if saveErr := auth.SaveCredentials(promptedUsername, promptedPassword); saveErr != nil {
		ui.Warn(fmt.Sprintf("Could not save credentials: %s", saveErr))
	}
	return state.applyLogin(result, promptedPassword)
}

// applySession records an established session on the state. The password is
// kept so a later token rejection can fall back to a full login.
func (s *LaunchState) applySession(sess *auth.Session, password string) {
	s.Username = sess.Username
	s.AccessToken = sess.AccessToken
	s.Password = password
	s.TokenCache = sess.Cache
	switch sess.Source {
	case auth.SourceCache:
		ui.Verbose("Using cached access token (still valid)", s.Config.Verbose)
	case auth.SourceRefresh:
		ui.Verbose("Refreshed the launcher session", s.Config.Verbose)
	default:
		ui.Verbose("Logged in with credentials", s.Config.Verbose)
	}
}

// applyLogin caches a fresh login result and records it on the state.
func (s *LaunchState) applyLogin(result *auth.LoginResult, password string) error {
	cache := auth.NewTokenCache(result)
	if saveErr := auth.SaveTokenCache(cache); saveErr != nil {
		ui.Verbose(fmt.Sprintf("Could not save token cache: %s", saveErr), s.Config.Verbose)
	}
	s.applySession(&auth.Session{Username: cache.Username, AccessToken: cache.AccessToken, Cache: cache, Source: auth.SourceLogin}, password)
	return nil
}

// stepLaunchAuth requests the per-launch artifacts (content bootstrap and
// launch token) from the gateway. If the access token is rejected it obtains
// a new session (refresh token first, then password) and retries once.
func stepLaunchAuth(ctx context.Context, state *LaunchState) error {
	build := resolveClientBuild(ctx, state)

	res, err := auth.LaunchAuth(ctx, state.Client, state.AccessToken, build)
	if err != nil {
		if !errors.Is(err, auth.ErrTokenRejected) {
			return err
		}
		ui.Verbose("Access token rejected during launch authorization, renewing session...", state.Config.Verbose)
		if state.TokenCache != nil {
			state.TokenCache.InvalidateAccess()
		}
		sess, sessErr := auth.EnsureSession(ctx, state.Client, auth.SessionRequest{
			Username: state.Username, Password: state.Password, Cache: state.TokenCache, Verbose: state.Config.Verbose,
		})
		if sessErr != nil {
			if errors.Is(sessErr, auth.ErrNoCredentials) {
				return &ui.UserError{
					Message:    "Your session expired and no saved credentials are available to renew it.",
					Suggestion: "Run 'cluckers login' and launch again.",
					Err:        sessErr,
				}
			}
			return sessErr
		}
		state.applySession(sess, state.Password)

		res, err = auth.LaunchAuth(ctx, state.Client, state.AccessToken, build)
		if err != nil {
			return err
		}
	}

	state.Bootstrap = res.Bootstrap
	state.LaunchToken = res.LaunchToken
	state.LaunchExpiresAt = res.LaunchExpiresAt
	if res.Bootstrap == nil {
		ui.Warn("No content bootstrap received (game may still work)")
	} else {
		ui.Verbose(fmt.Sprintf("Content bootstrap: %d bytes", len(res.Bootstrap)), state.Config.Verbose)
	}
	if !res.LaunchExpiresAt.IsZero() {
		ui.Verbose(fmt.Sprintf("Launch token valid for %s", time.Until(res.LaunchExpiresAt).Truncate(time.Second)), state.Config.Verbose)
	}
	return nil
}

// resolveClientBuild determines the installed game build version for the
// x-realm-client-build header. Failure is not fatal: the header is omitted
// and the gateway decides.
func resolveClientBuild(ctx context.Context, state *LaunchState) string {
	gameDir := state.GameDir
	if gameDir == "" {
		gameDir = state.Config.GameDir
		if gameDir == "" {
			gameDir = game.GameDir()
		}
	}
	build, err := game.InstalledVersion(ctx, gameDir, state.Config.PinnedVersion)
	if err != nil {
		ui.Verbose(fmt.Sprintf("Installed game version unknown, launch-auth will not send x-realm-client-build: %s", err), state.Config.Verbose)
		return ""
	}
	ui.Verbose("Installed game build: "+build, state.Config.Verbose)
	return build
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
	if state.LaunchToken == "" {
		return &ui.UserError{
			Message:    "No launch token available.",
			Suggestion: "This is a launcher bug: launch authorization did not run. Please report it.",
		}
	}

	// Write the launch token to a temp file passed to the game via -token_file.
	tokenPath, tokenCleanup, err := writeTokenFile(state.LaunchToken)
	if err != nil {
		return err
	}
	defer tokenCleanup()

	// Store path for signal handler cleanup (os.Exit bypasses defers).
	state.SetTokenTempFile(tokenPath)

	return LaunchGame(ctx, &LaunchConfig{
		ProtonScript:     state.ProtonScript,
		ProtonDir:        state.ProtonDir,
		CompatDataPath:   state.CompatDataPath,
		SteamInstallPath: state.SteamInstallPath,
		SteamGameId:      state.SteamGameId,
		GameDir:          state.GameDir,
		Username:         state.Username,
		LaunchToken:      state.LaunchToken,
		TokenPath:        tokenPath,
		ContentBootstrap: state.Bootstrap,
		Verbose:          state.Config.Verbose,
	})
}

// writeTokenFile writes the per-launch game token to a temp file. The path is
// passed to the game via -token_file (the game reads the token from disk).
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
