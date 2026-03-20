package gateway

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/0xc0re/cluckers/internal/ui"
)

// FlexBool handles JSON booleans that may arrive as bool, number, or string.
// Some APIs return SUCCESS as 1/0 instead of true/false.
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

// HealthResponse is the response from LAUNCHER_HEALTH.
type HealthResponse struct {
	Success FlexBool `json:"SUCCESS"`
}

// LoginRequest is the request body for LAUNCHER_LOGIN_OR_LINK.
type LoginRequest struct {
	UserName string `json:"user_name"`
	Password string `json:"password"`
}

// LoginResponse is the response from LAUNCHER_LOGIN_OR_LINK.
// PORTAL_INFO_1 is omitted — it contains a cosmetics list (JSON array), not
// the content bootstrap. Bootstrap comes from LAUNCHER_CONTENT_BOOTSTRAP.
type LoginResponse struct {
	Success     FlexBool `json:"SUCCESS"`
	AccessToken string   `json:"ACCESS_TOKEN"`
	TextValue   string   `json:"TEXT_VALUE"`
	LinkedFlag  FlexBool `json:"LINKED_FLAG"`
}

// OIDCTokenResponse is the response from LAUNCHER_EAC_OIDC_TOKEN.
type OIDCTokenResponse struct {
	Success     FlexBool `json:"SUCCESS"`
	PortalInfo1 string   `json:"PORTAL_INFO_1"`
	StringValue string   `json:"STRING_VALUE"`
	TextValue   string   `json:"TEXT_VALUE"`
}

// BootstrapResponse is the response from LAUNCHER_CONTENT_BOOTSTRAP.
type BootstrapResponse struct {
	Success     FlexBool    `json:"SUCCESS"`
	PortalInfo1 string      `json:"PORTAL_INFO_1"`
	StringValue string      `json:"STRING_VALUE"`
	SessionID   json.Number `json:"SESSION_ID"`
	Version     json.Number `json:"VERSION"`
}

// GenericRequest is a common request body for authenticated API calls.
type GenericRequest struct {
	UserName    string `json:"user_name"`
	AccessToken string `json:"access_token"`
}

// BotNameUpsertRequest is the request body for LAUNCHER_SUPPORTER_BOT_NAME_UPSERT.
// Each call sets one bot name at a specific slot index (1-indexed).
type BotNameUpsertRequest struct {
	UserName     string `json:"user_name"`
	AccessToken  string `json:"access_token"`
	TextValue    string `json:"text_value"`
	CustomValue1 int    `json:"custom_value_1"`
}

// BotNameDeleteRequest is the request body for LAUNCHER_SUPPORTER_BOT_NAME_DELETE.
type BotNameDeleteRequest struct {
	UserName     string `json:"user_name"`
	AccessToken  string `json:"access_token"`
	CustomValue1 int    `json:"custom_value_1"`
}

// BotNameResponse is the response from bot name upsert/delete/list commands.
type BotNameResponse struct {
	Success     FlexBool `json:"SUCCESS"`
	TextValue   string   `json:"TEXT_VALUE"`
	StringValue string   `json:"STRING_VALUE"`
}

// RegisterRequest is the request body for LAUNCHER_REGISTER.
type RegisterRequest struct {
	UserName string `json:"user_name"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

// RegisterResponse is the response from LAUNCHER_REGISTER.
type RegisterResponse struct {
	Success     FlexBool `json:"SUCCESS"`
	AccessToken string   `json:"ACCESS_TOKEN"`
	StringValue string   `json:"STRING_VALUE"`
	TextValue   string   `json:"TEXT_VALUE"`
}

// LinkCodeRequest is the request body for LAUNCHER_REQUEST_LINK_CODE.
// This endpoint requires both password and access_token.
type LinkCodeRequest struct {
	UserName    string `json:"user_name"`
	Password    string `json:"password"`
	AccessToken string `json:"access_token"`
}

// LinkCodeResponse is the response from LAUNCHER_REQUEST_LINK_CODE.
type LinkCodeResponse struct {
	Success     FlexBool `json:"SUCCESS"`
	LinkedFlag  FlexBool `json:"LINKED_FLAG"`
	StringValue string   `json:"STRING_VALUE"`
	TextValue   string   `json:"TEXT_VALUE"`
	AccessToken string   `json:"ACCESS_TOKEN"`
}

// DiscordStatusResponse is the response from LAUNCHER_DISCORD_STATUS.
type DiscordStatusResponse struct {
	Success    FlexBool `json:"SUCCESS"`
	LinkedFlag FlexBool `json:"LINKED_FLAG"`
}

// PasswordResetRequest is the request body for LAUNCHER_REQUEST_PASSWORD_RESET.
// Only user_name is required — the server looks up the email from registration.
type PasswordResetRequest struct {
	UserName string `json:"user_name"`
}

// PasswordResetResponse is the response from LAUNCHER_REQUEST_PASSWORD_RESET.
type PasswordResetResponse struct {
	Success     FlexBool `json:"SUCCESS"`
	StringValue string   `json:"STRING_VALUE"`
	TextValue   string   `json:"TEXT_VALUE"`
}
