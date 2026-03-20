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

func newPasswordResetServer(t *testing.T, resp map[string]interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestRequestPasswordReset_Success(t *testing.T) {
	srv := newPasswordResetServer(t, map[string]interface{}{
		"SUCCESS": 1,
	})
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	err := RequestPasswordReset(context.Background(), client, "testuser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequestPasswordReset_FailureWithTextValue(t *testing.T) {
	srv := newPasswordResetServer(t, map[string]interface{}{
		"SUCCESS":    0,
		"TEXT_VALUE": "User not found",
	})
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	err := RequestPasswordReset(context.Background(), client, "baduser")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ue *ui.UserError
	if !errors.As(err, &ue) {
		t.Fatalf("expected *ui.UserError, got %T: %v", err, err)
	}
	if !strings.Contains(ue.Message, "User not found") {
		t.Errorf("message = %q, want it to contain %q", ue.Message, "User not found")
	}
}

func TestRequestPasswordReset_FailureWithStringValue(t *testing.T) {
	srv := newPasswordResetServer(t, map[string]interface{}{
		"SUCCESS":      0,
		"STRING_VALUE": "Rate limited",
	})
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	err := RequestPasswordReset(context.Background(), client, "testuser")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ue *ui.UserError
	if !errors.As(err, &ue) {
		t.Fatalf("expected *ui.UserError, got %T: %v", err, err)
	}
	if !strings.Contains(ue.Message, "Rate limited") {
		t.Errorf("message = %q, want it to contain %q", ue.Message, "Rate limited")
	}
}

func TestRequestPasswordReset_FailureNoMessage(t *testing.T) {
	srv := newPasswordResetServer(t, map[string]interface{}{
		"SUCCESS": 0,
	})
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	err := RequestPasswordReset(context.Background(), client, "testuser")
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
