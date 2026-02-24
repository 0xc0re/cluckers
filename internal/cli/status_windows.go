//go:build windows

package cli

// platformStatusCheck returns nil for Wine and prefix status on Windows
// since Wine is not used.
func platformStatusCheck() (*wineStatusResult, *prefixStatusResult) {
	return nil, nil
}
