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

func TestRequestLinkCode_Success(t *testing.T) {
	srv := newJSONServer(t, http.StatusOK, map[string]interface{}{
		"code":         "ABC123",
		"access_token": "lpt_v1_x",
	})
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	code, err := RequestLinkCode(context.Background(), client, "user", "pass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != "ABC123" {
		t.Errorf("code = %q, want %q", code, "ABC123")
	}
}

func TestRequestLinkCode_Failure(t *testing.T) {
	srv := newJSONServer(t, http.StatusUnauthorized, map[string]interface{}{
		"detail": "Account not verified",
		"title":  "not_verified",
		"status": 401,
	})
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	_, err := RequestLinkCode(context.Background(), client, "user", "pass")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ue *ui.UserError
	if !errors.As(err, &ue) {
		t.Fatalf("expected *ui.UserError, got %T: %v", err, err)
	}
	if !strings.Contains(ue.Message, "Account not verified") {
		t.Errorf("message = %q, want it to contain %q", ue.Message, "Account not verified")
	}
}

func TestRequestLinkCode_EmptyCode(t *testing.T) {
	srv := newJSONServer(t, http.StatusOK, map[string]interface{}{
		"code": "",
	})
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	_, err := RequestLinkCode(context.Background(), client, "user", "pass")
	if err == nil {
		t.Fatal("expected error for empty code, got nil")
	}

	var ue *ui.UserError
	if !errors.As(err, &ue) {
		t.Fatalf("expected *ui.UserError, got %T: %v", err, err)
	}
	if !strings.Contains(ue.Message, "Link code response was empty") {
		t.Errorf("message = %q, want it to contain %q", ue.Message, "Link code response was empty")
	}
}

func TestRegister_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/launcher/v1/account" {
			t.Errorf("path = %q, want /launcher/v1/account", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user_name":    "newuser",
			"access_token": "lpt_v1_new",
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
}

func TestCheckDiscordStatus_Linked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("Authorization = %q, want Bearer tok", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"linked_flag": 1})
	}))
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	linked, err := CheckDiscordStatus(context.Background(), client, "user", "tok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !linked {
		t.Error("expected linked = true")
	}
}
