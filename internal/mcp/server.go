// internal/mcp/server.go
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"mcp-memory-server/internal/memory"
	"mcp-memory-server/pkg/logger"
)

// MCPRequest represents an MCP protocol request
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// MCPResponse represents an MCP protocol response
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an MCP protocol error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Server implements the MCP protocol for memory operations
type Server struct {
	store  *memory.Store
	logger *logger.Logger
}

// NewServer creates a new MCP server
func NewServer(store *memory.Store, logger *logger.Logger) *Server {
	return &Server{
		store:  store,
		logger: logger.WithComponent("mcp_server"),
	}
}

// Run starts the MCP server and handles requests
func (s *Server) Run(ctx context.Context) error {
	s.logger.Info("MCP server starting")

	// Don't send server info on startup - wait for initialize
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			s.logger.Info("MCP server shutting down")
			return nil
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		s.logger.Debug("Received request", "request", line)

		if err := s.handleRequest(line); err != nil {
			s.logger.WithError(err).Error("Failed to handle request")
			// Send error response with proper ID
			s.sendError(nil, -32603, "Internal error", err.Error())
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("error reading input: %w", err)
	}

	return nil
}

// handleRequest processes an MCP request
func (s *Server) handleRequest(requestLine string) error {
	var req MCPRequest
	if err := json.Unmarshal([]byte(requestLine), &req); err != nil {
		return s.sendError(nil, -32700, "Parse error", "Invalid JSON")
	}

	s.logger.Debug("Handling MCP request", "method", req.Method, "id", req.ID)

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	case "resources/list":
		return s.handleResourcesList(req)
	case "resources/read":
		return s.handleResourcesRead(req)
	default:
		return s.sendError(req.ID, -32601, "Method not found", fmt.Sprintf("Unknown method: %s", req.Method))
	}
}

// handleInitialize handles the MCP initialize method
func (s *Server) handleInitialize(req MCPRequest) error {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": false,
			},
			"resources": map[string]interface{}{
				"subscribe":   false,
				"listChanged": false,
			},
		},
		"serverInfo": map[string]interface{}{
			"name":    "memory-server",
			"version": "1.0.0",
		},
	}

	return s.sendResponse(req.ID, result)
}

// handleToolsList returns available tools
func (s *Server) handleToolsList(req MCPRequest) error {
	tools := []map[string]interface{}{
		{
			"name":        "remember",
			"description": "Store information in memory with optional categorization and tags",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The content to remember",
					},
					"summary": map[string]interface{}{
						"type":        "string",
						"description": "Optional summary of the content",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Optional category (e.g., 'code', 'concept', 'project')",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Optional tags for categorization",
					},
				},
				"required": []string{"content"},
			},
		},
		{
			"name":        "recall",
			"description": "Search for stored memories",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Optional category filter",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Optional tags filter",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results (default: 10)",
						"default":     10,
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "forget",
			"description": "Delete a stored memory by ID",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Memory ID to delete",
					},
				},
				"required": []string{"id"},
			},
		},
		{
			"name":        "list_memories",
			"description": "List all stored memories with optional filtering",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Optional category filter",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Optional tags filter",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results",
						"default":     20,
					},
				},
			},
		},
		{
			"name":        "memory_stats",
			"description": "Get statistics about stored memories",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}

	result := map[string]interface{}{
		"tools": tools,
	}

	return s.sendResponse(req.ID, result)
}

// handleToolsCall handles tool execution
func (s *Server) handleToolsCall(req MCPRequest) error {
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		return s.sendError(req.ID, -32602, "Invalid params", "Expected object")
	}

	toolName, ok := params["name"].(string)
	if !ok {
		return s.sendError(req.ID, -32602, "Invalid params", "Missing tool name")
	}

	arguments, ok := params["arguments"].(map[string]interface{})
	if !ok {
		arguments = make(map[string]interface{})
	}

	s.logger.Info("Executing tool", "tool", toolName, "arguments", arguments)

	var result interface{}
	var err error

	switch toolName {
	case "remember":
		result, err = s.handleRemember(arguments)
	case "recall":
		result, err = s.handleRecall(arguments)
	case "forget":
		result, err = s.handleForget(arguments)
	case "list_memories":
		result, err = s.handleListMemories(arguments)
	case "memory_stats":
		result, err = s.handleMemoryStats(arguments)
	default:
		return s.sendError(req.ID, -32602, "Unknown tool", toolName)
	}

	if err != nil {
		return s.sendError(req.ID, -32603, "Tool execution failed", err.Error())
	}

	toolResult := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": result,
			},
		},
	}

	return s.sendResponse(req.ID, toolResult)
}

// Tool implementations

func (s *Server) handleRemember(args map[string]interface{}) (string, error) {
	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("content is required")
	}

	summary, _ := args["summary"].(string)
	category, _ := args["category"].(string)

	var tags []string
	if tagsInterface, ok := args["tags"].([]interface{}); ok {
		for _, tag := range tagsInterface {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}

	memory, err := s.store.Store(content, summary, category, tags, nil)
	if err != nil {
		return "", fmt.Errorf("failed to store memory: %w", err)
	}

	return fmt.Sprintf("Memory stored successfully with ID: %s", memory.ID), nil
}

