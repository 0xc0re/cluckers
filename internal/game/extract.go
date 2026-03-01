package game

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xc0re/cluckers/internal/ui"
)

// ExtractProgressFunc is called during extraction with the number of files
// extracted so far and the total number of files in the archive.
type ExtractProgressFunc func(extracted, total int)

// ExtractZip extracts a zip archive to destDir with zip-slip protection.
// After successful extraction, the zip file is removed to reclaim disk space.
func ExtractZip(zipPath string, destDir string) error {
	return ExtractZipWithProgress(zipPath, destDir, nil)
}

// ExtractZipWithProgress extracts a zip archive to destDir with zip-slip protection.
// If onProgress is non-nil, it is called with extraction progress instead of
// printing to stdout. The callback is throttled to at most every 100ms.
// If onProgress is nil, terminal output is used (identical to CLI behavior).
// After successful extraction, the zip file is removed to reclaim disk space.
func ExtractZipWithProgress(zipPath string, destDir string, onProgress ExtractProgressFunc) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return &ui.UserError{
			Message:    "Failed to open game archive.",
			Detail:     err.Error(),
			Suggestion: "The download may be corrupted. Run `cluckers update` to re-download.",
			Err:        err,
		}
	}

	totalFiles := len(reader.File)
	var lastProgressCall time.Time

	for i, entry := range reader.File {
		target := filepath.Join(destDir, entry.Name)

		// Zip-slip protection: ensure the target path stays within destDir.
		rel, err := filepath.Rel(destDir, target)
		if err != nil || strings.HasPrefix(rel, "..") {
			ui.Warn(fmt.Sprintf("Skipping suspicious path in archive: %s", entry.Name))
			continue
		}

		// Progress reporting.
		if onProgress != nil {
			now := time.Now()
			if now.Sub(lastProgressCall) >= 100*time.Millisecond || i+1 == totalFiles {
				lastProgressCall = now
				onProgress(i+1, totalFiles)
			}
		} else {
			// Terminal progress indicator every 100 files.
			if (i+1)%100 == 0 || i+1 == totalFiles {
				fmt.Printf("\rExtracting... %d/%d files", i+1, totalFiles)
			}
		}

		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("creating directory %s: %w", target, err)
			}
			continue
		}

		if err := extractFile(entry, target); err != nil {
			return err
		}
	}

	// Newline after progress indicator (CLI mode only).
	if onProgress == nil {
		fmt.Println()
	}

	// Close the zip reader before removing the file. On Windows, os.Remove fails
	// if the file handle is still open ("The process cannot access the file").
	if err := reader.Close(); err != nil {
		ui.Warn(fmt.Sprintf("Could not close archive handle: %s", err))
	}

	// Remove the zip file to reclaim disk space (~5.3 GB).
	if err := os.Remove(zipPath); err != nil {
		// Non-fatal: warn but don't fail extraction.
		ui.Warn(fmt.Sprintf("Could not remove archive after extraction: %s", err))
	}

	return nil
}

// extractFile extracts a single file entry from a zip archive.
func extractFile(entry *zip.File, target string) error {
	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return fmt.Errorf("creating parent directory for %s: %w", target, err)
	}

	// On Windows, files extracted with mode 0444 get the read-only attribute,
	// which prevents subsequent extractions from overwriting them. Clear it first.
	prepareTarget(target)

	src, err := entry.Open()
	if err != nil {
		return fmt.Errorf("opening archive entry %s: %w", entry.Name, err)
	}
	defer src.Close()

	dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, entry.Mode())
	if err != nil {
		return fmt.Errorf("creating file %s: %w", target, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("extracting %s: %w", entry.Name, err)
	}

	return nil
}
