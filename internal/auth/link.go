package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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

// isFatalLinkError reports whether a Login failure should end the link wait:
// the PIN gate, a credential rejection (401/403), or context cancellation.
func isFatalLinkError(ctx context.Context, err error) bool {
	if ctx.Err() != nil || errors.Is(err, ErrPinRequired) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var ue *ui.UserError
	return errors.As(err, &ue) && ue.IsStatus(http.StatusUnauthorized, http.StatusForbidden)
}

// WaitForLink polls POST /launcher/v1/session-or-link until the account is
// linked to Discord and a real session is issued. Each unlinked reply carries
// the current link code; onCode is called whenever that code changes (the
// server may rotate it), so the caller can show it to the user. Returns the
// session on success; ErrPinRequired or a 401/403 credential rejection ends the
// wait immediately, ctx.Err() on cancellation, and a *ui.UserError on timeout.
// Transient gateway errors are logged and polling continues.
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
		switch {
		case errors.As(err, &nl):
			if nl.LinkCode != "" && nl.LinkCode != lastCode {
				lastCode = nl.LinkCode
				if onCode != nil {
					onCode(nl.LinkCode)
				}
			}
		case isFatalLinkError(ctx, err):
			return nil, err
		default:
			// Transient gateway trouble (5xx, HTML error page, timeout):
			// keep polling until the deadline, like the official client.
			ui.Verbose(fmt.Sprintf("Link poll failed, retrying: %s", err), true)
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
