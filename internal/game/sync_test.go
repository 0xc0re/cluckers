package game

import (
	"context"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/zeebo/blake3"
)

// blake3Hex returns the BLAKE3 hex digest of content.
func blake3Hex(content string) string {
	h := blake3.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// fakeUpdater builds an httptest server serving files[path]=content at
// <base>/<path>, plus a *VersionInfo and *Manifest describing them. requested
// records every path the server was asked for (for skip/redownload assertions).
type fakeUpdater struct {
	server    *httptest.Server
	info      *VersionInfo
	manifest  *Manifest
	mu        sync.Mutex
	requested map[string]int
}

func newFakeUpdater(t *testing.T, files map[string]string) *fakeUpdater {
	t.Helper()
	fu := &fakeUpdater{requested: map[string]int{}}
	fu.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(r.URL.Path, "/")
		fu.mu.Lock()
		fu.requested[rel]++
		fu.mu.Unlock()
		content, ok := files[rel]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(content))
	}))
	t.Cleanup(fu.server.Close)

	m := &Manifest{Schema: 1, Version: "test"}
	for path, content := range files {
		m.Files = append(m.Files, ManifestFile{
			Path: path,
			Hash: blake3Hex(content),
			Size: int64(len(content)),
		})
	}
	fu.manifest = m
	fu.info = &VersionInfo{BaseURL: fu.server.URL}
	return fu
}

func (fu *fakeUpdater) reqCount(path string) int {
	fu.mu.Lock()
	defer fu.mu.Unlock()
	return fu.requested[path]
}

func TestSyncManifest_DownloadsMissing(t *testing.T) {
	fu := newFakeUpdater(t, map[string]string{
		"a/b.txt":  "hello",
		"c.dat":    "world data",
		"deep/x.y": "nested content",
	})
	gameDir := t.TempDir()

	if err := SyncManifest(context.Background(), fu.info, fu.manifest, gameDir, nil); err != nil {
		t.Fatalf("SyncManifest: %v", err)
	}

	for path, want := range map[string]string{"a/b.txt": "hello", "c.dat": "world data", "deep/x.y": "nested content"} {
		got, err := os.ReadFile(filepath.Join(gameDir, filepath.FromSlash(path)))
		if err != nil {
			t.Errorf("reading %s: %v", path, err)
			continue
		}
		if string(got) != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
}

func TestSyncManifest_SkipsMatching(t *testing.T) {
	fu := newFakeUpdater(t, map[string]string{
		"keep.txt": "already here",
		"new.txt":  "needs download",
	})
	gameDir := t.TempDir()
	// Pre-place keep.txt with the correct content.
	if err := os.WriteFile(filepath.Join(gameDir, "keep.txt"), []byte("already here"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := SyncManifest(context.Background(), fu.info, fu.manifest, gameDir, nil); err != nil {
		t.Fatalf("SyncManifest: %v", err)
	}

	if n := fu.reqCount("keep.txt"); n != 0 {
		t.Errorf("keep.txt requested %d times, want 0 (matching file should be skipped)", n)
	}
	if n := fu.reqCount("new.txt"); n != 1 {
		t.Errorf("new.txt requested %d times, want 1", n)
	}
}

func TestSyncManifest_RedownloadsMismatch(t *testing.T) {
	fu := newFakeUpdater(t, map[string]string{"f.txt": "correct content"})
	gameDir := t.TempDir()
	// Pre-place f.txt with wrong content.
	if err := os.WriteFile(filepath.Join(gameDir, "f.txt"), []byte("stale wrong"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := SyncManifest(context.Background(), fu.info, fu.manifest, gameDir, nil); err != nil {
		t.Fatalf("SyncManifest: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(gameDir, "f.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "correct content" {
		t.Errorf("f.txt = %q, want %q (should re-download mismatched file)", got, "correct content")
	}
}

func TestSyncManifest_DeletesStale(t *testing.T) {
	fu := newFakeUpdater(t, map[string]string{"keep.txt": "keep me"})
	gameDir := t.TempDir()
	// Pre-place a stale file not in the manifest, plus a stale nested one.
	if err := os.WriteFile(filepath.Join(gameDir, "stale.txt"), []byte("delete me"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(gameDir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gameDir, "sub", "old.dll"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := SyncManifest(context.Background(), fu.info, fu.manifest, gameDir, nil); err != nil {
		t.Fatalf("SyncManifest: %v", err)
	}

	if _, err := os.Stat(filepath.Join(gameDir, "stale.txt")); !os.IsNotExist(err) {
		t.Errorf("stale.txt should have been deleted, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(gameDir, "sub", "old.dll")); !os.IsNotExist(err) {
		t.Errorf("sub/old.dll should have been deleted, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(gameDir, "keep.txt")); err != nil {
		t.Errorf("keep.txt should still exist: %v", err)
	}
}

func TestSyncManifest_BadHashFails(t *testing.T) {
	// Server serves content that does not match the manifest hash.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("tampered bytes"))
	}))
	defer srv.Close()

	info := &VersionInfo{BaseURL: srv.URL}
	m := &Manifest{Schema: 1, Files: []ManifestFile{
		{Path: "f.txt", Hash: blake3Hex("the real content"), Size: 16},
	}}
	gameDir := t.TempDir()

	if err := SyncManifest(context.Background(), info, m, gameDir, nil); err == nil {
		t.Fatal("expected error on BLAKE3 mismatch, got nil")
	}
	// A failed sync should leave the incomplete marker so the next run re-syncs.
	if !IsSyncIncomplete(gameDir) {
		t.Error("expected sync-incomplete marker to remain after failure")
	}
}

func TestSyncManifest_RejectsPathTraversal(t *testing.T) {
	fu := newFakeUpdater(t, map[string]string{"ok.txt": "fine"})
	// Inject a traversal path into the manifest.
	fu.manifest.Files = append(fu.manifest.Files, ManifestFile{
		Path: "../evil.txt", Hash: blake3Hex("evil"), Size: 4,
	})
	parent := t.TempDir()
	gameDir := filepath.Join(parent, "game")
	if err := os.MkdirAll(gameDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := SyncManifest(context.Background(), fu.info, fu.manifest, gameDir, nil); err == nil {
		t.Fatal("expected error for path traversal in manifest, got nil")
	}
	if _, err := os.Stat(filepath.Join(parent, "evil.txt")); !os.IsNotExist(err) {
		t.Errorf("traversal file escaped gameDir, stat err = %v", err)
	}
}

func TestSyncManifest_ReportsProgress(t *testing.T) {
	fu := newFakeUpdater(t, map[string]string{
		"a.txt": "12345",     // 5 bytes
		"b.txt": "678901234", // 9 bytes
	})
	gameDir := t.TempDir()

	var mu sync.Mutex
	var lastDownloaded, lastTotal int64
	onProgress := func(downloaded, total int64) {
		mu.Lock()
		lastDownloaded, lastTotal = downloaded, total
		mu.Unlock()
	}

	if err := SyncManifest(context.Background(), fu.info, fu.manifest, gameDir, onProgress); err != nil {
		t.Fatalf("SyncManifest: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if lastTotal != 14 {
		t.Errorf("total = %d, want 14 (sum of to-download sizes)", lastTotal)
	}
	if lastDownloaded != 14 {
		t.Errorf("final downloaded = %d, want 14", lastDownloaded)
	}
}

func TestSyncManifest_ContextCancel(t *testing.T) {
	fu := newFakeUpdater(t, map[string]string{"a.txt": "data"})
	gameDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before starting

	if err := SyncManifest(ctx, fu.info, fu.manifest, gameDir, nil); err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}
