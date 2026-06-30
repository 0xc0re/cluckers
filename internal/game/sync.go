package game

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/schollz/progressbar/v3"
	"github.com/zeebo/blake3"
)

// syncMarker is written to the game directory while a sync is in progress.
// Its presence indicates a sync was interrupted and the game files may be
// inconsistent, forcing a re-sync on the next check.
const syncMarker = ".cluckers-syncing"

// syncWorkers is the number of files downloaded concurrently.
const syncWorkers = 8

// ProgressFunc is called during download with cumulative bytes downloaded and
// the total expected bytes to download.
type ProgressFunc func(downloaded, total int64)

// IsSyncIncomplete reports whether a previous sync was interrupted, leaving
// the game directory in a potentially inconsistent state.
func IsSyncIncomplete(gameDir string) bool {
	_, err := os.Stat(filepath.Join(gameDir, syncMarker))
	return err == nil
}

// syncJob is a single file that must be downloaded.
type syncJob struct {
	file ManifestFile
	dest string // absolute destination path
}

// SyncManifest brings gameDir into exact agreement with the manifest: it
// downloads missing or changed files (verifying each against its BLAKE3 hash),
// then deletes any local files not present in the manifest (clean sync).
//
// If onProgress is non-nil it receives cumulative downloaded bytes and the
// total bytes to download; otherwise a terminal progress bar is shown.
func SyncManifest(ctx context.Context, info *VersionInfo, m *Manifest, gameDir string, onProgress ProgressFunc) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(gameDir, 0755); err != nil {
		return fmt.Errorf("creating game directory: %w", err)
	}

	// Diff the manifest against local files. Also build the set of expected
	// paths for the clean-sync deletion pass, and validate every path stays
	// within gameDir (manifest path-traversal guard).
	want := make(map[string]struct{}, len(m.Files))
	var jobs []syncJob
	var totalBytes int64
	for _, f := range m.Files {
		dest, err := safeJoin(gameDir, f.Path)
		if err != nil {
			return &ui.UserError{
				Message:    "Game manifest contained an unsafe file path.",
				Detail:     fmt.Sprintf("rejecting path %q", f.Path),
				Suggestion: "This may indicate a corrupted or tampered manifest. Try again later.",
			}
		}
		want[filepath.Clean(dest)] = struct{}{}
		if fileNeedsDownload(dest, f) {
			jobs = append(jobs, syncJob{file: f, dest: dest})
			totalBytes += f.Size
		}
	}

	if err := checkDiskSpace(gameDir, totalBytes); err != nil {
		return err
	}

	// Mark the sync as in progress; removed only on full success.
	markerPath := filepath.Join(gameDir, syncMarker)
	if err := os.WriteFile(markerPath, []byte("syncing"), 0644); err != nil {
		return fmt.Errorf("writing sync marker: %w", err)
	}

	if err := downloadJobs(ctx, info.BaseURL, jobs, totalBytes, onProgress); err != nil {
		return err // leave marker in place so the next run re-syncs
	}

	if err := removeStale(gameDir, want, markerPath); err != nil {
		return err
	}

	if err := os.Remove(markerPath); err != nil {
		ui.Warn(fmt.Sprintf("Could not remove sync marker: %s", err))
	}
	return nil
}

// safeJoin joins a forward-slash relative path onto base, rejecting any result
// that would escape base.
func safeJoin(base, rel string) (string, error) {
	dest := filepath.Join(base, filepath.FromSlash(rel))
	r, err := filepath.Rel(base, dest)
	if err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes base directory", rel)
	}
	return dest, nil
}

// fileNeedsDownload reports whether the local file at dest is missing or does
// not match the manifest entry's size and BLAKE3 hash.
func fileNeedsDownload(dest string, f ManifestFile) bool {
	info, err := os.Stat(dest)
	if err != nil {
		return true // missing or unreadable — (re)download
	}
	if info.Size() != f.Size {
		return true
	}
	local, err := blake3File(dest)
	if err != nil {
		return true
	}
	return local != f.Hash
}

// blake3File returns the BLAKE3 hex digest of the file at path.
func blake3File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := blake3.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// gameFileClient downloads game files. No client-level timeout: individual
// files can be large and slow; cancellation is handled via the request context.
var gameFileClient = &http.Client{}

// downloadJobs downloads all jobs using a bounded worker pool, reporting
// aggregated progress. It returns the first error encountered.
func downloadJobs(ctx context.Context, baseURL string, jobs []syncJob, totalBytes int64, onProgress ProgressFunc) error {
	rep := newProgressReporter(onProgress, totalBytes)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobCh := make(chan syncJob)
	var wg sync.WaitGroup
	var errOnce sync.Once
	var firstErr error
	setErr := func(e error) {
		errOnce.Do(func() {
			firstErr = e
			cancel()
		})
	}

	for i := 0; i < syncWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobCh {
				if ctx.Err() != nil {
					continue // drain remaining jobs so the feeder never blocks
				}
				if err := downloadOne(ctx, baseURL, j, rep); err != nil {
					setErr(err)
				}
			}
		}()
	}

