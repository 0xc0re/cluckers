package config

import (
	"os"
	"path/filepath"
)

// ConfigDir returns the config directory under the data dir.
func ConfigDir() string {
	return filepath.Join(DataDir(), "config")
}

// CacheDir returns the cache directory under the data dir.
func CacheDir() string {
	return filepath.Join(DataDir(), "cache")
}

// ConfigFile returns the path to the TOML settings file.
func ConfigFile() string {
	return filepath.Join(ConfigDir(), "settings.toml")
}

// CredentialsFile returns the path to the encrypted credentials file.
func CredentialsFile() string {
	return filepath.Join(ConfigDir(), "credentials.enc")
}

// EnsureDir creates a directory with 0700 permissions if it does not exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0700)
}
