package auth

import (
	"context"
	"encoding/base64"
	"strings"

	"github.com/cstory/cluckers/internal/gateway"
	"github.com/cstory/cluckers/internal/ui"
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

	// Fix base64 padding if needed (some servers omit trailing '=').
	if m := len(encoded) % 4; m != 0 {
		encoded += strings.Repeat("=", 4-m)
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, &ui.UserError{
			Message: "Failed to decode content bootstrap",
			Detail:  err.Error(),
		}
	}

	return data, nil
}
