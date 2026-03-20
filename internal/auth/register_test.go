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

func newLinkCodeServer(t *testing.T, resp map[string]interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestRequestLinkCode_Success(t *testing.T) {
	srv := newLinkCodeServer(t, map[string]interface{}{
		"SUCCESS":      1,
		"ACCESS_TOKEN": "ABC123",
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

func TestRequestLinkCode_FailureWithTextValue(t *testing.T) {
	srv := newLinkCodeServer(t, map[string]interface{}{
		"SUCCESS":    0,
		"TEXT_VALUE": "Account not verified",
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

func TestRequestLinkCode_FailureWithStringValue(t *testing.T) {
	srv := newLinkCodeServer(t, map[string]interface{}{
		"SUCCESS":      0,
		"STRING_VALUE": "Token expired",
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
	if !strings.Contains(ue.Message, "Token expired") {
		t.Errorf("message = %q, want it to contain %q", ue.Message, "Token expired")
	}
}

func TestRequestLinkCode_FailureNoMessage(t *testing.T) {
	srv := newLinkCodeServer(t, map[string]interface{}{
		"SUCCESS": 0,
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
	if !strings.Contains(ue.Message, "Unknown error") {
		t.Errorf("message = %q, want it to contain %q", ue.Message, "Unknown error")
	}
}

func TestRequestLinkCode_EmptyCode(t *testing.T) {
	srv := newLinkCodeServer(t, map[string]interface{}{
		"SUCCESS":      1,
		"ACCESS_TOKEN": "",
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
