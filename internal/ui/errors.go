package ui

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

// UserError is a user-facing error with optional technical detail and actionable suggestion.
type UserError struct {
	Message    string // User-friendly message (always shown).
	Detail     string // Technical detail (shown with -v).
	Suggestion string // Actionable hint (shown if non-empty).
	Err        error  // Wrapped underlying error.
}

// Error implements the error interface, returning the user-friendly message.
func (e *UserError) Error() string {
	return e.Message
}

// Unwrap returns the wrapped error for errors.Is/As compatibility.
func (e *UserError) Unwrap() error {
	return e.Err
}

// FormatError formats an error for display. If it is a UserError, the detail
// and suggestion are included based on verbose mode. Plain errors show their
// string representation.
func FormatError(err error, verbose bool) string {
	var ue *UserError
	if !errors.As(err, &ue) {
		return err.Error()
	}

	var b strings.Builder
	b.WriteString(ue.Message)

	if verbose && ue.Detail != "" {
		b.WriteString("\n  Detail: ")
		b.WriteString(ue.Detail)
	}
	if ue.Suggestion != "" {
		b.WriteString("\n  Suggestion: ")
		b.WriteString(ue.Suggestion)
	}
	return b.String()
}

// WineNotFoundError returns a UserError with per-distro Wine install instructions.
func WineNotFoundError() *UserError {
	distro := detectDistro()
	suggestion := wineInstallInstructions(distro)
	return &UserError{
		Message:    "Wine not found. Wine or Proton-GE is required to run Realm Royale.",
		Suggestion: suggestion,
	}
}

// detectDistro reads /etc/os-release and returns a distro identifier.
func detectDistro() string {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			return strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
		}
	}
	return ""
}

// wineInstallInstructions returns per-distro install instructions for Wine.
func wineInstallInstructions(distro string) string {
	switch distro {
	case "arch", "steamos":
		return "Install Wine: sudo pacman -S wine\n  Or install Proton-GE via ProtonUp-Qt for best compatibility."
	case "ubuntu", "debian", "linuxmint", "pop":
		return "Install Wine: sudo apt install wine\n  Or install Proton-GE via ProtonUp-Qt for best compatibility."
	case "fedora":
		return "Install Wine: sudo dnf install wine\n  Or install Proton-GE via ProtonUp-Qt for best compatibility."
	default:
		return fmt.Sprintf("Install Wine for your distribution, or install Proton-GE via ProtonUp-Qt.\n  See: https://github.com/DavidoTek/ProtonUp-Qt")
	}
}
