//go:build windows

package launch

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/0xc0re/cluckers/internal/config"
	"github.com/0xc0re/cluckers/internal/game"
	"github.com/0xc0re/cluckers/internal/ui"
)

// LaunchGame launches Realm Royale directly on Windows (no Wine needed).
// shm_launcher.exe runs natively to create shared memory for content bootstrap.
// It blocks until the game process exits. Temp files are cleaned up on exit.
func LaunchGame(ctx context.Context, cfg *LaunchConfig) error {
	// Validate game executable exists.
	gameExe := game.GameExePath(cfg.GameDir)
	if _, err := os.Stat(gameExe); err != nil {
		return &ui.UserError{
			Message:    "Game executable not found: " + gameExe,
			Detail:     err.Error(),
			Suggestion: "Set game_dir in your cluckers config to your Realm Royale install directory.",
		}
	}

	// Collect cleanup functions to run on exit.
	var cleanups []func()
	defer func() {
		for _, fn := range cleanups {
			fn()
		}
	}()

	// Build game args (same as Linux, but paths are native Windows paths).
	gameArgs := []string{
		fmt.Sprintf("-user=%s", cfg.Username),
		fmt.Sprintf("-token=%s", cfg.AccessToken),
		fmt.Sprintf("-eac_oidc_token_file=%s", cfg.OIDCTokenPath),
		fmt.Sprintf("-hostx=%s", cfg.HostX),
		"-Language=INT",
		"-dx11",
		"-seekfreeloadingpcconsole",
		"-nohomedir",
	}

	var cmdPath string
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
		gameArgs = append(gameArgs,
			fmt.Sprintf("-content_bootstrap_size=%d", len(cfg.ContentBootstrap)),
			fmt.Sprintf("-content_bootstrap_shm=%s", shmName),
		)

		// shm_launcher.exe runs natively on Windows.
		// Args: <bootstrap_file> <shm_name> <game_exe> [game_args...]
		cmdPath = shmPath
		args = append(args,
			bootstrapPath,
			shmName,
			gameExe,
		)
		args = append(args, gameArgs...)
	} else {
		// No bootstrap data -- launch game directly.
		cmdPath = gameExe
		args = gameArgs
	}

	if cfg.Verbose {
		ui.Verbose(fmt.Sprintf("Launch command: %s %v", cmdPath, args), true)
	}

	// Execute game process, blocking until it exits.
	cmd := exec.CommandContext(ctx, cmdPath, args...)
	cmd.Dir = cfg.GameDir
	cmd.Stdout = os.Stdout

	// Tee stderr to a log file for diagnostics.
	logDir := config.TmpDir()
	_ = config.EnsureDir(logDir)
	logPath := filepath.Join(logDir, "cluckers_game.log")
	gameLog, logErr := os.Create(logPath)
	if logErr == nil {
		cmd.Stderr = io.MultiWriter(os.Stderr, gameLog)
		cleanups = append(cleanups, func() { gameLog.Close() })
	} else {
		cmd.Stderr = os.Stderr
	}

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
