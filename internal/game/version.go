package game

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xc0re/cluckers/internal/config"
	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/zeebo/blake3"
)

// UpdaterURL is the endpoint for game version information.
const UpdaterURL = "https://updater.realmhub.io/builds/version.json"

// GameVersionDatRelPath is the manifest-relative path of the version marker
// used to decide whether the installed game matches a given build.
const GameVersionDatRelPath = "Realm-Royale/Binaries/GameVersion.dat"

// VersionInfo holds the response from the updater API.
type VersionInfo struct {
	LatestVersion        string `json:"latest_version"`
	BaseURL              string `json:"base_url"`
	ManifestURL          string `json:"manifest_url"`
	GameVersionDatPath   string `json:"gameversion_dat_path"`
	GameVersionDatBLAKE3 string `json:"gameversion_dat_blake3"`
	GameVersionDatSize   int64  `json:"gameversion_dat_size"`
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
// Also returns true if a previous sync was interrupted.
func NeedsUpdate(gameDir string, remote *VersionInfo) (bool, error) {
	// An interrupted sync means files are inconsistent — force re-download.
	if IsSyncIncomplete(gameDir) {
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

// PinVersionInfo returns a VersionInfo targeting a specific build instead of
// latest. The updater API only ever describes latest, but its URLs are
// version-templated, so a pinned VersionInfo is derived by substituting the
// version token in BaseURL/ManifestURL.
//
// The pinned build's GameVersion.dat hash and size are unknown (the API carries
// them only for latest), so they are cleared; callers must decide via the pinned
// manifest (see NeedsUpdateFromManifest / ResolveNeedsUpdate).
//
// When version is empty or already equals latest.LatestVersion, latest is
// returned unchanged. The input is never mutated.
func PinVersionInfo(latest *VersionInfo, version string) *VersionInfo {
	if version == "" || version == latest.LatestVersion || latest.LatestVersion == "" {
		return latest
	}
	pinned := *latest
	pinned.LatestVersion = version
	pinned.BaseURL = strings.ReplaceAll(latest.BaseURL, latest.LatestVersion, version)
	pinned.ManifestURL = strings.ReplaceAll(latest.ManifestURL, latest.LatestVersion, version)
	pinned.GameVersionDatBLAKE3 = ""
	pinned.GameVersionDatSize = 0
	return &pinned
}

// ResolveVersionInfo fetches the latest version info and, if pinned is set to a
// different build, rewrites it to target that pinned build instead.
func ResolveVersionInfo(ctx context.Context, pinned string) (*VersionInfo, error) {
	info, err := FetchVersionInfo(ctx)
	if err != nil {
		return nil, err
	}
	if pinned != "" && pinned != info.LatestVersion {
		if info.LatestVersion == "" {
			return nil, &ui.UserError{
				Message:    "Cannot pin the game version.",
				Detail:     "The updater did not report a latest version to derive pinned URLs from.",
				Suggestion: "Remove the version pin or try again later.",
			}
		}
		return PinVersionInfo(info, pinned), nil
	}
	return info, nil
}

// NeedsUpdateFromManifest reports whether the local install matches the
// GameVersion.dat entry in the manifest. It is source-agnostic and works for
// pinned builds whose dat hash is not carried by the updater API.
func NeedsUpdateFromManifest(gameDir string, m *Manifest) (bool, error) {
	if IsSyncIncomplete(gameDir) {
		return true, nil
	}

	var entry *ManifestFile
	for i := range m.Files {
		if m.Files[i].Path == GameVersionDatRelPath {
			entry = &m.Files[i]
			break
		}
	}
	if entry == nil {
		return false, fmt.Errorf("manifest has no %s entry", GameVersionDatRelPath)
	}

	datPath := filepath.Join(gameDir, filepath.FromSlash(entry.Path))
	data, err := os.ReadFile(datPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, fmt.Errorf("reading GameVersion.dat: %w", err)
	}
	if int64(len(data)) != entry.Size {
		return true, nil
	}
	hash := blake3.Sum256(data)
	if hex.EncodeToString(hash[:]) != entry.Hash {
		return true, nil
	}
	return false, nil
}

// ResolveNeedsUpdate decides whether gameDir needs syncing to info. For a latest
// build (info carries the GameVersion.dat hash) it compares cheaply without
// fetching the manifest. For a pinned build (hash unknown) it fetches the
// manifest and compares against it, returning that manifest so callers can reuse
// it for the subsequent sync.
func ResolveNeedsUpdate(ctx context.Context, gameDir string, info *VersionInfo) (bool, *Manifest, error) {
	if info.GameVersionDatBLAKE3 != "" {
		needs, err := NeedsUpdate(gameDir, info)
		return needs, nil, err
	}
	m, err := FetchManifest(ctx, info)
	if err != nil {
		return false, nil, err
	}
	needs, err := NeedsUpdateFromManifest(gameDir, m)
	return needs, m, err
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
