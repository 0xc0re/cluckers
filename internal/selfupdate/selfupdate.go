package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/schollz/progressbar/v3"
)

const (
	githubOwner    = "0xc0re"
	githubRepo     = "cluckers"
	latestReleaseURL = "https://api.github.com/repos/" + githubOwner + "/" + githubRepo + "/releases/latest"
)

// ReleaseInfo holds the parsed response from the GitHub releases API.
type ReleaseInfo struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a single downloadable asset in a GitHub release.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// CheckLatestVersion fetches the latest release information from GitHub.
func CheckLatestVersion(ctx context.Context) (*ReleaseInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		return nil, &ui.UserError{
			Message:    "Failed to create update check request.",
			Detail:     err.Error(),
			Suggestion: "Check your internet connection or try again later.",
			Err:        err,
		}
	}
	req.Header.Set("User-Agent", "cluckers-self-update")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, &ui.UserError{
			Message:    "Failed to check for launcher updates.",
			Detail:     err.Error(),
			Suggestion: "Check your internet connection or try again later.",
			Err:        err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &ui.UserError{
			Message:    fmt.Sprintf("GitHub API returned HTTP %d.", resp.StatusCode),
			Detail:     fmt.Sprintf("GET %s returned status %s", latestReleaseURL, resp.Status),
			Suggestion: "Check your internet connection or try again later.",
		}
	}

	var info ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, &ui.UserError{
			Message:    "Failed to parse release information.",
			Detail:     err.Error(),
			Suggestion: "Check your internet connection or try again later.",
			Err:        err,
		}
	}

	return &info, nil
}

// IsNewer compares the current version against the latest tag from GitHub.
// Returns true only if latestTag is strictly newer than currentVersion.
// Returns false if currentVersion is "dev" (dev builds should not self-update).
func IsNewer(currentVersion, latestTag string) bool {
	if currentVersion == "dev" {
		return false
	}

	current := strings.TrimPrefix(currentVersion, "v")
	latest := strings.TrimPrefix(latestTag, "v")

	currentParts := parseVersion(current)
	latestParts := parseVersion(latest)

	// Compare each segment.
	maxLen := len(currentParts)
	if len(latestParts) > maxLen {
		maxLen = len(latestParts)
	}

	for i := 0; i < maxLen; i++ {
		var c, l int
		if i < len(currentParts) {
			c = currentParts[i]
		}
		if i < len(latestParts) {
			l = latestParts[i]
		}
		if l > c {
			return true
		}
		if l < c {
			return false
		}
	}

	return false
}

// parseVersion splits a version string on "." and converts each segment to an int.
func parseVersion(v string) []int {
	parts := strings.Split(v, ".")
	result := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			result[i] = 0
		} else {
			result[i] = n
		}
	}
	return result
}

// assetName returns the expected archive asset name for the current platform.
func (r *ReleaseInfo) assetName() string {
	version := strings.TrimPrefix(r.TagName, "v")
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}

	return fmt.Sprintf("cluckers_%s_%s_%s.%s", version, goos, goarch, ext)
}

// FindAsset locates the platform-specific archive asset and the checksums.txt
// asset from the release. Returns (archive, checksums, error). The checksums
// asset may be nil if not found.
func FindAsset(info *ReleaseInfo) (*Asset, *Asset, error) {
	expectedName := info.assetName()

	var archive *Asset
	var checksums *Asset

	for i := range info.Assets {
		if info.Assets[i].Name == expectedName {
			archive = &info.Assets[i]
		}
		if info.Assets[i].Name == "checksums.txt" {
			checksums = &info.Assets[i]
		}
	}

	if archive == nil {
		return nil, nil, &ui.UserError{
			Message:    "No release asset found for this platform.",
			Detail:     fmt.Sprintf("Expected asset %q not found in release %s", expectedName, info.TagName),
			Suggestion: fmt.Sprintf("This release may not support %s/%s. Check GitHub for available assets.", runtime.GOOS, runtime.GOARCH),
		}
	}

	return archive, checksums, nil
}

