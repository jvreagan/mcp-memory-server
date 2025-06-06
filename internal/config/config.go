// internal/config/config.go
package config

import (
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
			DataDir:        getEnvString("MCP_DATA_DIR", defaultDataDir),
			MaxFileSize:    getEnvInt64("MCP_MAX_FILE_SIZE", 100*1024*1024),         // 100MB
			MaxStorageSize: getEnvInt64("MCP_MAX_STORAGE_SIZE", 100*1024*1024*1024), // 100GB
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

	return cfg, nil
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
