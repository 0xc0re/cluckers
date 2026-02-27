//go:build windows

package cli

// platformStatusCheck returns nil for Proton and compatdata status on Windows
// since Proton is not used.
func platformStatusCheck() (*protonStatusResult, *compatdataStatusResult) {
	return nil, nil
}
