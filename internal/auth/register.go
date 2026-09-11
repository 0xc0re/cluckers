package auth

import (
	"context"
	"net/http"

	"github.com/0xc0re/cluckers/internal/gateway"
)

// Register creates a new account on the Project Crown gateway via
// POST /launcher/v1/account. The reply has the same shape as a login reply:
// on success it carries session tokens (auto-login); a brand-new account is
// normally not linked to Discord yet, in which case the error chain contains
// ErrNotLinked and a *NotLinkedError with the link code to DM to the bot.
func Register(ctx context.Context, client *gateway.Client, username, password, email string) (*LoginResult, error) {
	req := gateway.RegisterRequest{
		UserName: username,
		Password: password,
		Email:    email,
	}

	var resp gateway.SessionResponse
	if err := client.Do(ctx, http.MethodPost, pathAccount, "", req, &resp); err != nil {
		return nil, err
	}

	return sessionResultFrom(&resp, username, "Registration", true)
}
