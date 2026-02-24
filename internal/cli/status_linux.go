//go:build linux

package cli

import (
	"path/filepath"

	"github.com/0xc0re/cluckers/internal/wine"
)

// platformStatusCheck returns Wine and prefix status on Linux.
func platformStatusCheck() (*wineStatusResult, *prefixStatusResult) {
	ws := checkWineStatus()
	ps := checkPrefixStatus()
	return &ws, &ps
}

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

	// Build base names for verbose display.
	dllNames := make([]string, len(wine.RequiredDLLs))
	for i, dll := range wine.RequiredDLLs {
		dllNames[i] = filepath.Base(dll)
	}

	return prefixStatusResult{
		path:         prefixPath,
		healthy:      healthy,
		missing:      missing,
		requiredDLLs: dllNames,
	}
}
