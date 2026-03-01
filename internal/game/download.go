package game

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/schollz/progressbar/v3"
	"github.com/zeebo/blake3"
)

// ProgressFunc is called during download with the number of bytes downloaded
// so far and the total expected bytes.
type ProgressFunc func(downloaded, total int64)

// DownloadGameZip downloads the game zip file with resume support and a terminal progress bar.
// It uses HTTP Range headers to resume interrupted downloads.
func DownloadGameZip(ctx context.Context, info *VersionInfo, destDir string) error {
	return DownloadGameZipWithProgress(ctx, info, destDir, nil)
}

// progressCallbackWriter wraps a ProgressFunc as an io.Writer, tracking cumulative
// bytes and throttling callbacks to at most every 250ms to avoid overwhelming the GUI.
type progressCallbackWriter struct {
	onProgress ProgressFunc
	total      int64
	written    int64
	mu         sync.Mutex
	lastCall   time.Time
}

func (w *progressCallbackWriter) Write(p []byte) (int, error) {
	n := len(p)
	w.mu.Lock()
	w.written += int64(n)
	now := time.Now()
	if now.Sub(w.lastCall) >= 250*time.Millisecond {
		w.lastCall = now
		downloaded := w.written
		w.mu.Unlock()
		w.onProgress(downloaded, w.total)
	} else {
		w.mu.Unlock()
	}
	return n, nil
}

// DownloadGameZipWithProgress downloads the game zip file with resume support.
// If onProgress is non-nil, it is called with download progress instead of
// printing a terminal progress bar. If onProgress is nil, the terminal progress
// bar is used (identical to CLI behavior).
func DownloadGameZipWithProgress(ctx context.Context, info *VersionInfo, destDir string, onProgress ProgressFunc) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("creating download directory: %w", err)
	}

	// Check available disk space (need ~2x zip size for zip + extraction).
	requiredBytes := info.ZipSize * 2
	if err := checkDiskSpace(destDir, requiredBytes); err != nil {
		return err
	}

	finalPath := filepath.Join(destDir, "game.zip")
	partialPath := finalPath + ".partial"

	// Check for existing partial download.
	var offset int64
	if stat, err := os.Stat(partialPath); err == nil {
		offset = stat.Size()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.ZipURL, nil)
	if err != nil {
		return &ui.UserError{
			Message:    "Failed to create download request.",
			Detail:     err.Error(),
			Suggestion: "Check your internet connection or try again later.",
			Err:        err,
		}
	}

	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}

	// No timeout on the client -- the download is ~5.3 GB and can take a long time.
	// The context handles cancellation.
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return &ui.UserError{
			Message:    "Failed to download game files.",
			Detail:     err.Error(),
			Suggestion: "Check your internet connection or try again later.",
			Err:        err,
		}
	}
	defer resp.Body.Close()

	var file *os.File
	switch resp.StatusCode {
	case http.StatusPartialContent:
		// Server supports Range -- resume from offset.
		file, err = os.OpenFile(partialPath, os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("opening partial download for append: %w", err)
		}
	case http.StatusOK:
		// Server did not support Range or this is a fresh download.
		offset = 0
		file, err = os.OpenFile(partialPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("creating download file: %w", err)
		}
	default:
		return &ui.UserError{
			Message:    fmt.Sprintf("Download server returned HTTP %d.", resp.StatusCode),
			Detail:     fmt.Sprintf("GET %s returned status %s", info.ZipURL, resp.Status),
			Suggestion: "Try again later or check the Cluckers Discord for server status.",
		}
	}
	defer file.Close()

	var writer io.Writer
	if onProgress != nil {
		// GUI mode: use callback writer instead of terminal progress bar.
		cbWriter := &progressCallbackWriter{
			onProgress: onProgress,
			total:      info.ZipSize,
			written:    offset,
		}
		// Send initial progress for resumed downloads.
		if offset > 0 {
			onProgress(offset, info.ZipSize)
		}
		writer = io.MultiWriter(file, cbWriter)
	} else {
		// CLI mode: use terminal progress bar.
		bar := progressbar.NewOptions64(
			info.ZipSize,
			progressbar.OptionSetDescription("Downloading game files"),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetPredictTime(true),
			progressbar.OptionFullWidth(),
			progressbar.OptionSetElapsedTime(true),
			progressbar.OptionShowTotalBytes(true),
		)

		// Set initial position for resume.
		if offset > 0 {
			_ = bar.Set64(offset)
		}

		writer = io.MultiWriter(file, bar)
	}

	if _, err := io.Copy(writer, resp.Body); err != nil {
		// Leave .partial file in place for future resume.
		return &ui.UserError{
			Message:    "Download interrupted.",
			Detail:     err.Error(),
			Suggestion: "Run the update command again to resume the download.",
			Err:        err,
		}
	}

	// Newline after progress bar (CLI mode only).
	if onProgress == nil {
		fmt.Println()
	}

	// Close file before rename so the handle is released.
	file.Close()

	// Move partial to final path.
	if err := os.Rename(partialPath, finalPath); err != nil {
		return fmt.Errorf("finalizing download: %w", err)
	}

	return nil
}

// VerifyBLAKE3 verifies that the file at filePath matches the expected BLAKE3 hash.
func VerifyBLAKE3(filePath string, expectedHash string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening file for verification: %w", err)
	}
	defer f.Close()

	hasher := blake3.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return fmt.Errorf("computing BLAKE3 hash: %w", err)
	}

	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if actualHash != expectedHash {
		return fmt.Errorf("BLAKE3 mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	return nil
}

// DownloadAndVerify downloads the game zip and verifies its BLAKE3 hash.
// If verification fails, the corrupt file is deleted.
func DownloadAndVerify(ctx context.Context, info *VersionInfo, destDir string) error {
	return DownloadAndVerifyWithProgress(ctx, info, destDir, nil)
}

// DownloadAndVerifyWithProgress downloads the game zip and verifies its BLAKE3 hash.
// If onProgress is non-nil, it is passed to DownloadGameZipWithProgress and the
// terminal "Verifying download integrity..." message is skipped (the GUI handles
// its own status text). If onProgress is nil, CLI behavior is identical to DownloadAndVerify.
func DownloadAndVerifyWithProgress(ctx context.Context, info *VersionInfo, destDir string, onProgress ProgressFunc) error {
	if err := DownloadGameZipWithProgress(ctx, info, destDir, onProgress); err != nil {
		return err
	}

	zipPath := filepath.Join(destDir, "game.zip")

	if onProgress == nil {
		ui.Info("Verifying download integrity...")
	}

	if err := VerifyBLAKE3(zipPath, info.ZipBLAKE3); err != nil {
		// Delete corrupt download so it doesn't get reused.
		_ = os.Remove(zipPath)
		return &ui.UserError{
			Message:    "Download verification failed.",
			Detail:     err.Error(),
			Suggestion: "Download was corrupted. Run `cluckers update` to re-download.",
			Err:        err,
		}
	}

	return nil
}

