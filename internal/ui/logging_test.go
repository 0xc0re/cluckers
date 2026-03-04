package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitLogging(t *testing.T) {
	dir := t.TempDir()
	InitLogging(dir)
	t.Cleanup(CloseLogging)

	logInfo("hello from test")
	CloseLogging()

	data, err := os.ReadFile(filepath.Join(dir, logFileName))
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(data), "[INFO] hello from test") {
		t.Errorf("log content missing expected entry, got: %s", data)
	}
}

func TestLogRotation(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, logFileName)

	// Create a file that exceeds the rotation threshold.
	bigData := make([]byte, logMaxBytes+1)
	for i := range bigData {
		bigData[i] = 'x'
	}
	if err := os.WriteFile(logPath, bigData, 0600); err != nil {
		t.Fatalf("write big log: %v", err)
	}

	InitLogging(dir)
	t.Cleanup(CloseLogging)

	// Backup should exist.
	backupPath := logPath + logBackupSuffix
	if _, err := os.Stat(backupPath); err != nil {
		t.Errorf("backup file not created: %v", err)
	}

	// New log should be small (just opened, no content yet from rotation).
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("stat new log: %v", err)
	}
	if info.Size() > logMaxBytes {
		t.Errorf("new log too large after rotation: %d bytes", info.Size())
	}
}

func TestNonFatalInit(t *testing.T) {
	// Pass a path that cannot be created (file exists as regular file).
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0600); err != nil {
		t.Fatalf("create blocker: %v", err)
	}
	// Try to use the file as a directory — should fail silently.
	InitLogging(filepath.Join(blocker, "logs"))
	t.Cleanup(CloseLogging)

	// UI functions must not panic.
	Success("test")
	Warn("test")
	Error("test")
	Info("test")
	Verbose("test", false)
}

func TestLogWriter(t *testing.T) {
	dir := t.TempDir()
	InitLogging(dir)
	t.Cleanup(CloseLogging)

	w := LogWriter()
	if _, err := w.Write([]byte("game stderr output\n")); err != nil {
		t.Errorf("LogWriter write failed: %v", err)
	}
	CloseLogging()

	data, err := os.ReadFile(filepath.Join(dir, logFileName))
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(data), "game stderr output") {
		t.Errorf("LogWriter content not in log file, got: %s", data)
	}
}

func TestVerboseAlwaysLogs(t *testing.T) {
	dir := t.TempDir()
	InitLogging(dir)
	t.Cleanup(CloseLogging)

	// isVerbose=false should still log to file.
	Verbose("secret debug info", false)
	CloseLogging()

	data, err := os.ReadFile(filepath.Join(dir, logFileName))
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(data), "[DEBUG] secret debug info") {
		t.Errorf("Verbose(false) did not log, got: %s", data)
	}
}