func (s *Server) handleRecall(args map[string]interface{}) (string, error) {
	query, ok := args["query"].(string)
	if !ok {
		return "", fmt.Errorf("query is required")
	}

	searchQuery := &memory.SearchQuery{
		Query: query,
		Limit: 10,
	}

	if category, ok := args["category"].(string); ok {
		searchQuery.Category = category
	}

	if tagsInterface, ok := args["tags"].([]interface{}); ok {
		for _, tag := range tagsInterface {
			if tagStr, ok := tag.(string); ok {
				searchQuery.Tags = append(searchQuery.Tags, tagStr)
			}
		}
	}

	if limit, ok := args["limit"].(float64); ok {
		searchQuery.Limit = int(limit)
	}

	memories, err := s.store.Search(searchQuery)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	if len(memories) == 0 {
		return "No memories found matching your query.", nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d matching memories:\n\n", len(memories)))

	for i, memory := range memories {
		result.WriteString(fmt.Sprintf("## Memory %d (ID: %s)\n", i+1, memory.ID))
		if memory.Category != "" {
			result.WriteString(fmt.Sprintf("**Category:** %s\n", memory.Category))
		}
		if len(memory.Tags) > 0 {
			result.WriteString(fmt.Sprintf("**Tags:** %s\n", strings.Join(memory.Tags, ", ")))
		}
		if memory.Summary != "" {
			result.WriteString(fmt.Sprintf("**Summary:** %s\n", memory.Summary))
		}
		result.WriteString(fmt.Sprintf("**Created:** %s\n", memory.CreatedAt.Format("2006-01-02 15:04:05")))
		result.WriteString(fmt.Sprintf("**Content:**\n%s\n\n", memory.Content))
		result.WriteString("---\n\n")
	}

	return result.String(), nil
}

func (s *Server) handleForget(args map[string]interface{}) (string, error) {
	id, ok := args["id"].(string)
	if !ok {
		return "", fmt.Errorf("id is required")
	}

	if err := s.store.Delete(id); err != nil {
		return "", fmt.Errorf("failed to delete memory: %w", err)
	}

	return fmt.Sprintf("Memory with ID %s has been forgotten.", id), nil
}

func (s *Server) handleListMemories(args map[string]interface{}) (string, error) {
	category, _ := args["category"].(string)
	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	var tags []string
	if tagsInterface, ok := args["tags"].([]interface{}); ok {
		for _, tag := range tagsInterface {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}

	memories, err := s.store.List(category, tags, limit)
	if err != nil {
		return "", fmt.Errorf("failed to list memories: %w", err)
	}

	if len(memories) == 0 {
		return "No memories found.", nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d memories:\n\n", len(memories)))

	for i, memory := range memories {
		result.WriteString(fmt.Sprintf("%d. **%s** (ID: %s)\n", i+1,
			memory.Summary, memory.ID))
		if memory.Category != "" {
			result.WriteString(fmt.Sprintf("   Category: %s\n", memory.Category))
		}
		if len(memory.Tags) > 0 {
			result.WriteString(fmt.Sprintf("   Tags: %s\n", strings.Join(memory.Tags, ", ")))
		}
		result.WriteString(fmt.Sprintf("   Created: %s, Accessed: %d times\n",
			memory.CreatedAt.Format("2006-01-02"), memory.AccessCount))

		// Show first 100 chars of content
		content := memory.Content
		if len(content) > 100 {
			content = content[:100] + "..."
		}
		result.WriteString(fmt.Sprintf("   Content: %s\n\n", content))
	}

	return result.String(), nil
}

func (s *Server) handleMemoryStats(args map[string]interface{}) (string, error) {
	stats := s.store.GetStats()

	var result strings.Builder
	result.WriteString("## Memory Statistics\n\n")
	result.WriteString(fmt.Sprintf("**Total Memories:** %d\n", stats["total_memories"]))
	result.WriteString(fmt.Sprintf("**Total Access Count:** %d\n", stats["total_access_count"]))
	result.WriteString(fmt.Sprintf("**Data Directory:** %s\n\n", stats["data_directory"]))

	if categories, ok := stats["categories"].(map[string]int); ok && len(categories) > 0 {
		result.WriteString("**Categories:**\n")
		for category, count := range categories {
			result.WriteString(fmt.Sprintf("- %s: %d\n", category, count))
		}
	}

	return result.String(), nil
}

// handleResourcesList handles resource listing (not implemented for now)
func (s *Server) handleResourcesList(req MCPRequest) error {
	result := map[string]interface{}{
		"resources": []interface{}{},
	}
	return s.sendResponse(req.ID, result)
}

// handleResourcesRead handles resource reading (not implemented for now)
func (s *Server) handleResourcesRead(req MCPRequest) error {
	return s.sendError(req.ID, -32601, "Not implemented", "Resource reading not implemented")
}

// Helper methods for MCP protocol

func (s *Server) sendResponse(id interface{}, result interface{}) error {
	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}

	return s.sendJSON(response)
}

// sendError sends an error response, handling null ID properly
func (s *Server) sendError(id interface{}, code int, message, data string) error {
	// Ensure ID is never null - use 0 if not provided
	responseID := id
	if responseID == nil {
		responseID = 0
	}

	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      responseID,
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}

	return s.sendJSON(response)
}

// sendJSON sends JSON to stdout and flushes immediately
func (s *Server) sendJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write to stdout followed by newline and flush
	_, err = fmt.Printf("%s\n", string(data))
	if err != nil {
		return fmt.Errorf("failed to write to stdout: %w", err)
	}

	// Force flush to ensure data is sent immediately
	os.Stdout.Sync()
	return nil
}

// Remove the sendServerInfo method as it's not needed
func (s *Server) sendServerInfo() error {
	// This method is no longer used
	return nil
}
