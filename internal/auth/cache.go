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
	// The v1 launcher access token (lpt_v1_...) is short-lived (~1 hour), so
	// this is kept conservatively under that window. Token rejection (HTTP 401)
	// also triggers transparent re-authentication.
	AccessTokenTTL = 45 * time.Minute
)

// TokenCache holds the cached launcher access token and its timestamp.
// The v1 API no longer issues a separate EAC/OIDC token, so only the access
// token is cached.
type TokenCache struct {
	AccessToken    string    `json:"access_token"`
	Username       string    `json:"username"`
	AccessCachedAt time.Time `json:"access_cached_at"`
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
// Creates the cache directory if needed. Callers are responsible for setting
// AccessCachedAt before calling this function.
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
