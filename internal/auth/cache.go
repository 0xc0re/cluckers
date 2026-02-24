package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/0xc0re/cluckers/internal/config"
)

// TTL constants for cached tokens.
const (
	// AccessTokenTTL is how long a cached access token is considered valid.
	AccessTokenTTL = 24 * time.Hour
	// OIDCTokenTTL is how long a cached OIDC token is considered valid.
	// Slightly under 1 hour to avoid edge-case expiry mid-launch.
	OIDCTokenTTL = 55 * time.Minute
)

// TokenCache holds cached tokens with independent per-token timestamps.
// Each token type has its own TTL timestamp so that refreshing one token
// does not reset the other's TTL.
type TokenCache struct {
	AccessToken    string    `json:"access_token"`
	OIDCToken      string    `json:"oidc_token"`
	Username       string    `json:"username"`
	AccessCachedAt time.Time `json:"access_cached_at"`
	OIDCCachedAt   time.Time `json:"oidc_cached_at"`
}

// tokenCachePath returns the path to the token cache file.
func tokenCachePath() string {
	return filepath.Join(config.CacheDir(), "tokens.json")
}

// AccessTokenValid returns true if the cached access token is still within its TTL.
func (c *TokenCache) AccessTokenValid() bool {
	if c.AccessToken == "" {
		return false
	}
	return time.Since(c.AccessCachedAt) < AccessTokenTTL
}

// OIDCTokenValid returns true if the cached OIDC token is still within its TTL.
func (c *TokenCache) OIDCTokenValid() bool {
	if c.OIDCToken == "" {
		return false
	}
	return time.Since(c.OIDCCachedAt) < OIDCTokenTTL
}

// LoadTokenCache reads the token cache from disk. Returns nil, nil if the file
// does not exist (first run, not an error).
func LoadTokenCache() (*TokenCache, error) {
	data, err := os.ReadFile(tokenCachePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // First run — no cache yet.
		}
		return nil, err
	}

	var cache TokenCache
	if err := json.Unmarshal(data, &cache); err != nil {
		// Corrupt cache — treat as missing.
		return nil, nil
	}

	return &cache, nil
}

// SaveTokenCache writes the token cache to disk with 0600 permissions.
// Creates the cache directory if needed.
// Callers are responsible for setting AccessCachedAt and/or OIDCCachedAt
// before calling this function. This ensures that refreshing one token
// does not reset the other's TTL.
func SaveTokenCache(cache *TokenCache) error {
	if err := config.EnsureDir(config.CacheDir()); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(tokenCachePath(), data, 0600)
}

// ClearTokenCache removes the token cache file from disk.
// Used by `cluckers logout` to wipe cached tokens.
func ClearTokenCache() error {
	err := os.Remove(tokenCachePath())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
