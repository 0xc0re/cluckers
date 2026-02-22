package crypto

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	// Use a fixed key for testing (not machine-id derived).
	var key [32]byte
	copy(key[:], []byte("test-key-32-bytes-long-padding!!"))

	plaintext := []byte(`{"username":"testuser","password":"secret123"}`)

	encrypted, err := Encrypt(plaintext, &key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Encrypted data must be longer than plaintext (nonce + overhead).
	if len(encrypted) <= len(plaintext) {
		t.Fatalf("encrypted data too short: got %d, plaintext %d", len(encrypted), len(plaintext))
	}

	decrypted, err := Decrypt(encrypted, &key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("round-trip mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	var key1, key2 [32]byte
	copy(key1[:], []byte("key-one-32-bytes-long-padding!!!"))
	copy(key2[:], []byte("key-two-32-bytes-long-padding!!!"))

	plaintext := []byte("secret data")

	encrypted, err := Encrypt(plaintext, &key1)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = Decrypt(encrypted, &key2)
	if err == nil {
		t.Fatal("Decrypt with wrong key should fail")
	}
	if err.Error() != "decryption failed (credentials may be from another machine)" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestDecryptTooShort(t *testing.T) {
	var key [32]byte
	copy(key[:], []byte("test-key-32-bytes-long-padding!!"))

	_, err := Decrypt([]byte("short"), &key)
	if err == nil {
		t.Fatal("Decrypt with too-short data should fail")
	}
}

func TestEncryptUniqueNonces(t *testing.T) {
	var key [32]byte
	copy(key[:], []byte("test-key-32-bytes-long-padding!!"))

	plaintext := []byte("same input")

	e1, err := Encrypt(plaintext, &key)
	if err != nil {
		t.Fatalf("Encrypt 1 failed: %v", err)
	}

	e2, err := Encrypt(plaintext, &key)
	if err != nil {
		t.Fatalf("Encrypt 2 failed: %v", err)
	}

	// Random nonces mean same plaintext should produce different ciphertext.
	if bytes.Equal(e1, e2) {
		t.Fatal("two encryptions of same plaintext should differ (unique nonces)")
	}
}
