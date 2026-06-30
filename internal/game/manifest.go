package game

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/0xc0re/cluckers/internal/ui"
)

// manifestSchema is the manifest schema version this client understands.
const manifestSchema = 1

// Manifest is the per-version file manifest served at VersionInfo.ManifestURL.
// It lists every game file with its relative path, BLAKE3 hash, and size.
type Manifest struct {
	Schema  int            `json:"schema"`
	Version string         `json:"version"`
	Files   []ManifestFile `json:"files"`
}

// ManifestFile describes a single game file. Path is relative to the game
// directory and uses forward slashes. Hash is the file's BLAKE3 digest in hex.
type ManifestFile struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

// FetchManifest retrieves and parses the file manifest referenced by the
// version info. The manifest endpoint does not require authentication.
func FetchManifest(ctx context.Context, info *VersionInfo) (*Manifest, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.ManifestURL, nil)
	if err != nil {
		return nil, &ui.UserError{
			Message:    "Failed to create manifest request.",
			Detail:     err.Error(),
			Suggestion: "Check your internet connection or try again later.",
			Err:        err,
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, &ui.UserError{
			Message:    "Failed to fetch the game file manifest.",
			Detail:     err.Error(),
			Suggestion: "Check your internet connection or try again later.",
			Err:        err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &ui.UserError{
			Message:    fmt.Sprintf("Manifest fetch returned HTTP %d.", resp.StatusCode),
			Detail:     fmt.Sprintf("GET %s returned status %s", info.ManifestURL, resp.Status),
			Suggestion: "Try again later or check the Cluckers Discord for server status.",
		}
	}

	var m Manifest
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, &ui.UserError{
			Message:    "Failed to parse the game file manifest.",
			Detail:     err.Error(),
			Suggestion: "Try again later or check the Cluckers Discord for server status.",
			Err:        err,
		}
	}

	if m.Schema != manifestSchema {
		return nil, &ui.UserError{
			Message:    "Unsupported game manifest version.",
			Detail:     fmt.Sprintf("manifest schema %d, this launcher supports %d", m.Schema, manifestSchema),
			Suggestion: "Update Cluckers with `cluckers self-update` and try again.",
		}
	}

	return &m, nil
}
