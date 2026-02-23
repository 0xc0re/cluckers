package launch

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xc0re/cluckers/assets"
)

// DeployXInputProxy writes the XInput remap proxy DLL to the game's binary
// directory. UE3 reserves XInput index 0 for the keyboard and polls indices
// 1-3, but Wine assigns the controller to index 0. The proxy remaps game
// index N to real index N-1, bridging this mismatch.
func DeployXInputProxy(gameDir string) error {
	dst := filepath.Join(gameDir, "Realm-Royale", "Binaries", "Win64", "xinput1_3.dll")

	// Skip if already deployed and same size.
	if info, err := os.Stat(dst); err == nil && info.Size() == int64(len(assets.XInputRemapDLL)) {
		return nil
	}

	if err := os.WriteFile(dst, assets.XInputRemapDLL, 0755); err != nil {
		return fmt.Errorf("deploy xinput proxy: %w", err)
	}
	return nil
}
