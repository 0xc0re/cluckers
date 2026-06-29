package auth

import (
	"context"
	"net/http"

	"github.com/0xc0re/cluckers/internal/gateway"
)

// RequestPasswordReset sends a POST /launcher/v1/password-reset request to the
// gateway. The server sends reset instructions to the user's registered email
// or Discord. This is a fire-and-forget operation — no polling or verification.
// Success is signalled by HTTP 2xx.
func RequestPasswordReset(ctx context.Context, client *gateway.Client, username string) error {
	req := gateway.PasswordResetRequest{
		UserName: username,
	}

	var resp gateway.PasswordResetResponse
	return client.Do(ctx, http.MethodPost, pathPasswordReset, "", req, &resp)
}
