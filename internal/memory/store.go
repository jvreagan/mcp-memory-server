// internal/memory/store.go
package memory

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"mcp-memory-server/internal/config"
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
	dataDir       string
	config        *config.StorageConfig
	logger        *logger.Logger
	mu            sync.RWMutex
	index         map[string]*Memory  // In-memory index for fast access
	categoryIndex map[string][]string // category -> memory IDs
	tagIndex      map[string][]string // tag -> memory IDs
	totalSize     int64               // total storage size in bytes
	memorySizes   map[string]int64    // memory ID -> file size
	saveQueue     chan *Memory        // async save queue
	wg            sync.WaitGroup      // wait group for worker goroutines
	shutdownCh    chan struct{}       // shutdown signal channel
}

// NewStore creates a new memory store
func NewStore(dataDir string, cfg *config.StorageConfig, log *logger.Logger) (*Store, error) {
	store := &Store{
		dataDir:       dataDir,
		config:        cfg,
		logger:        log.WithComponent("memory_store"),
		index:         make(map[string]*Memory),
		categoryIndex: make(map[string][]string),
		tagIndex:      make(map[string][]string),
		memorySizes:   make(map[string]int64),
		saveQueue:     make(chan *Memory, cfg.QueueSize), // Configurable queue size
		shutdownCh:    make(chan struct{}),
	}

	// Start async save workers if enabled
	if cfg.EnableAsync {
		for i := 0; i < cfg.WorkerThreads; i++ {
			store.wg.Add(1)
			go store.saveWorker()
		}
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
		"memories_loaded", len(store.index),
		"total_size", store.totalSize,
		"max_storage_size", cfg.MaxStorageSize,
		"async_enabled", cfg.EnableAsync,
		"worker_threads", cfg.WorkerThreads,
		"queue_size", cfg.QueueSize,
		"compression_enabled", cfg.EnableCompression,
		"compression_level", cfg.CompressionLevel)

	return store, nil
}

// Store saves a memory (fast synchronous path)
func (s *Store) Store(content, summary, category string, tags []string, metadata map[string]string) (*Memory, error) {
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

	// Fast synchronous update of in-memory index
	s.mu.Lock()
	// Check if memory already exists
	if existing, exists := s.index[id]; exists {
		memory.CreatedAt = existing.CreatedAt
		memory.AccessCount = existing.AccessCount
		s.logger.Debug("Updating existing memory", "id", id)
	} else {
		s.logger.Debug("Storing new memory", "id", id, "category", category)
	}

	// Update in-memory index immediately (fast)
	s.index[id] = memory
	s.updateIndices(memory)
	s.mu.Unlock()

	// Save to file based on async configuration
	if s.config.EnableAsync {
		// Queue for async file save (slow operations in background)
		// Use a goroutine to handle potential channel closure during shutdown
		go func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Warn("Save queue closed during shutdown, saving synchronously", "id", id)
					s.saveMemoryAsync(memory)
				}
			}()
			
			select {
			case s.saveQueue <- memory:
				// Successfully queued
			default:
				// Queue is full, log warning but don't block
				s.logger.Warn("Save queue full, memory will be saved synchronously", "id", id)
				// Save synchronously in current goroutine
				s.saveMemoryAsync(memory)
			}
		}()
	} else {
		// Synchronous save
		fileSize, err := s.saveMemoryToFile(memory)
		if err != nil {
			return nil, fmt.Errorf("failed to save memory: %w", err)
		}
		
		// Update storage tracking
		s.mu.Lock()
		oldSize := s.memorySizes[id]
		s.totalSize = s.totalSize - oldSize + fileSize
		s.memorySizes[id] = fileSize
		needsCleanup := s.totalSize > s.config.MaxStorageSize
		s.mu.Unlock()
		
		// Clean up if over limit
		if needsCleanup {
			if err := s.cleanupOldMemories(); err != nil {
				s.logger.WithError(err).Warn("Failed to cleanup old memories")
			}
		}
	}

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
	if _, err := s.saveMemoryToFile(memory); err != nil {
		s.logger.WithError(err).Warn("Failed to update memory access stats")
	}

	s.logger.Debug("Retrieved memory", "id", id, "access_count", memory.AccessCount)
	return memory, nil
}

