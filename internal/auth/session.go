package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/0xc0re/cluckers/internal/gateway"
	"github.com/0xc0re/cluckers/internal/ui"
)

// ErrRefreshRejected is returned when the server rejects a refresh token
// (HTTP 401/403); the caller must fall back to a password login.
var ErrRefreshRejected = errors.New("refresh token rejected by server")

// ErrNoCredentials is returned by EnsureSession when no cached session can be
// used and no username/password were supplied to log in with.
var ErrNoCredentials = errors.New("no credentials available")

// RefreshSession mints a new access token via POST /launcher/v1/session/refresh.
// The refresh token is the credential; no Authorization header is sent. If the
// reply omits refresh_token or user_name, the corresponding result fields are
// empty and the caller should keep its previous values.
func RefreshSession(ctx context.Context, client *gateway.Client, refreshToken string) (*LoginResult, error) {
	req := gateway.RefreshRequest{RefreshToken: refreshToken}

	var resp gateway.SessionResponse
	if err := client.Do(ctx, http.MethodPost, pathSessionRefresh, "", req, &resp); err != nil {
		var ue *ui.UserError
		if errors.As(err, &ue) && ue.IsStatus(http.StatusUnauthorized, http.StatusForbidden) {
			return nil, &ui.UserError{
				Message:    "Session refresh failed: " + ue.Message,
				Detail:     ue.Detail,
				Suggestion: "Log in again with 'cluckers login'.",
				Err:        ErrRefreshRejected,
				Status:     ue.Status,
				Code:       ue.Code,
			}
		}
		return nil, err
	}

	return sessionResultFrom(&resp, "", "Session refresh", false)
}

// SessionSource says how EnsureSession obtained the session.
type SessionSource int

const (
	SourceCache   SessionSource = iota // Cached access token was still valid.
	SourceRefresh                      // Minted via the refresh token.
	SourceLogin                        // Full username/password login.
)

// SessionRequest is the input to EnsureSession.
type SessionRequest struct {
	Username string // Optional; when set, a cached session for another user is ignored.
	Password string // Optional; required only if a full login turns out to be necessary.
	Cache    *TokenCache
	Verbose  bool
}

// Session is a usable launcher session.
type Session struct {
	Username    string
	AccessToken string
	Cache       *TokenCache // The (possibly updated) cache entry, already saved to disk.
	Source      SessionSource
}

// EnsureSession returns a valid session with the least intrusive method:
// the cached access token if still valid, else a refresh, else a password
// login. It never prompts. A full login may fail with ErrNotLinked or
// ErrPinRequired exactly like Login; when no password is available it fails
// with ErrNoCredentials. Cache save failures are non-fatal.
func EnsureSession(ctx context.Context, client *gateway.Client, req SessionRequest) (*Session, error) {
	cache := req.Cache
	sameUser := cache != nil && (req.Username == "" || cache.Username == req.Username)

	if sameUser && cache.AccessTokenValid() {
		return &Session{Username: cache.Username, AccessToken: cache.AccessToken, Cache: cache, Source: SourceCache}, nil
	}

	if sameUser && cache.RefreshTokenValid() {
		result, err := RefreshSession(ctx, client, cache.RefreshToken)
		if err == nil {
			merged := NewTokenCache(result)
			if merged.RefreshToken == "" {
				merged.RefreshToken = cache.RefreshToken
				merged.RefreshExpiresAt = cache.RefreshExpiresAt
			}
			if merged.Username == "" {
				merged.Username = cache.Username
			}
			if saveErr := SaveTokenCache(merged); saveErr != nil {
				ui.Verbose(fmt.Sprintf("Could not save token cache: %s", saveErr), req.Verbose)
			}
			return &Session{Username: merged.Username, AccessToken: merged.AccessToken, Cache: merged, Source: SourceRefresh}, nil
		}
		ui.Verbose(fmt.Sprintf("Session refresh failed, falling back to login: %s", err), req.Verbose)
	}

	if req.Username == "" || req.Password == "" {
		return nil, ErrNoCredentials
	}

	result, err := Login(ctx, client, req.Username, req.Password)
	if err != nil {
		return nil, err
	}
	fresh := NewTokenCache(result)
	if saveErr := SaveTokenCache(fresh); saveErr != nil {
		ui.Verbose(fmt.Sprintf("Could not save token cache: %s", saveErr), req.Verbose)
	}
	return &Session{Username: fresh.Username, AccessToken: fresh.AccessToken, Cache: fresh, Source: SourceLogin}, nil
}
