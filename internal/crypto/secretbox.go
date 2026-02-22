// Package crypto provides NaCl secretbox encryption with machine-id-derived keys
// for encrypting user credentials at rest.
package crypto

import (
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/denisbrodbeck/machineid"
	"golang.org/x/crypto/nacl/secretbox"
	"golang.org/x/crypto/scrypt"
)

const (
	// appSalt is a fixed salt for scrypt key derivation, scoped to this application.
	appSalt = "cluckers-credential-encryption-v1"

	// nonceSize is the size of the NaCl secretbox nonce in bytes.
	nonceSize = 24
)

// DeriveKey reads the machine ID and derives a 32-byte key using scrypt.
// The key is deterministic per machine: same machine always produces the same key.
func DeriveKey() ([32]byte, error) {
	var key [32]byte

	machineID, err := machineid.ID()
	if err != nil {
		return key, fmt.Errorf("read machine ID: %w", err)
	}

	derived, err := scrypt.Key([]byte(machineID), []byte(appSalt), 32768, 8, 1, 32)
	if err != nil {
		return key, fmt.Errorf("scrypt key derivation: %w", err)
	}

	copy(key[:], derived)
	return key, nil
}

// Encrypt seals plaintext using NaCl secretbox with a random 24-byte nonce.
// The returned ciphertext has the nonce prepended: [nonce (24 bytes) | sealed data].
func Encrypt(plaintext []byte, key *[32]byte) ([]byte, error) {
	var nonce [nonceSize]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	// secretbox.Seal appends the sealed message to the first argument (nonce[:]).
	sealed := secretbox.Seal(nonce[:], plaintext, &nonce, key)
	return sealed, nil
}

// Decrypt opens ciphertext produced by Encrypt. It extracts the 24-byte nonce
// from the front of the data and opens the secretbox.
func Decrypt(ciphertext []byte, key *[32]byte) ([]byte, error) {
	if len(ciphertext) < nonceSize+secretbox.Overhead {
		return nil, errors.New("ciphertext too short")
	}

	var nonce [nonceSize]byte
	copy(nonce[:], ciphertext[:nonceSize])

	plaintext, ok := secretbox.Open(nil, ciphertext[nonceSize:], &nonce, key)
	if !ok {
		return nil, errors.New("decryption failed (credentials may be from another machine)")
	}

	return plaintext, nil
}
