package launch

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/0xc0re/cluckers/internal/ui"
)

// The game (UE3 RealmGame) writes tab-separated diagnostic logs to
// <gameDir>/Realm-Royale/Binaries/Logs/MCTS-<timestamp>_<pid>.log. The first
// line is a column header and the file ends with an "End" marker; notable
// events (including the fatal error that aborts startup) are logged as data
// rows whose second column is the human-readable message. Surfacing that
// message turns an opaque "Proton launch failed" into the game's actual error.

// gameLogError returns a concise summary of the most recent game log's error
// rows, or "" if no log or no usable message is found.
func gameLogError(gameDir string) string {
	path := latestGameLog(gameDir)
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return summarizeGameLogError(string(data))
}

// latestGameLog returns the path to the most recently modified MCTS-*.log under
// the game's Binaries/Logs directory, or "" if none exists.
func latestGameLog(gameDir string) string {
	dir := filepath.Join(gameDir, "Realm-Royale", "Binaries", "Logs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var newest string
	var newestMod int64 = -1
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, "MCTS-") || !strings.HasSuffix(name, ".log") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if mod := info.ModTime().UnixNano(); mod > newestMod {
			newestMod = mod
			newest = filepath.Join(dir, name)
		}
	}
	return newest
}

// newLaunchError builds the UserError for a failed game launch. When the game
// logged its own error, that becomes the Message (always shown, even without
// -v) so the real cause is visible; otherwise fallbackMsg is used. The host
// process's stderr tail and the game log path go in Detail (shown with -v).
func newLaunchError(gameDir, fallbackMsg, stderr, suggestion string) *ui.UserError {
	msg := fallbackMsg
	if gameErr := gameLogError(gameDir); gameErr != "" {
		msg = "Game crashed on startup — " + gameErr
	}

	detail := lastNLines(stderr, 10)
	if logPath := latestGameLog(gameDir); logPath != "" {
		if strings.TrimSpace(detail) != "" {
			detail += "\n"
		}
		detail += "Game log: " + logPath
	}

	return &ui.UserError{Message: msg, Detail: detail, Suggestion: suggestion}
}

// lastNLines returns the last n lines from a string. If the string has fewer
// than n lines, it is returned as-is. An empty string returns empty.
func lastNLines(s string, n int) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

// summarizeGameLogError extracts the message column from the data rows of an
// MCTS log, skipping the header row and the trailing "End" marker. It returns
// up to the last three messages joined by "; ", or "" if there are none.
func summarizeGameLogError(content string) string {
	lines := strings.Split(content, "\n")
	var msgs []string
	for i, line := range lines {
		line = strings.TrimRight(line, "\r")
		if i == 0 { // column header
			continue
		}
		if line == "" || line == "End" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}
		if msg := strings.TrimSpace(fields[1]); msg != "" {
			msgs = append(msgs, msg)
		}
	}
	if len(msgs) == 0 {
		return ""
	}
	if len(msgs) > 3 {
		msgs = msgs[len(msgs)-3:]
	}
	return strings.Join(msgs, "; ")
}