// DownloadAndReplace downloads the archive asset, optionally verifies its
// SHA256 checksum, extracts the binary, and atomically replaces the running
// executable.
func DownloadAndReplace(ctx context.Context, asset *Asset, checksumsAsset *Asset) error {
	// Download the archive to a temp file.
	archivePath, err := downloadAsset(ctx, asset)
	if err != nil {
		return err
	}
	defer os.Remove(archivePath)

	// Verify checksum if checksums.txt is available.
	if checksumsAsset != nil {
		if err := verifyChecksum(ctx, archivePath, asset.Name, checksumsAsset); err != nil {
			return err
		}
	}

	// Determine the running executable path.
	execPath, err := os.Executable()
	if err != nil {
		return &ui.UserError{
			Message:    "Failed to determine current executable path.",
			Detail:     err.Error(),
			Suggestion: "Ensure the cluckers binary is accessible.",
			Err:        err,
		}
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return &ui.UserError{
			Message:    "Failed to resolve executable symlinks.",
			Detail:     err.Error(),
			Suggestion: "Ensure the cluckers binary path is accessible.",
			Err:        err,
		}
	}

	// Extract the binary from the archive to a temp file in the same directory.
	execDir := filepath.Dir(execPath)
	tmpBin, err := extractBinary(archivePath, execDir)
	if err != nil {
		return err
	}

	// Set executable permissions.
	if err := os.Chmod(tmpBin, 0755); err != nil {
		os.Remove(tmpBin)
		return &ui.UserError{
			Message:    "Failed to set permissions on new binary.",
			Detail:     err.Error(),
			Suggestion: "Check filesystem permissions.",
			Err:        err,
		}
	}

	// Atomic rename: replace the current executable.
	if err := os.Rename(tmpBin, execPath); err != nil {
		os.Remove(tmpBin)
		return &ui.UserError{
			Message:    "Failed to replace the current binary.",
			Detail:     err.Error(),
			Suggestion: "Ensure you have write permission to " + execDir + ".",
			Err:        err,
		}
	}

	return nil
}

// downloadAsset downloads a release asset to a temp file, showing a progress bar.
func downloadAsset(ctx context.Context, asset *Asset) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		return "", &ui.UserError{
			Message:    "Failed to create download request.",
			Detail:     err.Error(),
			Suggestion: "Check your internet connection or try again later.",
			Err:        err,
		}
	}
	req.Header.Set("User-Agent", "cluckers-self-update")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", &ui.UserError{
			Message:    "Failed to download update.",
			Detail:     err.Error(),
			Suggestion: "Check your internet connection or try again later.",
			Err:        err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", &ui.UserError{
			Message:    fmt.Sprintf("Download returned HTTP %d.", resp.StatusCode),
			Detail:     fmt.Sprintf("GET %s returned status %s", asset.BrowserDownloadURL, resp.Status),
			Suggestion: "Try again later or check GitHub for the release.",
		}
	}

	tmpFile, err := os.CreateTemp("", "cluckers-update-*")
	if err != nil {
		return "", &ui.UserError{
			Message:    "Failed to create temporary file for download.",
			Detail:     err.Error(),
			Suggestion: "Check disk space and /tmp permissions.",
			Err:        err,
		}
	}
	defer tmpFile.Close()

	bar := progressbar.NewOptions64(
		asset.Size,
		progressbar.OptionSetDescription("Downloading update"),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetElapsedTime(true),
		progressbar.OptionShowTotalBytes(true),
	)

	writer := io.MultiWriter(tmpFile, bar)
	if _, err := io.Copy(writer, resp.Body); err != nil {
		os.Remove(tmpFile.Name())
		return "", &ui.UserError{
			Message:    "Download interrupted.",
			Detail:     err.Error(),
			Suggestion: "Check your internet connection and try again.",
			Err:        err,
		}
	}

	fmt.Println()

	return tmpFile.Name(), nil
}

// verifyChecksum downloads checksums.txt and verifies the archive SHA256 hash.
func verifyChecksum(ctx context.Context, archivePath, assetName string, checksumsAsset *Asset) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumsAsset.BrowserDownloadURL, nil)
	if err != nil {
		return &ui.UserError{
			Message:    "Failed to create checksum download request.",
			Detail:     err.Error(),
			Suggestion: "Check your internet connection or try again later.",
			Err:        err,
		}
	}
	req.Header.Set("User-Agent", "cluckers-self-update")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return &ui.UserError{
			Message:    "Failed to download checksums.",
			Detail:     err.Error(),
			Suggestion: "Check your internet connection or try again later.",
			Err:        err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Non-fatal: skip checksum verification if checksums.txt fails to download.
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil // Non-fatal: skip verification.
	}

	// Parse checksums.txt -- format: "sha256hash  filename\n"
	expectedHash := ""
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == assetName {
			expectedHash = parts[0]
			break
		}
	}

	if expectedHash == "" {
		// No matching checksum found; skip verification.
		return nil
	}

	// Compute SHA256 of the downloaded archive.
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening archive for checksum verification: %w", err)
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return fmt.Errorf("computing SHA256 hash: %w", err)
	}

	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if actualHash != expectedHash {
		return &ui.UserError{
			Message:    "Checksum verification failed.",
			Detail:     fmt.Sprintf("Expected SHA256 %s, got %s", expectedHash, actualHash),
			Suggestion: "The download may be corrupted. Run `cluckers self-update` to try again.",
		}
	}

	return nil
}

