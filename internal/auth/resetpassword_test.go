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

func TestRequestPasswordReset_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/launcher/v1/password-reset" {
			t.Errorf("path = %q, want /launcher/v1/password-reset", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"request_id":   "r1",
			"access_token": "RESET42",
			"text_value":   "If the account exists, DM this code to the Discord bot, then reply with your new password.",
		})
	}))
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	res, err := RequestPasswordReset(context.Background(), client, "testuser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.RequestID != "r1" {
		t.Errorf("RequestID = %q, want r1", res.RequestID)
	}
	if res.Code != "RESET42" {
		t.Errorf("Code = %q, want RESET42", res.Code)
	}
	if !strings.Contains(res.Message, "DM this code") {
		t.Errorf("Message = %q, want the server instructions", res.Message)
	}
}

func TestRequestPasswordReset_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"detail": "User not found",
			"title":  "not_found",
			"status": 404,
		})
	}))
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	_, err := RequestPasswordReset(context.Background(), client, "baduser")
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
