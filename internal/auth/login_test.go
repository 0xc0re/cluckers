package auth

import (
	"context"
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

// TestLogin_MissingLinkedFlagWithRealToken guards against showing a session
// token as a link code when the server omits linked_flag.
func TestLogin_MissingLinkedFlagWithRealToken(t *testing.T) {
	srv := newJSONServer(t, http.StatusOK, map[string]interface{}{"user_name": "user", "access_token": "lpt_v1_real"})
	defer srv.Close()
	res, err := Login(context.Background(), gateway.NewClient(srv.URL, false), "user", "pass")
	if err != nil || res.AccessToken != "lpt_v1_real" {
		t.Fatalf("res = %+v, err = %v; want the token accepted as a session", res, err)
	}
}
