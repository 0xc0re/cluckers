package auth

import (
	"context"

	"github.com/0xc0re/cluckers/internal/gateway"
	"github.com/0xc0re/cluckers/internal/ui"
)

// RequestPasswordReset sends a LAUNCHER_REQUEST_PASSWORD_RESET request to the
// gateway. The server sends reset instructions to the user's registered email
// or Discord. This is a fire-and-forget operation — no polling or verification.
func RequestPasswordReset(ctx context.Context, client *gateway.Client, username string) error {
	req := gateway.PasswordResetRequest{
		UserName: username,
	}

	var resp gateway.PasswordResetResponse
	if err := client.Post(ctx, "LAUNCHER_REQUEST_PASSWORD_RESET", req, &resp); err != nil {
		return err
	}

	if !bool(resp.Success) {
		msg := resp.TextValue
		if msg == "" {
			msg = resp.StringValue
		}
		if msg == "" {
			msg = "Unknown error"
		}
		return &ui.UserError{
			Message:    "Password reset failed: " + msg,
			Suggestion: "Make sure the username is correct and try again.",
		}
	}

	return nil
}
