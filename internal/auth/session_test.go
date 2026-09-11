package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/0xc0re/cluckers/internal/gateway"
)

// countingServer records hits per path and dispatches to handlers.
type countingServer struct {
	*httptest.Server
	mu   sync.Mutex
	hits map[string]int
}

func newCountingServer(t *testing.T, handlers map[string]http.HandlerFunc) *countingServer {
	t.Helper()
	cs := &countingServer{hits: map[string]int{}}
	cs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cs.mu.Lock()
		cs.hits[r.URL.Path]++
		cs.mu.Unlock()
		if h, ok := handlers[r.URL.Path]; ok {
			h(w, r)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(cs.Close)
	return cs
}

func (cs *countingServer) count(path string) int {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.hits[path]
}

func jsonReply(status int, body map[string]interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(body)
	}
}

func TestRefreshSession(t *testing.T) {
	t.Run("success without bearer", func(t *testing.T) {
		srv := newCountingServer(t, map[string]http.HandlerFunc{
			"/launcher/v1/session/refresh": func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "" {
					t.Error("refresh must not send Authorization")
				}
				var body map[string]string
				json.NewDecoder(r.Body).Decode(&body)
				if body["refresh_token"] != "lrt_old" {
					t.Errorf("refresh_token = %q, want lrt_old", body["refresh_token"])
				}
				jsonReply(http.StatusOK, map[string]interface{}{"access_token": "lpt_new", "access_expires_at_unix": 1800000000})(w, r)
			},
		})
		res, err := RefreshSession(context.Background(), gateway.NewClient(srv.URL, false), "lrt_old")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.AccessToken != "lpt_new" || res.RefreshToken != "" || res.Username != "" {
			t.Errorf("result = %+v", res)
		}
	})

	t.Run("401 is ErrRefreshRejected", func(t *testing.T) {
		srv := newCountingServer(t, map[string]http.HandlerFunc{
			"/launcher/v1/session/refresh": jsonReply(http.StatusUnauthorized, map[string]interface{}{
				"detail": "Launcher refresh session is invalid or expired", "title": "invalid_refresh_session", "status": 401,
			}),
		})
		_, err := RefreshSession(context.Background(), gateway.NewClient(srv.URL, false), "stale")
		if !errors.Is(err, ErrRefreshRejected) {
			t.Errorf("expected ErrRefreshRejected, got %v", err)
		}
	})
}

