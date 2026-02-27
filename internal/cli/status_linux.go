//go:build linux

package cli

import (
	"github.com/0xc0re/cluckers/internal/wine"
)

// platformStatusCheck returns Proton and compatdata status on Linux.
func platformStatusCheck() (*protonStatusResult, *compatdataStatusResult) {
	ps := checkProtonStatus()
	cs := checkCompatdataStatus()
	return &ps, &cs
}

func checkProtonStatus() protonStatusResult {
	install, err := wine.FindProton(Cfg.WinePath)
	if err != nil {
		return protonStatusResult{found: false, err: err}
	}
	return protonStatusResult{
		found:     true,
		version:   install.DisplayVersion(),
		protonDir: install.ProtonDir,
	}
}

func checkCompatdataStatus() compatdataStatusResult {
	compatdata := wine.CompatdataPath()
	healthy := wine.CompatdataHealthy(compatdata)
	return compatdataStatusResult{
		path:    compatdata,
		healthy: healthy,
	}
}
