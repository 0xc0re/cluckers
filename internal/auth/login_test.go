package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0xc0re/cluckers/internal/gateway"
	"github.com/0xc0re/cluckers/internal/ui"
)

// bootstrapPayload is a test payload mimicking the real BPS1 magic header.
// It includes bytes 0xFF/0xFE which produce +/ in standard base64 and -_ in
// URL-safe base64, ensuring the two encodings are actually different.
var bootstrapPayload = append(
	[]byte("BPS1"),
	append([]byte{0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA}, // force +/ vs -_
		bytes.Repeat([]byte("x"), 126)...)...,
) // 136 bytes total

// newBootstrapServer returns an httptest.Server that responds to
// GET /launcher/v1/content-bootstrap with the given base64-encoded value in
// portal_info_1. If encoded is empty, portal_info_1 is omitted. It also
// verifies the request carries the Bearer token.
func newBootstrapServer(t *testing.T, encoded string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer token")
		}
		resp := map[string]interface{}{
			"session_id": "abc",
			"version":    1,
		}
		if encoded != "" {
			resp["portal_info_1"] = encoded
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestGetContentBootstrap_StandardBase64(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString(bootstrapPayload)
	srv := newBootstrapServer(t, encoded)
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	data, err := GetContentBootstrap(context.Background(), client, "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != string(bootstrapPayload) {
		t.Errorf("payload mismatch: got %d bytes, want %d bytes", len(data), len(bootstrapPayload))
	}
}

func TestGetContentBootstrap_URLSafeBase64(t *testing.T) {
	encoded := base64.URLEncoding.EncodeToString(bootstrapPayload)
	srv := newBootstrapServer(t, encoded)
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	data, err := GetContentBootstrap(context.Background(), client, "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != string(bootstrapPayload) {
		t.Errorf("payload mismatch: got %d bytes, want %d bytes", len(data), len(bootstrapPayload))
	}
}

func TestGetContentBootstrap_RawUnpadded(t *testing.T) {
	encoded := base64.RawStdEncoding.EncodeToString(bootstrapPayload)
	srv := newBootstrapServer(t, encoded)
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	data, err := GetContentBootstrap(context.Background(), client, "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != string(bootstrapPayload) {
		t.Errorf("payload mismatch: got %d bytes, want %d bytes", len(data), len(bootstrapPayload))
	}
}

func TestGetContentBootstrap_RawURLSafe(t *testing.T) {
	encoded := base64.RawURLEncoding.EncodeToString(bootstrapPayload)
	srv := newBootstrapServer(t, encoded)
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	data, err := GetContentBootstrap(context.Background(), client, "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != string(bootstrapPayload) {
		t.Errorf("payload mismatch: got %d bytes, want %d bytes", len(data), len(bootstrapPayload))
	}
}

func TestGetContentBootstrap_EmptyResponse(t *testing.T) {
	srv := newBootstrapServer(t, "")
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	data, err := GetContentBootstrap(context.Background(), client, "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data != nil {
		t.Errorf("expected nil data for empty response, got %d bytes", len(data))
	}
}

func TestGetContentBootstrap_InvalidBase64(t *testing.T) {
	srv := newBootstrapServer(t, "!!!not-base64-at-all!!!")
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	_, err := GetContentBootstrap(context.Background(), client, "token")
	if err == nil {
		t.Fatal("expected error for invalid base64, got nil")
	}

	var ue *ui.UserError
	if !errors.As(err, &ue) {
		t.Fatalf("expected *ui.UserError, got %T: %v", err, err)
	}
	if ue.Message != "Failed to decode content bootstrap" {
		t.Errorf("message = %q, want %q", ue.Message, "Failed to decode content bootstrap")
	}
	if ue.Suggestion == "" {
		t.Error("expected non-empty Suggestion for decode error")
	}
}

func TestGetContentBootstrap_Whitespace(t *testing.T) {
	// Encode with standard base64 and insert whitespace/newlines.
	encoded := base64.StdEncoding.EncodeToString(bootstrapPayload)
	// Insert newlines and spaces (simulating line-wrapped responses).
	withWhitespace := " " + encoded[:20] + "\n" + encoded[20:40] + "\r\n" + encoded[40:] + " "

	srv := newBootstrapServer(t, withWhitespace)
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	data, err := GetContentBootstrap(context.Background(), client, "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != string(bootstrapPayload) {
		t.Errorf("payload mismatch: got %d bytes, want %d bytes", len(data), len(bootstrapPayload))
	}
}

// TestGetContentBootstrap_TokenRejected verifies that an HTTP 401 from the
// gateway is surfaced as ErrTokenRejected so the pipeline can re-authenticate.
func TestGetContentBootstrap_TokenRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"detail": "Invalid or expired access token",
			"title":  "invalid_token",
			"status": 401,
		})
	}))
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	data, err := GetContentBootstrap(context.Background(), client, "stale-token")
	if data != nil {
		t.Errorf("expected nil data on rejection, got %d bytes", len(data))
	}
	if err == nil {
		t.Fatal("expected error for rejected token, got nil")
	}
	if !errors.Is(err, ErrTokenRejected) {
		t.Errorf("expected ErrTokenRejected, got %v", err)
	}
}

// TestLogin_Success verifies a 200 session-or-link response yields the token.
func TestLogin_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/launcher/v1/session-or-link" {
			t.Errorf("path = %q, want /launcher/v1/session-or-link", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user_name":    "user",
			"access_token": "lpt_v1_abc",
			"linked_flag":  1,
		})
	}))
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	res, err := Login(context.Background(), client, "user", "pass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.AccessToken != "lpt_v1_abc" {
		t.Errorf("access token = %q, want lpt_v1_abc", res.AccessToken)
	}
	if !res.Linked {
		t.Error("expected Linked = true")
	}
}

// TestLogin_BadCredentials verifies a 401 problem+json is surfaced to the user.
func TestLogin_BadCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"detail": "The supplied credentials did not match",
			"title":  "invalid_credentials",
			"status": 401,
		})
	}))
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	_, err := Login(context.Background(), client, "user", "wrong")
	if err == nil {
		t.Fatal("expected error for bad credentials, got nil")
	}
	var ue *ui.UserError
	if !errors.As(err, &ue) {
		t.Fatalf("expected *ui.UserError, got %T", err)
	}
	if ue.Message != "The supplied credentials did not match" {
		t.Errorf("message = %q, want the problem detail", ue.Message)
	}
}
