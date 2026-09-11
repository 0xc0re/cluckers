package launch

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0xc0re/cluckers/internal/auth"
	"github.com/0xc0re/cluckers/internal/config"
	"github.com/0xc0re/cluckers/internal/gateway"
)

// noopReporter is a ProgressReporter that does nothing (for tests).
type noopReporter struct{}

func (n *noopReporter) StepStarted(name string)           {}
func (n *noopReporter) StepCompleted(name string)         {}
func (n *noopReporter) StepFailed(name string, err error) {}
func (n *noopReporter) StepSkipped(name string)           {}
func (n *noopReporter) StepPaused(name string)            {}

// gatewayStub is an httptest gateway with per-path handlers and hit counts.
type gatewayStub struct {
	*httptest.Server
	mu   sync.Mutex
	hits map[string]int
}

func newGatewayStub(t *testing.T, handlers map[string]http.HandlerFunc) *gatewayStub {
	t.Helper()
	g := &gatewayStub{hits: map[string]int{}}
	g.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.mu.Lock()
		g.hits[r.URL.Path]++
		g.mu.Unlock()
		if h, ok := handlers[r.URL.Path]; ok {
			h(w, r)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(g.Close)
	return g
}

func (g *gatewayStub) count(path string) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.hits[path]
}

func reply(status int, body map[string]interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(body)
	}
}

var testBootstrap = []byte("BPS1" + strings.Repeat("x", 132))

// newTestState builds a GUI-style state (credentials pre-populated so no
// prompting can happen) pointed at the stub gateway. PinnedVersion is set so
// resolveClientBuild never touches the network.
func newTestState(t *testing.T, g *gatewayStub) *LaunchState {
	t.Helper()
	t.Setenv("CLUCKERS_HOME", t.TempDir())
	return &LaunchState{
		Config:   &config.Config{Gateway: g.URL, PinnedVersion: "0.39.6969.0"},
		Client:   gateway.NewClient(g.URL, false),
		Reporter: &noopReporter{},
		Username: "alice",
		Password: "hunter2",
		GameDir:  t.TempDir(),
	}
}

