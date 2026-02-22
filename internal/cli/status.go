package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/0xc0re/cluckers/internal/game"
	"github.com/0xc0re/cluckers/internal/gateway"
	"github.com/0xc0re/cluckers/internal/wine"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show launcher status and system info",
	Long:  "Displays Wine detection, prefix health, game version, server version, and gateway connectivity. Use -v for verbose details.",
	RunE: func(cmd *cobra.Command, args []string) error {
		verbose := Cfg.Verbose

		// Collect all status checks (non-fatal -- gather results then display).
		wineStatus := checkWineStatus()
		prefixStatus := checkPrefixStatus()
		gameStatus := checkGameStatus(cmd.Context())
		gatewayStatus := checkGatewayStatus(cmd.Context())

		if verbose {
			printVerboseStatus(wineStatus, prefixStatus, gameStatus, gatewayStatus)
		} else {
			printCompactStatus(wineStatus, prefixStatus, gameStatus, gatewayStatus)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

// Status check result types.

type wineStatusResult struct {
	found    bool
	path     string
	wineType string // "Proton-GE" or "System Wine"
	err      error
}

type prefixStatusResult struct {
	path    string
	healthy bool
	missing []string
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
	url     string
	online  bool
	err     error
}

// Status check functions with short timeouts.

func checkWineStatus() wineStatusResult {
	path, err := wine.FindWine(Cfg.WinePath)
	if err != nil {
		return wineStatusResult{found: false, err: err}
	}
	wineType := "System Wine"
	if wine.IsProtonGE(path) {
		wineType = "Proton-GE"
	}
	return wineStatusResult{found: true, path: path, wineType: wineType}
}

func checkPrefixStatus() prefixStatusResult {
	prefixPath := Cfg.WinePrefix
	if prefixPath == "" {
		prefixPath = wine.PrefixPath()
	}

	healthy, missing := wine.VerifyPrefix(prefixPath)
	return prefixStatusResult{path: prefixPath, healthy: healthy, missing: missing}
}

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

func printCompactStatus(ws wineStatusResult, ps prefixStatusResult, gs gameStatusResult, gws gatewayStatusResult) {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	fmt.Println(bold("Cluckers Status"))
	fmt.Println()

	// Wine
	if ws.found {
		label := truncatePath(ws.path, 40)
		fmt.Printf("  %-10s %-45s %s\n", "Wine:", ws.wineType+" ("+label+")", green("[OK]"))
	} else {
		fmt.Printf("  %-10s %-45s %s\n", "Wine:", "", red("[NOT FOUND]"))
	}

	// Prefix
	if ps.healthy {
		fmt.Printf("  %-10s %-45s %s\n", "Prefix:", truncatePath(ps.path, 40), green("[OK]"))
	} else if len(ps.missing) > 0 {
		detail := fmt.Sprintf("%d missing DLLs", len(ps.missing))
		fmt.Printf("  %-10s %-45s %s\n", "Prefix:", truncatePath(ps.path, 40), red("["+detail+"]"))
	} else {
		fmt.Printf("  %-10s %-45s %s\n", "Prefix:", truncatePath(ps.path, 40), red("[not created]"))
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
	printHints(ws, ps, gs, gws)
}

// Verbose output.

func printVerboseStatus(ws wineStatusResult, ps prefixStatusResult, gs gameStatusResult, gws gatewayStatusResult) {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	fmt.Println(bold("Cluckers Status (verbose)"))
	fmt.Println()

	// Wine
	fmt.Println(bold("Wine:"))
	if ws.found {
		fmt.Printf("  Binary:  %s\n", ws.path)
		fmt.Printf("  Type:    %s\n", ws.wineType)
	} else {
		fmt.Printf("  Status:  %s\n", red("Not found"))
	}
	fmt.Println()

	// Prefix
	fmt.Println(bold("Prefix:"))
	fmt.Printf("  Path:    %s\n", ps.path)
	if ps.healthy {
		fmt.Printf("  DLLs:    ")
		for i, dll := range wine.RequiredDLLs {
			name := filepath.Base(dll)
			if i > 0 {
				fmt.Print("  ")
			}
			fmt.Printf("%s %s", name, green("[OK]"))
		}
		fmt.Println()
	} else {
		fmt.Printf("  DLLs:    ")
		missingSet := make(map[string]bool)
		for _, m := range ps.missing {
			missingSet[m] = true
		}
		for i, dll := range wine.RequiredDLLs {
			name := filepath.Base(dll)
			if i > 0 {
				fmt.Print("  ")
			}
			if missingSet[name] {
				fmt.Printf("%s %s", name, red("[MISSING]"))
			} else {
				fmt.Printf("%s %s", name, green("[OK]"))
			}
		}
		fmt.Println()
	}
	fmt.Println()

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
	printHints(ws, ps, gs, gws)
}

// printHints shows actionable fix suggestions for detected problems.
func printHints(ws wineStatusResult, ps prefixStatusResult, gs gameStatusResult, gws gatewayStatusResult) {
	dim := color.New(color.Faint).SprintFunc()
	hasHints := false

	if !ws.found {
		if !hasHints {
			fmt.Println(dim("Hints:"))
			hasHints = true
		}
		fmt.Println(dim("  - Install Wine or Proton-GE for your distribution"))
	}

	if !ps.healthy && len(ps.missing) > 0 {
		if !hasHints {
			fmt.Println(dim("Hints:"))
			hasHints = true
		}
		fmt.Println(dim("  - Run `cluckers launch` to auto-create prefix"))
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