// Search searches for memories based on query
func (s *Store) Search(query *SearchQuery) ([]*Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type scoredMemory struct {
		memory *Memory
		score  float64
	}

	var results []scoredMemory
	queryLower := strings.ToLower(query.Query)

	// Use indices for faster search if category or tags are specified
	var candidateIDs map[string]bool
	if query.Category != "" {
		candidateIDs = make(map[string]bool)
		for _, id := range s.categoryIndex[query.Category] {
			candidateIDs[id] = true
		}
	}

	if len(query.Tags) > 0 {
		tagCandidates := make(map[string]bool)
		for _, tag := range query.Tags {
			for _, id := range s.tagIndex[strings.ToLower(tag)] {
				tagCandidates[id] = true
			}
		}
		if candidateIDs != nil {
			// Intersection of category and tag candidates
			for id := range candidateIDs {
				if !tagCandidates[id] {
					delete(candidateIDs, id)
				}
			}
		} else {
			candidateIDs = tagCandidates
		}
	}

	// Search through candidates or all memories
	for id, memory := range s.index {
		if candidateIDs != nil && !candidateIDs[id] {
			continue
		}
		score := s.calculateRelevanceScore(memory, query, queryLower)
		if score > 0 {
			results = append(results, scoredMemory{memory: memory, score: score})
		}
	}

	// Sort by relevance score
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	limit := query.Limit
	if limit == 0 || limit > 50 {
		limit = 20 // Default limit
	}

	var memories []*Memory
	for i, result := range results {
		if i >= limit {
			break
		}
		memories = append(memories, result.memory)
	}

	s.logger.Info("Search completed",
		"query", query.Query,
		"results", len(memories),
		"total_memories", len(s.index))

	return memories, nil
}

// List lists all memories with optional filtering
func (s *Store) List(category string, tags []string, limit int) ([]*Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*Memory

	// Use indices for faster filtering
	var candidateIDs map[string]bool
	if category != "" {
		candidateIDs = make(map[string]bool)
		for _, id := range s.categoryIndex[category] {
			candidateIDs[id] = true
		}
	}

	if len(tags) > 0 {
		tagCandidates := make(map[string]bool)
		for _, tag := range tags {
			for _, id := range s.tagIndex[strings.ToLower(tag)] {
				tagCandidates[id] = true
			}
		}
		if candidateIDs != nil {
			// Intersection
			for id := range candidateIDs {
				if !tagCandidates[id] {
					delete(candidateIDs, id)
				}
			}
		} else {
			candidateIDs = tagCandidates
		}
	}

	// Collect results
	if candidateIDs != nil {
		for id := range candidateIDs {
			if memory, exists := s.index[id]; exists {
				results = append(results, memory)
			}
		}
	} else {
		// No filters, return all
		for _, memory := range s.index {
			results = append(results, memory)
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

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

	// Update storage size
	s.totalSize -= s.memorySizes[id]
	delete(s.memorySizes, id)

	// Remove from indices
	memory := s.index[id]
	s.removeFromIndices(memory)
	delete(s.index, id)

	s.logger.Info("Memory deleted", "id", id)
	return nil
}

// Close gracefully shuts down the store
func (s *Store) Close() error {
	s.logger.Info("Closing memory store")
	
	// Only proceed with shutdown if async is enabled
	if !s.config.EnableAsync {
		s.logger.Info("Memory store closed (sync mode)")
		return nil
	}
	
	// Signal workers to start shutdown
	close(s.shutdownCh)
	
	// Wait a moment for workers to start draining
	time.Sleep(100 * time.Millisecond)
	
	// Close the save queue to prevent new saves
	close(s.saveQueue)
	
	// Log queue status
	queueLen := len(s.saveQueue)
	if queueLen > 0 {
		s.logger.Info("Waiting for pending saves to complete", "pending_saves", queueLen)
	}
	
	// Wait for all workers to finish with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		s.logger.Info("All save workers completed successfully")
	case <-time.After(30 * time.Second):
		s.logger.Warn("Timeout waiting for save workers to complete")
		return fmt.Errorf("timeout waiting for workers to complete")
	}
	
	s.logger.Info("Memory store closed successfully")
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
		"total_size":         s.totalSize,
		"max_storage_size":   s.config.MaxStorageSize,
		"storage_used_pct":   float64(s.totalSize) / float64(s.config.MaxStorageSize) * 100,
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

func (s *Store) saveMemoryToFile(memory *Memory) (int64, error) {
	var filename string
	if s.config.EnableCompression {
		filename = fmt.Sprintf("%s.json.gz", memory.ID)
	} else {
		filename = fmt.Sprintf("%s.json", memory.ID)
	}
	filepath := filepath.Join(s.dataDir, "memories", filename)

	data, err := json.Marshal(memory)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal memory: %w", err)
	}

	var fileData []byte
	if s.config.EnableCompression {
		// Compress data
		var compressed bytes.Buffer
		gzipWriter, err := gzip.NewWriterLevel(&compressed, s.config.CompressionLevel)
		if err != nil {
			return 0, fmt.Errorf("failed to create gzip writer: %w", err)
		}
		if _, err := gzipWriter.Write(data); err != nil {
			return 0, fmt.Errorf("failed to compress data: %w", err)
		}
		if err := gzipWriter.Close(); err != nil {
			return 0, fmt.Errorf("failed to close gzip writer: %w", err)
		}
		fileData = compressed.Bytes()
	} else {
		// Use uncompressed data
		fileData = data
	}

	// Check file size limit
	if int64(len(fileData)) > s.config.MaxFileSize {
		return 0, fmt.Errorf("memory file size %d exceeds limit %d", len(fileData), s.config.MaxFileSize)
	}

	// Atomic write
	tempFile := filepath + ".tmp"
	if err := os.WriteFile(tempFile, fileData, 0644); err != nil {
		return 0, fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tempFile, filepath); err != nil {
		os.Remove(tempFile)
		return 0, fmt.Errorf("failed to rename temp file: %w", err)
	}

	return int64(len(fileData)), nil
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
		if !strings.HasSuffix(entry.Name(), ".json.gz") && !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filepath := filepath.Join(memoriesDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			s.logger.WithError(err).Warn("Failed to get file info", "file", entry.Name())
			continue
		}

		data, err := os.ReadFile(filepath)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to read memory file", "file", entry.Name())
			continue
		}

		// Decompress if gzipped
		var jsonData []byte
		if strings.HasSuffix(entry.Name(), ".gz") {
			gzipReader, err := gzip.NewReader(bytes.NewReader(data))
			if err != nil {
				s.logger.WithError(err).Warn("Failed to create gzip reader", "file", entry.Name())
				continue
			}
			jsonData, err = io.ReadAll(gzipReader)
			gzipReader.Close()
			if err != nil {
				s.logger.WithError(err).Warn("Failed to decompress memory", "file", entry.Name())
				continue
			}
		} else {
			jsonData = data
		}

		var memory Memory
		if err := json.Unmarshal(jsonData, &memory); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal memory", "file", entry.Name())
			continue
		}

		s.index[memory.ID] = &memory
		s.memorySizes[memory.ID] = info.Size()
		s.totalSize += info.Size()
		s.updateIndices(&memory)
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

