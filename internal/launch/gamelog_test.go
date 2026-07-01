package launch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const sampleMCTSLog = "Date & Time\tMessage\tType\tDetail\tSource\tProcess\tThread\n" +
	"30-06-2026 13:05:23.985\tBad marshal [UNKNOWN] id=[0/0x00000000] flags=0x00 size=594600 place=61\t1\t1\tPlatform\t300\t304\n" +
	"End\n"

func TestSummarizeGameLogError_ExtractsMessage(t *testing.T) {
	got := summarizeGameLogError(sampleMCTSLog)
	if got == "" {
		t.Fatal("expected a non-empty summary from a log containing an error row")
	}
	if want := "Bad marshal"; !contains(got, want) {
		t.Errorf("summary = %q, want it to contain %q", got, want)
	}
	// The header row and the "End" footer must not appear in the summary.
	if contains(got, "Date & Time") || contains(got, "End") {
		t.Errorf("summary should exclude header/footer, got %q", got)
	}
}

func TestSummarizeGameLogError_HeaderOnly(t *testing.T) {
	headerOnly := "Date & Time\tMessage\tType\tDetail\tSource\nEnd\n"
	if got := summarizeGameLogError(headerOnly); got != "" {
		t.Errorf("expected empty summary for header/footer-only log, got %q", got)
	}
}

func TestSummarizeGameLogError_Empty(t *testing.T) {
	if got := summarizeGameLogError(""); got != "" {
		t.Errorf("expected empty summary for empty input, got %q", got)
	}
}

func TestLatestGameLog_PicksNewest(t *testing.T) {
	gameDir := t.TempDir()
	logsDir := filepath.Join(gameDir, "Realm-Royale", "Binaries", "Logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		t.Fatal(err)
	}
	older := filepath.Join(logsDir, "MCTS-2026-06-30_12.00.00_00100.log")
	newer := filepath.Join(logsDir, "MCTS-2026-06-30_13.00.00_00200.log")
	// A non-MCTS log that must be ignored even if newest.
	other := filepath.Join(logsDir, "system-2026-06-30.log")
	for _, p := range []string{older, newer, other} {
		if err := os.WriteFile(p, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	// Make mtimes explicit: older < newer, and the non-MCTS file newest of all.
	now := time.Now()
	os.Chtimes(older, now.Add(-2*time.Hour), now.Add(-2*time.Hour))
	os.Chtimes(newer, now.Add(-1*time.Hour), now.Add(-1*time.Hour))
	os.Chtimes(other, now, now)

	got := latestGameLog(gameDir)
	if got != newer {
		t.Errorf("latestGameLog = %q, want %q", got, newer)
	}
}

func TestLatestGameLog_NoLogs(t *testing.T) {
	if got := latestGameLog(t.TempDir()); got != "" {
		t.Errorf("expected empty path when no logs exist, got %q", got)
	}
}

func TestGameLogError_ReadsLatest(t *testing.T) {
	gameDir := t.TempDir()
	logsDir := filepath.Join(gameDir, "Realm-Royale", "Binaries", "Logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logsDir, "MCTS-2026-06-30_13.05.23_00300.log"), []byte(sampleMCTSLog), 0644); err != nil {
		t.Fatal(err)
	}
	got := gameLogError(gameDir)
	if !contains(got, "Bad marshal") {
		t.Errorf("gameLogError = %q, want it to contain the Bad marshal message", got)
	}
}

func TestNewLaunchError_LeadsWithGameError(t *testing.T) {
	gameDir := t.TempDir()
	logsDir := filepath.Join(gameDir, "Realm-Royale", "Binaries", "Logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logsDir, "MCTS-x.log"), []byte(sampleMCTSLog), 0644); err != nil {
		t.Fatal(err)
	}

	ue := newLaunchError(gameDir, "Proton launch failed", "some proton stderr", "do the thing")
	// The real cause must be in the Message (shown even without -v), not buried in Detail.
	if !contains(ue.Message, "Bad marshal") {
		t.Errorf("Message = %q, want it to contain the game's logged error", ue.Message)
	}
	if ue.Message == "Proton launch failed" {
		t.Error("Message should not be the bare fallback when the game logged an error")
	}
	if ue.Suggestion != "do the thing" {
		t.Errorf("Suggestion = %q, want it preserved", ue.Suggestion)
	}
}

func TestNewLaunchError_FallbackWhenNoLog(t *testing.T) {
	ue := newLaunchError(t.TempDir(), "Proton launch failed", "stderr tail", "hint")
	if ue.Message != "Proton launch failed" {
		t.Errorf("Message = %q, want the fallback when no game log exists", ue.Message)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
