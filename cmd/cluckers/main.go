package main

import (
	"fmt"
	"os"

	"github.com/cstory/cluckers/internal/cli"
)

// Set via ldflags at build time (goreleaser).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cli.SetVersion(fmt.Sprintf("%s (commit %s, built %s)", version, commit, date))

	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
