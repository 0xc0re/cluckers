package main

import (
	"fmt"
	"os"

	"github.com/0xc0re/cluckers/internal/cli"
	"github.com/0xc0re/cluckers/internal/config"
	"github.com/0xc0re/cluckers/internal/ui"
)

// Set via ldflags at build time (goreleaser).
var (
	version    = "dev"
	commit     = "none"
	date       = "unknown"
	gatewayURL = "https://gateway-dev.project-crown.com"
	hostxIP    = "157.90.131.105"
)

func main() {
	cli.SetVersion(fmt.Sprintf("%s (commit %s, built %s)", version, commit, date))
	config.SetBuildDefaults(gatewayURL, hostxIP)
	cli.InitFlags()

	if err := cli.Execute(); err != nil {
		// Use FormatError to show suggestions and details for UserErrors.
		// Verbose is not available here (config loads inside Cobra), so
		// check if -v was passed by looking at os.Args.
		verbose := false
		for _, arg := range os.Args {
			if arg == "-v" || arg == "--verbose" {
				verbose = true
				break
			}
		}
		fmt.Fprintln(os.Stderr, ui.FormatError(err, verbose))
		os.Exit(1)
	}
}