feed:
	for _, j := range jobs {
		select {
		case <-ctx.Done():
			break feed
		case jobCh <- j:
		}
	}
	close(jobCh)
	wg.Wait()

	if firstErr != nil {
		return firstErr
	}
	rep.finish()
	return nil
}

// downloadOne fetches a single file to a temp file in the destination
// directory, verifies its BLAKE3 hash, and atomically renames it into place.
func downloadOne(ctx context.Context, baseURL string, j syncJob, rep *progressReporter) error {
	url := strings.TrimRight(baseURL, "/") + "/" + j.file.Path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return &ui.UserError{
			Message:    "Failed to create download request.",
			Detail:     err.Error(),
			Suggestion: "Check your internet connection or try again later.",
			Err:        err,
		}
	}

	resp, err := gameFileClient.Do(req)
	if err != nil {
		return &ui.UserError{
			Message:    "Failed to download game files.",
			Detail:     fmt.Sprintf("%s: %s", j.file.Path, err),
			Suggestion: "Check your internet connection and run `cluckers update` again.",
			Err:        err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &ui.UserError{
			Message:    fmt.Sprintf("Download server returned HTTP %d.", resp.StatusCode),
			Detail:     fmt.Sprintf("GET %s returned status %s", url, resp.Status),
			Suggestion: "Try again later or check the Cluckers Discord for server status.",
		}
	}

	if err := os.MkdirAll(filepath.Dir(j.dest), 0755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", j.file.Path, err)
	}
	prepareTarget(j.dest) // clear read-only attribute if the file already exists

	tmp, err := os.CreateTemp(filepath.Dir(j.dest), ".dl-*")
	if err != nil {
		return fmt.Errorf("creating temp file for %s: %w", j.file.Path, err)
	}
	tmpPath := tmp.Name()
	// Best-effort cleanup if we don't rename it into place.
	defer func() { _ = os.Remove(tmpPath) }()

	hasher := blake3.New()
	w := io.MultiWriter(tmp, hasher, progressWriter{rep: rep})
	if _, err := io.Copy(w, resp.Body); err != nil {
		tmp.Close()
		return &ui.UserError{
			Message:    "Download interrupted.",
			Detail:     fmt.Sprintf("%s: %s", j.file.Path, err),
			Suggestion: "Run `cluckers update` again to resume.",
			Err:        err,
		}
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("finalizing %s: %w", j.file.Path, err)
	}

	got := hex.EncodeToString(hasher.Sum(nil))
	if got != j.file.Hash {
		return &ui.UserError{
			Message:    "Downloaded file failed integrity check.",
			Detail:     fmt.Sprintf("%s: BLAKE3 expected %s, got %s", j.file.Path, j.file.Hash, got),
			Suggestion: "The download may be corrupted. Run `cluckers update` to retry.",
		}
	}

	if err := os.Rename(tmpPath, j.dest); err != nil {
		return fmt.Errorf("installing %s: %w", j.file.Path, err)
	}
	return nil
}

// removeStale deletes any regular file under gameDir whose path is not in the
// manifest's expected set. The sync marker is preserved (removed separately).
func removeStale(gameDir string, want map[string]struct{}, markerPath string) error {
	return filepath.Walk(gameDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}
		if path == markerPath {
			return nil
		}
		if _, ok := want[filepath.Clean(path)]; ok {
			return nil
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("removing stale file %s: %w", path, err)
		}
		return nil
	})
}

// progressReporter aggregates download progress across workers. With a callback
// it throttles to at most every 250ms; otherwise it drives a terminal bar.
type progressReporter struct {
	onProgress ProgressFunc
	bar        *progressbar.ProgressBar
	total      int64
	downloaded int64
	mu         sync.Mutex
	lastCall   time.Time
}

func newProgressReporter(onProgress ProgressFunc, total int64) *progressReporter {
	r := &progressReporter{onProgress: onProgress, total: total}
	if onProgress == nil {
		r.bar = progressbar.NewOptions64(
			total,
			progressbar.OptionSetDescription("Downloading game files"),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetPredictTime(true),
			progressbar.OptionFullWidth(),
			progressbar.OptionSetElapsedTime(true),
			progressbar.OptionShowTotalBytes(true),
		)
	}
	return r
}

func (r *progressReporter) add(n int64) {
	d := atomic.AddInt64(&r.downloaded, n)
	if r.bar != nil {
		_ = r.bar.Add64(n)
		return
	}
	if r.onProgress == nil {
		return
	}
	r.mu.Lock()
	now := time.Now()
	if now.Sub(r.lastCall) >= 250*time.Millisecond {
		r.lastCall = now
		r.mu.Unlock()
		r.onProgress(d, r.total)
	} else {
		r.mu.Unlock()
	}
}

func (r *progressReporter) finish() {
	if r.bar != nil {
		_ = r.bar.Finish()
		fmt.Println()
		return
	}
	if r.onProgress != nil {
		r.onProgress(atomic.LoadInt64(&r.downloaded), r.total)
	}
}

// progressWriter forwards byte counts to a progressReporter.
type progressWriter struct {
	rep *progressReporter
}

func (w progressWriter) Write(p []byte) (int, error) {
	w.rep.add(int64(len(p)))
	return len(p), nil
}
