package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/0xc0re/cluckers/internal/ui"
	"github.com/hashicorp/go-retryablehttp"
)

// UserAgent identifies the launcher to the gateway. Kept in sync with the
// official Project Crown launcher version that the API expects.
const UserAgent = "CluckersCentral/1.2.54"

// sensitiveKeys are JSON field names whose values are redacted in verbose logs.
var sensitiveKeys = map[string]bool{
	"password":      true,
	"access_token":  true,
	"portal_info_1": true,
	"text_value":    true,
}

// sanitizeJSON redacts sensitive field values in a JSON payload for safe logging.
// Returns the original string if the payload is not valid JSON.
func sanitizeJSON(data []byte) string {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return string(data)
	}
	for key := range obj {
		if sensitiveKeys[key] {
			obj[key] = "[REDACTED]"
		}
	}
	out, err := json.Marshal(obj)
	if err != nil {
		return string(data)
	}
	return string(out)
}

// Client is an HTTP client for the Project Crown launcher REST API.
type Client struct {
	httpClient *retryablehttp.Client
	baseURL    string
	verbose    bool
}

// NewClient creates a new gateway client with retry and backoff configured.
func NewClient(baseURL string, verbose bool) *Client {
	rc := retryablehttp.NewClient()
	rc.RetryMax = 3
	rc.RetryWaitMin = 500 * time.Millisecond
	rc.RetryWaitMax = 5 * time.Second
	rc.Logger = nil // Suppress default retryablehttp logging.
	rc.HTTPClient.Timeout = 15 * time.Second

	return &Client{
		httpClient: rc,
		baseURL:    strings.TrimRight(baseURL, "/"),
		verbose:    verbose,
	}
}

// problemDetails is the RFC 7807 problem+json error body returned by the v1 API.
type problemDetails struct {
	Detail string `json:"detail"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Type   string `json:"type"`
}

// Do sends an HTTP request to path (e.g. "/launcher/v1/session"), optionally with
// a Bearer token and a JSON request body, decoding a 2xx JSON response into result.
//
// Success is indicated by a 2xx status code (there is no SUCCESS field). Non-2xx
// responses are parsed as RFC 7807 problem+json and returned as *ui.UserError.
// A nil body sends no request payload; a nil result skips response decoding.
func (c *Client) Do(ctx context.Context, method, path, bearer string, body, result interface{}) error {
	url := c.baseURL + path

	var reader io.Reader
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := retryablehttp.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &ui.UserError{
			Message:    "Could not connect to the Project Crown servers.",
			Detail:     err.Error(),
			Suggestion: "Check your internet connection or try again later.",
			Err:        err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ui.UserError{
			Message: "Failed to read gateway response.",
			Detail:  err.Error(),
			Err:     err,
		}
	}

	if c.verbose {
		if payload != nil {
			ui.Verbose(fmt.Sprintf("Gateway %s %s request: %s", method, path, sanitizeJSON(payload)), true)
		} else {
			ui.Verbose(fmt.Sprintf("Gateway %s %s request", method, path), true)
		}
		ui.Verbose(fmt.Sprintf("Gateway %s %s response (HTTP %d): %s", method, path, resp.StatusCode, sanitizeJSON(respBody)), true)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.errorFromResponse(method, path, resp.StatusCode, respBody)
	}

	if result == nil || len(respBody) == 0 {
		return nil
	}

	if err := json.Unmarshal(respBody, result); err != nil {
		return &ui.UserError{
			Message: "Failed to parse gateway response.",
			Detail:  fmt.Sprintf("JSON unmarshal error: %s; body: %s", err.Error(), string(respBody)),
			Err:     err,
		}
	}

	return nil
}

// errorFromResponse builds a *ui.UserError from a non-2xx response, parsing the
// RFC 7807 problem+json body when present.
func (c *Client) errorFromResponse(method, path string, status int, body []byte) error {
	// Cloudflare / nginx HTML error page (gateway down, not the app).
	if strings.Contains(strings.ToLower(string(body)), "<html") {
		return &ui.UserError{
			Message:    "Gateway unreachable (received HTML error page).",
			Detail:     fmt.Sprintf("HTTP %d from %s %s", status, method, path),
			Suggestion: "The Project Crown servers may be down. Try again later.",
		}
	}

	var pd problemDetails
	if err := json.Unmarshal(body, &pd); err == nil && (pd.Detail != "" || pd.Title != "") {
		msg := pd.Detail
		if msg == "" {
			msg = pd.Title
		}
		return &ui.UserError{
			Message: msg,
			Detail:  fmt.Sprintf("HTTP %d (%s)", status, pd.Title),
		}
	}

	return &ui.UserError{
		Message: fmt.Sprintf("Gateway error (HTTP %d).", status),
		Detail:  string(body),
	}
}

// HealthCheck calls GET /healthz and returns an error if the gateway is not healthy.
func (c *Client) HealthCheck(ctx context.Context) error {
	var resp HealthResponse
	if err := c.Do(ctx, http.MethodGet, "/healthz", "", nil, &resp); err != nil {
		return &ui.UserError{
			Message:    "Could not reach the Project Crown servers. Check your internet connection or try again later.",
			Detail:     err.Error(),
			Suggestion: "If this persists, the gateway may be down for maintenance.",
			Err:        err,
		}
	}
	if !strings.EqualFold(resp.Status, "ok") {
		return &ui.UserError{
			Message:    "Gateway is down.",
			Detail:     fmt.Sprintf("/healthz returned status=%q", resp.Status),
			Suggestion: "The Project Crown servers may be undergoing maintenance. Try again later.",
		}
	}
	return nil
}
