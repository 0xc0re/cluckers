package gateway

// HealthResponse is the response from LAUNCHER_HEALTH.
type HealthResponse struct {
	Success bool `json:"SUCCESS"`
}

// LoginRequest is the request body for LAUNCHER_LOGIN_OR_LINK.
type LoginRequest struct {
	UserName string `json:"user_name"`
	Password string `json:"password"`
}

// LoginResponse is the response from LAUNCHER_LOGIN_OR_LINK.
type LoginResponse struct {
	Success     bool   `json:"SUCCESS"`
	AccessToken string `json:"ACCESS_TOKEN"`
	TextValue   string `json:"TEXT_VALUE"`
	LinkedFlag  bool   `json:"LINKED_FLAG"`
	PortalInfo1 string `json:"PORTAL_INFO_1"`
}

// OIDCTokenResponse is the response from LAUNCHER_EAC_OIDC_TOKEN.
type OIDCTokenResponse struct {
	PortalInfo1 string `json:"PORTAL_INFO_1"`
	StringValue string `json:"STRING_VALUE"`
	TextValue   string `json:"TEXT_VALUE"`
}

// BootstrapResponse is the response from LAUNCHER_CONTENT_BOOTSTRAP.
type BootstrapResponse struct {
	PortalInfo1 string `json:"PORTAL_INFO_1"`
	StringValue string `json:"STRING_VALUE"`
	SessionID   string `json:"SESSION_ID"`
	Version     string `json:"VERSION"`
}

// GenericRequest is a common request body for authenticated API calls.
type GenericRequest struct {
	UserName    string `json:"user_name"`
	AccessToken string `json:"access_token"`
}
