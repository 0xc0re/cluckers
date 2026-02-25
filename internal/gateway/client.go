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

// Client is an HTTP client for the Project Crown gateway API.
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

// Post sends a JSON POST to the gateway at /json/<command>, marshalling body
// and unmarshalling the response into result.
func (c *Client) Post(ctx context.Context, command string, body interface{}, result interface{}) error {
	return c.postInternal(ctx, command, body, result, nil)
}

// PostWithRaw sends a JSON POST like Post, but also returns the raw response
// body via rawOut (if non-nil). This allows callers to inspect the exact server
// response for debugging purposes, even when the typed result unmarshals to
// zero values.
func (c *Client) PostWithRaw(ctx context.Context, command string, body interface{}, result interface{}, rawOut *[]byte) error {
	return c.postInternal(ctx, command, body, result, rawOut)
}

func (c *Client) postInternal(ctx context.Context, command string, body interface{}, result interface{}, rawOut *[]byte) error {
	url := fmt.Sprintf("%s/json/%s", c.baseURL, command)

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CluckersCentral/1.1.68")
	req.Header.Set("Accept", "*/*")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &ui.UserError{
			Message:    "Could not connect to gateway.",
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

	// Provide raw response to caller if requested.
	if rawOut != nil {
		*rawOut = respBody
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Check for HTML error page (Cloudflare, nginx, etc.)
		if strings.Contains(strings.ToLower(string(respBody)), "<html") {
			return &ui.UserError{
				Message:    "Gateway unreachable (received HTML error page).",
				Detail:     fmt.Sprintf("HTTP %d from %s", resp.StatusCode, url),
				Suggestion: "The Project Crown servers may be down. Try again later.",
			}
		}
		return &ui.UserError{
			Message: fmt.Sprintf("Gateway error (HTTP %d).", resp.StatusCode),
			Detail:  string(respBody),
		}
	}

	if c.verbose {
		ui.Verbose(fmt.Sprintf("Gateway %s request: %s", command, string(payload)), true)
		ui.Verbose(fmt.Sprintf("Gateway %s response: %s", command, string(respBody)), true)
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

// HealthCheck calls LAUNCHER_HEALTH and returns an error if the gateway is not healthy.
func (c *Client) HealthCheck(ctx context.Context) error {
	var resp HealthResponse
	err := c.Post(ctx, "LAUNCHER_HEALTH", map[string]string{
		"user_name": "health",
	}, &resp)
	if err != nil {
		return &ui.UserError{
			Message:    "Could not reach the Project Crown servers. Check your internet connection or try again later.",
			Detail:     err.Error(),
			Suggestion: "If this persists, the gateway may be down for maintenance.",
			Err:        err,
		}
	}
	if !bool(resp.Success) {
		return &ui.UserError{
			Message:    "Gateway is down.",
			Detail:     "LAUNCHER_HEALTH returned SUCCESS=false",
			Suggestion: "The Project Crown servers may be undergoing maintenance. Try again later.",
		}
	}
	return nil
}
