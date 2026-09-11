package auth

import (
	"context"
	"errors"
	"time"

	"github.com/0xc0re/cluckers/internal/gateway"
	"github.com/0xc0re/cluckers/internal/ui"
)

// Link polling parameters. The official launcher re-checks every 3 seconds.
// Package variables so tests can shorten them.
var (
	linkPollInterval = 3 * time.Second
	linkTimeout      = 5 * time.Minute
)

// WaitForLink polls POST /launcher/v1/session-or-link until the account is
// linked to Discord and a real session is issued. Each unlinked reply carries
// the current link code; onCode is called whenever that code changes (the
// server may rotate it), so the caller can show it to the user. Returns the
// session on success, the underlying error on any non-link failure (including
// ErrPinRequired), ctx.Err() on cancellation, or a *ui.UserError on timeout.
func WaitForLink(ctx context.Context, client *gateway.Client, username, password string, onCode func(code string)) (*LoginResult, error) {
	deadline := time.NewTimer(linkTimeout)
	defer deadline.Stop()

	lastCode := ""
	for {
		result, err := Login(ctx, client, username, password)
		if err == nil {
			return result, nil
		}
		var nl *NotLinkedError
		if !errors.As(err, &nl) {
			return nil, err
		}
		if nl.LinkCode != "" && nl.LinkCode != lastCode {
			lastCode = nl.LinkCode
			if onCode != nil {
				onCode(nl.LinkCode)
			}
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline.C:
			return nil, &ui.UserError{
				Message:    "Timed out waiting for the Discord link.",
				Suggestion: "DM the code to the Project Crown bot, then run 'cluckers login' again.",
			}
		case <-time.After(linkPollInterval):
		}
	}
}
