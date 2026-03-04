package ui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	logFileName    = "cluckers.log"
	logMaxBytes    = 5 * 1024 * 1024 // 5 MB
	logBackupSuffix = ".1"
)

var (
	logMu   sync.Mutex
	logFile *os.File
)

// InitLogging creates the log directory, rotates the log file if it exceeds
// 5 MB, and opens cluckers.log for append. Failures are non-fatal: all log
// calls silently no-op if the file cannot be opened.
func InitLogging(logDir string) {
	logMu.Lock()
	defer logMu.Unlock()

	if err := os.MkdirAll(logDir, 0700); err != nil {
		return
	}

	logPath := filepath.Join(logDir, logFileName)

	// Rotate if existing log exceeds max size.
	if info, err := os.Stat(logPath); err == nil && info.Size() > logMaxBytes {
		backupPath := logPath + logBackupSuffix
		_ = os.Remove(backupPath)
		_ = os.Rename(logPath, backupPath)
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return
	}
	logFile = f
}

// CloseLogging flushes and closes the log file.
func CloseLogging() {
	logMu.Lock()
	defer logMu.Unlock()

	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

// LogWriter returns the log file as an io.Writer, or io.Discard if logging
// is not initialised. Useful for teeing game stderr into the unified log.
func LogWriter() io.Writer {
	logMu.Lock()
	defer logMu.Unlock()

	if logFile != nil {
		return logFile
	}
	return io.Discard
}

// writeLog writes a timestamped, level-prefixed line to the log file.
func writeLog(level, msg string) {
	logMu.Lock()
	defer logMu.Unlock()

	if logFile == nil {
		return
	}
	ts := time.Now().Format("2006-01-02T15:04:05")
	fmt.Fprintf(logFile, "%s [%s] %s\n", ts, level, msg)
}

func logInfo(msg string)  { writeLog("INFO", msg) }
func logWarn(msg string)  { writeLog("WARN", msg) }
func logError(msg string) { writeLog("ERROR", msg) }
func logDebug(msg string) { writeLog("DEBUG", msg) }
