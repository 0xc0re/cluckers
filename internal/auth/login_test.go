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
// LAUNCHER_CONTENT_BOOTSTRAP with the given base64-encoded value in
// PORTAL_INFO_1. If encoded is empty, PORTAL_INFO_1 is omitted.
func newBootstrapServer(t *testing.T, encoded string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"SESSION_ID": 1,
			"VERSION":    1,
		}
		if encoded != "" {
			resp["PORTAL_INFO_1"] = encoded
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
	data, err := GetContentBootstrap(context.Background(), client, "user", "token")
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
	data, err := GetContentBootstrap(context.Background(), client, "user", "token")
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
	data, err := GetContentBootstrap(context.Background(), client, "user", "token")
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
	data, err := GetContentBootstrap(context.Background(), client, "user", "token")
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
	data, err := GetContentBootstrap(context.Background(), client, "user", "token")
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
	_, err := GetContentBootstrap(context.Background(), client, "user", "token")
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
	data, err := GetContentBootstrap(context.Background(), client, "user", "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != string(bootstrapPayload) {
		t.Errorf("payload mismatch: got %d bytes, want %d bytes", len(data), len(bootstrapPayload))
	}
}
