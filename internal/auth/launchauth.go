package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/0xc0re/cluckers/internal/gateway"
	"github.com/0xc0re/cluckers/internal/ui"
)

// pathLaunchAuth issues the per-launch artifacts (bootstrap + launch token).
const pathLaunchAuth = "/launcher/v1/launch-auth"

// clientBuildHeader carries the installed game build version on launch-auth.
const clientBuildHeader = "x-realm-client-build"

// bootstrapSize is the size of the BPS1 content bootstrap the game expects.
const bootstrapSize = 136

// LaunchAuthResult holds the artifacts the game needs to start a session.
type LaunchAuthResult struct {
	Bootstrap       []byte    // Raw 136-byte BPS1 blob for the shared-memory mapping; nil if absent.
	LaunchToken     string    // Short-lived token written to the -token_file the game reads.
	LaunchExpiresAt time.Time // Zero when the server sent nothing usable.
}

// LaunchAuth calls POST /launcher/v1/launch-auth with the session access
// token as a Bearer credential and an empty JSON object body. clientBuild is
// the installed game build version (e.g. "0.39.6969.0") sent as the
// x-realm-client-build header; pass "" to omit the header.
//
// Returns an error wrapping ErrTokenRejected on HTTP 401/403 so callers can
// refresh or re-authenticate. An empty launch token is an error: the game
// cannot authenticate without it. A missing bootstrap is not (Bootstrap is
// nil), matching the previous graceful degradation.
func LaunchAuth(ctx context.Context, client *gateway.Client, accessToken, clientBuild string) (*LaunchAuthResult, error) {
	var headers map[string]string
	if clientBuild != "" {
		headers = map[string]string{clientBuildHeader: clientBuild}
	}

	var resp gateway.LaunchAuthResponse
	if err := client.DoWithHeaders(ctx, http.MethodPost, pathLaunchAuth, accessToken, headers, struct{}{}, &resp); err != nil {
		return nil, classifyTokenError(err, "Launch authorization failed")
	}

	if resp.LaunchToken == "" {
		return nil, &ui.UserError{
			Message:    "Gateway returned an empty launch token.",
			Detail:     "POST " + pathLaunchAuth + " succeeded without launch_token",
			Suggestion: "This is a server-side issue. Try again later or ask on the Project Crown Discord.",
		}
	}

	result := &LaunchAuthResult{
		LaunchToken:     resp.LaunchToken,
		LaunchExpiresAt: expiryFrom(resp.LaunchExpiresAtUnix, resp.LaunchExpirationDatetime),
	}

	if resp.PortalInfo1 != "" {
		data, err := decodeBootstrap(resp.PortalInfo1)
		if err != nil {
			return nil, err
		}
		if len(data) != bootstrapSize {
			ui.Warn(fmt.Sprintf("Content bootstrap is %d bytes (expected %d)", len(data), bootstrapSize))
		}
		result.Bootstrap = data
	}

	return result, nil
}
