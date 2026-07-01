package game

import (
	"context"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/zeebo/blake3"
)

func TestPinVersionInfo_RewritesURLs(t *testing.T) {
	latest := &VersionInfo{
		LatestVersion:        "0.37.6744.0",
		BaseURL:              "https://updater.realmhub.io/builds/0.37.6744.0",
		ManifestURL:          "https://updater.realmhub.io/builds/manifest-v0.37.6744.0.json",
		GameVersionDatPath:   "Realm-Royale/Binaries/GameVersion.dat",
		GameVersionDatBLAKE3: "deadbeef",
		GameVersionDatSize:   19,
	}

	pinned := PinVersionInfo(latest, "0.37.6742.0")

	if pinned.LatestVersion != "0.37.6742.0" {
		t.Errorf("LatestVersion = %q, want 0.37.6742.0", pinned.LatestVersion)
	}
	if pinned.BaseURL != "https://updater.realmhub.io/builds/0.37.6742.0" {
		t.Errorf("BaseURL = %q", pinned.BaseURL)
	}
	if pinned.ManifestURL != "https://updater.realmhub.io/builds/manifest-v0.37.6742.0.json" {
		t.Errorf("ManifestURL = %q", pinned.ManifestURL)
	}
	// The pinned build's dat hash/size are unknown; must be cleared so callers
	// fall back to the manifest for the update decision.
	if pinned.GameVersionDatBLAKE3 != "" || pinned.GameVersionDatSize != 0 {
		t.Errorf("dat hash/size not cleared: %q %d", pinned.GameVersionDatBLAKE3, pinned.GameVersionDatSize)
	}
	if pinned.GameVersionDatPath != latest.GameVersionDatPath {
		t.Errorf("GameVersionDatPath = %q, want unchanged", pinned.GameVersionDatPath)
	}
	// Original must not be mutated.
	if latest.LatestVersion != "0.37.6744.0" || latest.GameVersionDatBLAKE3 != "deadbeef" {
		t.Errorf("PinVersionInfo mutated the original: %+v", latest)
	}
}

func TestPinVersionInfo_SameOrEmptyVersionReturnsLatest(t *testing.T) {
	latest := &VersionInfo{
		LatestVersion:        "0.37.6744.0",
		BaseURL:              "https://x/builds/0.37.6744.0",
		ManifestURL:          "https://x/builds/manifest-v0.37.6744.0.json",
		GameVersionDatBLAKE3: "abc",
	}
	for _, v := range []string{"", "0.37.6744.0"} {
		got := PinVersionInfo(latest, v)
		if got.LatestVersion != "0.37.6744.0" || got.GameVersionDatBLAKE3 != "abc" {
			t.Errorf("PinVersionInfo(latest, %q) = %+v, want latest unchanged", v, got)
		}
	}
}

