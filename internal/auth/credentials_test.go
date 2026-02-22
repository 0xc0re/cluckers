package auth

import (
	"os"
	"testing"
)

func TestSaveLoadCredentialsRoundTrip(t *testing.T) {
	// Use a temp dir to avoid touching real config.
	tmp := t.TempDir()
	t.Setenv("CLUCKERS_HOME", tmp)

	err := SaveCredentials("testuser", "secret123")
	if err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}

	creds, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials failed: %v", err)
	}
	if creds == nil {
		t.Fatal("LoadCredentials returned nil")
	}
	if creds.Username != "testuser" {
		t.Fatalf("username mismatch: got %q, want %q", creds.Username, "testuser")
	}
	if creds.Password != "secret123" {
		t.Fatalf("password mismatch: got %q, want %q", creds.Password, "secret123")
	}
}

func TestLoadCredentialsMissingFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLUCKERS_HOME", tmp)

	creds, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials should not error on missing file: %v", err)
	}
	if creds != nil {
		t.Fatal("LoadCredentials should return nil for missing file")
	}
}

func TestDeleteCredentials(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLUCKERS_HOME", tmp)

	// Save, then delete.
	if err := SaveCredentials("testuser", "secret123"); err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}

	if err := DeleteCredentials(); err != nil {
		t.Fatalf("DeleteCredentials failed: %v", err)
	}

	// File should be gone.
	creds, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials after delete failed: %v", err)
	}
	if creds != nil {
		t.Fatal("credentials should be nil after deletion")
	}
}

func TestDeleteCredentialsIdempotent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLUCKERS_HOME", tmp)

	// Delete non-existent file should not error.
	if err := DeleteCredentials(); err != nil {
		t.Fatalf("DeleteCredentials on non-existent should not error: %v", err)
	}
}

func TestSaveCredentialsFilePermissions(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLUCKERS_HOME", tmp)

	if err := SaveCredentials("testuser", "secret123"); err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}

	info, err := os.Stat(tmp + "/config/credentials.enc")
	if err != nil {
		t.Fatalf("stat credentials file: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Fatalf("credentials file permissions: got %04o, want 0600", perm)
	}
}
