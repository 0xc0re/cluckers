package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

type launchAuthCapture struct {
	method, path, auth, build, body string
}

// newLaunchAuthServer replies to POST /launcher/v1/launch-auth with the given
// body and records the request.
func newLaunchAuthServer(t *testing.T, status int, resp map[string]interface{}, cap *launchAuthCapture) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cap != nil {
			b, _ := io.ReadAll(r.Body)
			*cap = launchAuthCapture{r.Method, r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("x-realm-client-build"), strings.TrimSpace(string(b))}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestLaunchAuth_Success(t *testing.T) {
	var cap launchAuthCapture
	srv := newLaunchAuthServer(t, http.StatusOK, map[string]interface{}{
		"launch_token":           "lt_abc",
		"launch_expires_at_unix": 1800000000,
		"portal_info_1":          base64.StdEncoding.EncodeToString(bootstrapPayload),
	}, &cap)
	defer srv.Close()

	res, err := LaunchAuth(context.Background(), gateway.NewClient(srv.URL, false), "lpt_tok", "0.39.6969.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cap.method != http.MethodPost || cap.path != "/launcher/v1/launch-auth" {
		t.Errorf("request = %s %s, want POST /launcher/v1/launch-auth", cap.method, cap.path)
	}
	if cap.auth != "Bearer lpt_tok" {
		t.Errorf("Authorization = %q, want Bearer lpt_tok", cap.auth)
	}
	if cap.build != "0.39.6969.0" {
		t.Errorf("x-realm-client-build = %q, want 0.39.6969.0", cap.build)
	}
	if cap.body != "{}" {
		t.Errorf("body = %q, want {}", cap.body)
	}
	if res.LaunchToken != "lt_abc" {
		t.Errorf("LaunchToken = %q, want lt_abc", res.LaunchToken)
	}
	if res.LaunchExpiresAt.Unix() != 1800000000 {
		t.Errorf("LaunchExpiresAt = %v, want unix 1800000000", res.LaunchExpiresAt)
	}
	if !bytes.Equal(res.Bootstrap, bootstrapPayload) {
		t.Errorf("Bootstrap mismatch: %d bytes", len(res.Bootstrap))
	}
}

func TestLaunchAuth_NoBuildHeaderWhenEmpty(t *testing.T) {
	var cap launchAuthCapture
	srv := newLaunchAuthServer(t, http.StatusOK, map[string]interface{}{"launch_token": "lt"}, &cap)
	defer srv.Close()
	res, err := LaunchAuth(context.Background(), gateway.NewClient(srv.URL, false), "tok", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cap.build != "" {
		t.Errorf("x-realm-client-build = %q, want absent", cap.build)
	}
	if res.Bootstrap != nil {
		t.Errorf("Bootstrap = %v, want nil when portal_info_1 absent", res.Bootstrap)
	}
}

func TestLaunchAuth_EmptyLaunchToken(t *testing.T) {
	srv := newLaunchAuthServer(t, http.StatusOK, map[string]interface{}{
		"portal_info_1": base64.StdEncoding.EncodeToString(bootstrapPayload),
	}, nil)
	defer srv.Close()
	_, err := LaunchAuth(context.Background(), gateway.NewClient(srv.URL, false), "tok", "1.0")
	var ue *ui.UserError
	if !errors.As(err, &ue) || !strings.Contains(ue.Message, "empty launch token") {
		t.Fatalf("expected empty-launch-token UserError, got %v", err)
	}
	if errors.Is(err, ErrTokenRejected) {
		t.Error("empty token must not be classified as a rejected session")
	}
}

func TestLaunchAuth_TokenRejected(t *testing.T) {
	srv := newLaunchAuthServer(t, http.StatusUnauthorized, map[string]interface{}{
		"detail": "Launcher session is invalid or expired", "title": "invalid_session", "status": 401,
	}, nil)
	defer srv.Close()
	_, err := LaunchAuth(context.Background(), gateway.NewClient(srv.URL, false), "stale", "1.0")
	if !errors.Is(err, ErrTokenRejected) {
		t.Errorf("expected ErrTokenRejected, got %v", err)
	}
	var ue *ui.UserError
	if !errors.As(err, &ue) || ue.Code != "invalid_session" {
		t.Errorf("expected problem code invalid_session, got %v", err)
	}
}

func TestLaunchAuth_InvalidBootstrap(t *testing.T) {
	srv := newLaunchAuthServer(t, http.StatusOK, map[string]interface{}{
		"launch_token":  "lt",
		"portal_info_1": "!!!not-base64-at-all!!!",
	}, nil)
	defer srv.Close()
	_, err := LaunchAuth(context.Background(), gateway.NewClient(srv.URL, false), "tok", "1.0")
	var ue *ui.UserError
	if !errors.As(err, &ue) || ue.Message != "Failed to decode content bootstrap" {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestDecodeBase64Resilient(t *testing.T) {
	std := base64.StdEncoding.EncodeToString(bootstrapPayload)
	cases := map[string]string{
		"standard":        std,
		"url-safe":        base64.URLEncoding.EncodeToString(bootstrapPayload),
		"raw standard":    base64.RawStdEncoding.EncodeToString(bootstrapPayload),
		"raw url-safe":    base64.RawURLEncoding.EncodeToString(bootstrapPayload),
		"whitespace":      " " + std[:20] + "\n" + std[20:40] + "\r\n" + std[40:] + " ",
		"missing padding": strings.TrimRight(std, "="),
	}
	for name, encoded := range cases {
		got, err := decodeBase64Resilient(encoded)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", name, err)
			continue
		}
		if !bytes.Equal(got, bootstrapPayload) {
			t.Errorf("%s: payload mismatch (%d bytes)", name, len(got))
		}
	}
	if _, err := decodeBase64Resilient("!!!not-base64-at-all!!!"); err == nil {
		t.Error("expected error for invalid input")
	}
}
