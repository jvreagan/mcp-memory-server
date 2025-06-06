package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mcp-memory-server/internal/config"
	"mcp-memory-server/pkg/logger"
)

func TestStoreGracefulShutdown(t *testing.T) {
	// Create temporary directory for test data
	tempDir, err := os.MkdirTemp("", "memory-store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test configuration with async enabled
	cfg := &config.StorageConfig{
		DataDir:           tempDir,
		MaxFileSize:       10 * 1024 * 1024,  // 10MB
		MaxStorageSize:    100 * 1024 * 1024, // 100MB
		EnableAsync:       true,
		QueueSize:         100,
		WorkerThreads:     2,
		EnableCompression: false,
	}

	// Create logger
	log := logger.New("debug", "text")

	// Create store
	store, err := NewStore(tempDir, cfg, log)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Store some memories to ensure workers are active
	for i := 0; i < 10; i++ {
		content := fmt.Sprintf("Test memory content %d", i)
		_, err := store.Store(content, "Test summary", "test", []string{"test"}, nil)
		if err != nil {
			t.Errorf("Failed to store memory %d: %v", i, err)
		}
	}

	// Give workers a moment to start processing
	time.Sleep(100 * time.Millisecond)

	// Close the store
	err = store.Close()
	if err != nil {
		t.Errorf("Failed to close store: %v", err)
	}

	// Verify all memories were saved
	memoriesDir := filepath.Join(tempDir, "memories")
	files, err := os.ReadDir(memoriesDir)
	if err != nil {
		t.Fatalf("Failed to read memories directory: %v", err)
	}

	if len(files) != 10 {
		t.Errorf("Expected 10 memory files, got %d", len(files))
	}
}

func TestStoreCloseWithPendingSaves(t *testing.T) {
	// Create temporary directory for test data
	tempDir, err := os.MkdirTemp("", "memory-store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test configuration with async enabled and small queue
	cfg := &config.StorageConfig{
		DataDir:           tempDir,
		MaxFileSize:       10 * 1024 * 1024,  // 10MB
		MaxStorageSize:    100 * 1024 * 1024, // 100MB
		EnableAsync:       true,
		QueueSize:         5,  // Small queue to test overflow
		WorkerThreads:     1,  // Single worker to slow processing
		EnableCompression: false,
	}

	// Create logger
	log := logger.New("debug", "text")

	// Create store
	store, err := NewStore(tempDir, cfg, log)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Store many memories quickly to fill the queue
	for i := 0; i < 20; i++ {
		content := fmt.Sprintf("Test memory content with more data to ensure save takes time %d %s", i, time.Now().String())
		go func(idx int) {
			_, err := store.Store(content, "Test summary", "test", []string{"test"}, nil)
			if err != nil {
				t.Logf("Failed to store memory %d: %v", idx, err)
			}
		}(i)
	}

	// Give stores a moment to queue up
	time.Sleep(200 * time.Millisecond)

	// Close the store - should wait for pending saves
	start := time.Now()
	err = store.Close()
	duration := time.Since(start)
	
	if err != nil {
		t.Errorf("Failed to close store: %v", err)
	}

	// Verify close waited for saves (should take some time)
	if duration < 100*time.Millisecond {
		t.Logf("Close completed very quickly (%v), might not have waited for saves", duration)
	}

	t.Logf("Store close took %v", duration)
}

func TestStoreSyncModeClose(t *testing.T) {
	// Create temporary directory for test data
	tempDir, err := os.MkdirTemp("", "memory-store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test configuration with async disabled
	cfg := &config.StorageConfig{
		DataDir:           tempDir,
		MaxFileSize:       10 * 1024 * 1024,  // 10MB
		MaxStorageSize:    100 * 1024 * 1024, // 100MB
		EnableAsync:       false, // Sync mode
		EnableCompression: false,
	}

	// Create logger
	log := logger.New("debug", "text")

	// Create store
	store, err := NewStore(tempDir, cfg, log)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Store a memory in sync mode
	_, err = store.Store("Test content", "Test summary", "test", []string{"test"}, nil)
	if err != nil {
		t.Errorf("Failed to store memory: %v", err)
	}

	// Close should be immediate in sync mode
	start := time.Now()
	err = store.Close()
	duration := time.Since(start)
	
	if err != nil {
		t.Errorf("Failed to close store: %v", err)
	}

	// Verify close was quick (sync mode has nothing to wait for)
	if duration > 100*time.Millisecond {
		t.Errorf("Close took too long in sync mode: %v", duration)
	}
}