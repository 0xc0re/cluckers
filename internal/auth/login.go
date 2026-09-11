package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/0xc0re/cluckers/internal/gateway"
	"github.com/0xc0re/cluckers/internal/ui"
)

// REST API paths for the v1 launcher gateway.
const (
	pathSessionOrLink  = "/launcher/v1/session-or-link"
	pathSessionRefresh = "/launcher/v1/session/refresh"
	pathAccount        = "/launcher/v1/account"
	pathPasswordReset  = "/launcher/v1/password-reset"
	pathBotNames       = "/launcher/v1/supporter/bot-names"
)

// text_value sentinels the gateway uses on an otherwise successful session
// reply when the server is running in developer-only mode.
const (
	textPinRequired = "PIN_REQUIRED"
	textPinInvalid  = "PIN_INVALID"
)

// DiscordInviteURL is the public Project Crown Discord server.
const DiscordInviteURL = "https://discord.gg/realmroyale"

// DiscordBotUserID is the Discord user id of the Project Crown bot that link
// codes and password-reset codes are DM'd to.
const DiscordBotUserID = "1404860983419211839"

// DiscordBotDMURL opens a DM with the Project Crown bot.
const DiscordBotDMURL = "https://discord.com/users/" + DiscordBotUserID

// ErrTokenRejected is returned when the server rejects a cached access token
// (e.g. after a server restart or token revocation). Callers can check
// errors.Is(err, ErrTokenRejected) to trigger re-authentication.
var ErrTokenRejected = errors.New("access token rejected by server")

// ErrPinRequired is returned when the gateway is in developer-only mode: the
// session reply is 2xx with text_value PIN_REQUIRED / PIN_INVALID and no
// token. The official launcher cannot send a PIN either, so this is a hard stop.
var ErrPinRequired = errors.New("server is in developer-only mode")

// ErrNotLinked is returned when the account has not been linked to Discord
// yet. The error chain also contains a *NotLinkedError carrying the link code.
var ErrNotLinked = errors.New("discord account not linked")

// NotLinkedError carries the Discord link code the gateway handed out. It is
// wrapped inside a *ui.UserError and unwraps to ErrNotLinked, so callers can
// use errors.Is(err, ErrNotLinked) to detect it and errors.As to read the code.
type NotLinkedError struct {
	LinkCode string
	Detail   string // Server-supplied text_value, may be empty.
}

func (e *NotLinkedError) Error() string { return ErrNotLinked.Error() }
func (e *NotLinkedError) Unwrap() error { return ErrNotLinked }

// LoginResult holds the successful result of a gateway login, registration,
// or session refresh.
type LoginResult struct {
	AccessToken      string
	RefreshToken     string
	Username         string
	AccessExpiresAt  time.Time // Zero when the server sent nothing usable.
	RefreshExpiresAt time.Time // Zero when the server sent nothing usable.
	Linked           bool
	SupporterTier    string // custom_message on the login reply; may be empty.
}

// Login authenticates with the Project Crown gateway via the session-or-link
// endpoint (POST /launcher/v1/session-or-link). On success it returns the
// session tokens. Special 2xx outcomes are surfaced as errors:
//   - errors.Is(err, ErrNotLinked): the account must be linked to Discord first;
//     errors.As(err, &*NotLinkedError) yields the link code to DM to the bot.
//   - errors.Is(err, ErrPinRequired): the server is in developer-only mode.
func Login(ctx context.Context, client *gateway.Client, username, password string) (*LoginResult, error) {
	req := gateway.LoginRequest{UserName: username, Password: password}

	var resp gateway.SessionResponse
	if err := client.Do(ctx, http.MethodPost, pathSessionOrLink, "", req, &resp); err != nil {
		return nil, err
	}

	return sessionResultFrom(&resp, username, "Login", true)
}

