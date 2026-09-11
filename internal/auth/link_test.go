package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0xc0re/cluckers/internal/gateway"
	"github.com/0xc0re/cluckers/internal/ui"
)

func TestWaitForLink_PollsUntilLinked(t *testing.T) {
	linkPollInterval = 5 * time.Millisecond
	t.Cleanup(func() { linkPollInterval = 3 * time.Second })

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		switch n {
		case 1:
			json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "CODE1", "linked_flag": 0})
		case 2:
			json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "CODE1", "linked_flag": 0})
		case 3:
			json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "CODE2", "linked_flag": 0})
		default:
			json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "lpt_v1_ok", "refresh_token": "r", "linked_flag": 1, "user_name": "u"})
		}
	}))
	defer srv.Close()

	var codes []string
	res, err := WaitForLink(context.Background(), gateway.NewClient(srv.URL, false), "u", "p", func(c string) { codes = append(codes, c) })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.AccessToken != "lpt_v1_ok" {
		t.Errorf("AccessToken = %q, want lpt_v1_ok", res.AccessToken)
	}
	if got := atomic.LoadInt32(&calls); got != 4 {
		t.Errorf("requests = %d, want 4", got)
	}
	if len(codes) != 2 || codes[0] != "CODE1" || codes[1] != "CODE2" {
		t.Errorf("onCode calls = %v, want [CODE1 CODE2] (called only when the code changes)", codes)
	}
}

func TestWaitForLink_StopsOnPinGate(t *testing.T) {
	srv := newJSONServer(t, http.StatusOK, map[string]interface{}{"linked_flag": 1, "text_value": "PIN_REQUIRED"})
	defer srv.Close()
	_, err := WaitForLink(context.Background(), gateway.NewClient(srv.URL, false), "u", "p", nil)
	if !errors.Is(err, ErrPinRequired) {
		t.Errorf("expected ErrPinRequired, got %v", err)
	}
}

func TestWaitForLink_Timeout(t *testing.T) {
	linkPollInterval = 5 * time.Millisecond
	linkTimeout = 30 * time.Millisecond
	t.Cleanup(func() { linkPollInterval = 3 * time.Second; linkTimeout = 5 * time.Minute })

	srv := newJSONServer(t, http.StatusOK, map[string]interface{}{"access_token": "CODE", "linked_flag": 0})
	defer srv.Close()
	_, err := WaitForLink(context.Background(), gateway.NewClient(srv.URL, false), "u", "p", nil)
	if err == nil || errors.Is(err, ErrNotLinked) {
		t.Fatalf("expected a timeout UserError, got %v", err)
	}
}

func TestWaitForLink_ContextCancel(t *testing.T) {
	srv := newJSONServer(t, http.StatusOK, map[string]interface{}{"access_token": "CODE", "linked_flag": 0})
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := WaitForLink(ctx, gateway.NewClient(srv.URL, false), "u", "p", nil)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestWaitForLink_SurvivesTransientErrors(t *testing.T) {
	linkPollInterval = 5 * time.Millisecond
	t.Cleanup(func() { linkPollInterval = 3 * time.Second })

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		switch n {
		case 1:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "CODE", "linked_flag": 0})
		case 2, 3, 4, 5:
			// Gateway hiccup: retryablehttp retries 5xx up to 3 times, then errors.
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte("<html>bad gateway</html>"))
		case 6:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("not json"))
		default:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "lpt_v1_ok", "linked_flag": 1})
		}
	}))
	defer srv.Close()

	res, err := WaitForLink(context.Background(), gateway.NewClient(srv.URL, false), "u", "p", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.AccessToken != "lpt_v1_ok" || atomic.LoadInt32(&calls) != 7 {
		t.Errorf("token = %q, calls = %d; want lpt_v1_ok after 7 calls", res.AccessToken, calls)
	}
}

func TestWaitForLink_StopsOnCredentialRejection(t *testing.T) {
	srv := newJSONServer(t, http.StatusUnauthorized, map[string]interface{}{"detail": "bad password", "title": "invalid_credentials", "status": 401})
	defer srv.Close()
	_, err := WaitForLink(context.Background(), gateway.NewClient(srv.URL, false), "u", "p", nil)
	var ue *ui.UserError
	if !errors.As(err, &ue) || !ue.IsStatus(http.StatusUnauthorized) {
		t.Fatalf("expected a 401 UserError, got %v", err)
	}
}
