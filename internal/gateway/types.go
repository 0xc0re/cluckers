package gateway

import (
	"encoding/json"
	"strings"
)

// LinkFlag decodes linked_flag exactly like the official launcher: linked only
// when the value is the number 1, the bool true, or the strings "1"/"true".
// Any other value (0, -1, null, missing, other strings) means not linked.
type LinkFlag bool

func (b *LinkFlag) UnmarshalJSON(data []byte) error {
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	switch v := raw.(type) {
	case bool:
		*b = LinkFlag(v)
	case float64:
		*b = LinkFlag(v == 1)
	case string:
		*b = LinkFlag(v == "1" || strings.EqualFold(v, "true"))
	default:
		*b = false
	}
	return nil
}

// HealthResponse is the response from GET /healthz.
type HealthResponse struct {
	Service string `json:"service"`
	Status  string `json:"status"`
}

// LoginRequest is the request body for the session endpoints
// (/launcher/v1/session-or-link and /launcher/v1/session).
type LoginRequest struct {
	UserName string `json:"user_name"`
	Password string `json:"password"`
}

// IsLinked reports whether the reply says the account is Discord-linked
// (linked_flag == 1 / true, nothing else).
func (r *SessionResponse) IsLinked() bool { return bool(r.LinkedFlag) }

// SessionResponse is the 200 response from the session endpoints
// (/launcher/v1/session-or-link, /launcher/v1/session/refresh) and account
// creation. Success is signalled by HTTP 2xx; there is no SUCCESS field. Errors
// arrive as RFC 7807 problem+json and are surfaced as *ui.UserError before
// unmarshalling.
//
// Semantics of a 2xx body (matching the official 1.6.3 launcher):
//   - linked_flag != 1: the account is not linked to Discord yet and
//     access_token carries the Discord LINK CODE, not a session token.
//   - linked_flag == 1 and text_value is PIN_REQUIRED / PIN_INVALID: the server
//     is in developer-only mode; no token is issued.
//   - otherwise access_token is a real session token (lpt_v1_...), and
//     refresh_token / *_expires_at_unix describe the session lifetime.
//   - custom_message is the supporter tier name, custom_value_1 the tier level,
//     custom_value_2/3 bot-name slots total/used, custom_value_4 announcement
//     seconds, text_value announcement text, portal_info_1 a JSON array of bot
//     names (on the login response).
type SessionResponse struct {
	AccountID                 json.Number `json:"account_id"`
	UserName                  string      `json:"user_name"`
	SessionID                 string      `json:"session_id"`
	AccessToken               string      `json:"access_token"`
	ExpirationDatetime        string      `json:"expiration_datetime"`
	AccessExpiresAtUnix       json.Number `json:"access_expires_at_unix"`
	RefreshToken              string      `json:"refresh_token"`
	RefreshExpirationDatetime string      `json:"refresh_expiration_datetime"`
	RefreshExpiresAtUnix      json.Number `json:"refresh_expires_at_unix"`
	LinkedFlag                LinkFlag    `json:"linked_flag"`
	CustomMessage             string      `json:"custom_message"`
	CustomValue1              json.Number `json:"custom_value_1"`
	CustomValue2              json.Number `json:"custom_value_2"`
	CustomValue3              json.Number `json:"custom_value_3"`
	CustomValue4              json.Number `json:"custom_value_4"`
	TextValue                 string      `json:"text_value"`
	PortalInfo1               string      `json:"portal_info_1"`
}

// RefreshRequest is the request body for POST /launcher/v1/session/refresh.
// The call carries no Authorization header; the refresh token is the credential.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// LaunchAuthResponse is the response from POST /launcher/v1/launch-auth
// (Bearer access token, empty JSON object body, x-realm-client-build header).
// portal_info_1 holds the base64-encoded 136-byte BPS1 content bootstrap and
// launch_token is the short-lived token the game reads from -token_file.
type LaunchAuthResponse struct {
	AccountID                json.Number `json:"account_id"`
	SessionID                string      `json:"session_id"`
	LaunchToken              string      `json:"launch_token"`
	LaunchExpirationDatetime string      `json:"launch_expiration_datetime"`
	LaunchExpiresAtUnix      json.Number `json:"launch_expires_at_unix"`
	ExpirationDatetime       string      `json:"expiration_datetime"`
	CustomValue1             json.Number `json:"custom_value_1"`
	CustomValue2             json.Number `json:"custom_value_2"`
	PortalInfo1              string      `json:"portal_info_1"`
}

// RegisterRequest is the request body for POST /launcher/v1/account.
type RegisterRequest struct {
	UserName string `json:"user_name"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

// PasswordResetRequest is the request body for POST /launcher/v1/password-reset.
type PasswordResetRequest struct {
	UserName string `json:"user_name"`
}

// PasswordResetResponse is the response from POST /launcher/v1/password-reset.
// access_token carries the RESET CODE the user must DM to the Discord bot, and
// text_value the human-readable instructions.
type PasswordResetResponse struct {
	RequestID   string `json:"request_id"`
	AccessToken string `json:"access_token"`
	TextValue   string `json:"text_value"`
}

// BotNameUpsertRequest is the request body for
// PUT /launcher/v1/supporter/bot-names/{slot}. The slot index is in the path.
type BotNameUpsertRequest struct {
	BotName string `json:"bot_name"`
}
