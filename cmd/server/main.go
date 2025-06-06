// cmd/server/main.go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mcp-memory-server/internal/config"
	"mcp-memory-server/internal/mcp"
	"mcp-memory-server/internal/memory"
	"mcp-memory-server/pkg/logger"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger := logger.New(cfg.Logging.Level, cfg.Logging.Format)
	logger.Info("Starting MCP Memory Server", "version", "1.0.0")

	// Initialize memory store
	memoryStore, err := memory.NewStore(cfg.Storage.DataDir, &cfg.Storage, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize memory store")
	}

	// Initialize MCP server
	mcpServer := mcp.NewServer(memoryStore, logger)

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Channel to signal when shutdown is complete
	shutdownComplete := make(chan struct{})

	go func() {
		<-sigChan
		logger.Info("Shutdown signal received")
		
		// Cancel the context to stop the MCP server
		cancel()
		
		// Close the memory store to ensure all pending saves complete
		if err := memoryStore.Close(); err != nil {
			logger.WithError(err).Error("Error closing memory store")
		}
		
		close(shutdownComplete)
	}()

	// Start MCP server
	logger.Info("MCP Memory Server ready", "data_dir", cfg.Storage.DataDir)
	if err := mcpServer.Run(ctx); err != nil {
		logger.WithError(err).Fatal("MCP server failed")
	}

	// Wait for shutdown to complete
	select {
	case <-shutdownComplete:
		logger.Info("Shutdown complete")
	case <-time.After(35 * time.Second):
		logger.Error("Shutdown timeout - forcing exit")
	}

	logger.Info("MCP Memory Server stopped")
}
