package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// PromptUsername prints "Username: " and reads a line from stdin.
// Returns a *UserError if stdin is not a terminal.
func PromptUsername() (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", &UserError{
			Message:    "Cannot prompt for username: no terminal available",
			Suggestion: "Run 'cluckers login' from a terminal first to save credentials.",
		}
	}

	fmt.Print("Username: ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read username: %w", err)
	}
	username := strings.TrimSpace(line)
	if username == "" {
		return "", fmt.Errorf("username cannot be empty")
	}
	return username, nil
}

// PromptPassword prints "Password: " and reads hidden input from stdin.
// Returns a *UserError if stdin is not a terminal.
func PromptPassword() (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", &UserError{
			Message:    "Cannot prompt for password: no terminal available",
			Suggestion: "Run 'cluckers login' from a terminal first to save credentials.",
		}
	}

	fmt.Print("Password: ")
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // ReadPassword does not echo a newline.
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	password := string(pw)
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}
	return password, nil
}

// PromptEmail prints "Email: " and reads a line from stdin.
// Returns a *UserError if stdin is not a terminal.
func PromptEmail() (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", &UserError{
			Message:    "Cannot prompt for email: no terminal available",
			Suggestion: "Run 'cluckers register' from a terminal first to create an account.",
		}
	}

	fmt.Print("Email: ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read email: %w", err)
	}
	email := strings.TrimSpace(line)
	if email == "" {
		return "", fmt.Errorf("email cannot be empty")
	}
	return email, nil
}