// updateIndices adds memory to category and tag indices
func (s *Store) updateIndices(memory *Memory) {
	// Update category index
	if memory.Category != "" {
		category := strings.ToLower(memory.Category)
		found := false
		for _, id := range s.categoryIndex[category] {
			if id == memory.ID {
				found = true
				break
			}
		}
		if !found {
			s.categoryIndex[category] = append(s.categoryIndex[category], memory.ID)
		}
	}

	// Update tag index
	for _, tag := range memory.Tags {
		tagKey := strings.ToLower(tag)
		found := false
		for _, id := range s.tagIndex[tagKey] {
			if id == memory.ID {
				found = true
				break
			}
		}
		if !found {
			s.tagIndex[tagKey] = append(s.tagIndex[tagKey], memory.ID)
		}
	}
}

// removeFromIndices removes memory from category and tag indices
func (s *Store) removeFromIndices(memory *Memory) {
	// Remove from category index
	if memory.Category != "" {
		category := strings.ToLower(memory.Category)
		ids := s.categoryIndex[category]
		for i, id := range ids {
			if id == memory.ID {
				s.categoryIndex[category] = append(ids[:i], ids[i+1:]...)
				break
			}
		}
		if len(s.categoryIndex[category]) == 0 {
			delete(s.categoryIndex, category)
		}
	}

	// Remove from tag index
	for _, tag := range memory.Tags {
		tagKey := strings.ToLower(tag)
		ids := s.tagIndex[tagKey]
		for i, id := range ids {
			if id == memory.ID {
				s.tagIndex[tagKey] = append(ids[:i], ids[i+1:]...)
				break
			}
		}
		if len(s.tagIndex[tagKey]) == 0 {
			delete(s.tagIndex, tagKey)
		}
	}
}

