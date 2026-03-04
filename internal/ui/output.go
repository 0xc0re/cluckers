package ui

import (
	"fmt"

	"github.com/fatih/color"
)

var (
	greenBold = color.New(color.FgGreen, color.Bold)
	yellow    = color.New(color.FgYellow)
	redBold   = color.New(color.FgRed, color.Bold)
	dim       = color.New(color.Faint)
)

// Success prints a green checkmark with the given message.
func Success(msg string) {
	logInfo(msg)
	greenBold.Print("\u2713 ")
	fmt.Println(msg)
}

// Warn prints a yellow warning with the given message.
func Warn(msg string) {
	logWarn(msg)
	yellow.Print("\u26a0 ")
	fmt.Println(msg)
}

// Error prints a red cross with the given message.
func Error(msg string) {
	logError(msg)
	redBold.Print("\u2717 ")
	fmt.Println(msg)
}

// Info prints a plain informational message.
func Info(msg string) {
	logInfo(msg)
	fmt.Println(msg)
}

// Verbose prints a dimmed message only when verbose mode is enabled.
// The message is always written to the log file regardless of the -v flag.
func Verbose(msg string, isVerbose bool) {
	logDebug(msg)
	if !isVerbose {
		return
	}
	dim.Println(msg)
}
