package game

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchManifest_ParsesFiles(t *testing.T) {
	const body = `{
		"schema": 1,
		"version": "0.37.6744.0",
		"files": [
			{"path": "Realm-Royale/Binaries/GameVersion.dat", "hash": "abc123", "size": 19},
			{"path": "Realm-Royale/Binaries/Win64/X.dll", "hash": "def456", "size": 1024}
		]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	info := &VersionInfo{ManifestURL: srv.URL}
	m, err := FetchManifest(context.Background(), info)
	if err != nil {
		t.Fatalf("FetchManifest: %v", err)
	}
	if m.Schema != 1 {
		t.Errorf("Schema = %d, want 1", m.Schema)
	}
	if m.Version != "0.37.6744.0" {
		t.Errorf("Version = %q, want 0.37.6744.0", m.Version)
	}
	if len(m.Files) != 2 {
		t.Fatalf("len(Files) = %d, want 2", len(m.Files))
	}
	if m.Files[1].Path != "Realm-Royale/Binaries/Win64/X.dll" {
		t.Errorf("Files[1].Path = %q", m.Files[1].Path)
	}
	if m.Files[1].Hash != "def456" || m.Files[1].Size != 1024 {
		t.Errorf("Files[1] = %+v", m.Files[1])
	}
}

func TestFetchManifest_RejectsWrongSchema(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"schema": 2, "version": "x", "files": []}`))
	}))
	defer srv.Close()

	info := &VersionInfo{ManifestURL: srv.URL}
	if _, err := FetchManifest(context.Background(), info); err == nil {
		t.Fatal("expected error for unsupported schema, got nil")
	}
}

func TestFetchManifest_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	info := &VersionInfo{ManifestURL: srv.URL}
	if _, err := FetchManifest(context.Background(), info); err == nil {
		t.Fatal("expected error for HTTP 404, got nil")
	}
}
