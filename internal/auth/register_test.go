package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xc0re/cluckers/internal/gateway"
	"github.com/0xc0re/cluckers/internal/ui"
)

// newJSONServer returns a server that replies with the given status and body.
func newJSONServer(t *testing.T, status int, resp map[string]interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if status != 0 {
			w.WriteHeader(status)
		}
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestRegister_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/launcher/v1/account" {
			t.Errorf("path = %q, want /launcher/v1/account", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user_name":     "newuser",
			"access_token":  "lpt_v1_new",
			"refresh_token": "lrt_new",
			"linked_flag":   1,
		})
	}))
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	res, err := Register(context.Background(), client, "newuser", "pass", "e@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.AccessToken != "lpt_v1_new" {
		t.Errorf("access token = %q, want lpt_v1_new", res.AccessToken)
	}
	if res.RefreshToken != "lrt_new" {
		t.Errorf("refresh token = %q, want lrt_new", res.RefreshToken)
	}
}

// TestRegister_NotLinked verifies a fresh account (linked_flag 0) surfaces the
// link code carried in access_token instead of treating it as a session.
func TestRegister_NotLinked(t *testing.T) {
	srv := newJSONServer(t, http.StatusOK, map[string]interface{}{
		"user_name":    "newuser",
		"access_token": "ABC123",
		"linked_flag":  0,
		"text_value":   "DM the code to the bot",
	})
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	_, err := Register(context.Background(), client, "newuser", "pass", "e@example.com")
	if !errors.Is(err, ErrNotLinked) {
		t.Fatalf("expected ErrNotLinked, got %v", err)
	}
	var nl *NotLinkedError
	if !errors.As(err, &nl) {
		t.Fatalf("expected *NotLinkedError in chain, got %T", err)
	}
	if nl.LinkCode != "ABC123" {
		t.Errorf("LinkCode = %q, want ABC123", nl.LinkCode)
	}
	var ue *ui.UserError
	if !errors.As(err, &ue) || !strings.Contains(ue.Message, "not linked") {
		t.Errorf("expected a UserError mentioning not linked, got %v", err)
	}
}
