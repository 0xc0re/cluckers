//go:build linux

package launch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xc0re/cluckers/assets"
	"github.com/0xc0re/cluckers/internal/config"
	"github.com/0xc0re/cluckers/internal/game"
	"github.com/0xc0re/cluckers/internal/gateway"
	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/0xc0re/cluckers/internal/wine"
)

// Fixed SHM name for prep mode. Only one Steam instance runs at a time,
// so a PID-based name is unnecessary.
const prepSHMName = `Local\realm_content_bootstrap_cluckers`

// RunPrep executes the prep pipeline: auth, tokens, bootstrap, platform setup,
// version check, download, then writes persistent config files for Steam-managed
// launch via %command%.
func RunPrep(ctx context.Context, cfg *config.Config) error {
	return RunPrepWithReporter(ctx, cfg, NewCLIReporter())
}

// RunPrepWithReporter executes the prep pipeline with a custom reporter.
func RunPrepWithReporter(ctx context.Context, cfg *config.Config, reporter ProgressReporter) error {
	client := gateway.NewClient(cfg.Gateway, cfg.Verbose)

	state := &LaunchState{
		Config:   cfg,
		Client:   client,
		Reporter: reporter,
	}

	steps := buildPrepSteps(state)

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

// buildPrepSteps constructs the ordered list of prep pipeline steps.
// Same as buildSteps but replaces stepLaunchGame with stepWriteLaunchConfig.
func buildPrepSteps(state *LaunchState) []Step {
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
	steps = append(steps, Step{Name: "Writing launch config", Fn: stepWriteLaunchConfig})
	return steps
}

// stepWriteLaunchConfig writes persistent files for Steam-managed launch:
//   - ~/.cluckers/cache/bootstrap.bin   — content bootstrap bytes
//   - ~/.cluckers/cache/oidc-token.txt  — OIDC token string
//   - ~/.cluckers/bin/shm_launcher.exe  — extracted from embedded asset
//   - ~/.cluckers/bin/launch-config.txt — Wine-path args for shm_launcher
func stepWriteLaunchConfig(_ context.Context, state *LaunchState) error {
	if state.Bootstrap == nil || len(state.Bootstrap) == 0 {
		return &ui.UserError{
			Message:    "Content bootstrap is required for prep mode",
			Suggestion: "Try again — the gateway may have been temporarily unavailable.",
		}
	}

	cacheDir := config.CacheDir()
	binDir := config.BinDir()

	if err := config.EnsureDir(cacheDir); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}
	if err := config.EnsureDir(binDir); err != nil {
		return fmt.Errorf("creating bin directory: %w", err)
	}

	// 1. Write bootstrap.bin
	bootstrapPath := filepath.Join(cacheDir, "bootstrap.bin")
	if err := os.WriteFile(bootstrapPath, state.Bootstrap, 0600); err != nil {
		return fmt.Errorf("writing bootstrap.bin: %w", err)
	}

	// 2. Write oidc-token.txt
	oidcPath := filepath.Join(cacheDir, "oidc-token.txt")
	if err := os.WriteFile(oidcPath, []byte(state.OIDCToken), 0600); err != nil {
		return fmt.Errorf("writing oidc-token.txt: %w", err)
	}

	// 3. Extract shm_launcher.exe to bin dir (idempotent).
	shmDest := filepath.Join(binDir, "shm_launcher.exe")
	if err := ExtractSHMLauncherTo(shmDest); err != nil {
		return fmt.Errorf("extracting shm_launcher.exe: %w", err)
	}

	// 4. Build and write launch-config.txt
	gameExe := game.GameExePath(state.GameDir)

	lines := []string{
		wine.LinuxToWinePath(bootstrapPath),
		prepSHMName,
		wine.LinuxToWinePath(gameExe),
		fmt.Sprintf("-user=%s", state.Username),
		fmt.Sprintf("-token=%s", state.AccessToken),
		fmt.Sprintf("-eac_oidc_token_file=%s", wine.LinuxToWinePath(oidcPath)),
		fmt.Sprintf("-hostx=%s", state.Config.HostX),
		"-Language=INT",
		"-dx11",
		fmt.Sprintf("-content_bootstrap_size=%d", len(state.Bootstrap)),
		"-seekfreeloadingpcconsole",
		"-nohomedir",
		fmt.Sprintf("-content_bootstrap_shm=%s", prepSHMName),
	}

	configContent := ""
	for _, line := range lines {
		configContent += line + "\n"
	}

	configPath := filepath.Join(binDir, "launch-config.txt")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("writing launch-config.txt: %w", err)
	}

	ui.Verbose(fmt.Sprintf("Launch config written to %s", configPath), state.Config.Verbose)
	return nil
}

// ExtractSHMLauncherTo writes the embedded shm_launcher.exe to a fixed path.
// Skips the write if the destination exists and has the same size as the
// embedded binary (idempotent across launches).
func ExtractSHMLauncherTo(destPath string) error {
	info, err := os.Stat(destPath)
	if err == nil && info.Size() == int64(len(assets.SHMLauncherExe)) {
		return nil // Already up to date.
	}

	if err := os.WriteFile(destPath, assets.SHMLauncherExe, 0755); err != nil {
		return fmt.Errorf("writing shm_launcher.exe: %w", err)
	}

	return nil
}
