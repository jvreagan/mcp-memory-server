// internal/memory/store.go
package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mcp-memory-server/pkg/logger"
)

// Memory represents a stored memory item
type Memory struct {
	ID          string            `json:"id"`
	Content     string            `json:"content"`
	Summary     string            `json:"summary,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Category    string            `json:"category,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	AccessCount int               `json:"access_count"`
	LastAccess  time.Time         `json:"last_access"`
}

// SearchQuery represents a search request
type SearchQuery struct {
	Query    string   `json:"query"`
	Tags     []string `json:"tags,omitempty"`
	Category string   `json:"category,omitempty"`
	Limit    int      `json:"limit,omitempty"`
}

// Store manages memory storage and retrieval
type Store struct {
	dataDir string
	logger  *logger.Logger
	mu      sync.RWMutex
	index   map[string]*Memory // In-memory index for fast access
}

// NewStore creates a new memory store
func NewStore(dataDir string, log *logger.Logger) (*Store, error) {
	store := &Store{
		dataDir: dataDir,
		logger:  log.WithComponent("memory_store"),
		index:   make(map[string]*Memory),
	}

	// Ensure directories exist
	if err := store.ensureDirectories(); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	// Load existing memories into index
	if err := store.loadIndex(); err != nil {
		return nil, fmt.Errorf("failed to load memory index: %w", err)
	}

	store.logger.Info("Memory store initialized",
		"data_dir", dataDir,
		"memories_loaded", len(store.index))

	return store, nil
}

// Store saves a memory
func (s *Store) Store(content, summary, category string, tags []string, metadata map[string]string) (*Memory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate ID from content hash
	id := s.generateID(content)
	now := time.Now()

	memory := &Memory{
		ID:          id,
		Content:     content,
		Summary:     summary,
		Tags:        tags,
		Category:    category,
		Metadata:    metadata,
		CreatedAt:   now,
		UpdatedAt:   now,
		AccessCount: 0,
		LastAccess:  now,
	}

	// Check if memory already exists
	if existing, exists := s.index[id]; exists {
		memory.CreatedAt = existing.CreatedAt
		memory.AccessCount = existing.AccessCount
		s.logger.Info("Updating existing memory", "id", id)
	} else {
		s.logger.Info("Storing new memory", "id", id, "category", category)
	}

	// Save to file
	if err := s.saveMemoryToFile(memory); err != nil {
		return nil, fmt.Errorf("failed to save memory to file: %w", err)
	}

	// Update index
	s.index[id] = memory

	return memory, nil
}

// Get retrieves a memory by ID
func (s *Store) Get(id string) (*Memory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	memory, exists := s.index[id]
	if !exists {
		return nil, fmt.Errorf("memory not found: %s", id)
	}

	// Update access statistics
	memory.AccessCount++
	memory.LastAccess = time.Now()

	// Save updated stats
	if err := s.saveMemoryToFile(memory); err != nil {
		s.logger.WithError(err).Warn("Failed to update memory access stats")
	}

	s.logger.Debug("Retrieved memory", "id", id, "access_count", memory.AccessCount)
	return memory, nil
}

// Search searches for memories based on query
func (s *Store) Search(query *SearchQuery) ([]*Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*Memory
	queryLower := strings.ToLower(query.Query)

	for _, memory := range s.index {
		score := s.calculateRelevanceScore(memory, query, queryLower)
		if score > 0 {
			results = append(results, memory)
		}
	}

	// Sort by relevance (simple scoring for now)
	// TODO: Implement proper ranking algorithm

	limit := query.Limit
	if limit == 0 || limit > 50 {
		limit = 20 // Default limit
	}

	if len(results) > limit {
		results = results[:limit]
	}

	s.logger.Info("Search completed",
		"query", query.Query,
		"results", len(results),
		"total_memories", len(s.index))

	return results, nil
}

// List lists all memories with optional filtering
func (s *Store) List(category string, tags []string, limit int) ([]*Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*Memory

	for _, memory := range s.index {
		// Filter by category
		if category != "" && memory.Category != category {
			continue
		}

		// Filter by tags
		if len(tags) > 0 && !s.hasAnyTag(memory.Tags, tags) {
			continue
		}

		results = append(results, memory)
	}

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// Delete removes a memory
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.index[id]; !exists {
		return fmt.Errorf("memory not found: %s", id)
	}

	// Remove file
	filename := fmt.Sprintf("%s.json", id)
	filepath := filepath.Join(s.dataDir, "memories", filename)
	if err := os.Remove(filepath); err != nil {
		return fmt.Errorf("failed to remove memory file: %w", err)
	}

	// Remove from index
	delete(s.index, id)

	s.logger.Info("Memory deleted", "id", id)
	return nil
}

// GetStats returns store statistics
func (s *Store) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	categories := make(map[string]int)
	totalAccess := 0

	for _, memory := range s.index {
		if memory.Category != "" {
			categories[memory.Category]++
		}
		totalAccess += memory.AccessCount
	}

	return map[string]interface{}{
		"total_memories":     len(s.index),
		"categories":         categories,
		"total_access_count": totalAccess,
		"data_directory":     s.dataDir,
	}
}

// Helper methods

func (s *Store) generateID(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])[:16] // Use first 16 chars
}

func (s *Store) ensureDirectories() error {
	dirs := []string{
		filepath.Join(s.dataDir, "memories"),
		filepath.Join(s.dataDir, "index"),
		filepath.Join(s.dataDir, "logs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func (s *Store) saveMemoryToFile(memory *Memory) error {
	filename := fmt.Sprintf("%s.json", memory.ID)
	filepath := filepath.Join(s.dataDir, "memories", filename)

	data, err := json.MarshalIndent(memory, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal memory: %w", err)
	}

	// Atomic write
	tempFile := filepath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tempFile, filepath); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

func (s *Store) loadIndex() error {
	memoriesDir := filepath.Join(s.dataDir, "memories")

	entries, err := os.ReadDir(memoriesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No memories directory yet
		}
		return fmt.Errorf("failed to read memories directory: %w", err)
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filepath := filepath.Join(memoriesDir, entry.Name())
		data, err := os.ReadFile(filepath)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to read memory file", "file", entry.Name())
			continue
		}

		var memory Memory
		if err := json.Unmarshal(data, &memory); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal memory", "file", entry.Name())
			continue
		}

		s.index[memory.ID] = &memory
	}

	return nil
}

func (s *Store) calculateRelevanceScore(memory *Memory, query *SearchQuery, queryLower string) float64 {
	score := 0.0

	// Content matching
	if strings.Contains(strings.ToLower(memory.Content), queryLower) {
		score += 1.0
	}

	// Summary matching
	if memory.Summary != "" && strings.Contains(strings.ToLower(memory.Summary), queryLower) {
		score += 0.8
	}

	// Category matching
	if query.Category != "" && memory.Category == query.Category {
		score += 0.5
	}

	// Tag matching
	if len(query.Tags) > 0 && s.hasAnyTag(memory.Tags, query.Tags) {
		score += 0.3
	}

	// Recent access boost
	if time.Since(memory.LastAccess) < 24*time.Hour {
		score += 0.1
	}

	return score
}

func (s *Store) hasAnyTag(memoryTags, queryTags []string) bool {
	for _, queryTag := range queryTags {
		for _, memoryTag := range memoryTags {
			if strings.EqualFold(memoryTag, queryTag) {
				return true
			}
		}
	}
	return false
}
