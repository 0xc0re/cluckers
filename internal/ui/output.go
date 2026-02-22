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
	greenBold.Print("\u2713 ")
	fmt.Println(msg)
}

// Warn prints a yellow warning with the given message.
func Warn(msg string) {
	yellow.Print("\u26a0 ")
	fmt.Println(msg)
}

// Error prints a red cross with the given message.
func Error(msg string) {
	redBold.Print("\u2717 ")
	fmt.Println(msg)
}

// Info prints a plain informational message.
func Info(msg string) {
	fmt.Println(msg)
}

// Verbose prints a dimmed message only when verbose mode is enabled.
func Verbose(msg string, isVerbose bool) {
	if !isVerbose {
		return
	}
	dim.Println(msg)
}
