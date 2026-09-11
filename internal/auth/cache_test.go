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

func TestAccessTokenValid_ServerExpiry(t *testing.T) {
	future := TokenCache{AccessToken: "tok", AccessCachedAt: time.Now().Add(-3 * time.Hour), AccessExpiresAt: time.Now().Add(10 * time.Minute)}
	if !future.AccessTokenValid() {
		t.Error("token with a future server expiry should be valid even if cached long ago")
	}
	if rem := future.AccessRemaining(); rem <= 8*time.Minute || rem > 10*time.Minute {
		t.Errorf("AccessRemaining = %v, want ~9m (expiry minus skew)", rem)
	}
	soon := TokenCache{AccessToken: "tok", AccessCachedAt: time.Now(), AccessExpiresAt: time.Now().Add(30 * time.Second)}
	if soon.AccessTokenValid() {
		t.Error("token expiring within the skew window should be invalid")
	}
	if soon.AccessRemaining() != 0 {
		t.Error("AccessRemaining should be 0 for an invalid token")
	}
}

func TestRefreshTokenValid(t *testing.T) {
	if (&TokenCache{}).RefreshTokenValid() {
		t.Error("empty refresh token should be invalid")
	}
	if !(&TokenCache{RefreshToken: "r"}).RefreshTokenValid() {
		t.Error("refresh token with unknown expiry should be valid")
	}
	if (&TokenCache{RefreshToken: "r", RefreshExpiresAt: time.Now().Add(-time.Minute)}).RefreshTokenValid() {
		t.Error("expired refresh token should be invalid")
	}
	if !(&TokenCache{RefreshToken: "r", RefreshExpiresAt: time.Now().Add(time.Hour)}).RefreshTokenValid() {
		t.Error("future refresh token should be valid")
	}
}

func TestNewTokenCacheAndInvalidate(t *testing.T) {
	exp := time.Now().Add(time.Hour)
	c := NewTokenCache(&LoginResult{AccessToken: "a", RefreshToken: "r", Username: "u", AccessExpiresAt: exp, RefreshExpiresAt: exp.Add(time.Hour)})
	if c.AccessToken != "a" || c.RefreshToken != "r" || c.Username != "u" || !c.AccessExpiresAt.Equal(exp) {
		t.Errorf("NewTokenCache = %+v", c)
	}
	if time.Since(c.AccessCachedAt) > time.Minute {
		t.Error("AccessCachedAt not stamped")
	}
	c.InvalidateAccess()
	if c.AccessTokenValid() || c.AccessToken != "" {
		t.Error("InvalidateAccess should drop the access token")
	}
	if !c.RefreshTokenValid() || c.RefreshToken != "r" {
		t.Error("InvalidateAccess must keep the refresh token")
	}
}

func TestSaveAndLoadTokenCache_RefreshRoundTrip(t *testing.T) {
	t.Setenv("CLUCKERS_HOME", t.TempDir())
	exp := time.Now().Add(time.Hour).Truncate(time.Second)
	if err := SaveTokenCache(&TokenCache{AccessToken: "a", RefreshToken: "r", Username: "u", AccessCachedAt: time.Now(), AccessExpiresAt: exp, RefreshExpiresAt: exp}); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadTokenCache()
	if err != nil || loaded == nil {
		t.Fatalf("LoadTokenCache: %v %v", loaded, err)
	}
	if loaded.RefreshToken != "r" || !loaded.AccessExpiresAt.Equal(exp) || !loaded.RefreshExpiresAt.Equal(exp) {
		t.Errorf("round trip lost fields: %+v", loaded)
	}
}

// TestLoadTokenCache_LegacyThreeField verifies a tokens.json written by the
// v1.2 launcher (no refresh/expiry fields) loads and uses the legacy TTL rule.
func TestLoadTokenCache_LegacyThreeField(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLUCKERS_HOME", tmp)
	legacy := map[string]interface{}{
		"access_token":     "old-access",
		"username":         "olduser",
		"access_cached_at": time.Now().Add(-10 * time.Minute).Format(time.RFC3339Nano),
	}
	data, _ := json.MarshalIndent(legacy, "", "  ")
	if err := os.MkdirAll(filepath.Join(tmp, "cache"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "cache", "tokens.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadTokenCache()
	if err != nil || loaded == nil {
		t.Fatalf("LoadTokenCache: %v %v", loaded, err)
	}
	if !loaded.AccessTokenValid() {
		t.Error("legacy cache 10 minutes old should still be valid under the 45-minute rule")
	}
	if loaded.RefreshTokenValid() {
		t.Error("legacy cache has no refresh token")
	}
}
