//go:build linux

package wine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RequiredDLLs lists the DLLs that must exist in the Wine prefix for the game to work.
var RequiredDLLs = []string{
	"drive_c/windows/system32/vcruntime140.dll",
	"drive_c/windows/system32/msvcp140.dll",
	"drive_c/windows/system32/d3dx11_43.dll",
	"drive_c/windows/system32/d3d11.dll", // DXVK
}

// VerifyPrefix checks that all required DLLs exist in the Wine prefix.
// Returns true if all DLLs are present, and a list of missing DLL base names.
func VerifyPrefix(prefixPath string) (healthy bool, missing []string) {
	for _, dll := range RequiredDLLs {
		fullPath := filepath.Join(prefixPath, dll)
		if _, err := os.Stat(fullPath); err != nil {
			missing = append(missing, filepath.Base(dll))
		}
	}
	return len(missing) == 0, missing
}

// RepairInstructions returns actionable repair instructions based on Wine type
// and the list of missing DLLs.
func RepairInstructions(winePath string, missing []string) string {
	missingList := strings.Join(missing, ", ")

	if IsProtonGE(winePath) {
		return fmt.Sprintf(
			"Missing DLLs: %s\n"+
				"  To repair: delete your Wine prefix directory and re-launch Cluckers.\n"+
				"  The prefix will be recreated from the Proton-GE template.\n"+
				"  Prefix location: %s",
			missingList, PrefixPath(),
		)
	}

	return fmt.Sprintf(
		"Missing DLLs: %s\n"+
			"  To repair, run:\n"+
			"    WINEPREFIX=%s winetricks vcrun2022 d3dx11_43 dxvk\n"+
			"  Or delete the prefix directory and re-launch Cluckers to recreate it.",
		missingList, PrefixPath(),
	)
}
