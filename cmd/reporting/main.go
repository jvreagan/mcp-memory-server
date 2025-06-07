// cmd/reporting/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"mcp-memory-server/internal/config"
	"mcp-memory-server/internal/memory"
	"mcp-memory-server/internal/reporting"
	"mcp-memory-server/pkg/logger"
)

func main() {
	// Parse command line flags
	port := flag.Int("port", 9000, "Web server port")
	host := flag.String("host", "localhost", "Web server host")
	dataDir := flag.String("data-dir", "", "MCP memory data directory (auto-detected if not specified)")
	flag.Parse()

	// Load configuration to get default data directory
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Use provided data directory or default from config
	memoryDataDir := *dataDir
	if memoryDataDir == "" {
		memoryDataDir = cfg.Storage.DataDir
	}

	// Initialize logger
	logger := logger.New("info", "text")
	logger.Info("Starting MCP Memory Reporting Server", 
		"version", "1.0.0",
		"data_dir", memoryDataDir,
		"port", *port)

	// Initialize read-only memory store with encryption config if enabled
	memoryStore, err := memory.NewReadOnlyStoreWithConfig(memoryDataDir, &cfg.Storage, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize read-only memory store")
	}

	// Initialize reporting server
	reportingServer := reporting.NewServer(*host, *port, memoryStore, logger)

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Shutdown signal received")
		cancel()
	}()

	// Start reporting server
	logger.Info("Memory reporting dashboard available", "url", fmt.Sprintf("http://%s:%d", *host, *port))
	
	if err := reportingServer.Start(ctx); err != nil && err != http.ErrServerClosed {
		logger.WithError(err).Fatal("Reporting server failed")
	}

	logger.Info("MCP Memory Reporting Server stopped")
}