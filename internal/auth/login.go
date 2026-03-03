package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/0xc0re/cluckers/internal/gateway"
	"github.com/0xc0re/cluckers/internal/ui"
)

// LoginResult holds the successful result of a gateway login.
type LoginResult struct {
	AccessToken string
	Username    string
}

// Login authenticates with the Project Crown gateway via LAUNCHER_LOGIN_OR_LINK.
// Returns a LoginResult with the access token on success, or a UserError with
// a clear message on failure.
func Login(ctx context.Context, client *gateway.Client, username, password string) (*LoginResult, error) {
	req := gateway.LoginRequest{
		UserName: username,
		Password: password,
	}

	var resp gateway.LoginResponse
	if err := client.Post(ctx, "LAUNCHER_LOGIN_OR_LINK", req, &resp); err != nil {
		return nil, err
	}

	if !bool(resp.Success) {
		msg := resp.TextValue
		if msg == "" {
			msg = "Unknown error"
		}
		return nil, &ui.UserError{
			Message: "Login failed: " + msg,
		}
	}

	if resp.AccessToken == "" {
		return nil, &ui.UserError{
			Message: "Login succeeded but no access token received",
		}
	}

	return &LoginResult{
		AccessToken: resp.AccessToken,
		Username:    username,
	}, nil
}

// GetOIDCToken retrieves an EAC OIDC JWT token from the gateway via
// LAUNCHER_EAC_OIDC_TOKEN. Tries response fields in order: PortalInfo1,
// StringValue, TextValue (matching the POC's fallback pattern).
func GetOIDCToken(ctx context.Context, client *gateway.Client, username, accessToken string) (string, error) {
	req := gateway.GenericRequest{
		UserName:    username,
		AccessToken: accessToken,
	}

	var resp gateway.OIDCTokenResponse
	if err := client.Post(ctx, "LAUNCHER_EAC_OIDC_TOKEN", req, &resp); err != nil {
		return "", err
	}

	// Try fields in order, matching POC: PORTAL_INFO_1 -> STRING_VALUE -> TEXT_VALUE
	token := resp.PortalInfo1
	if token == "" {
		token = resp.StringValue
	}
	if token == "" {
		token = resp.TextValue
	}

	if token == "" {
		return "", &ui.UserError{
			Message: "Failed to get OIDC token",
		}
	}

	return token, nil
}

// GetContentBootstrap retrieves the content bootstrap from the gateway via
// LAUNCHER_CONTENT_BOOTSTRAP. The bootstrap is base64-encoded in the response
// and is typically 136 bytes with a BPS1 magic header.
//
// Returns nil, nil if no bootstrap data is present (not an error -- the game
// can launch without it).
func GetContentBootstrap(ctx context.Context, client *gateway.Client, username, accessToken string) ([]byte, error) {
	req := gateway.GenericRequest{
		UserName:    username,
		AccessToken: accessToken,
	}

	var resp gateway.BootstrapResponse
	if err := client.Post(ctx, "LAUNCHER_CONTENT_BOOTSTRAP", req, &resp); err != nil {
		return nil, err
	}

	// Try fields in order, matching POC: PORTAL_INFO_1 -> STRING_VALUE
	encoded := resp.PortalInfo1
	if encoded == "" {
		encoded = resp.StringValue
	}

	if encoded == "" {
		return nil, nil // No bootstrap data — not an error.
	}

	data, err := decodeBase64Resilient(encoded)
	if err != nil {
		return nil, &ui.UserError{
			Message:    "Failed to decode content bootstrap",
			Detail:     err.Error(),
			Suggestion: "This may be a server-side issue. Try again later or contact support on Discord.",
		}
	}

	return data, nil
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
