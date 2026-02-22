package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// PromptUsername prints "Username: " and reads a line from stdin.
// Returns an error if stdin is not a terminal.
func PromptUsername() (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("cannot prompt for username: stdin is not a terminal")
	}

	fmt.Print("Username: ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read username: %w", err)
	}
	return strings.TrimSpace(line), nil
}

// PromptPassword prints "Password: " and reads hidden input from stdin.
// Returns an error if stdin is not a terminal.
func PromptPassword() (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("cannot prompt for password: stdin is not a terminal")
	}

	fmt.Print("Password: ")
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // ReadPassword does not echo a newline.
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	return string(pw), nil
}