// cleanupOldMemories removes oldest memories to stay under storage limit
func (s *Store) cleanupOldMemories() error {
	// Sort memories by last access time (oldest first)
	type memoryWithTime struct {
		id         string
		lastAccess time.Time
		size       int64
	}

	var memories []memoryWithTime
	for id, memory := range s.index {
		memories = append(memories, memoryWithTime{
			id:         id,
			lastAccess: memory.LastAccess,
			size:       s.memorySizes[id],
		})
	}

	sort.Slice(memories, func(i, j int) bool {
		return memories[i].lastAccess.Before(memories[j].lastAccess)
	})

	// Remove memories until we're under the limit
	targetSize := int64(float64(s.config.MaxStorageSize) * 0.9) // Clean to 90% of limit
	for _, mem := range memories {
		if s.totalSize <= targetSize {
			break
		}

		if err := s.Delete(mem.id); err != nil {
			s.logger.WithError(err).Warn("Failed to delete memory during cleanup", "id", mem.id)
			continue
		}

		s.logger.Info("Cleaned up old memory", "id", mem.id, "size", mem.size, "last_access", mem.lastAccess)
	}

	return nil
}

// GetTimeline returns memory creation timeline data for charts
func (s *Store) GetTimeline() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Group memories by day for the last 30 days
	now := time.Now()
	days := make(map[string]int)
	labels := make([]string, 0, 30)
	data := make([]int, 0, 30)

	// Initialize last 30 days
	for i := 29; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		dayStr := day.Format("2006-01-02")
		days[dayStr] = 0
		labels = append(labels, day.Format("Jan 2"))
	}

	// Count memories per day
	for _, memory := range s.index {
		dayStr := memory.CreatedAt.Format("2006-01-02")
		if _, exists := days[dayStr]; exists {
			days[dayStr]++
		}
	}

	// Convert to array in chronological order
	for i := 29; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		dayStr := day.Format("2006-01-02")
		data = append(data, days[dayStr])
	}

	return map[string]interface{}{
		"labels": labels,
		"data":   data,
	}
}

// ReadOnlyStore provides read-only access to memory data for reporting
type ReadOnlyStore struct {
	dataDir string
	logger  *logger.Logger
	mu      sync.RWMutex
	index   map[string]*Memory
}

// NewReadOnlyStore creates a new read-only memory store for reporting
func NewReadOnlyStore(dataDir string, log *logger.Logger) (*ReadOnlyStore, error) {
	store := &ReadOnlyStore{
		dataDir: dataDir,
		logger:  log.WithComponent("readonly_memory_store"),
		index:   make(map[string]*Memory),
	}

	// Load existing memories into index
	if err := store.loadIndex(); err != nil {
		return nil, fmt.Errorf("failed to load memory index: %w", err)
	}

	store.logger.Info("Read-only memory store initialized",
		"data_dir", dataDir,
		"memories_loaded", len(store.index))

	return store, nil
}

// Refresh reloads the memory index from disk
func (s *ReadOnlyStore) Refresh() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear existing index
	s.index = make(map[string]*Memory)

	// Reload from disk
	return s.loadIndex()
}

// GetStats returns store statistics (read-only version)
func (s *ReadOnlyStore) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	categories := make(map[string]int)
	totalAccess := 0
	var totalSize int64

	for _, memory := range s.index {
		if memory.Category != "" {
			categories[memory.Category]++
		}
		totalAccess += memory.AccessCount
	}

	// Calculate approximate total size by examining files
	memoriesDir := filepath.Join(s.dataDir, "memories")
	if entries, err := os.ReadDir(memoriesDir); err == nil {
		for _, entry := range entries {
			if info, err := entry.Info(); err == nil {
				totalSize += info.Size()
			}
		}
	}

	return map[string]interface{}{
		"total_memories":     len(s.index),
		"categories":         categories,
		"total_access_count": totalAccess,
		"data_directory":     s.dataDir,
		"total_size":         totalSize,
		"storage_used_pct":   0, // We don't know the limit in read-only mode
	}
}

