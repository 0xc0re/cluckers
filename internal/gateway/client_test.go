package gateway

import (
	"context"
	"encoding/json"
	"errors"
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

func TestSanitizeJSON(t *testing.T) {
	in := []byte(`{"user_name":"alice","password":"hunter2","access_token":"lpt_v1_abc"}`)
	out := sanitizeJSON(in)
	if strings.Contains(out, "hunter2") || strings.Contains(out, "lpt_v1_abc") {
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
