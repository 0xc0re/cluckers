//go:build linux

package cli

import (
	"github.com/0xc0re/cluckers/internal/wine"
)

// platformStatusCheck returns Proton and compatdata status on Linux.
// Uses the legacy wineStatusResult/prefixStatusResult types for compatibility
// with the shared status.go display code (will be fully rewritten in Plan 02).
func platformStatusCheck() (*wineStatusResult, *prefixStatusResult) {
	ws := checkProtonStatus()
	ps := checkCompatdataStatus()
	return &ws, &ps
}

func checkProtonStatus() wineStatusResult {
	install, err := wine.FindProton(Cfg.WinePath)
	if err != nil {
		return wineStatusResult{found: false, err: err}
	}
	return wineStatusResult{found: true, path: install.ProtonDir, wineType: "Proton-GE"}
}

func checkCompatdataStatus() prefixStatusResult {
	compatdata := wine.CompatdataPath()
	healthy := wine.CompatdataHealthy(compatdata)
	return prefixStatusResult{
		path:    compatdata,
		healthy: healthy,
	}
}
