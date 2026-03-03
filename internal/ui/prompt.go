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
	username := strings.TrimSpace(line)
	if username == "" {
		return "", fmt.Errorf("username cannot be empty")
	}
	return username, nil
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
	password := string(pw)
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}
	return password, nil
}

// PromptEmail prints "Email: " and reads a line from stdin.
// Returns an error if stdin is not a terminal.
func PromptEmail() (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("cannot prompt for email: stdin is not a terminal")
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