func TestStepAuthenticate_ValidCacheMakesNoRequests(t *testing.T) {
	g := newGatewayStub(t, nil)
	state := newTestState(t, g)
	if err := auth.SaveTokenCache(&auth.TokenCache{AccessToken: "cached", Username: "alice", AccessCachedAt: time.Now(), AccessExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err := stepAuthenticate(context.Background(), state); err != nil {
		t.Fatalf("stepAuthenticate: %v", err)
	}
	if state.AccessToken != "cached" || state.Password != "hunter2" {
		t.Errorf("state = token %q password %q", state.AccessToken, state.Password)
	}
	if n := g.count("/launcher/v1/session-or-link") + g.count("/launcher/v1/session/refresh"); n != 0 {
		t.Errorf("made %d gateway requests, want 0", n)
	}
}

func TestStepAuthenticate_GUIUnlinkedReturnsErrNotLinked(t *testing.T) {
	g := newGatewayStub(t, map[string]http.HandlerFunc{
		"/launcher/v1/session-or-link": reply(http.StatusOK, map[string]interface{}{"access_token": "CODE1", "linked_flag": 0}),
	})
	state := newTestState(t, g)
	err := stepAuthenticate(context.Background(), state)
	if !errors.Is(err, auth.ErrNotLinked) {
		t.Fatalf("expected ErrNotLinked, got %v", err)
	}
	var nl *auth.NotLinkedError
	if !errors.As(err, &nl) || nl.LinkCode != "CODE1" {
		t.Errorf("expected link code CODE1 in %v", err)
	}
	if g.count("/launcher/v1/session-or-link") != 1 {
		t.Errorf("GUI mode must not poll; hits = %v", g.hits)
	}
}

func TestStepAuthenticate_GUIPinGate(t *testing.T) {
	g := newGatewayStub(t, map[string]http.HandlerFunc{
		"/launcher/v1/session-or-link": reply(http.StatusOK, map[string]interface{}{"linked_flag": 1, "text_value": "PIN_REQUIRED"}),
	})
	state := newTestState(t, g)
	if err := stepAuthenticate(context.Background(), state); !errors.Is(err, auth.ErrPinRequired) {
		t.Fatalf("expected ErrPinRequired, got %v", err)
	}
}

func TestStepLaunchAuth_Success(t *testing.T) {
	var gotBuild, gotBody string
	g := newGatewayStub(t, map[string]http.HandlerFunc{
		"/launcher/v1/launch-auth": func(w http.ResponseWriter, r *http.Request) {
			gotBuild = r.Header.Get("x-realm-client-build")
			b := make([]byte, 16)
			n, _ := r.Body.Read(b)
			gotBody = strings.TrimSpace(string(b[:n]))
			reply(http.StatusOK, map[string]interface{}{
				"launch_token":           "lt_1",
				"launch_expires_at_unix": time.Now().Add(15 * time.Minute).Unix(),
				"portal_info_1":          base64.StdEncoding.EncodeToString(testBootstrap),
			})(w, r)
		},
	})
	state := newTestState(t, g)
	state.AccessToken = "lpt_ok"
	if err := stepLaunchAuth(context.Background(), state); err != nil {
		t.Fatalf("stepLaunchAuth: %v", err)
	}
	if state.LaunchToken != "lt_1" || string(state.Bootstrap) != string(testBootstrap) {
		t.Errorf("state = token %q bootstrap %d bytes", state.LaunchToken, len(state.Bootstrap))
	}
	if state.LaunchExpiresAt.IsZero() {
		t.Error("LaunchExpiresAt not set")
	}
	if gotBuild != "0.39.6969.0" || gotBody != "{}" {
		t.Errorf("request: build=%q body=%q", gotBuild, gotBody)
	}
}

func TestStepLaunchAuth_RejectedTokenRefreshesAndRetries(t *testing.T) {
	var calls int
	g := newGatewayStub(t, map[string]http.HandlerFunc{
		"/launcher/v1/launch-auth": func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.Header.Get("Authorization") != "Bearer lpt_fresh" {
				reply(http.StatusUnauthorized, map[string]interface{}{"detail": "expired", "title": "invalid_session", "status": 401})(w, r)
				return
			}
			reply(http.StatusOK, map[string]interface{}{"launch_token": "lt_2"})(w, r)
		},
		"/launcher/v1/session/refresh": reply(http.StatusOK, map[string]interface{}{"access_token": "lpt_fresh", "refresh_token": "lrt_2"}),
	})
	state := newTestState(t, g)
	state.AccessToken = "lpt_stale"
	state.TokenCache = &auth.TokenCache{AccessToken: "lpt_stale", RefreshToken: "lrt_1", Username: "alice", AccessCachedAt: time.Now()}

	if err := stepLaunchAuth(context.Background(), state); err != nil {
		t.Fatalf("stepLaunchAuth: %v", err)
	}
	if state.LaunchToken != "lt_2" || state.AccessToken != "lpt_fresh" {
		t.Errorf("state = launch %q access %q", state.LaunchToken, state.AccessToken)
	}
	if calls != 2 || g.count("/launcher/v1/session/refresh") != 1 || g.count("/launcher/v1/session-or-link") != 0 {
		t.Errorf("launch-auth calls = %d, hits = %v; want one refresh, one retry, no password login", calls, g.hits)
	}
	if state.Bootstrap != nil {
		t.Error("Bootstrap should be nil when portal_info_1 is absent")
	}
}

func TestStepLaunchAuth_RejectedTwiceFails(t *testing.T) {
	g := newGatewayStub(t, map[string]http.HandlerFunc{
		"/launcher/v1/launch-auth":     reply(http.StatusUnauthorized, map[string]interface{}{"detail": "expired", "title": "invalid_session", "status": 401}),
		"/launcher/v1/session-or-link": reply(http.StatusOK, map[string]interface{}{"access_token": "lpt_new", "linked_flag": 1, "user_name": "alice"}),
	})
	state := newTestState(t, g)
	state.AccessToken = "lpt_stale"
	err := stepLaunchAuth(context.Background(), state)
	if !errors.Is(err, auth.ErrTokenRejected) {
		t.Fatalf("expected ErrTokenRejected after retry, got %v", err)
	}
	if g.count("/launcher/v1/launch-auth") != 2 || g.count("/launcher/v1/session-or-link") != 1 {
		t.Errorf("hits = %v; want two launch-auth attempts and one login", g.hits)
	}
}

func TestStepLaunchAuth_EmptyLaunchToken(t *testing.T) {
	g := newGatewayStub(t, map[string]http.HandlerFunc{
		"/launcher/v1/launch-auth": reply(http.StatusOK, map[string]interface{}{"portal_info_1": base64.StdEncoding.EncodeToString(testBootstrap)}),
	})
	state := newTestState(t, g)
	state.AccessToken = "lpt_ok"
	if err := stepLaunchAuth(context.Background(), state); err == nil || !strings.Contains(err.Error(), "launch token") {
		t.Fatalf("expected empty-launch-token error, got %v", err)
	}
}

func TestStepLaunchGame_RefusesWithoutLaunchToken(t *testing.T) {
	state := newTestState(t, newGatewayStub(t, nil))
	state.LaunchToken = ""
	if err := stepLaunchGame(context.Background(), state); err == nil || !strings.Contains(err.Error(), "launch token") {
		t.Fatalf("expected no-launch-token error, got %v", err)
	}
}

func TestWriteTokenFile(t *testing.T) {
	t.Setenv("CLUCKERS_HOME", t.TempDir())
	path, cleanup, err := writeTokenFile("lt_secret")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "lt_secret" {
		t.Errorf("token file = %q, %v", data, err)
	}
	if info, _ := os.Stat(path); info != nil && info.Mode().Perm() != 0600 {
		t.Errorf("token file mode = %v, want 0600", info.Mode().Perm())
	}
	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("cleanup did not remove the token file")
	}
}

func TestBuildSteps_LaunchAuthAfterVerify(t *testing.T) {
	names := StepNames(&config.Config{})
	idx := func(name string) int {
		for i, n := range names {
			if n == name {
				return i
			}
		}
		return -1
	}
	v, l, g := idx("Verifying game installation"), idx("Requesting launch authorization"), idx("Launching game")
	if v < 0 || l < 0 || g < 0 || !(v < l && l < g) {
		t.Errorf("step order = %v; want verify < launch-auth < launch", names)
	}
	for _, n := range names {
		if strings.Contains(n, "bootstrap") {
			t.Errorf("legacy bootstrap step still present: %v", names)
		}
	}
}
