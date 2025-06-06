// internal/api/server.go
package api

import (
	"encoding/json"
	"net/http"

	"mcp-memory-server/internal/memory"
	"mcp-memory-server/pkg/logger"
)

type Server struct {
	store  *memory.Store
	logger *logger.Logger
}

func NewServer(store *memory.Store, logger *logger.Logger) *Server {
	return &Server{
		store:  store,
		logger: logger.WithComponent("api_server"),
	}
}

type RememberRequest struct {
	Content  string   `json:"content"`
	Summary  string   `json:"summary,omitempty"`
	Category string   `json:"category,omitempty"`
	Tags     []string `json:"tags,omitempty"`
}

type RememberResponse struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Message string `json:"message"`
}

type RecallRequest struct {
	Query    string   `json:"query"`
	Category string   `json:"category,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Limit    int      `json:"limit,omitempty"`
}

func (s *Server) Start(port string) error {
	http.HandleFunc("/remember", s.handleRemember)
	http.HandleFunc("/recall", s.handleRecall)
	http.HandleFunc("/stats", s.handleStats)
	http.HandleFunc("/health", s.handleHealth)

	s.logger.Info("Starting API server", map[string]interface{}{
		"port": port,
	})

	return http.ListenAndServe(":"+port, nil)
}

func (s *Server) handleRemember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RememberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	// Store memory using the async store
	mem, err := s.store.Store(req.Content, req.Summary, req.Category, req.Tags, nil)
	if err != nil {
		s.logger.Error("Failed to store memory", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Failed to store memory", http.StatusInternalServerError)
		return
	}

	resp := RememberResponse{
		Success: true,
		ID:      mem.ID,
		Message: "Memory stored successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleRecall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RecallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	if req.Limit == 0 {
		req.Limit = 10
	}

	searchQuery := &memory.SearchQuery{
		Query:    req.Query,
		Category: req.Category,
		Tags:     req.Tags,
		Limit:    req.Limit,
	}
	
	memories, err := s.store.Search(searchQuery)
	if err != nil {
		s.logger.Error("Failed to search memories", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Failed to search memories", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(memories)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := s.store.GetStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}