func writeDat(t *testing.T, gameDir, rel string, content []byte) string {
	t.Helper()
	p := filepath.Join(gameDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, content, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func manifestFor(content []byte) *Manifest {
	h := blake3.Sum256(content)
	return &Manifest{
		Schema:  1,
		Version: "0.37.6742.0",
		Files: []ManifestFile{
			{Path: GameVersionDatRelPath, Hash: hex.EncodeToString(h[:]), Size: int64(len(content))},
		},
	}
}

func TestNeedsUpdateFromManifest_Match(t *testing.T) {
	dir := t.TempDir()
	content := []byte("gameversion-6742\x00\x00")
	writeDat(t, dir, GameVersionDatRelPath, content)

	needs, err := NeedsUpdateFromManifest(dir, manifestFor(content))
	if err != nil {
		t.Fatalf("NeedsUpdateFromManifest: %v", err)
	}
	if needs {
		t.Error("needs = true, want false for matching file")
	}
}

func TestNeedsUpdateFromManifest_Mismatch(t *testing.T) {
	dir := t.TempDir()
	writeDat(t, dir, GameVersionDatRelPath, []byte("old-content"))

	needs, err := NeedsUpdateFromManifest(dir, manifestFor([]byte("new-content-different")))
	if err != nil {
		t.Fatalf("NeedsUpdateFromManifest: %v", err)
	}
	if !needs {
		t.Error("needs = false, want true for mismatched hash/size")
	}
}

func TestNeedsUpdateFromManifest_MissingFile(t *testing.T) {
	dir := t.TempDir()
	needs, err := NeedsUpdateFromManifest(dir, manifestFor([]byte("whatever")))
	if err != nil {
		t.Fatalf("NeedsUpdateFromManifest: %v", err)
	}
	if !needs {
		t.Error("needs = false, want true when local file is absent")
	}
}

func TestNeedsUpdateFromManifest_NoDatEntry(t *testing.T) {
	dir := t.TempDir()
	m := &Manifest{Schema: 1, Files: []ManifestFile{{Path: "Realm-Royale/Binaries/Win64/X.dll", Hash: "x", Size: 1}}}
	if _, err := NeedsUpdateFromManifest(dir, m); err == nil {
		t.Fatal("expected error when manifest has no GameVersion.dat entry, got nil")
	}
}

func TestResolveNeedsUpdate_LatestUsesHashNoManifest(t *testing.T) {
	dir := t.TempDir()
	content := []byte("latest-dat")
	writeDat(t, dir, GameVersionDatRelPath, content)
	h := blake3.Sum256(content)

	info := &VersionInfo{
		GameVersionDatPath:   GameVersionDatRelPath,
		GameVersionDatBLAKE3: hex.EncodeToString(h[:]),
		GameVersionDatSize:   int64(len(content)),
	}
	needs, m, err := ResolveNeedsUpdate(context.Background(), dir, info)
	if err != nil {
		t.Fatalf("ResolveNeedsUpdate: %v", err)
	}
	if needs {
		t.Error("needs = true, want false")
	}
	if m != nil {
		t.Error("manifest should be nil on the latest path (no fetch)")
	}
}

func TestResolveNeedsUpdate_PinnedFetchesManifest(t *testing.T) {
	dir := t.TempDir()
	content := []byte("pinned-dat-6742")
	writeDat(t, dir, GameVersionDatRelPath, content)
	h := blake3.Sum256(content)

	body := `{"files":[{"path":"` + GameVersionDatRelPath + `","hash":"` + hex.EncodeToString(h[:]) + `","size":` + itoa(len(content)) + `}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	// Pinned info: no dat hash, manifest URL points at the legacy schema-less manifest.
	info := &VersionInfo{ManifestURL: srv.URL}
	needs, m, err := ResolveNeedsUpdate(context.Background(), dir, info)
	if err != nil {
		t.Fatalf("ResolveNeedsUpdate: %v", err)
	}
	if needs {
		t.Error("needs = true, want false (local matches pinned manifest)")
	}
	if m == nil {
		t.Error("manifest should be returned on the pinned path for reuse")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func TestFetchManifest_AcceptsLegacySchemaless(t *testing.T) {
	body := `{"files":[{"path":"Realm-Royale/Binaries/GameVersion.dat","hash":"abc","size":19}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	m, err := FetchManifest(context.Background(), &VersionInfo{ManifestURL: srv.URL})
	if err != nil {
		t.Fatalf("FetchManifest rejected legacy schema-less manifest: %v", err)
	}
	if len(m.Files) != 1 {
		t.Fatalf("len(Files) = %d, want 1", len(m.Files))
	}
}

func TestFetchManifest_RejectsEmptyFiles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"files":[]}`))
	}))
	defer srv.Close()

	if _, err := FetchManifest(context.Background(), &VersionInfo{ManifestURL: srv.URL}); err == nil {
		t.Fatal("expected error for empty file list, got nil")
	}
}
