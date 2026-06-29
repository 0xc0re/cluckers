package auth

import (
	"context"
	"net/http"

	"github.com/0xc0re/cluckers/internal/gateway"
	"github.com/0xc0re/cluckers/internal/ui"
)

// RegisterResult holds the successful result of account registration.
type RegisterResult struct {
	AccessToken string
	Username    string
}

// Register creates a new account on the Project Crown gateway via
// POST /launcher/v1/account. Returns a RegisterResult with the access token on
// success (auto-login). Success is signalled by HTTP 2xx.
func Register(ctx context.Context, client *gateway.Client, username, password, email string) (*RegisterResult, error) {
	req := gateway.RegisterRequest{
		UserName: username,
		Password: password,
		Email:    email,
	}

	var resp gateway.SessionResponse
	if err := client.Do(ctx, http.MethodPost, pathAccount, "", req, &resp); err != nil {
		return nil, err
	}

	if resp.AccessToken == "" {
		return nil, &ui.UserError{
			Message: "Registration succeeded but no access token received",
		}
	}

	uname := resp.UserName
	if uname == "" {
		uname = username
	}

	return &RegisterResult{
		AccessToken: resp.AccessToken,
		Username:    uname,
	}, nil
}

// RequestLinkCode requests a Discord verification code from the gateway via
// POST /launcher/v1/discord/link/code. The returned code must be DM'd to the
// Discord bot to complete account linking. This endpoint uses password auth.
func RequestLinkCode(ctx context.Context, client *gateway.Client, username, password string) (string, error) {
	req := gateway.LinkCodeRequest{
		UserName: username,
		Password: password,
	}

	var resp gateway.LinkCodeResponse
	if err := client.Do(ctx, http.MethodPost, pathDiscordLinkCode, "", req, &resp); err != nil {
		return "", err
	}

	if resp.Code == "" {
		return "", &ui.UserError{
			Message:    "Link code response was empty",
			Suggestion: "This may be a temporary server issue. Try again later.",
		}
	}

	return resp.Code, nil
}

// CheckDiscordStatus checks whether the user's Discord account has been linked
// by calling GET /launcher/v1/discord/link with the access token as a Bearer
// credential. Returns true if linked_flag is set.
func CheckDiscordStatus(ctx context.Context, client *gateway.Client, username, accessToken string) (bool, error) {
	_ = username // The v1 endpoint identifies the user from the Bearer token.
	var resp gateway.DiscordStatusResponse
	if err := client.Do(ctx, http.MethodGet, pathDiscordLink, accessToken, nil, &resp); err != nil {
		return false, err
	}
	return bool(resp.LinkedFlag), nil
}
