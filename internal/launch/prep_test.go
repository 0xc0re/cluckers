//go:build linux

package launch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xc0re/cluckers/assets"
	"github.com/0xc0re/cluckers/internal/config"
)

// newTestPrepState creates a LaunchState populated with test data suitable for
// stepWriteLaunchConfig. Sets CLUCKERS_HOME to a temp dir so file writes are isolated.
func newTestPrepState(t *testing.T) *LaunchState {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("CLUCKERS_HOME", tmp)

	gameDir := filepath.Join(tmp, "game")
	if err := os.MkdirAll(gameDir, 0755); err != nil {
		t.Fatal(err)
	}

	return &LaunchState{
		Config: &config.Config{
			HostX:   "157.90.131.105",
			Verbose: false,
		},
		Username:    "testuser",
		AccessToken: "test-access-token",
		OIDCToken:   "test-oidc-token",
		Bootstrap:   []byte("BPS1" + strings.Repeat("\x00", 132)), // 136 bytes with magic header
		GameDir:     gameDir,
		Reporter:    &noopReporter{},
	}
}

// noopReporter is a ProgressReporter that does nothing (for tests).
type noopReporter struct{}

func (n *noopReporter) StepStarted(name string)          {}
func (n *noopReporter) StepCompleted(name string)        {}
func (n *noopReporter) StepFailed(name string, err error) {}
func (n *noopReporter) StepSkipped(name string)          {}
func (n *noopReporter) StepPaused(name string)           {}

func TestStepWriteLaunchConfig_WritesAllFiles(t *testing.T) {
	state := newTestPrepState(t)

	if err := stepWriteLaunchConfig(context.Background(), state); err != nil {
		t.Fatalf("stepWriteLaunchConfig() error: %v", err)
	}

	// Verify all 4 files exist.
	cacheDir := config.CacheDir()
	binDir := config.BinDir()

	files := map[string]string{
		"bootstrap.bin":    filepath.Join(cacheDir, "bootstrap.bin"),
		"oidc-token.txt":   filepath.Join(cacheDir, "oidc-token.txt"),
		"shm_launcher.exe": filepath.Join(binDir, "shm_launcher.exe"),
		"launch-config.txt": filepath.Join(binDir, "launch-config.txt"),
	}

	for name, path := range files {
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("%s not created: %v", name, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("%s is empty", name)
		}
	}

	// Verify bootstrap.bin content matches state.Bootstrap.
	data, err := os.ReadFile(files["bootstrap.bin"])
	if err != nil {
		t.Fatalf("reading bootstrap.bin: %v", err)
	}
	if len(data) != len(state.Bootstrap) {
		t.Errorf("bootstrap.bin size = %d, want %d", len(data), len(state.Bootstrap))
	}

	// Verify oidc-token.txt content.
	oidc, err := os.ReadFile(files["oidc-token.txt"])
	if err != nil {
		t.Fatalf("reading oidc-token.txt: %v", err)
	}
	if string(oidc) != "test-oidc-token" {
		t.Errorf("oidc-token.txt = %q, want %q", string(oidc), "test-oidc-token")
	}
}

