package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/0xc0re/cluckers/internal/game"
	"github.com/0xc0re/cluckers/internal/gateway"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show launcher status and system info",
	Long:  "Displays system info, game version, server version, and gateway connectivity. Use -v for verbose details.",
	RunE: func(cmd *cobra.Command, args []string) error {
		verbose := Cfg.Verbose

		// Collect all status checks (non-fatal -- gather results then display).
		ps, cs := platformStatusCheck()
		gameStatus := checkGameStatus(cmd.Context())
		gatewayStatus := checkGatewayStatus(cmd.Context())

		if verbose {
			printVerboseStatus(ps, cs, gameStatus, gatewayStatus)
		} else {
			printCompactStatus(ps, cs, gameStatus, gatewayStatus)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

// Status check result types.

type protonStatusResult struct {
	found     bool
	version   string // e.g. "GE-Proton10-1"
	protonDir string // e.g. "/home/user/.steam/..."
	err       error
}

type compatdataStatusResult struct {
	path    string // e.g. "/home/user/.cluckers/compatdata"
	healthy bool
}

type gameStatusResult struct {
	gameDir       string
	localVersion  string
	remoteVersion string
	remoteErr     error
	needsUpdate   bool
	exeExists     bool
}

type gatewayStatusResult struct {
	url    string
	online bool
	err    error
}

// Status check functions with short timeouts.

func checkGameStatus(ctx context.Context) gameStatusResult {
	gameDir := Cfg.GameDir
	if gameDir == "" {
		gameDir = game.GameDir()
	}

	localVer := game.LocalVersion(gameDir)
	exeExists := false
	if _, err := os.Stat(game.GameExePath(gameDir)); err == nil {
		exeExists = true
	}

	// Short timeout for remote version check.
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	info, err := game.FetchVersionInfo(timeoutCtx)
	if err != nil {
		return gameStatusResult{
			gameDir:      gameDir,
			localVersion: localVer,
			remoteErr:    err,
			exeExists:    exeExists,
		}
	}

	needsUpdate, _ := game.NeedsUpdate(gameDir, info)
	return gameStatusResult{
		gameDir:       gameDir,
		localVersion:  localVer,
		remoteVersion: info.LatestVersion,
		needsUpdate:   needsUpdate,
		exeExists:     exeExists,
	}
}

func checkGatewayStatus(ctx context.Context) gatewayStatusResult {
	url := Cfg.Gateway

	// Short timeout for gateway health check.
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	client := gateway.NewClient(url, false)
	err := client.HealthCheck(timeoutCtx)
	return gatewayStatusResult{url: url, online: err == nil, err: err}
}

// Compact (default) output.

func printCompactStatus(ps *protonStatusResult, cs *compatdataStatusResult, gs gameStatusResult, gws gatewayStatusResult) {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	fmt.Println(bold("Cluckers Status"))
	fmt.Println()

	// Proton (Linux only)
	if ps != nil {
		if ps.found {
			fmt.Printf("  %-10s %-45s %s\n", "Proton:", ps.version, green("[OK]"))
		} else {
			fmt.Printf("  %-10s %-45s %s\n", "Proton:", "", red("[NOT FOUND]"))
		}
	}

	// Compatibility data (Linux only)
	if cs != nil {
		if cs.healthy {
			fmt.Printf("  %-10s %-45s %s\n", "Compat:", truncatePath(cs.path, 40), green("[OK]"))
		} else {
			fmt.Printf("  %-10s %-45s %s\n", "Compat:", truncatePath(cs.path, 40), red("[not created]"))
		}
	}

	// Game
	if gs.localVersion == "not installed" {
		fmt.Printf("  %-10s %-45s %s\n", "Game:", "not installed", red("[not installed]"))
	} else if gs.needsUpdate {
		fmt.Printf("  %-10s %-45s %s\n", "Game:", gs.localVersion, yellow("[Update available]"))
	} else {
		fmt.Printf("  %-10s %-45s %s\n", "Game:", gs.localVersion, green("[OK]"))
	}

	// Server
	if gs.remoteErr != nil {
		fmt.Printf("  %-10s %-45s %s\n", "Server:", "", red("[Unavailable]"))
	} else if gs.needsUpdate {
		fmt.Printf("  %-10s %-45s %s\n", "Server:", "v"+gs.remoteVersion, yellow("[Update available]"))
	} else {
		fmt.Printf("  %-10s %-45s %s\n", "Server:", "v"+gs.remoteVersion, green("[Up to date]"))
	}

	// Gateway
	if gws.online {
		fmt.Printf("  %-10s %-45s %s\n", "Gateway:", gws.url, green("[Online]"))
	} else {
		fmt.Printf("  %-10s %-45s %s\n", "Gateway:", gws.url, red("[Offline]"))
	}

	// Actionable hints
	fmt.Println()
	printHints(ps, cs, gs, gws)
}

// Verbose output.

func printVerboseStatus(ps *protonStatusResult, cs *compatdataStatusResult, gs gameStatusResult, gws gatewayStatusResult) {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	fmt.Println(bold("Cluckers Status (verbose)"))
	fmt.Println()

	// Proton (Linux only)
	if ps != nil {
		fmt.Println(bold("Proton:"))
		if ps.found {
			fmt.Printf("  Version: %s\n", ps.version)
			fmt.Printf("  Path:    %s\n", ps.protonDir)
		} else {
			fmt.Printf("  Status:  %s\n", red("Not found"))
		}
		fmt.Println()
	}

	// Compatibility data (Linux only)
	if cs != nil {
		fmt.Println(bold("Compatibility Data:"))
		fmt.Printf("  Path:    %s\n", cs.path)
		if cs.healthy {
			fmt.Printf("  Status:  %s\n", green("Ready"))
		} else {
			fmt.Printf("  Status:  %s\n", yellow("Not yet created (will be created on first launch)"))
		}
		fmt.Println()
	}

	// Game
	fmt.Println(bold("Game:"))
	fmt.Printf("  Path:    %s\n", gs.gameDir)
	fmt.Printf("  Version: %s\n", gs.localVersion)
	exePath := game.GameExePath(gs.gameDir)
	if gs.exeExists {
		fmt.Printf("  Exe:     %s %s\n", exePath, green("[OK]"))
	} else {
		fmt.Printf("  Exe:     %s %s\n", exePath, red("[MISSING]"))
	}
	fmt.Println()

	// Server
	fmt.Println(bold("Server:"))
	if gs.remoteErr != nil {
		fmt.Printf("  Status:  %s\n", red("Unavailable"))
		fmt.Printf("  Error:   %s\n", gs.remoteErr)
	} else {
		fmt.Printf("  Latest:  %s\n", gs.remoteVersion)
		if gs.needsUpdate {
			fmt.Printf("  Status:  %s\n", yellow("Update available"))
		} else {
			fmt.Printf("  Status:  %s\n", green("Up to date"))
		}
	}
	fmt.Println()

	// Gateway
	fmt.Println(bold("Gateway:"))
	fmt.Printf("  URL:     %s\n", gws.url)
	if gws.online {
		fmt.Printf("  Health:  %s\n", green("Online"))
	} else {
		fmt.Printf("  Health:  %s\n", red("Offline"))
		if gws.err != nil {
			fmt.Printf("  Error:   %s\n", gws.err)
		}
	}
	fmt.Println()

	// Actionable hints
	printHints(ps, cs, gs, gws)
}

// printHints shows actionable fix suggestions for detected problems.
func printHints(ps *protonStatusResult, cs *compatdataStatusResult, gs gameStatusResult, gws gatewayStatusResult) {
	dim := color.New(color.Faint).SprintFunc()
	hasHints := false

	if ps != nil && !ps.found {
		if !hasHints {
			fmt.Println(dim("Hints:"))
			hasHints = true
		}
		fmt.Println(dim("  - Install Proton-GE for your distribution (see https://github.com/GloriousEggroll/proton-ge-custom)"))
	}

	if cs != nil && !cs.healthy {
		if !hasHints {
			fmt.Println(dim("Hints:"))
			hasHints = true
		}
		fmt.Println(dim("  - Run `cluckers launch` to auto-create Proton environment"))
	}

	if gs.localVersion == "not installed" || gs.needsUpdate {
		if !hasHints {
			fmt.Println(dim("Hints:"))
			hasHints = true
		}
		fmt.Println(dim("  - Run `cluckers update` to download game files"))
	}

	if !gws.online {
		if !hasHints {
			fmt.Println(dim("Hints:"))
			hasHints = true
		}
		fmt.Println(dim("  - Check your internet connection"))
	}
}

// truncatePath shortens a path for compact display.
func truncatePath(p string, maxLen int) string {
	if len(p) <= maxLen {
		return p
	}
	return "..." + p[len(p)-maxLen+3:]
}
