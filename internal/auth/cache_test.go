package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAccessTokenValid(t *testing.T) {
	tests := []struct {
		name     string
		cache    TokenCache
		expected bool
	}{
		{
			name: "valid token with recent timestamp",
			cache: TokenCache{
				AccessToken:    "tok123",
				AccessCachedAt: time.Now().Add(-10 * time.Minute),
			},
			expected: true,
		},
		{
			name: "expired token with old timestamp",
			cache: TokenCache{
				AccessToken:    "tok123",
				AccessCachedAt: time.Now().Add(-1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "empty token regardless of timestamp",
			cache: TokenCache{
				AccessToken:    "",
				AccessCachedAt: time.Now(),
			},
			expected: false,
		},
		{
			name: "zero timestamp",
			cache: TokenCache{
				AccessToken: "tok123",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cache.AccessTokenValid()
			if got != tt.expected {
				t.Errorf("AccessTokenValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSaveAndLoadTokenCache(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLUCKERS_HOME", tmp)

	now := time.Now().Truncate(time.Second) // JSON truncates to seconds

	original := &TokenCache{
		AccessToken:    "access-tok",
		Username:       "testuser",
		AccessCachedAt: now.Add(-5 * time.Minute),
	}

	// Save.
	if err := SaveTokenCache(original); err != nil {
		t.Fatalf("SaveTokenCache() error: %v", err)
	}

	// Load.
	loaded, err := LoadTokenCache()
	if err != nil {
		t.Fatalf("LoadTokenCache() error: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadTokenCache() returned nil")
	}

	// Verify fields.
	if loaded.AccessToken != original.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, original.AccessToken)
	}
	if loaded.Username != original.Username {
		t.Errorf("Username = %q, want %q", loaded.Username, original.Username)
	}

	// Verify timestamp round-trips (truncated to second precision for JSON).
	if !loaded.AccessCachedAt.Equal(original.AccessCachedAt) {
		t.Errorf("AccessCachedAt = %v, want %v", loaded.AccessCachedAt, original.AccessCachedAt)
	}

	// Verify token is valid.
	if !loaded.AccessTokenValid() {
		t.Error("loaded AccessTokenValid() should be true")
	}
}

func TestLoadTokenCache_OldFormat(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLUCKERS_HOME", tmp)

	// Write a cache file in the old format (with an oidc_token field that the
	// new struct ignores, and no access_cached_at).
	oldCache := map[string]interface{}{
		"access_token": "old-access",
		"oidc_token":   "old-oidc",
		"username":     "olduser",
		"cached_at":    time.Now().Add(-10 * time.Minute).Format(time.RFC3339Nano),
	}

	data, err := json.MarshalIndent(oldCache, "", "  ")
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	cacheDir := filepath.Join(tmp, "cache")
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "tokens.json"), data, 0600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	// Load the old-format cache.
	loaded, err := LoadTokenCache()
	if err != nil {
		t.Fatalf("LoadTokenCache() error: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadTokenCache() returned nil")
	}

	// The access token should be present.
	if loaded.AccessToken != "old-access" {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, "old-access")
	}

	// But Valid should return false (zero-value AccessCachedAt = expired), so the
	// user is transparently re-authenticated.
	if loaded.AccessTokenValid() {
		t.Error("AccessTokenValid() should be false for old-format cache (zero AccessCachedAt)")
	}
}
