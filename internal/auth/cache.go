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
	// AccessTokenTTL is how long a cached access token is considered valid when
	// the server did not report an expiry (legacy cache files). Current gateway
	// replies carry access_expires_at_unix, which takes precedence.
	AccessTokenTTL = 45 * time.Minute

	// expirySkew is subtracted from server-reported expiries so a token is
	// refreshed slightly before the server would reject it.
	expirySkew = 60 * time.Second
)

// TokenCache holds the cached launcher session: the short-lived access token,
// the refresh token used to mint a new one without a password, and their
// expiries. Files written by older versions lack the refresh/expiry fields and
// fall back to the AccessCachedAt + AccessTokenTTL rule.
type TokenCache struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token,omitempty"`
	Username         string    `json:"username"`
	AccessCachedAt   time.Time `json:"access_cached_at"`
	AccessExpiresAt  time.Time `json:"access_expires_at,omitzero"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at,omitzero"`
}

// NewTokenCache builds a cache entry from a fresh session, stamping AccessCachedAt.
func NewTokenCache(r *LoginResult) *TokenCache {
	return &TokenCache{
		AccessToken:      r.AccessToken,
		RefreshToken:     r.RefreshToken,
		Username:         r.Username,
		AccessCachedAt:   time.Now(),
		AccessExpiresAt:  r.AccessExpiresAt,
		RefreshExpiresAt: r.RefreshExpiresAt,
	}
}

// tokenCachePath returns the path to the token cache file.
func tokenCachePath() string {
	return filepath.Join(config.CacheDir(), "tokens.json")
}

// AccessTokenValid returns true if the cached access token is still usable:
// before its server-reported expiry (minus skew), or within AccessTokenTTL of
// when it was cached if no expiry is known.
func (c *TokenCache) AccessTokenValid() bool {
	if c.AccessToken == "" {
		return false
	}
	if !c.AccessExpiresAt.IsZero() {
		return time.Now().Before(c.AccessExpiresAt.Add(-expirySkew))
	}
	return time.Since(c.AccessCachedAt) < AccessTokenTTL
}

// AccessRemaining returns how long the cached access token stays valid, or
// zero if it is already invalid.
func (c *TokenCache) AccessRemaining() time.Duration {
	if !c.AccessTokenValid() {
		return 0
	}
	if !c.AccessExpiresAt.IsZero() {
		return time.Until(c.AccessExpiresAt.Add(-expirySkew))
	}
	return AccessTokenTTL - time.Since(c.AccessCachedAt)
}

// RefreshTokenValid returns true if a refresh token is cached and not known to
// be expired. An unknown expiry is treated as valid; the server's 401 is the
// backstop.
func (c *TokenCache) RefreshTokenValid() bool {
	if c.RefreshToken == "" {
		return false
	}
	if c.RefreshExpiresAt.IsZero() {
		return true
	}
	return time.Now().Before(c.RefreshExpiresAt.Add(-expirySkew))
}

// InvalidateAccess forgets the access token (e.g. after the server rejected
// it) while keeping the refresh token, so the next EnsureSession refreshes
// instead of asking for a password.
func (c *TokenCache) InvalidateAccess() {
	c.AccessToken = ""
	c.AccessExpiresAt = time.Time{}
	c.AccessCachedAt = time.Time{}
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
// AccessCachedAt before calling this function (NewTokenCache does).
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