// extractBinary extracts the cluckers binary from a .tar.gz or .zip archive
// into a temp file in destDir. Returns the path to the temp file.
func extractBinary(archivePath, destDir string) (string, error) {
	if strings.HasSuffix(archivePath, ".zip") || runtime.GOOS == "windows" {
		return extractFromZip(archivePath, destDir)
	}
	return extractFromTarGz(archivePath, destDir)
}

// extractFromTarGz extracts the cluckers binary from a .tar.gz archive.
func extractFromTarGz(archivePath, destDir string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", &ui.UserError{
			Message:    "Failed to open downloaded archive.",
			Detail:     err.Error(),
			Suggestion: "Run `cluckers self-update` to try again.",
			Err:        err,
		}
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", &ui.UserError{
			Message:    "Failed to decompress archive.",
			Detail:     err.Error(),
			Suggestion: "The download may be corrupted. Run `cluckers self-update` to try again.",
			Err:        err,
		}
	}
	defer gz.Close()

	tr := tar.NewReader(gz)

	binaryName := "cluckers"
	if runtime.GOOS == "windows" {
		binaryName = "cluckers.exe"
	}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", &ui.UserError{
				Message:    "Failed to read archive contents.",
				Detail:     err.Error(),
				Suggestion: "The download may be corrupted. Run `cluckers self-update` to try again.",
				Err:        err,
			}
		}

		// Skip directory entries and look for the binary.
		if header.Typeflag == tar.TypeDir {
			continue
		}

		name := filepath.Base(header.Name)
		if name != binaryName {
			continue
		}

		// Write to a temp file in the destination directory.
		tmpFile, err := os.CreateTemp(destDir, "cluckers-new-*")
		if err != nil {
			return "", &ui.UserError{
				Message:    "Failed to create temporary file for new binary.",
				Detail:     err.Error(),
				Suggestion: "Check disk space and permissions in " + destDir + ".",
				Err:        err,
			}
		}

		if _, err := io.Copy(tmpFile, tr); err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			return "", &ui.UserError{
				Message:    "Failed to extract binary from archive.",
				Detail:     err.Error(),
				Suggestion: "Run `cluckers self-update` to try again.",
				Err:        err,
			}
		}
		tmpFile.Close()
		return tmpFile.Name(), nil
	}

	return "", &ui.UserError{
		Message:    "Binary not found in archive.",
		Detail:     fmt.Sprintf("Could not find %q in the downloaded archive", binaryName),
		Suggestion: "The release archive may be malformed. Check GitHub for the latest release.",
	}
}

// extractFromZip extracts the cluckers binary from a .zip archive.
func extractFromZip(archivePath, destDir string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", &ui.UserError{
			Message:    "Failed to open downloaded archive.",
			Detail:     err.Error(),
			Suggestion: "Run `cluckers self-update` to try again.",
			Err:        err,
		}
	}
	defer r.Close()

	binaryName := "cluckers"
	if runtime.GOOS == "windows" {
		binaryName = "cluckers.exe"
	}

	for _, f := range r.File {
		name := filepath.Base(f.Name)
		if name != binaryName {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return "", &ui.UserError{
				Message:    "Failed to read archive contents.",
				Detail:     err.Error(),
				Suggestion: "The download may be corrupted. Run `cluckers self-update` to try again.",
				Err:        err,
			}
		}
		defer rc.Close()

		tmpFile, err := os.CreateTemp(destDir, "cluckers-new-*")
		if err != nil {
			return "", &ui.UserError{
				Message:    "Failed to create temporary file for new binary.",
				Detail:     err.Error(),
				Suggestion: "Check disk space and permissions in " + destDir + ".",
				Err:        err,
			}
		}

		if _, err := io.Copy(tmpFile, rc); err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			return "", &ui.UserError{
				Message:    "Failed to extract binary from archive.",
				Detail:     err.Error(),
				Suggestion: "Run `cluckers self-update` to try again.",
				Err:        err,
			}
		}
		tmpFile.Close()
		return tmpFile.Name(), nil
	}

	return "", &ui.UserError{
		Message:    "Binary not found in archive.",
		Detail:     fmt.Sprintf("Could not find %q in the downloaded archive", binaryName),
		Suggestion: "The release archive may be malformed. Check GitHub for the latest release.",
	}
}
