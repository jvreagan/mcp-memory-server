// internal/config/config.go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	Storage StorageConfig `json:"storage"`
	Logging LoggingConfig `json:"logging"`
	Search  SearchConfig  `json:"search"`
	Web     WebConfig     `json:"web"`
}

// StorageConfig holds data storage configuration
type StorageConfig struct {
	DataDir        string `json:"data_dir"`
	MaxFileSize    int64  `json:"max_file_size"`    // bytes per file
	MaxStorageSize int64  `json:"max_storage_size"` // total storage limit in bytes
	
	// Async behavior configuration
	EnableAsync   bool `json:"enable_async"`    // Enable async save operations
	QueueSize     int  `json:"queue_size"`      // Size of async save queue
	WorkerThreads int  `json:"worker_threads"`  // Number of worker threads for async saves
	
	// Compression configuration
	EnableCompression bool   `json:"enable_compression"` // Enable gzip compression
	CompressionLevel  int    `json:"compression_level"`  // Gzip compression level (1-9)
	
	// Encryption configuration
	EnableEncryption  bool   `json:"enable_encryption"`  // Enable AES-256-GCM encryption
	EncryptionKeyPath string `json:"encryption_key_path"` // Path to encryption key file
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `json:"level"`  // "debug", "info", "warn", "error"
	Format string `json:"format"` // "json", "text"
}

// SearchConfig holds search configuration
type SearchConfig struct {
	EnableEmbeddings bool   `json:"enable_embeddings"`
	EmbeddingModel   string `json:"embedding_model"`
	MaxResults       int    `json:"max_results"`
}

// WebConfig holds web server configuration
type WebConfig struct {
	Enabled bool   `json:"enabled"`
	Port    int    `json:"port"`
	Host    string `json:"host"`
}

// Load loads configuration from environment variables with sensible defaults
func Load() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	defaultDataDir := filepath.Join(homeDir, ".mcp-memory")

	cfg := &Config{
		Storage: StorageConfig{
			DataDir:           getEnvString("MCP_DATA_DIR", defaultDataDir),
			MaxFileSize:       getEnvInt64("MCP_MAX_FILE_SIZE", 100*1024*1024),         // 100MB
			MaxStorageSize:    getEnvInt64("MCP_MAX_STORAGE_SIZE", 100*1024*1024*1024), // 100GB
			EnableAsync:       getEnvBool("MCP_ENABLE_ASYNC", true),                    // Async enabled by default
			QueueSize:         getEnvInt("MCP_QUEUE_SIZE", 1000),                       // Default queue size
			WorkerThreads:     getEnvInt("MCP_WORKER_THREADS", 2),                      // Default 2 workers
			EnableCompression: getEnvBool("MCP_ENABLE_COMPRESSION", true),              // Compression enabled by default
			CompressionLevel:  getEnvInt("MCP_COMPRESSION_LEVEL", 6),                   // Default gzip level (1-9, 6 is balanced)
			EnableEncryption:  getEnvBool("MCP_ENABLE_ENCRYPTION", false),              // Encryption disabled by default
			EncryptionKeyPath: getEnvString("MCP_ENCRYPTION_KEY_PATH", filepath.Join(homeDir, ".mcp-memory", "encryption.key")),
		},
		Logging: LoggingConfig{
			Level:  getEnvString("MCP_LOG_LEVEL", "info"),
			Format: getEnvString("MCP_LOG_FORMAT", "json"),
		},
		Search: SearchConfig{
			EnableEmbeddings: getEnvBool("MCP_ENABLE_EMBEDDINGS", false),
			EmbeddingModel:   getEnvString("MCP_EMBEDDING_MODEL", "text-embedding-ada-002"),
			MaxResults:       getEnvInt("MCP_MAX_RESULTS", 20),
		},
		Web: WebConfig{
			Enabled: getEnvBool("MCP_WEB_ENABLED", true),
			Port:    getEnvInt("MCP_WEB_PORT", 9000),
			Host:    getEnvString("MCP_WEB_HOST", "localhost"),
		},
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate validates the configuration values
func (c *Config) Validate() error {
	// Validate compression level
	if c.Storage.EnableCompression {
		if c.Storage.CompressionLevel < 1 || c.Storage.CompressionLevel > 9 {
			return fmt.Errorf("compression level must be between 1 and 9, got %d", c.Storage.CompressionLevel)
		}
	}
	
	// Validate encryption configuration
	if c.Storage.EnableEncryption && c.Storage.EncryptionKeyPath == "" {
		return fmt.Errorf("encryption key path must be specified when encryption is enabled")
	}
	
	// Validate queue size
	if c.Storage.EnableAsync && c.Storage.QueueSize < 1 {
		return fmt.Errorf("queue size must be at least 1 when async is enabled, got %d", c.Storage.QueueSize)
	}
	
	// Validate worker threads
	if c.Storage.EnableAsync && c.Storage.WorkerThreads < 1 {
		return fmt.Errorf("worker threads must be at least 1 when async is enabled, got %d", c.Storage.WorkerThreads)
	}
	
	// Validate storage limits
	if c.Storage.MaxFileSize <= 0 {
		return fmt.Errorf("max file size must be positive, got %d", c.Storage.MaxFileSize)
	}
	
	if c.Storage.MaxStorageSize <= 0 {
		return fmt.Errorf("max storage size must be positive, got %d", c.Storage.MaxStorageSize)
	}
	
	if c.Storage.MaxFileSize > c.Storage.MaxStorageSize {
		return fmt.Errorf("max file size (%d) cannot exceed max storage size (%d)", c.Storage.MaxFileSize, c.Storage.MaxStorageSize)
	}
	
	return nil
}

// Helper functions for environment variable parsing
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if str := os.Getenv(key); str != "" {
		if val, err := strconv.Atoi(str); err == nil {
			return val
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if str := os.Getenv(key); str != "" {
		if val, err := strconv.ParseInt(str, 10, 64); err == nil {
			return val
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if str := os.Getenv(key); str != "" {
		return str == "true" || str == "1"
	}
	return defaultValue
}
