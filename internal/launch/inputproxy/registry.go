//go:build linux

package inputproxy

// Stub file for registry patching -- to be implemented in GREEN phase.

// winebusRegContent is the Windows registry content for winebus configuration.
// Stub: empty string -- will be populated in GREEN phase.
var winebusRegContent = ""

// WriteWinebusRegFile creates a temporary .reg file with winebus configuration.
// Stub -- will be implemented in GREEN phase.
func WriteWinebusRegFile() (string, func(), error) {
	return "", func() {}, nil
}