func TestStepWriteLaunchConfig_LaunchConfigFormat(t *testing.T) {
	state := newTestPrepState(t)

	if err := stepWriteLaunchConfig(context.Background(), state); err != nil {
		t.Fatalf("stepWriteLaunchConfig() error: %v", err)
	}

	configPath := filepath.Join(config.BinDir(), "launch-config.txt")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading launch-config.txt: %v", err)
	}

	content := string(data)
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")

	// Minimum 3 lines: bootstrap_file, shm_name, game_exe
	if len(lines) < 3 {
		t.Fatalf("launch-config.txt has %d lines, need at least 3", len(lines))
	}

	// Line 1: bootstrap file (Wine path)
	if !strings.HasPrefix(lines[0], "Z:\\") {
		t.Errorf("line 1 (bootstrap) should be a Wine path, got %q", lines[0])
	}
	if !strings.Contains(lines[0], "bootstrap.bin") {
		t.Errorf("line 1 should reference bootstrap.bin, got %q", lines[0])
	}

	// Line 2: SHM name
	if lines[1] != prepSHMName {
		t.Errorf("line 2 (shm_name) = %q, want %q", lines[1], prepSHMName)
	}

	// Line 3: game exe (Wine path)
	if !strings.HasPrefix(lines[2], "Z:\\") {
		t.Errorf("line 3 (game_exe) should be a Wine path, got %q", lines[2])
	}
	if !strings.HasSuffix(lines[2], "ShippingPC-RealmGameNoEditor.exe") {
		t.Errorf("line 3 should end with game exe, got %q", lines[2])
	}

	// Verify expected game args are present.
	fullContent := content
	expectedArgs := []string{
		"-user=testuser",
		"-token=test-access-token",
		"-eac_oidc_token_file=",
		"-hostx=157.90.131.105",
		"-Language=INT",
		"-dx11",
		fmt.Sprintf("-content_bootstrap_size=%d", len(state.Bootstrap)),
		"-seekfreeloadingpcconsole",
		"-nohomedir",
		"-content_bootstrap_shm=" + prepSHMName,
	}
	for _, arg := range expectedArgs {
		if !strings.Contains(fullContent, arg) {
			t.Errorf("launch-config.txt missing %q", arg)
		}
	}
}

func TestStepWriteLaunchConfig_IdempotentSHMExtraction(t *testing.T) {
	state := newTestPrepState(t)

	// First write.
	if err := stepWriteLaunchConfig(context.Background(), state); err != nil {
		t.Fatalf("first stepWriteLaunchConfig() error: %v", err)
	}

	shmPath := filepath.Join(config.BinDir(), "shm_launcher.exe")
	info1, err := os.Stat(shmPath)
	if err != nil {
		t.Fatalf("stat shm_launcher.exe after first write: %v", err)
	}
	modTime1 := info1.ModTime()

	// Verify size matches embedded asset.
	if info1.Size() != int64(len(assets.SHMLauncherExe)) {
		t.Errorf("shm_launcher.exe size = %d, want %d", info1.Size(), len(assets.SHMLauncherExe))
	}

	// Second write — should be a no-op (same size).
	if err := stepWriteLaunchConfig(context.Background(), state); err != nil {
		t.Fatalf("second stepWriteLaunchConfig() error: %v", err)
	}

	info2, err := os.Stat(shmPath)
	if err != nil {
		t.Fatalf("stat shm_launcher.exe after second write: %v", err)
	}

	// ModTime should not change if the file was not rewritten.
	if !modTime1.Equal(info2.ModTime()) {
		t.Errorf("shm_launcher.exe was rewritten (modtime changed from %v to %v)", modTime1, info2.ModTime())
	}
}

func TestStepWriteLaunchConfig_NilBootstrap(t *testing.T) {
	state := newTestPrepState(t)
	state.Bootstrap = nil

	err := stepWriteLaunchConfig(context.Background(), state)
	if err == nil {
		t.Fatal("expected error for nil bootstrap, got nil")
	}

	if !strings.Contains(err.Error(), "bootstrap") {
		t.Errorf("error should mention bootstrap, got: %v", err)
	}
}

func TestStepWriteLaunchConfig_DynamicBootstrapSize(t *testing.T) {
	state := newTestPrepState(t)
	// Use a small bootstrap (21 bytes) to prove size is dynamic, not hardcoded.
	state.Bootstrap = []byte("BPS1" + strings.Repeat("\x00", 17))

	if err := stepWriteLaunchConfig(context.Background(), state); err != nil {
		t.Fatalf("stepWriteLaunchConfig() error: %v", err)
	}

	configPath := filepath.Join(config.BinDir(), "launch-config.txt")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading launch-config.txt: %v", err)
	}

	content := string(data)
	expected := fmt.Sprintf("-content_bootstrap_size=%d", len(state.Bootstrap))
	if !strings.Contains(content, expected) {
		t.Errorf("launch-config.txt should contain %q, got:\n%s", expected, content)
	}
	// Verify the old hardcoded value is NOT present.
	if strings.Contains(content, "-content_bootstrap_size=136") {
		t.Error("launch-config.txt still contains hardcoded -content_bootstrap_size=136")
	}
}
