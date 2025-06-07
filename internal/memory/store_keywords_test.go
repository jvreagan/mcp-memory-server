// internal/memory/store_keywords_test.go
package memory

import (
	"os"
	"testing"
	"time"

	"mcp-memory-server/internal/config"
	"mcp-memory-server/pkg/logger"
)

func TestStoreKeywordExtraction(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "memory-test-keywords-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create store
	cfg := &config.StorageConfig{
		MaxStorageSize:    10 * 1024 * 1024, // 10MB
		MaxFileSize:       1 * 1024 * 1024,  // 1MB
		EnableAsync:       false,
		EnableCompression: false,
	}
	log := logger.New("info", "text")
	store, err := NewStore(tmpDir, cfg, log)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Test 1: Store memory with technical content
	content1 := "We're using Golang and PostgreSQL for the backend API. Frontend uses React with TypeScript."
	memory1, err := store.Store(content1, "Tech stack overview", "technical", []string{"stack", "architecture"}, nil)
	if err != nil {
		t.Fatalf("Failed to store memory: %v", err)
	}

	// Check keywords were extracted
	if len(memory1.Keywords) == 0 {
		t.Error("Expected keywords to be extracted, got none")
	}

	// Check for specific keywords
	hasGolang := false
	hasReact := false
	for _, kw := range memory1.Keywords {
		if kw == "golang" {
			hasGolang = true
		}
		if kw == "react" {
			hasReact = true
		}
	}
	if !hasGolang {
		t.Error("Expected 'golang' in keywords")
	}
	if !hasReact {
		t.Error("Expected 'react' in keywords")
	}

	// Test 2: Store memory with person names
	content2 := "John Smith and Sarah Johnson are working on the mcp-memory-server project."
	memory2, err := store.Store(content2, "Team update", "team", []string{"people"}, nil)
	if err != nil {
		t.Fatalf("Failed to store memory: %v", err)
	}

	hasPersonName := false
	hasProjectName := false
	for _, kw := range memory2.Keywords {
		if kw == "John Smith" || kw == "Sarah Johnson" {
			hasPersonName = true
		}
		if kw == "mcp-memory-server" {
			hasProjectName = true
		}
	}
	if !hasPersonName {
		t.Error("Expected person names in keywords")
	}
	if !hasProjectName {
		t.Error("Expected project name in keywords")
	}

	// Test 3: Search by keyword
	searchQuery := &SearchQuery{
		Query: "golang",
		Limit: 10,
	}
	results, err := store.Search(searchQuery)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected search results for 'golang'")
	}

	// Test 4: Get memories by keyword
	golangMemories, err := store.GetByKeyword("golang", 10)
	if err != nil {
		t.Fatalf("Failed to get by keyword: %v", err)
	}

	if len(golangMemories) != 1 {
		t.Errorf("Expected 1 memory with 'golang' keyword, got %d", len(golangMemories))
	}

	// Test 5: Get top keywords
	topKeywords := store.GetTopKeywords(5)
	if len(topKeywords) == 0 {
		t.Error("Expected top keywords")
	}

	// Test 6: Verify keyword index is updated
	stats := store.GetStats()
	if uniqueKeywords, ok := stats["unique_keywords"].(int); ok {
		if uniqueKeywords == 0 {
			t.Error("Expected unique keywords in stats")
		}
	} else {
		t.Error("unique_keywords not found in stats")
	}
}

func TestKeywordSearchRelevance(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "memory-test-relevance-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create store
	cfg := &config.StorageConfig{
		MaxStorageSize:    10 * 1024 * 1024,
		MaxFileSize:       1 * 1024 * 1024,
		EnableAsync:       false,
		EnableCompression: false,
	}
	log := logger.New("info", "text")
	store, err := NewStore(tmpDir, cfg, log)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Store memories with varying relevance
	store.Store("Python is great for data science", "Python overview", "language", nil, nil)
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	store.Store("Machine learning with Python and TensorFlow", "ML guide", "ml", nil, nil)
	time.Sleep(10 * time.Millisecond)
	store.Store("Web development with Django Python framework", "Web dev", "web", nil, nil)

	// Search for Python - should get all three, but ML guide should rank higher due to keywords
	results, err := store.Search(&SearchQuery{Query: "python machine learning", Limit: 10})
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}

	if len(results) < 2 {
		t.Fatalf("Expected at least 2 results, got %d", len(results))
	}

	// The ML guide should rank first due to matching both "python" and "machine learning" keywords
	if results[0].Summary != "ML guide" {
		t.Errorf("Expected ML guide to rank first, got %s", results[0].Summary)
	}
}