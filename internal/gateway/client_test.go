package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xc0re/cluckers/internal/ui"
)

func TestDoSuccessDecodesResult(t *testing.T) {
	var gotUserAgent, gotAccept, gotContentType string
	var gotBody map[string]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserAgent = r.Header.Get("User-Agent")
		gotAccept = r.Header.Get("Accept")
		gotContentType = r.Header.Get("Content-Type")
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","version":"1.2"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, false)
	var resp HealthResponse
	err := c.Do(context.Background(), http.MethodPost, "/healthz", "", map[string]string{"user_name": "alice"}, &resp)
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("decoded status = %q, want %q", resp.Status, "ok")
	}
	if gotUserAgent != UserAgent {
		t.Errorf("User-Agent = %q, want %q", gotUserAgent, UserAgent)
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q, want application/json", gotAccept)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
	if gotBody["user_name"] != "alice" {
		t.Errorf("request body user_name = %q, want %q", gotBody["user_name"], "alice")
	}
}

func TestDoBearerHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, false)

	if err := c.Do(context.Background(), http.MethodGet, "/launcher/v1/content-bootstrap", "lpt_v1_secret", nil, nil); err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	if gotAuth != "Bearer lpt_v1_secret" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer lpt_v1_secret")
	}

	if err := c.Do(context.Background(), http.MethodGet, "/healthz", "", nil, nil); err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	if gotAuth != "" {
		t.Errorf("Authorization = %q, want empty when no bearer given", gotAuth)
	}
}

func TestDoProblemJSONError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"detail":"Invalid username or password.","title":"Unauthorized","status":401}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, false)
	err := c.Do(context.Background(), http.MethodPost, "/launcher/v1/session", "", map[string]string{"user_name": "alice"}, nil)
	if err == nil {
		t.Fatal("Do returned nil error for HTTP 401")
	}

	var ue *ui.UserError
	if !errors.As(err, &ue) {
		t.Fatalf("error type = %T, want *ui.UserError", err)
	}
	if ue.Message != "Invalid username or password." {
		t.Errorf("Message = %q, want problem detail", ue.Message)
	}
	if !strings.Contains(ue.Detail, "401") || !strings.Contains(ue.Detail, "Unauthorized") {
		t.Errorf("Detail = %q, want it to mention status 401 and title", ue.Detail)
	}
	if ue.Status != http.StatusUnauthorized {
		t.Errorf("Status = %d, want 401", ue.Status)
	}
	if ue.Code != "Unauthorized" {
		t.Errorf("Code = %q, want problem title", ue.Code)
	}
	if !ue.IsStatus(http.StatusUnauthorized, http.StatusForbidden) {
		t.Error("IsStatus(401, 403) = false, want true")
	}
	if ue.IsStatus(http.StatusForbidden) {
		t.Error("IsStatus(403) = true, want false")
	}
}

func TestDoHTMLErrorPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 403 (not retried by retryablehttp) with a Cloudflare-style HTML page.
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("<html><body>error</body></html>"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, false)
	err := c.Do(context.Background(), http.MethodGet, "/healthz", "", nil, nil)
	if err == nil {
		t.Fatal("Do returned nil error for HTML error page")
	}

	var ue *ui.UserError
	if !errors.As(err, &ue) {
		t.Fatalf("error type = %T, want *ui.UserError", err)
	}
	if !strings.Contains(ue.Message, "Gateway unreachable") {
		t.Errorf("Message = %q, want gateway-unreachable message", ue.Message)
	}
	if ue.Status != http.StatusForbidden {
		t.Errorf("Status = %d, want 403", ue.Status)
	}
}

func TestDoMalformedJSONResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, false)
	var resp HealthResponse
	err := c.Do(context.Background(), http.MethodGet, "/healthz", "", nil, &resp)
	if err == nil {
		t.Fatal("Do returned nil error for malformed JSON body")
	}
	var ue *ui.UserError
	if !errors.As(err, &ue) {
		t.Fatalf("error type = %T, want *ui.UserError", err)
	}
}

