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
		json.NewEncoder(w).Encode(map[string]interface{}{"request_id": "r1"})
	}))
	defer srv.Close()

	client := gateway.NewClient(srv.URL, false)
	if err := RequestPasswordReset(context.Background(), client, "testuser"); err != nil {
		t.Fatalf("unexpected error: %v", err)
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
