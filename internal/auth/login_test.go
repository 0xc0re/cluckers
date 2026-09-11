package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["pin"]; ok {
			t.Error("request body must not carry a pin field")
		}
		if r.Header.Get("Authorization") != "" {
			t.Error("login must not send an Authorization header")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user_name":               "user",
			"access_token":            "lpt_v1_abc",
			"refresh_token":           "lrt_v1_abc",
			"access_expires_at_unix":  1800000000,
			"refresh_expires_at_unix": "1800086400",
			"linked_flag":             1,
			"custom_message":          "cluck_commander",
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
	if res.RefreshToken != "lrt_v1_abc" {
		t.Errorf("refresh token = %q, want lrt_v1_abc", res.RefreshToken)
	}
	if res.AccessExpiresAt.Unix() != 1800000000 {
		t.Errorf("AccessExpiresAt = %v, want unix 1800000000", res.AccessExpiresAt)
	}
	if res.RefreshExpiresAt.Unix() != 1800086400 {
		t.Errorf("RefreshExpiresAt = %v, want unix 1800086400", res.RefreshExpiresAt)
	}
	if !res.Linked {
		t.Error("expected Linked = true")
	}
	if res.SupporterTier != "cluck_commander" {
		t.Errorf("SupporterTier = %q, want cluck_commander", res.SupporterTier)
	}
}

// TestLogin_PinRequired verifies the developer-only-mode gate is surfaced as
// ErrPinRequired rather than a generic "no token" failure.
func TestLogin_PinRequired(t *testing.T) {
	for _, text := range []string{"PIN_REQUIRED", "PIN_INVALID"} {
		srv := newJSONServer(t, http.StatusOK, map[string]interface{}{
			"user_name":   "user",
			"linked_flag": 1,
			"text_value":  text,
		})
		client := gateway.NewClient(srv.URL, false)
		_, err := Login(context.Background(), client, "user", "pass")
		srv.Close()
		if !errors.Is(err, ErrPinRequired) {
			t.Errorf("text_value=%s: expected ErrPinRequired, got %v", text, err)
		}
		var ue *ui.UserError
		if !errors.As(err, &ue) || ue.Suggestion == "" {
			t.Errorf("text_value=%s: expected a UserError with a suggestion, got %v", text, err)
		}
	}
}

// TestLogin_NotLinked verifies an unlinked account returns the link code that
// the server smuggles in access_token.
func TestLogin_NotLinked(t *testing.T) {
	srv := newJSONServer(t, http.StatusOK, map[string]interface{}{
		"user_name":    "user",
		"access_token": " LINK99 ",
		"linked_flag":  0,
	})
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	res, err := Login(context.Background(), client, "user", "pass")
	if res != nil {
		t.Errorf("expected nil result, got %+v", res)
	}
	if !errors.Is(err, ErrNotLinked) {
		t.Fatalf("expected ErrNotLinked, got %v", err)
	}
	var nl *NotLinkedError
	if !errors.As(err, &nl) || nl.LinkCode != "LINK99" {
		t.Errorf("expected NotLinkedError with code LINK99, got %v", err)
	}
}

// TestLogin_NoToken verifies a 2xx without a token (and no known sentinel)
// still fails, with the server text preserved for -v.
func TestLogin_NoToken(t *testing.T) {
	srv := newJSONServer(t, http.StatusOK, map[string]interface{}{
		"user_name":      "user",
		"linked_flag":    1,
		"text_value":     "Something odd",
		"custom_message": "extra",
	})
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	_, err := Login(context.Background(), client, "user", "pass")
	var ue *ui.UserError
	if !errors.As(err, &ue) {
		t.Fatalf("expected *ui.UserError, got %T: %v", err, err)
	}
	if !strings.Contains(ue.Message, "no access token") {
		t.Errorf("Message = %q, want no-access-token message", ue.Message)
	}
	if !strings.Contains(ue.Detail, "Something odd") || !strings.Contains(ue.Detail, "extra") {
		t.Errorf("Detail = %q, want server text preserved", ue.Detail)
	}
	if errors.Is(err, ErrNotLinked) || errors.Is(err, ErrPinRequired) {
		t.Errorf("unexpected sentinel in %v", err)
	}
}

func TestExpiryFrom(t *testing.T) {
	cases := []struct {
		name string
		unix json.Number
		dt   string
		want time.Time
	}{
		{"unix wins", "1800000000", "2020-01-01T00:00:00Z", time.Unix(1800000000, 0)},
		{"rfc3339 fallback", "", "2020-01-02T03:04:05Z", time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)},
		{"naive datetime", "", "2020-01-02T03:04:05", time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)},
		{"nothing", "", "", time.Time{}},
		{"garbage", "abc", "soon", time.Time{}},
		{"zero unix ignored", "0", "", time.Time{}},
	}
	for _, c := range cases {
		got := expiryFrom(c.unix, c.dt)
		if !got.Equal(c.want) {
			t.Errorf("%s: expiryFrom(%q, %q) = %v, want %v", c.name, c.unix, c.dt, got, c.want)
		}
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
