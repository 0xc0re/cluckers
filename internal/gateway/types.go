package gateway

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/0xc0re/cluckers/internal/ui"
)

// FlexBool handles JSON booleans that may arrive as bool, number, or string.
// The v1 REST API returns linked_flag as 1/0 instead of true/false.
type FlexBool bool

func (b *FlexBool) UnmarshalJSON(data []byte) error {
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	switch v := raw.(type) {
	case bool:
		*b = FlexBool(v)
	case float64:
		*b = FlexBool(v != 0)
	case string:
		parsed, parseErr := strconv.ParseBool(v)
		if parseErr != nil {
			ui.Warn(fmt.Sprintf("FlexBool: unexpected string %q, defaulting to false", v))
		}
		*b = FlexBool(parsed)
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

// SessionResponse is the 200 response from the session endpoints
// (/launcher/v1/session-or-link, /launcher/v1/session) and account creation.
// Success is signalled by HTTP 2xx; there is no SUCCESS field. Errors arrive as
// RFC 7807 problem+json and are surfaced as *ui.UserError before unmarshalling.
type SessionResponse struct {
	AccountID          json.Number `json:"account_id"`
	UserName           string      `json:"user_name"`
	SessionID          string      `json:"session_id"`
	AccessToken        string      `json:"access_token"`
	ExpirationDatetime string      `json:"expiration_datetime"`
	LinkedFlag         FlexBool    `json:"linked_flag"`
	CustomMessage      string      `json:"custom_message"`
	CustomValue1       json.Number `json:"custom_value_1"`
	TextValue          string      `json:"text_value"`
	PortalInfo1        string      `json:"portal_info_1"`
}

// BootstrapResponse is the response from GET /launcher/v1/content-bootstrap.
// portal_info_1 holds the base64-encoded BPS1 content bootstrap blob.
type BootstrapResponse struct {
	AccountID          json.Number `json:"account_id"`
	SessionID          string      `json:"session_id"`
	Version            json.Number `json:"version"`
	CustomValue1       json.Number `json:"custom_value_1"`
	CustomValue2       json.Number `json:"custom_value_2"`
	ExpirationDatetime string      `json:"expiration_datetime"`
	PortalInfo1        string      `json:"portal_info_1"`
}

// RegisterRequest is the request body for POST /launcher/v1/account.
type RegisterRequest struct {
	UserName string `json:"user_name"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

// LinkCodeRequest is the request body for POST /launcher/v1/discord/link/code.
type LinkCodeRequest struct {
	UserName string `json:"user_name"`
	Password string `json:"password"`
}

// LinkCodeResponse is the response from POST /launcher/v1/discord/link/code.
type LinkCodeResponse struct {
	Code        string   `json:"code"`
	AccessToken string   `json:"access_token"`
	LinkedFlag  FlexBool `json:"linked_flag"`
}

// DiscordStatusResponse is the response from GET /launcher/v1/discord/link.
type DiscordStatusResponse struct {
	LinkedFlag     FlexBool `json:"linked_flag"`
	PortalUserID   string   `json:"portal_userid"`
	PortalUsername string   `json:"portal_username"`
}

// PasswordResetRequest is the request body for POST /launcher/v1/password-reset.
type PasswordResetRequest struct {
	UserName string `json:"user_name"`
}

// PasswordResetResponse is the response from POST /launcher/v1/password-reset.
type PasswordResetResponse struct {
	RequestID string `json:"request_id"`
	TextValue string `json:"text_value"`
}

// BotNameUpsertRequest is the request body for
// PUT /launcher/v1/supporter/bot-names/{slot}. The slot index is in the path.
type BotNameUpsertRequest struct {
	BotName string `json:"bot_name"`
}
