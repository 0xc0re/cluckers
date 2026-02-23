package launch

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/0xc0re/cluckers/internal/game"
	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/0xc0re/cluckers/internal/wine"
)

// LaunchConfig holds all parameters needed to launch the game under Wine.
type LaunchConfig struct {
	WinePath         string
	WinePrefix       string
	GameDir          string
	Username         string
	AccessToken      string
	OIDCTokenPath    string
	ContentBootstrap []byte
	HostX            string
	Verbose          bool
}

// LaunchGame launches Realm Royale under Wine with the correct arguments.
// It blocks until the game process exits. Temp files (OIDC token, bootstrap,
// shm_launcher.exe) are cleaned up after the game exits or on context cancellation.
func LaunchGame(ctx context.Context, cfg *LaunchConfig) error {
	// Validate game executable exists.
	gameExe := game.GameExePath(cfg.GameDir)
	if _, err := os.Stat(gameExe); err != nil {
		return &ui.UserError{
			Message:    "Game executable not found: " + gameExe,
			Detail:     err.Error(),
			Suggestion: "Set game_dir in ~/.cluckers/config/settings.toml to your Realm Royale install directory.",
		}
	}

	// Collect cleanup functions to run on exit.
	var cleanups []func()
	defer func() {
		for _, fn := range cleanups {
			fn()
		}
	}()

	// Build game args matching POC exactly.
	gameArgs := []string{
		fmt.Sprintf("-user=%s", cfg.Username),
		fmt.Sprintf("-token=%s", cfg.AccessToken),
		fmt.Sprintf("-eac_oidc_token_file=%s", wine.LinuxToWinePath(cfg.OIDCTokenPath)),
		fmt.Sprintf("-hostx=%s", cfg.HostX),
		"-Language=INT",
		"-dx11",
		"-content_bootstrap_size=136",
		"-seekfreeloadingpcconsole",
		"-nohomedir",
	}

	var args []string

	if cfg.ContentBootstrap != nil && len(cfg.ContentBootstrap) > 0 {
		// Extract shm_launcher.exe from embedded binary.
		shmPath, shmCleanup, err := ExtractSHMLauncher()
		if err != nil {
			return fmt.Errorf("extract shm_launcher: %w", err)
		}
		cleanups = append(cleanups, shmCleanup)

		// Write bootstrap data to temp file.
		bootstrapPath, bootstrapCleanup, err := WriteBootstrapFile(cfg.ContentBootstrap)
		if err != nil {
			return fmt.Errorf("write bootstrap file: %w", err)
		}
		cleanups = append(cleanups, bootstrapCleanup)

		// Build SHM name using current process PID.
		shmName := fmt.Sprintf(`Local\realm_content_bootstrap_%d`, os.Getpid())
		gameArgs = append(gameArgs, fmt.Sprintf("-content_bootstrap_shm=%s", shmName))

		// shm_launcher.exe args: <bootstrap_file> <shm_name> <game_exe> [game_args...]
		args = append(args,
			cfg.WinePath,
			shmPath,
			wine.LinuxToWinePath(bootstrapPath),
			shmName,
			wine.LinuxToWinePath(gameExe),
		)
		args = append(args, gameArgs...)
	} else {
		// No bootstrap data -- launch game directly.
		args = append(args, cfg.WinePath, gameExe)
		args = append(args, gameArgs...)
	}

	// Deploy XInput remap proxy for controller support on Linux/Wine.
	if err := DeployXInputProxy(cfg.GameDir); err != nil {
		return fmt.Errorf("deploy xinput proxy: %w", err)
	}

	// Set up environment.
	env := os.Environ()
	if cfg.WinePrefix != "" {
		env = append(env, "WINEPREFIX="+cfg.WinePrefix)
	}
	if wine.IsProtonGE(cfg.WinePath) {
		env = append(env, "WINEFSYNC=1")
	}
	env = append(env, "WINEDLLOVERRIDES=dxgi=n,xinput1_3=n")

	if cfg.Verbose {
		ui.Verbose(fmt.Sprintf("Wine command: %v", args), true)
	}

	// Execute wine process, blocking until game exits.
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = env
	cmd.Dir = cfg.GameDir
	cmd.Stdout = os.Stdout

	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// If context was cancelled (Ctrl+C), don't treat as an error.
		if ctx.Err() != nil {
			return nil
		}
		return &ui.UserError{
			Message: "Game exited with an error.",
			Detail:  err.Error(),
		}
	}

	return nil
}