// sessionResultFrom interprets a 2xx session reply (login, register, refresh).
// checkLink controls whether linked_flag != 1 is treated as "not linked"; the
// refresh endpoint is only ever called for linked accounts and may omit the flag.
func sessionResultFrom(resp *gateway.SessionResponse, fallbackUser, what string, checkLink bool) (*LoginResult, error) {
	uname := resp.UserName
	if uname == "" {
		uname = fallbackUser
	}

	text := strings.TrimSpace(resp.TextValue)
	if strings.EqualFold(text, textPinRequired) || strings.EqualFold(text, textPinInvalid) {
		return nil, &ui.UserError{
			Message:    "The Project Crown server is in developer-only mode and requires an access PIN.",
			Detail:     "text_value=" + text,
			Suggestion: "The launcher cannot supply a developer PIN. Wait for the server to leave developer-only mode, or ask the Project Crown team on Discord (" + DiscordInviteURL + ").",
			Err:        ErrPinRequired,
		}
	}

	if checkLink && !resp.IsLinked() && resp.AccessToken != "" {
		return nil, &ui.UserError{
			Message:    "Your account is not linked to Discord yet.",
			Detail:     text,
			Suggestion: "DM the link code to the Project Crown bot (" + DiscordBotDMURL + "; join via " + DiscordInviteURL + "), then log in again.",
			Err:        &NotLinkedError{LinkCode: strings.TrimSpace(resp.AccessToken), Detail: text},
		}
	}

	if resp.AccessToken == "" {
		detail := text
		if resp.CustomMessage != "" {
			detail = strings.TrimSpace(detail + " " + resp.CustomMessage)
		}
		return nil, &ui.UserError{
			Message: what + " succeeded but no access token received",
			Detail:  detail,
		}
	}

	return &LoginResult{
		AccessToken:      resp.AccessToken,
		RefreshToken:     resp.RefreshToken,
		Username:         uname,
		AccessExpiresAt:  expiryFrom(resp.AccessExpiresAtUnix, resp.ExpirationDatetime),
		RefreshExpiresAt: expiryFrom(resp.RefreshExpiresAtUnix, resp.RefreshExpirationDatetime),
		Linked:           resp.IsLinked(),
		SupporterTier:    strings.TrimSpace(resp.CustomMessage),
	}, nil
}

// expiryFrom derives an expiry time from a unix timestamp (preferred) or an
// RFC 3339 datetime. Returns the zero time when neither is usable.
func expiryFrom(unix json.Number, datetime string) time.Time {
	if unix != "" {
		if secs, err := unix.Int64(); err == nil && secs > 0 {
			return time.Unix(secs, 0)
		}
	}
	datetime = strings.TrimSpace(datetime)
	if datetime == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05"} {
		if ts, err := time.Parse(layout, datetime); err == nil {
			return ts
		}
	}
	return time.Time{}
}

// decodeBootstrap base64-decodes a portal_info_1 value into raw bootstrap bytes.
func decodeBootstrap(encoded string) ([]byte, error) {
	data, err := decodeBase64Resilient(encoded)
	if err != nil {
		return nil, &ui.UserError{
			Message:    "Failed to decode content bootstrap",
			Detail:     err.Error(),
			Suggestion: "This may be a server-side issue. Try again later or contact support on Discord.",
		}
	}
	return data, nil
}

// classifyTokenError marks 401/403 gateway errors as ErrTokenRejected so that
// callers can transparently re-authenticate. Other errors pass through.
func classifyTokenError(err error, message string) error {
	var ue *ui.UserError
	if errors.As(err, &ue) && ue.IsStatus(http.StatusUnauthorized, http.StatusForbidden) {
		return &ui.UserError{
			Message:    message + ": " + ue.Message,
			Detail:     ue.Detail,
			Suggestion: "Your session may have expired. Try logging out and back in.",
			Err:        ErrTokenRejected,
			Status:     ue.Status,
			Code:       ue.Code,
		}
	}
	return err
}

// decodeBase64Resilient tries multiple base64 encoding strategies to handle
// standard (+/), URL-safe (-_), padded, and unpadded variants. It also strips
// whitespace/newlines that some APIs may include in responses.
func decodeBase64Resilient(encoded string) ([]byte, error) {
	// Strip whitespace/newlines (some APIs wrap long base64 lines).
	encoded = strings.NewReplacer(" ", "", "\n", "", "\r", "", "\t", "").Replace(encoded)

	// Try padded variants first (add padding if missing).
	padded := encoded
	if m := len(padded) % 4; m != 0 {
		padded += strings.Repeat("=", 4-m)
	}

	// 1. Standard base64 with padding (+/ alphabet).
	if data, err := base64.StdEncoding.DecodeString(padded); err == nil {
		return data, nil
	}

	// 2. URL-safe base64 with padding (-_ alphabet).
	if data, err := base64.URLEncoding.DecodeString(padded); err == nil {
		return data, nil
	}

	// 3. Raw standard base64 without padding.
	if data, err := base64.RawStdEncoding.DecodeString(encoded); err == nil {
		return data, nil
	}

	// 4. Raw URL-safe base64 without padding.
	if data, err := base64.RawURLEncoding.DecodeString(encoded); err == nil {
		return data, nil
	}

	return nil, fmt.Errorf("all base64 decode strategies failed for input of length %d", len(encoded))
}
