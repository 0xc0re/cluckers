package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/0xc0re/cluckers/internal/gateway"
)

// PasswordResetResult is the outcome of a password reset request. Code is the
// reset code the user must DM to the Project Crown Discord bot; Message is the
// server's human-readable instruction text. Either may be empty.
type PasswordResetResult struct {
	RequestID string
	Code      string
	Message   string
}

// RequestPasswordReset sends POST /launcher/v1/password-reset. The gateway
// replies with a reset code (in access_token) and instructions (text_value);
// the user completes the reset by DMing the code to the Discord bot. Success is
// signalled by HTTP 2xx.
func RequestPasswordReset(ctx context.Context, client *gateway.Client, username string) (*PasswordResetResult, error) {
	req := gateway.PasswordResetRequest{
		UserName: username,
	}

	var resp gateway.PasswordResetResponse
	if err := client.Do(ctx, http.MethodPost, pathPasswordReset, "", req, &resp); err != nil {
		return nil, err
	}

	return &PasswordResetResult{
		RequestID: resp.RequestID,
		Code:      strings.TrimSpace(resp.AccessToken),
		Message:   strings.TrimSpace(resp.TextValue),
	}, nil
}
