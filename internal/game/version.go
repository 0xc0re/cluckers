package game

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/0xc0re/cluckers/internal/config"
	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/zeebo/blake3"
)

// UpdaterURL is the endpoint for game version information.
const UpdaterURL = "https://updater.realmhub.io/builds/version.json"

// VersionInfo holds the response from the updater API.
type VersionInfo struct {
	LatestVersion        string `json:"latest_version"`
	BaseURL              string `json:"base_url"`
	ManifestURL          string `json:"manifest_url"`
	GameVersionDatPath   string `json:"gameversion_dat_path"`
	GameVersionDatBLAKE3 string `json:"gameversion_dat_blake3"`
	GameVersionDatSize   int64  `json:"gameversion_dat_size"`
	ZipURL               string `json:"zip_url"`
	ZipBLAKE3            string `json:"zip_blake3"`
	ZipSize              int64  `json:"zip_size"`
}

// FetchVersionInfo retrieves the current version information from the updater API.
// The updater API does not require authentication.
func FetchVersionInfo(ctx context.Context) (*VersionInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, UpdaterURL, nil)
	if err != nil {
		return nil, &ui.UserError{
			Message:    "Failed to create version check request.",
			Detail:     err.Error(),
			Suggestion: "Check your internet connection or try again later.",
			Err:        err,
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, &ui.UserError{
			Message:    "Failed to check for game updates.",
			Detail:     err.Error(),
			Suggestion: "Check your internet connection or try again later.",
			Err:        err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &ui.UserError{
			Message:    fmt.Sprintf("Version check returned HTTP %d.", resp.StatusCode),
			Detail:     fmt.Sprintf("GET %s returned status %s", UpdaterURL, resp.Status),
			Suggestion: "Check your internet connection or try again later.",
		}
	}

	var info VersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, &ui.UserError{
			Message:    "Failed to parse version information.",
			Detail:     err.Error(),
			Suggestion: "Check your internet connection or try again later.",
			Err:        err,
		}
	}

	return &info, nil
}

// NeedsUpdate checks whether the local game files need updating by comparing
// the local GameVersion.dat BLAKE3 hash against the remote version.
// Also returns true if a previous extraction was interrupted.
func NeedsUpdate(gameDir string, remote *VersionInfo) (bool, error) {
	// Interrupted extraction means files are inconsistent — force re-download.
	if IsExtractionIncomplete(gameDir) {
		return true, nil
	}

	datPath := filepath.Join(gameDir, remote.GameVersionDatPath)

	data, err := os.ReadFile(datPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, fmt.Errorf("reading GameVersion.dat: %w", err)
	}

	// Check size first as a quick comparison.
	if int64(len(data)) != remote.GameVersionDatSize {
		return true, nil
	}

	// Compute BLAKE3 hash and compare.
	hash := blake3.Sum256(data)
	localHash := hex.EncodeToString(hash[:])
	if localHash != remote.GameVersionDatBLAKE3 {
		return true, nil
	}

	return false, nil
}

// LocalVersion returns a human-readable summary of the locally installed game version.
func LocalVersion(gameDir string) string {
	datPath := filepath.Join(gameDir, "Realm-Royale", "Binaries", "GameVersion.dat")

	info, err := os.Stat(datPath)
	if err != nil {
		return "not installed"
	}

	return fmt.Sprintf("present (%d bytes)", info.Size())
}

// GameDir returns the default game directory under the Cluckers data directory.
func GameDir() string {
	return filepath.Join(config.DataDir(), "game")
}

// GameExePath returns the full path to the game executable within a game directory.
func GameExePath(gameDir string) string {
	return filepath.Join(gameDir, "Realm-Royale", "Binaries", "Win64", "ShippingPC-RealmGameNoEditor.exe")
}