func TestEnsureSession(t *testing.T) {
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)

	t.Run("valid cache makes no requests", func(t *testing.T) {
		t.Setenv("CLUCKERS_HOME", t.TempDir())
		srv := newCountingServer(t, nil)
		cache := &TokenCache{AccessToken: "a", Username: "u", AccessCachedAt: time.Now(), AccessExpiresAt: future}
		s, err := EnsureSession(context.Background(), gateway.NewClient(srv.URL, false), SessionRequest{Username: "u", Cache: cache})
		if err != nil || s.Source != SourceCache || s.AccessToken != "a" {
			t.Fatalf("session = %+v, err = %v", s, err)
		}
		if n := srv.count("/launcher/v1/session-or-link") + srv.count("/launcher/v1/session/refresh"); n != 0 {
			t.Errorf("made %d requests, want 0", n)
		}
	})

	t.Run("cache for another user is ignored", func(t *testing.T) {
		t.Setenv("CLUCKERS_HOME", t.TempDir())
		srv := newCountingServer(t, map[string]http.HandlerFunc{
			"/launcher/v1/session-or-link": jsonReply(http.StatusOK, map[string]interface{}{"access_token": "lpt_b", "linked_flag": 1, "user_name": "bob"}),
		})
		cache := &TokenCache{AccessToken: "a", RefreshToken: "r", Username: "alice", AccessCachedAt: time.Now(), AccessExpiresAt: future}
		s, err := EnsureSession(context.Background(), gateway.NewClient(srv.URL, false), SessionRequest{Username: "bob", Password: "p", Cache: cache})
		if err != nil || s.Source != SourceLogin || s.Username != "bob" {
			t.Fatalf("session = %+v, err = %v", s, err)
		}
		if srv.count("/launcher/v1/session/refresh") != 0 {
			t.Error("must not refresh another user's session")
		}
	})

	t.Run("expired access with refresh token refreshes once and keeps old refresh token", func(t *testing.T) {
		t.Setenv("CLUCKERS_HOME", t.TempDir())
		srv := newCountingServer(t, map[string]http.HandlerFunc{
			"/launcher/v1/session/refresh": jsonReply(http.StatusOK, map[string]interface{}{"access_token": "lpt_new", "access_expires_at_unix": future.Unix()}),
		})
		cache := &TokenCache{AccessToken: "old", RefreshToken: "lrt_keep", Username: "u", AccessCachedAt: past, AccessExpiresAt: past, RefreshExpiresAt: future}
		s, err := EnsureSession(context.Background(), gateway.NewClient(srv.URL, false), SessionRequest{Username: "u", Password: "p", Cache: cache})
		if err != nil || s.Source != SourceRefresh || s.AccessToken != "lpt_new" || s.Username != "u" {
			t.Fatalf("session = %+v, err = %v", s, err)
		}
		if srv.count("/launcher/v1/session/refresh") != 1 || srv.count("/launcher/v1/session-or-link") != 0 {
			t.Errorf("hits = %v, want exactly one refresh and no login", srv.hits)
		}
		saved, _ := LoadTokenCache()
		if saved == nil || saved.AccessToken != "lpt_new" || saved.RefreshToken != "lrt_keep" || !saved.RefreshExpiresAt.Equal(cache.RefreshExpiresAt) {
			t.Errorf("saved cache = %+v, want new access and the old refresh token", saved)
		}
	})

	t.Run("rotated refresh token is stored", func(t *testing.T) {
		t.Setenv("CLUCKERS_HOME", t.TempDir())
		srv := newCountingServer(t, map[string]http.HandlerFunc{
			"/launcher/v1/session/refresh": jsonReply(http.StatusOK, map[string]interface{}{"access_token": "lpt_new", "refresh_token": "lrt_rotated", "user_name": "u"}),
		})
		cache := &TokenCache{AccessToken: "old", RefreshToken: "lrt_old", Username: "u", AccessCachedAt: past}
		s, err := EnsureSession(context.Background(), gateway.NewClient(srv.URL, false), SessionRequest{Cache: cache})
		if err != nil || s.Cache.RefreshToken != "lrt_rotated" {
			t.Fatalf("session = %+v, err = %v", s, err)
		}
	})

	t.Run("refresh rejected falls back to login", func(t *testing.T) {
		t.Setenv("CLUCKERS_HOME", t.TempDir())
		srv := newCountingServer(t, map[string]http.HandlerFunc{
			"/launcher/v1/session/refresh": jsonReply(http.StatusUnauthorized, map[string]interface{}{"detail": "expired", "title": "invalid_refresh_session", "status": 401}),
			"/launcher/v1/session-or-link": jsonReply(http.StatusOK, map[string]interface{}{"access_token": "lpt_login", "refresh_token": "lrt_login", "linked_flag": 1, "user_name": "u"}),
		})
		cache := &TokenCache{AccessToken: "old", RefreshToken: "lrt_stale", Username: "u", AccessCachedAt: past}
		s, err := EnsureSession(context.Background(), gateway.NewClient(srv.URL, false), SessionRequest{Username: "u", Password: "p", Cache: cache})
		if err != nil || s.Source != SourceLogin || s.AccessToken != "lpt_login" {
			t.Fatalf("session = %+v, err = %v", s, err)
		}
		if srv.count("/launcher/v1/session/refresh") != 1 || srv.count("/launcher/v1/session-or-link") != 1 {
			t.Errorf("hits = %v", srv.hits)
		}
	})

	t.Run("no cache and no credentials", func(t *testing.T) {
		t.Setenv("CLUCKERS_HOME", t.TempDir())
		srv := newCountingServer(t, nil)
		_, err := EnsureSession(context.Background(), gateway.NewClient(srv.URL, false), SessionRequest{})
		if !errors.Is(err, ErrNoCredentials) {
			t.Errorf("expected ErrNoCredentials, got %v", err)
		}
	})

	t.Run("legacy cache without refresh token logs in", func(t *testing.T) {
		t.Setenv("CLUCKERS_HOME", t.TempDir())
		srv := newCountingServer(t, map[string]http.HandlerFunc{
			"/launcher/v1/session-or-link": jsonReply(http.StatusOK, map[string]interface{}{"access_token": "lpt_login", "linked_flag": 1}),
		})
		cache := &TokenCache{AccessToken: "old", Username: "u", AccessCachedAt: past}
		s, err := EnsureSession(context.Background(), gateway.NewClient(srv.URL, false), SessionRequest{Username: "u", Password: "p", Cache: cache})
		if err != nil || s.Source != SourceLogin {
			t.Fatalf("session = %+v, err = %v", s, err)
		}
		if srv.count("/launcher/v1/session/refresh") != 0 {
			t.Error("must not call refresh without a refresh token")
		}
	})

	t.Run("login sentinels propagate", func(t *testing.T) {
		t.Setenv("CLUCKERS_HOME", t.TempDir())
		srv := newCountingServer(t, map[string]http.HandlerFunc{
			"/launcher/v1/session-or-link": jsonReply(http.StatusOK, map[string]interface{}{"access_token": "CODE", "linked_flag": 0}),
		})
		_, err := EnsureSession(context.Background(), gateway.NewClient(srv.URL, false), SessionRequest{Username: "u", Password: "p"})
		if !errors.Is(err, ErrNotLinked) {
			t.Errorf("expected ErrNotLinked, got %v", err)
		}
	})
}
