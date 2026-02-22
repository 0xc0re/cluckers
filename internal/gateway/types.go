package gateway

import (
	"encoding/json"
	"strconv"
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
		parsed, _ := strconv.ParseBool(v)
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
	PortalInfo1 string `json:"PORTAL_INFO_1"`
	StringValue string `json:"STRING_VALUE"`
	TextValue   string `json:"TEXT_VALUE"`
}

// BootstrapResponse is the response from LAUNCHER_CONTENT_BOOTSTRAP.
type BootstrapResponse struct {
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