// List lists all memories with optional filtering (read-only version)
func (s *ReadOnlyStore) List(category string, tags []string, limit int) ([]*Memory, error) {
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

	// Sort by creation time (newest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// saveWorker processes the async save queue
func (s *Store) saveWorker() {
	defer s.wg.Done()
	
	for {
		select {
		case memory, ok := <-s.saveQueue:
			if !ok {
				// Channel closed, worker should exit
				s.logger.Debug("Save worker exiting - queue closed")
				return
			}
			s.saveMemoryAsync(memory)
		case <-s.shutdownCh:
			// Shutdown signal received, drain the queue
			s.logger.Debug("Save worker received shutdown signal, draining queue")
			for {
				select {
				case memory, ok := <-s.saveQueue:
					if !ok {
						s.logger.Debug("Save worker exiting - queue drained")
						return
					}
					s.saveMemoryAsync(memory)
				default:
					// Queue is empty
					s.logger.Debug("Save worker exiting - queue empty")
					return
				}
			}
		}
	}
}

// saveMemoryAsync handles the slow file operations asynchronously
func (s *Store) saveMemoryAsync(memory *Memory) {
	// Save to file (slow operation)
	fileSize, err := s.saveMemoryToFile(memory)
	if err != nil {
		s.logger.WithError(err).Error("Failed to save memory file asynchronously", "id", memory.ID)
		return
	}

	// Update storage tracking
	s.mu.Lock()
	oldSize := s.memorySizes[memory.ID]
	s.totalSize = s.totalSize - oldSize + fileSize
	s.memorySizes[memory.ID] = fileSize

	// Check if cleanup is needed
	needsCleanup := s.totalSize > s.config.MaxStorageSize
	s.mu.Unlock()

	// Clean up if over limit (slow operation)
	if needsCleanup {
		if err := s.cleanupOldMemories(); err != nil {
			s.logger.WithError(err).Warn("Failed to cleanup old memories")
		}
	}

	s.logger.Debug("Memory saved asynchronously", "id", memory.ID, "size", fileSize)
}

// GetTimeline returns memory creation timeline data for charts (read-only version)
func (s *ReadOnlyStore) GetTimeline() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Group memories by day for the last 30 days
	now := time.Now()
	days := make(map[string]int)
	labels := make([]string, 0, 30)
	data := make([]int, 0, 30)

	// Initialize last 30 days
	for i := 29; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		dayStr := day.Format("2006-01-02")
		days[dayStr] = 0
		labels = append(labels, day.Format("Jan 2"))
	}

	// Count memories per day
	for _, memory := range s.index {
		dayStr := memory.CreatedAt.Format("2006-01-02")
		if _, exists := days[dayStr]; exists {
			days[dayStr]++
		}
	}

	// Convert to array in chronological order
	for i := 29; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		dayStr := day.Format("2006-01-02")
		data = append(data, days[dayStr])
	}

	return map[string]interface{}{
		"labels": labels,
		"data":   data,
	}
}

// loadIndex loads memories from disk (read-only version)
func (s *ReadOnlyStore) loadIndex() error {
	memoriesDir := filepath.Join(s.dataDir, "memories")

	entries, err := os.ReadDir(memoriesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No memories directory yet
		}
		return fmt.Errorf("failed to read memories directory: %w", err)
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json.gz") && !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filepath := filepath.Join(memoriesDir, entry.Name())

		data, err := os.ReadFile(filepath)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to read memory file", "file", entry.Name())
			continue
		}

		// Decompress if gzipped
		var jsonData []byte
		if strings.HasSuffix(entry.Name(), ".gz") {
			gzipReader, err := gzip.NewReader(bytes.NewReader(data))
			if err != nil {
				s.logger.WithError(err).Warn("Failed to create gzip reader", "file", entry.Name())
				continue
			}
			jsonData, err = io.ReadAll(gzipReader)
			gzipReader.Close()
			if err != nil {
				s.logger.WithError(err).Warn("Failed to decompress memory", "file", entry.Name())
				continue
			}
		} else {
			jsonData = data
		}

		var memory Memory
		if err := json.Unmarshal(jsonData, &memory); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal memory", "file", entry.Name())
			continue
		}

		s.index[memory.ID] = &memory
	}

	return nil
}

// hasAnyTag checks if memory has any of the query tags (read-only version)
func (s *ReadOnlyStore) hasAnyTag(memoryTags, queryTags []string) bool {
	for _, queryTag := range queryTags {
		for _, memoryTag := range memoryTags {
			if strings.EqualFold(memoryTag, queryTag) {
				return true
			}
		}
	}
	return false
}
