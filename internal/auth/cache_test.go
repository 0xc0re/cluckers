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
				AccessCachedAt: time.Now().Add(-1 * time.Hour),
			},
			expected: true,
		},
		{
			name: "expired token with old timestamp",
			cache: TokenCache{
				AccessToken:    "tok123",
				AccessCachedAt: time.Now().Add(-25 * time.Hour),
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

func TestOIDCTokenValid(t *testing.T) {
	tests := []struct {
		name     string
		cache    TokenCache
		expected bool
	}{
		{
			name: "valid token with recent timestamp",
			cache: TokenCache{
				OIDCToken:    "oidc123",
				OIDCCachedAt: time.Now().Add(-30 * time.Minute),
			},
			expected: true,
		},
		{
			name: "expired token with old timestamp",
			cache: TokenCache{
				OIDCToken:    "oidc123",
				OIDCCachedAt: time.Now().Add(-56 * time.Minute),
			},
			expected: false,
		},
		{
			name: "empty token regardless of timestamp",
			cache: TokenCache{
				OIDCToken:    "",
				OIDCCachedAt: time.Now(),
			},
			expected: false,
		},
		{
			name: "zero timestamp",
			cache: TokenCache{
				OIDCToken: "oidc123",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cache.OIDCTokenValid()
			if got != tt.expected {
				t.Errorf("OIDCTokenValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTokenTTLIndependence(t *testing.T) {
	t.Run("access valid but OIDC expired", func(t *testing.T) {
		cache := TokenCache{
			AccessToken:    "access123",
			OIDCToken:      "oidc123",
			AccessCachedAt: time.Now().Add(-1 * time.Hour), // within 24h TTL
			OIDCCachedAt:   time.Now().Add(-56 * time.Minute), // beyond 55min TTL
		}

		if !cache.AccessTokenValid() {
			t.Error("AccessTokenValid() should be true (within 24h TTL)")
		}
		if cache.OIDCTokenValid() {
			t.Error("OIDCTokenValid() should be false (beyond 55min TTL)")
		}
	})

	t.Run("OIDC valid but access expired", func(t *testing.T) {
		cache := TokenCache{
			AccessToken:    "access123",
			OIDCToken:      "oidc123",
			AccessCachedAt: time.Now().Add(-25 * time.Hour), // beyond 24h TTL
			OIDCCachedAt:   time.Now().Add(-30 * time.Minute), // within 55min TTL
		}

		if cache.AccessTokenValid() {
			t.Error("AccessTokenValid() should be false (beyond 24h TTL)")
		}
		if !cache.OIDCTokenValid() {
			t.Error("OIDCTokenValid() should be true (within 55min TTL)")
		}
	})

	t.Run("access valid with zero OIDC timestamp", func(t *testing.T) {
		cache := TokenCache{
			AccessToken:    "access123",
			OIDCToken:      "oidc123",
			AccessCachedAt: time.Now(),
			// OIDCCachedAt is zero
		}

		if !cache.AccessTokenValid() {
			t.Error("AccessTokenValid() should be true")
		}
		if cache.OIDCTokenValid() {
			t.Error("OIDCTokenValid() should be false (zero timestamp)")
		}
	})
}

func TestSaveAndLoadTokenCache(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLUCKERS_HOME", tmp)

	now := time.Now().Truncate(time.Second) // JSON truncates to seconds

	original := &TokenCache{
		AccessToken:    "access-tok",
		OIDCToken:      "oidc-tok",
		Username:       "testuser",
		AccessCachedAt: now.Add(-5 * time.Minute),
		OIDCCachedAt:   now.Add(-2 * time.Minute),
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
	if loaded.OIDCToken != original.OIDCToken {
		t.Errorf("OIDCToken = %q, want %q", loaded.OIDCToken, original.OIDCToken)
	}
	if loaded.Username != original.Username {
		t.Errorf("Username = %q, want %q", loaded.Username, original.Username)
	}

	// Verify timestamps round-trip (truncated to second precision for JSON).
	if !loaded.AccessCachedAt.Equal(original.AccessCachedAt) {
		t.Errorf("AccessCachedAt = %v, want %v", loaded.AccessCachedAt, original.AccessCachedAt)
	}
	if !loaded.OIDCCachedAt.Equal(original.OIDCCachedAt) {
		t.Errorf("OIDCCachedAt = %v, want %v", loaded.OIDCCachedAt, original.OIDCCachedAt)
	}

	// Verify tokens are valid.
	if !loaded.AccessTokenValid() {
		t.Error("loaded AccessTokenValid() should be true")
	}
	if !loaded.OIDCTokenValid() {
		t.Error("loaded OIDCTokenValid() should be true")
	}
}

func TestLoadTokenCache_OldFormat(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLUCKERS_HOME", tmp)

	// Write a cache file in the old format (single cached_at field).
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

	// Tokens should be present.
	if loaded.AccessToken != "old-access" {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, "old-access")
	}
	if loaded.OIDCToken != "old-oidc" {
		t.Errorf("OIDCToken = %q, want %q", loaded.OIDCToken, "old-oidc")
	}

	// But both Valid methods should return false (zero-value new timestamps = expired).
	if loaded.AccessTokenValid() {
		t.Error("AccessTokenValid() should be false for old-format cache (zero AccessCachedAt)")
	}
	if loaded.OIDCTokenValid() {
		t.Error("OIDCTokenValid() should be false for old-format cache (zero OIDCCachedAt)")
	}
}