func TestHealthCheck(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/healthz" {
				t.Errorf("path = %q, want /healthz", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		}))
		defer srv.Close()

		c := NewClient(srv.URL, false)
		if err := c.HealthCheck(context.Background()); err != nil {
			t.Errorf("HealthCheck returned error: %v", err)
		}
	})

	t.Run("degraded", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"degraded"}`))
		}))
		defer srv.Close()

		c := NewClient(srv.URL, false)
		err := c.HealthCheck(context.Background())
		if err == nil {
			t.Fatal("HealthCheck returned nil for non-ok status")
		}
		var ue *ui.UserError
		if !errors.As(err, &ue) {
			t.Fatalf("error type = %T, want *ui.UserError", err)
		}
	})
}

func TestDoWithHeadersSendsHeaderAndEmptyObject(t *testing.T) {
	var gotBuild, gotContentType string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBuild = r.Header.Get("x-realm-client-build")
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, false)
	err := c.DoWithHeaders(context.Background(), http.MethodPost, "/launcher/v1/launch-auth", "tok",
		map[string]string{"x-realm-client-build": "0.39.6969.0"}, struct{}{}, nil)
	if err != nil {
		t.Fatalf("DoWithHeaders returned error: %v", err)
	}
	if gotBuild != "0.39.6969.0" {
		t.Errorf("x-realm-client-build = %q, want 0.39.6969.0", gotBuild)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
	if strings.TrimSpace(string(gotBody)) != "{}" {
		t.Errorf("body = %q, want {}", gotBody)
	}

	// No headers map: header absent.
	gotBuild = "unset"
	if err := c.Do(context.Background(), http.MethodPost, "/x", "", struct{}{}, nil); err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	if gotBuild != "" {
		t.Errorf("x-realm-client-build = %q, want empty when no headers given", gotBuild)
	}
}

func TestSessionResponseDecodesNewFields(t *testing.T) {
	for _, body := range []string{
		`{"access_token":"a","refresh_token":"r","access_expires_at_unix":1700000000,"refresh_expires_at_unix":1700003600,"linked_flag":1}`,
		`{"access_token":"a","refresh_token":"r","access_expires_at_unix":"1700000000","refresh_expires_at_unix":"1700003600","linked_flag":"1"}`,
	} {
		var resp SessionResponse
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			t.Fatalf("unmarshal %s: %v", body, err)
		}
		if resp.RefreshToken != "r" {
			t.Errorf("RefreshToken = %q, want r", resp.RefreshToken)
		}
		if v, err := resp.AccessExpiresAtUnix.Int64(); err != nil || v != 1700000000 {
			t.Errorf("AccessExpiresAtUnix = %v (%v), want 1700000000", resp.AccessExpiresAtUnix, err)
		}
		if v, err := resp.RefreshExpiresAtUnix.Int64(); err != nil || v != 1700003600 {
			t.Errorf("RefreshExpiresAtUnix = %v (%v), want 1700003600", resp.RefreshExpiresAtUnix, err)
		}
		if !bool(resp.LinkedFlag) {
			t.Error("LinkedFlag = false, want true")
		}
	}
}

func TestSanitizeJSON(t *testing.T) {
	in := []byte(`{"user_name":"alice","password":"hunter2","access_token":"lpt_v1_abc","refresh_token":"lrt_1","launch_token":"lt_1"}`)
	out := sanitizeJSON(in)
	if strings.Contains(out, "hunter2") || strings.Contains(out, "lpt_v1_abc") || strings.Contains(out, "lrt_1") || strings.Contains(out, "lt_1") {
		t.Errorf("sanitizeJSON leaked sensitive values: %s", out)
	}
	if !strings.Contains(out, "alice") {
		t.Errorf("sanitizeJSON dropped non-sensitive value: %s", out)
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Errorf("sanitizeJSON did not redact: %s", out)
	}

	// Non-JSON input passes through unchanged.
	if got := sanitizeJSON([]byte("not json")); got != "not json" {
		t.Errorf("sanitizeJSON(non-JSON) = %q, want passthrough", got)
	}
}
