package auth

import (
	"context"

	"github.com/0xc0re/cluckers/internal/gateway"
	"github.com/0xc0re/cluckers/internal/ui"
)

// RegisterResult holds the successful result of account registration.
type RegisterResult struct {
	AccessToken string
	Username    string
}

// Register creates a new account on the Project Crown gateway via LAUNCHER_REGISTER.
// Returns a RegisterResult with the access token on success (auto-login).
func Register(ctx context.Context, client *gateway.Client, username, password, email string) (*RegisterResult, error) {
	req := gateway.RegisterRequest{
		UserName: username,
		Password: password,
		Email:    email,
	}

	var resp gateway.RegisterResponse
	if err := client.Post(ctx, "LAUNCHER_REGISTER", req, &resp); err != nil {
		return nil, err
	}

	if !bool(resp.Success) {
		msg := resp.TextValue
		if msg == "" {
			msg = resp.StringValue
		}
		if msg == "" {
			msg = "Unknown error"
		}
		return nil, &ui.UserError{
			Message:    "Registration failed: " + msg,
			Suggestion: "The username may already be taken, or the email may be invalid.",
		}
	}

	if resp.AccessToken == "" {
		return nil, &ui.UserError{
			Message: "Registration succeeded but no access token received",
		}
	}

	return &RegisterResult{
		AccessToken: resp.AccessToken,
		Username:    username,
	}, nil
}

// RequestLinkCode requests a Discord verification code from the gateway via
// LAUNCHER_REQUEST_LINK_CODE. The returned code must be DM'd to the Discord bot
// to complete account linking.
func RequestLinkCode(ctx context.Context, client *gateway.Client, username, accessToken string) (string, error) {
	req := gateway.LinkCodeRequest{
		UserName:    username,
		AccessToken: accessToken,
	}

	var resp gateway.LinkCodeResponse
	if err := client.Post(ctx, "LAUNCHER_REQUEST_LINK_CODE", req, &resp); err != nil {
		return "", err
	}

	if !bool(resp.Success) {
		return "", &ui.UserError{
			Message:    "Failed to get Discord link code",
			Suggestion: "Try running 'cluckers login' first, then request a link code.",
		}
	}

	code := resp.StringValue
	if code == "" {
		return "", &ui.UserError{
			Message: "Link code response was empty",
		}
	}

	return code, nil
}

// CheckDiscordStatus checks whether the user's Discord account has been linked
// by calling LAUNCHER_DISCORD_STATUS. Returns true if LINKED_FLAG is set.
func CheckDiscordStatus(ctx context.Context, client *gateway.Client, username, accessToken string) (bool, error) {
	req := gateway.GenericRequest{
		UserName:    username,
		AccessToken: accessToken,
	}
	var resp gateway.DiscordStatusResponse
	if err := client.Post(ctx, "LAUNCHER_DISCORD_STATUS", req, &resp); err != nil {
		return false, err
	}
	return bool(resp.LinkedFlag), nil
}
