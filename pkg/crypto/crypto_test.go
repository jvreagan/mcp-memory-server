// pkg/crypto/crypto_test.go
package crypto

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCrypto(t *testing.T) {
	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "crypto-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	keyPath := filepath.Join(tempDir, "test.key")

	// Test creating new crypto instance
	crypto, err := New(keyPath)
	if err != nil {
		t.Fatalf("Failed to create crypto: %v", err)
	}

	// Test that key file was created
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error("Key file was not created")
	}

	// Test encryption and decryption
	testData := []byte("This is a test message for encryption")
	
	encrypted, err := crypto.Encrypt(testData)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Encrypted data should be different from original
	if bytes.Equal(encrypted, testData) {
		t.Error("Encrypted data is the same as original")
	}

	// Test decryption
	decrypted, err := crypto.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	// Decrypted data should match original
	if !bytes.Equal(decrypted, testData) {
		t.Errorf("Decrypted data doesn't match original. Got: %s, Want: %s", decrypted, testData)
	}

	// Test string encryption
	testString := "Hello, encrypted world!"
	encryptedStr, err := crypto.EncryptString(testString)
	if err != nil {
		t.Fatalf("Failed to encrypt string: %v", err)
	}

	decryptedStr, err := crypto.DecryptString(encryptedStr)
	if err != nil {
		t.Fatalf("Failed to decrypt string: %v", err)
	}

	if decryptedStr != testString {
		t.Errorf("Decrypted string doesn't match. Got: %s, Want: %s", decryptedStr, testString)
	}

	// Test loading existing key
	crypto2, err := New(keyPath)
	if err != nil {
		t.Fatalf("Failed to load existing key: %v", err)
	}

	// Should be able to decrypt with the loaded key
	decrypted2, err := crypto2.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt with loaded key: %v", err)
	}

	if !bytes.Equal(decrypted2, testData) {
		t.Error("Decryption with loaded key failed")
	}
}

func TestInvalidDecryption(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "crypto-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	crypto, err := New(filepath.Join(tempDir, "test.key"))
	if err != nil {
		t.Fatalf("Failed to create crypto: %v", err)
	}

	// Test decrypting invalid data
	invalidData := []byte("not encrypted data")
	_, err = crypto.Decrypt(invalidData)
	if err == nil {
		t.Error("Expected error when decrypting invalid data")
	}

	// Test decrypting too short data
	shortData := []byte("short")
	_, err = crypto.Decrypt(shortData)
	if err == nil {
		t.Error("Expected error when decrypting short data")
	}
}