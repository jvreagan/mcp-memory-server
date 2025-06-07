// internal/memory/store_encryption_test.go
package memory

import (
	"os"
	"path/filepath"
	"testing"

	"mcp-memory-server/internal/config"
	"mcp-memory-server/pkg/logger"
)

func TestStoreWithEncryption(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "memory-encryption-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create logger
	log := logger.New("debug", "text")

	// Test configuration with encryption enabled
	cfg := &config.StorageConfig{
		DataDir:           tempDir,
		MaxFileSize:       1024 * 1024,
		MaxStorageSize:    10 * 1024 * 1024,
		EnableAsync:       false,
		EnableCompression: true,
		CompressionLevel:  6,
		EnableEncryption:  true,
		EncryptionKeyPath: filepath.Join(tempDir, "test.key"),
	}

	// Create store with encryption
	store, err := NewStore(tempDir, cfg, log)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Store a memory
	content := "This is a test memory that should be encrypted"
	summary := "Test encrypted memory"
	category := "test"
	tags := []string{"encryption", "test"}

	memory, err := store.Store(content, summary, category, tags, nil)
	if err != nil {
		t.Fatalf("Failed to store memory: %v", err)
	}

	// Verify the memory was stored
	retrieved, err := store.Get(memory.ID)
	if err != nil {
		t.Fatalf("Failed to get memory: %v", err)
	}

	if retrieved.Content != content {
		t.Errorf("Retrieved content doesn't match. Got: %s, Want: %s", retrieved.Content, content)
	}

	// Close the store to ensure files are written
	store.Close()

	// Create a new store instance to test loading encrypted memories
	store2, err := NewStore(tempDir, cfg, log)
	if err != nil {
		t.Fatalf("Failed to create second store: %v", err)
	}
	defer store2.Close()

	// Verify the memory can be loaded from encrypted file
	retrieved2, err := store2.Get(memory.ID)
	if err != nil {
		t.Fatalf("Failed to get memory from second store: %v", err)
	}

	if retrieved2.Content != content {
		t.Errorf("Retrieved content from second store doesn't match. Got: %s, Want: %s", retrieved2.Content, content)
	}

	// Test that without the correct key, data cannot be read
	wrongKeyCfg := &config.StorageConfig{
		DataDir:           tempDir,
		MaxFileSize:       1024 * 1024,
		MaxStorageSize:    10 * 1024 * 1024,
		EnableAsync:       false,
		EnableCompression: true,
		CompressionLevel:  6,
		EnableEncryption:  true,
		EncryptionKeyPath: filepath.Join(tempDir, "wrong.key"),
	}

	// This should create a new key, so loading should fail
	store3, err := NewStore(tempDir, wrongKeyCfg, log)
	if err != nil {
		t.Fatalf("Failed to create store with wrong key: %v", err)
	}
	defer store3.Close()

	// The store should load but have no memories (decryption will fail)
	stats := store3.GetStats()
	totalMemories := stats["total_memories"].(int)
	if totalMemories != 0 {
		t.Errorf("Expected 0 memories with wrong key, got %d", totalMemories)
	}
}

func TestReadOnlyStoreWithEncryption(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "memory-readonly-encryption-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create logger
	log := logger.New("debug", "text")

	// Test configuration with encryption enabled
	cfg := &config.StorageConfig{
		DataDir:           tempDir,
		MaxFileSize:       1024 * 1024,
		MaxStorageSize:    10 * 1024 * 1024,
		EnableAsync:       false,
		EnableCompression: true,
		CompressionLevel:  6,
		EnableEncryption:  true,
		EncryptionKeyPath: filepath.Join(tempDir, "test.key"),
	}

	// First create a store and add some encrypted memories
	store, err := NewStore(tempDir, cfg, log)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Store a memory
	content := "This is a test memory for read-only access"
	summary := "Test read-only encrypted memory"
	category := "readonly-test"
	tags := []string{"encryption", "readonly"}

	_, err = store.Store(content, summary, category, tags, nil)
	if err != nil {
		t.Fatalf("Failed to store memory: %v", err)
	}
	store.Close()

	// Now create a read-only store with the same encryption config
	roStore, err := NewReadOnlyStoreWithConfig(tempDir, cfg, log)
	if err != nil {
		t.Fatalf("Failed to create read-only store: %v", err)
	}

	// Verify we can list the encrypted memories
	memories, err := roStore.List("", nil, 10)
	if err != nil {
		t.Fatalf("Failed to list memories: %v", err)
	}

	if len(memories) != 1 {
		t.Errorf("Expected 1 memory, got %d", len(memories))
	}

	if memories[0].Content != content {
		t.Errorf("Content doesn't match. Got: %s, Want: %s", memories[0].Content, content)
	}

	// Test stats
	stats := roStore.GetStats()
	totalMemories := stats["total_memories"].(int)
	if totalMemories != 1 {
		t.Errorf("Expected 1 memory in stats, got %d", totalMemories)
	}
}