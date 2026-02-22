// Package auth provides gateway authentication and credential management.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xc0re/cluckers/internal/config"
	"github.com/0xc0re/cluckers/internal/crypto"
)

// Credentials holds a user's login credentials for encrypted persistence.
type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// SaveCredentials encrypts and writes credentials to disk.
// The file is written to config.CredentialsFile() with 0600 permissions.
func SaveCredentials(username, password string) error {
	creds := Credentials{Username: username, Password: password}

	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	key, err := crypto.DeriveKey()
	if err != nil {
		return fmt.Errorf("derive encryption key: %w", err)
	}

	encrypted, err := crypto.Encrypt(data, &key)
	if err != nil {
		return fmt.Errorf("encrypt credentials: %w", err)
	}

	path := config.CredentialsFile()
	if err := config.EnsureDir(filepath.Dir(path)); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	if err := os.WriteFile(path, encrypted, 0600); err != nil {
		return fmt.Errorf("write credentials file: %w", err)
	}

	return nil
}

// LoadCredentials reads and decrypts credentials from disk.
// Returns nil, nil if the credentials file does not exist (first-time user).
// Returns an error if decryption fails (e.g., credentials from another machine).
func LoadCredentials() (*Credentials, error) {
	data, err := os.ReadFile(config.CredentialsFile())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil // First-time user — no saved credentials.
		}
		return nil, fmt.Errorf("read credentials file: %w", err)
	}

	key, err := crypto.DeriveKey()
	if err != nil {
		return nil, fmt.Errorf("derive encryption key: %w", err)
	}

	plaintext, err := crypto.Decrypt(data, &key)
	if err != nil {
		return nil, fmt.Errorf("decrypt credentials: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(plaintext, &creds); err != nil {
		return nil, fmt.Errorf("unmarshal credentials: %w", err)
	}

	return &creds, nil
}

// DeleteCredentials removes the encrypted credentials file.
// Returns nil if the file does not exist (idempotent).
func DeleteCredentials() error {
	err := os.Remove(config.CredentialsFile())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove credentials file: %w", err)
	}
	return nil
}
