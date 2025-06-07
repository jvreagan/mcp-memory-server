// pkg/crypto/crypto.go
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	// KeySize is the size of the AES-256 key in bytes
	KeySize = 32
	// NonceSize is the size of the GCM nonce in bytes
	NonceSize = 12
)

// Crypto handles encryption and decryption using AES-256-GCM
type Crypto struct {
	key    []byte
	cipher cipher.AEAD
}

// New creates a new Crypto instance with the key from the specified file
func New(keyPath string) (*Crypto, error) {
	// Ensure directory exists
	dir := filepath.Dir(keyPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create key directory: %w", err)
	}

	// Load or generate key
	key, err := loadOrGenerateKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load or generate key: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM cipher: %w", err)
	}

	return &Crypto{
		key:    key,
		cipher: gcm,
	}, nil
}

// Encrypt encrypts the given data
func (c *Crypto) Encrypt(data []byte) ([]byte, error) {
	// Generate random nonce
	nonce := make([]byte, c.cipher.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and prepend nonce
	ciphertext := c.cipher.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// Decrypt decrypts the given data
func (c *Crypto) Decrypt(data []byte) ([]byte, error) {
	if len(data) < c.cipher.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := data[:c.cipher.NonceSize()], data[c.cipher.NonceSize():]

	// Decrypt
	plaintext, err := c.cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// EncryptString encrypts a string and returns base64 encoded result
func (c *Crypto) EncryptString(plaintext string) (string, error) {
	encrypted, err := c.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptString decrypts a base64 encoded string
func (c *Crypto) DecryptString(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	decrypted, err := c.Decrypt(data)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}

// loadOrGenerateKey loads an existing key or generates a new one
func loadOrGenerateKey(keyPath string) ([]byte, error) {
	// Try to load existing key
	if key, err := os.ReadFile(keyPath); err == nil {
		if len(key) != KeySize {
			return nil, fmt.Errorf("invalid key size: expected %d bytes, got %d", KeySize, len(key))
		}
		return key, nil
	}

	// Generate new key
	key := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// Save key with restricted permissions
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return nil, fmt.Errorf("failed to save key: %w", err)
	}

	return key, nil
}

// GetKey returns the encryption key (for sharing with other services)
func (c *Crypto) GetKey() []byte {
	keyCopy := make([]byte, len(c.key))
	copy(keyCopy, c.key)
	return keyCopy
}