package main

import (
	"fmt"
	"os"

	"github.com/0xc0re/cluckers/internal/cli"
	"github.com/0xc0re/cluckers/internal/config"
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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
