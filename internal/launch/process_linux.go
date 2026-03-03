//go:build linux

package launch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/0xc0re/cluckers/internal/game"
	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/0xc0re/cluckers/internal/wine"
)

// LaunchGame launches Realm Royale under Proton with the correct arguments.
// It blocks until the game process exits. Temp files (OIDC token, bootstrap,
// shm_launcher.exe) are cleaned up after the game exits or on context cancellation.
// Stderr is captured for error diagnostics; SHM bridge failures produce distinct
// error messages separate from general Proton crashes.
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

	// Build game args matching POC exactly. These are consumed by the Windows
	// game process running under Proton.
	gameArgs := []string{
		fmt.Sprintf("-user=%s", cfg.Username),
		fmt.Sprintf("-token=%s", cfg.AccessToken),
		fmt.Sprintf("-eac_oidc_token_file=%s", wine.LinuxToWinePath(cfg.OIDCTokenPath)),
		fmt.Sprintf("-hostx=%s", cfg.HostX),
		"-Language=INT",
		"-dx11",
		"-seekfreeloadingpcconsole",
		"-nohomedir",
	}

	var shmPath, bootstrapPath, shmName string

	if cfg.ContentBootstrap != nil && len(cfg.ContentBootstrap) > 0 {
		// Extract shm_launcher.exe from embedded binary.
		var shmCleanup func()
		var err error
		shmPath, shmCleanup, err = ExtractSHMLauncher()
		if err != nil {
			return fmt.Errorf("extract shm_launcher: %w", err)
		}
		cleanups = append(cleanups, shmCleanup)

		// Write bootstrap data to temp file.
		var bootstrapCleanup func()
		bootstrapPath, bootstrapCleanup, err = WriteBootstrapFile(cfg.ContentBootstrap)
		if err != nil {
			return fmt.Errorf("write bootstrap file: %w", err)
		}
		cleanups = append(cleanups, bootstrapCleanup)

		// Build SHM name using current process PID.
		shmName = fmt.Sprintf(`Local\realm_content_bootstrap_%d`, os.Getpid())
		gameArgs = append(gameArgs,
			fmt.Sprintf("-content_bootstrap_size=%d", len(cfg.ContentBootstrap)),
			fmt.Sprintf("-content_bootstrap_shm=%s", shmName),
		)
	}

	// Build proton command using helpers from proton_env.go.
	cmdName, cmdArgs := buildProtonCommand(cfg.ProtonScript, shmPath, bootstrapPath, shmName, gameExe, gameArgs)

	// Build proton environment.
	env := buildProtonEnv(cfg.CompatDataPath, cfg.SteamInstallPath, cfg.SteamGameId, cfg.Verbose)

	if cfg.Verbose {
		ui.Verbose(fmt.Sprintf("Proton: %s", cfg.ProtonScript), true)
		ui.Verbose(fmt.Sprintf("Compatdata: %s", cfg.CompatDataPath), true)
		ui.Verbose(fmt.Sprintf("Command: %s %v", cmdName, cmdArgs), true)
	}

	// Execute proton process, blocking until game exits.
	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)
	cmd.Env = env
	cmd.Dir = cfg.GameDir

	// Capture stderr for error diagnostics.
	var stderrBuf bytes.Buffer
	if cfg.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	} else {
		// Suppress Proton noise in non-verbose mode.
		cmd.Stderr = &stderrBuf
	}

	if err := cmd.Run(); err != nil {
		// If context was cancelled (Ctrl+C), don't treat as an error.
		if ctx.Err() != nil {
			return nil
		}

		// Check for SHM bridge-specific failure first (distinct error message).
		if shmErr := shmBridgeError(err, stderrBuf.String(), cfg.CompatDataPath); shmErr != nil {
			return shmErr
		}

		// General Proton launch failure with last 10 stderr lines.
		return &ui.UserError{
			Message:    "Proton launch failed",
			Detail:     lastNLines(stderrBuf.String(), 10),
			Suggestion: protonErrorSuggestion(cfg.CompatDataPath),
		}
	}

	return nil
}
