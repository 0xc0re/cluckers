package game

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeVersionJSON serves version.json describing latestVersion with the given
// GameVersion.dat content, and points UpdaterURL at it for the test.
func fakeVersionJSON(t *testing.T, latestVersion, datContent string) *int {
	t.Helper()
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		json.NewEncoder(w).Encode(map[string]interface{}{
			"latest_version":         latestVersion,
			"base_url":               "http://example.invalid/" + latestVersion,
			"manifest_url":           "http://example.invalid/manifest.json",
			"gameversion_dat_path":   GameVersionDatRelPath,
			"gameversion_dat_blake3": blake3Hex(datContent),
			"gameversion_dat_size":   len(datContent),
		})
	}))
	t.Cleanup(srv.Close)
	old := UpdaterURL
	UpdaterURL = srv.URL
	t.Cleanup(func() { UpdaterURL = old })
	return &hits
}

func TestInstalledVersion_Marker(t *testing.T) {
	dir := t.TempDir()
	hits := fakeVersionJSON(t, "9.9.9.9", "x")
	if err := WriteInstalledVersion(dir, "0.39.6969.0"); err != nil {
		t.Fatal(err)
	}
	v, err := InstalledVersion(context.Background(), dir, "1.2.3.4")
	if err != nil || v != "0.39.6969.0" {
		t.Fatalf("InstalledVersion = %q, %v; want marker value", v, err)
	}
	if *hits != 0 {
		t.Error("marker present: must not hit the updater")
	}
}

func TestInstalledVersion_Pinned(t *testing.T) {
	dir := t.TempDir()
	hits := fakeVersionJSON(t, "9.9.9.9", "x")
	v, err := InstalledVersion(context.Background(), dir, " 0.38.1.0 ")
	if err != nil || v != "0.38.1.0" {
		t.Fatalf("InstalledVersion = %q, %v; want pinned value", v, err)
	}
	if *hits != 0 {
		t.Error("pinned: must not hit the updater")
	}
}

func TestInstalledVersion_LatestMatchWritesMarker(t *testing.T) {
	dir := t.TempDir()
	writeDat(t, dir, GameVersionDatRelPath, []byte("dat-bytes"))
	hits := fakeVersionJSON(t, "0.39.6969.0", "dat-bytes")
	v, err := InstalledVersion(context.Background(), dir, "")
	if err != nil || v != "0.39.6969.0" {
		t.Fatalf("InstalledVersion = %q, %v; want latest", v, err)
	}
	if *hits != 1 {
		t.Errorf("updater hits = %d, want 1", *hits)
	}
	if got := readInstalledVersion(dir); got != "0.39.6969.0" {
		t.Errorf("marker after match = %q, want 0.39.6969.0", got)
	}
	// Second call is served from the marker.
	if _, err := InstalledVersion(context.Background(), dir, ""); err != nil || *hits != 1 {
		t.Errorf("second call: err=%v hits=%d, want marker hit and no network", err, *hits)
	}
}

func TestInstalledVersion_Mismatch(t *testing.T) {
	dir := t.TempDir()
	writeDat(t, dir, GameVersionDatRelPath, []byte("old-dat"))
	fakeVersionJSON(t, "0.40.0.0", "new-dat")
	v, err := InstalledVersion(context.Background(), dir, "")
	if err == nil || v != "" {
		t.Fatalf("InstalledVersion = %q, %v; want error on mismatch", v, err)
	}
	if !strings.Contains(err.Error(), "cluckers update") {
		t.Errorf("error should point at cluckers update: %v", err)
	}
	if readInstalledVersion(dir) != "" {
		t.Error("marker must not be written on mismatch")
	}
}

func TestSyncManifest_WritesInstalledVersionAndKeepsIt(t *testing.T) {
	fu := newFakeUpdater(t, map[string]string{
		"Realm-Royale/Binaries/GameVersion.dat": "v",
		"Realm-Royale/a.txt":                    "hello",
	})
	fu.manifest.Version = "0.39.6969.0"
	dir := t.TempDir()
	if err := SyncManifest(context.Background(), fu.info, fu.manifest, dir, func(int64, int64) {}); err != nil {
		t.Fatal(err)
	}
	if got := readInstalledVersion(dir); got != "0.39.6969.0" {
		t.Fatalf("marker after sync = %q", got)
	}
	// A second (no-op) sync must not delete the marker as a stale file.
	if err := SyncManifest(context.Background(), fu.info, fu.manifest, dir, func(int64, int64) {}); err != nil {
		t.Fatal(err)
	}
	if got := readInstalledVersion(dir); got != "0.39.6969.0" {
		t.Errorf("marker after re-sync = %q, want preserved", got)
	}
}
