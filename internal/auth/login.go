package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/0xc0re/cluckers/internal/gateway"
	"github.com/0xc0re/cluckers/internal/ui"
)

// REST API paths for the v1 launcher gateway.
const (
	pathSessionOrLink    = "/launcher/v1/session-or-link"
	pathSession          = "/launcher/v1/session"
	pathContentBootstrap = "/launcher/v1/content-bootstrap"
	pathAccount          = "/launcher/v1/account"
	pathDiscordLink      = "/launcher/v1/discord/link"
	pathDiscordLinkCode  = "/launcher/v1/discord/link/code"
	pathPasswordReset    = "/launcher/v1/password-reset"
	pathBotNames         = "/launcher/v1/supporter/bot-names"
)

// ErrTokenRejected is returned when the server rejects a cached access token
// (e.g. after a server restart or token revocation). Callers can check
// errors.Is(err, ErrTokenRejected) to trigger re-authentication.
var ErrTokenRejected = errors.New("access token rejected by server")

// LoginResult holds the successful result of a gateway login.
type LoginResult struct {
	AccessToken string
	Username    string
	Linked      bool
}

// Login authenticates with the Project Crown gateway via the session-or-link
// endpoint (POST /launcher/v1/session-or-link). This also links the account to
// Discord on first login. Returns a LoginResult with the access token on success,
// or a *ui.UserError with a clear message on failure.
func Login(ctx context.Context, client *gateway.Client, username, password string) (*LoginResult, error) {
	req := gateway.LoginRequest{UserName: username, Password: password}

	var resp gateway.SessionResponse
	if err := client.Do(ctx, http.MethodPost, pathSessionOrLink, "", req, &resp); err != nil {
		return nil, err
	}

	if resp.AccessToken == "" {
		return nil, &ui.UserError{
			Message: "Login succeeded but no access token received",
		}
	}

	uname := resp.UserName
	if uname == "" {
		uname = username
	}

	return &LoginResult{
		AccessToken: resp.AccessToken,
		Username:    uname,
		Linked:      bool(resp.LinkedFlag),
	}, nil
}

// GetContentBootstrap retrieves the content bootstrap from the gateway via
// GET /launcher/v1/content-bootstrap using the access token as a Bearer
// credential. The bootstrap is base64-encoded in portal_info_1 and is a BPS1
// blob consumed by the game via shared memory.
//
// Returns nil, nil if no bootstrap data is present (not an error -- the game
// can launch without it). Returns an error wrapping ErrTokenRejected when the
// server rejects the token (HTTP 401), so callers can re-authenticate.
func GetContentBootstrap(ctx context.Context, client *gateway.Client, accessToken string) ([]byte, error) {
	var resp gateway.BootstrapResponse
	if err := client.Do(ctx, http.MethodGet, pathContentBootstrap, accessToken, nil, &resp); err != nil {
		return nil, classifyTokenError(err, "Content bootstrap request failed")
	}

	if resp.PortalInfo1 == "" {
		return nil, nil // No bootstrap data — not an error.
	}

	data, err := decodeBase64Resilient(resp.PortalInfo1)
	if err != nil {
		return nil, &ui.UserError{
			Message:    "Failed to decode content bootstrap",
			Detail:     err.Error(),
			Suggestion: "This may be a server-side issue. Try again later or contact support on Discord.",
		}
	}

	return data, nil
}

// classifyTokenError marks 401/403 gateway errors as ErrTokenRejected so that
// callers can transparently re-authenticate. Other errors pass through.
func classifyTokenError(err error, message string) error {
	var ue *ui.UserError
	if errors.As(err, &ue) {
		if strings.Contains(ue.Detail, "HTTP 401") || strings.Contains(ue.Detail, "HTTP 403") {
			return &ui.UserError{
				Message:    message + ": " + ue.Message,
				Suggestion: "Your session may have expired. Try logging out and back in.",
				Err:        ErrTokenRejected,
			}
		}
	}
	return err
}

// decodeBase64Resilient tries multiple base64 encoding strategies to handle
// standard (+/), URL-safe (-_), padded, and unpadded variants. It also strips
// whitespace/newlines that some APIs may include in responses.
func decodeBase64Resilient(encoded string) ([]byte, error) {
	// Strip whitespace/newlines (some APIs wrap long base64 lines).
	encoded = strings.NewReplacer(" ", "", "\n", "", "\r", "", "\t", "").Replace(encoded)

	// Try padded variants first (add padding if missing).
	padded := encoded
	if m := len(padded) % 4; m != 0 {
		padded += strings.Repeat("=", 4-m)
	}

	// 1. Standard base64 with padding (+/ alphabet).
	if data, err := base64.StdEncoding.DecodeString(padded); err == nil {
		return data, nil
	}

	// 2. URL-safe base64 with padding (-_ alphabet).
	if data, err := base64.URLEncoding.DecodeString(padded); err == nil {
		return data, nil
	}

	// 3. Raw standard base64 without padding.
	if data, err := base64.RawStdEncoding.DecodeString(encoded); err == nil {
		return data, nil
	}

	// 4. Raw URL-safe base64 without padding.
	if data, err := base64.RawURLEncoding.DecodeString(encoded); err == nil {
		return data, nil
	}

	return nil, fmt.Errorf("all base64 decode strategies failed for input of length %d", len(encoded))
}
