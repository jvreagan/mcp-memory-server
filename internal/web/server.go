// internal/web/server.go
package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"mcp-memory-server/internal/config"
	"mcp-memory-server/internal/memory"
	"mcp-memory-server/pkg/logger"
)

// Server provides a web interface for memory statistics
type Server struct {
	config *config.WebConfig
	store  *memory.Store
	logger *logger.Logger
	server *http.Server
}

// NewServer creates a new web server
func NewServer(cfg *config.WebConfig, store *memory.Store, logger *logger.Logger) *Server {
	return &Server{
		config: cfg,
		store:  store,
		logger: logger.WithComponent("web_server"),
	}
}

// Start starts the web server
func (s *Server) Start(ctx context.Context) error {
	if !s.config.Enabled {
		s.logger.Info("Web server disabled")
		return nil
	}

	mux := http.NewServeMux()

	// Static routes
	mux.HandleFunc("/", s.handleDashboard)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/memories", s.handleMemories)
	mux.HandleFunc("/api/timeline", s.handleTimeline)

	address := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.server = &http.Server{
		Addr:    address,
		Handler: mux,
	}

	s.logger.Info("Starting web server", "address", address)

	// Start server in goroutine
	go func() {
		s.logger.Info("Web server listening", "address", address)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.WithError(err).Error("Web server failed to start", "address", address)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		s.logger.WithError(err).Error("Failed to shutdown web server gracefully")
		return err
	}

	s.logger.Info("Web server stopped")
	return nil
}

// Stop stops the web server
func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// handleDashboard serves the main dashboard HTML
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MCP Memory Server Dashboard</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        .header {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .stat-card {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .stat-value {
            font-size: 2em;
            font-weight: bold;
            color: #2563eb;
        }
        .stat-label {
            color: #6b7280;
            margin-top: 5px;
        }
        .chart-container {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }
        .memories-table {
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        table {
            width: 100%;
            border-collapse: collapse;
        }
        th, td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #e5e7eb;
        }
        th {
            background-color: #f9fafb;
            font-weight: 600;
        }
        .progress-bar {
            width: 100%;
            height: 20px;
            background-color: #e5e7eb;
            border-radius: 10px;
            overflow: hidden;
        }
        .progress-fill {
            height: 100%;
            background-color: #10b981;
            transition: width 0.3s ease;
        }
        .error {
            color: #dc2626;
            background-color: #fef2f2;
            padding: 10px;
            border-radius: 4px;
            margin: 10px 0;
        }
        .loading {
            text-align: center;
            padding: 40px;
            color: #6b7280;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>MCP Memory Server Dashboard</h1>
            <p>Real-time statistics and insights for your memory storage</p>
        </div>

        <div id="loading" class="loading">Loading...</div>
        <div id="error" class="error" style="display: none;"></div>

        <div id="dashboard" style="display: none;">
            <div class="stats-grid">
                <div class="stat-card">
                    <div class="stat-value" id="total-memories">-</div>
                    <div class="stat-label">Total Memories</div>
                </div>
                <div class="stat-card">
                    <div class="stat-value" id="total-access">-</div>
                    <div class="stat-label">Total Access Count</div>
                </div>
                <div class="stat-card">
                    <div class="stat-value" id="storage-used">-</div>
                    <div class="stat-label">Storage Used</div>
                </div>
                <div class="stat-card">
                    <div class="stat-value" id="storage-percent">-</div>
                    <div class="stat-label">Storage Usage</div>
                    <div class="progress-bar" style="margin-top: 10px;">
                        <div class="progress-fill" id="storage-progress" style="width: 0%;"></div>
                    </div>
                </div>
            </div>

            <div class="chart-container">
                <h3>Categories Distribution</h3>
                <canvas id="categories-chart" width="400" height="200"></canvas>
            </div>

            <div class="chart-container">
                <h3>Memory Creation Timeline</h3>
                <canvas id="timeline-chart" width="400" height="200"></canvas>
            </div>

            <div class="memories-table">
                <h3 style="margin: 0; padding: 20px 20px 0 20px;">Recent Memories</h3>
                <table>
                    <thead>
                        <tr>
                            <th>Summary</th>
                            <th>Category</th>
                            <th>Tags</th>
                            <th>Created</th>
                            <th>Access Count</th>
                        </tr>
                    </thead>
                    <tbody id="memories-tbody">
                    </tbody>
                </table>
            </div>
        </div>
    </div>

    <script>
        let categoriesChart, timelineChart;

        async function fetchStats() {
            try {
                const response = await fetch('/api/stats');
                const data = await response.json();
                return data;
            } catch (error) {
                throw new Error('Failed to fetch stats: ' + error.message);
            }
        }

        async function fetchMemories() {
            try {
                const response = await fetch('/api/memories?limit=10');
                const data = await response.json();
                return data;
            } catch (error) {
                throw new Error('Failed to fetch memories: ' + error.message);
            }
        }

        async function fetchTimeline() {
            try {
                const response = await fetch('/api/timeline');
                const data = await response.json();
                return data;
            } catch (error) {
                throw new Error('Failed to fetch timeline: ' + error.message);
            }
        }

        function formatBytes(bytes) {
            if (bytes === 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }

        function updateStats(stats) {
            document.getElementById('total-memories').textContent = stats.total_memories;
            document.getElementById('total-access').textContent = stats.total_access_count;
            document.getElementById('storage-used').textContent = formatBytes(stats.total_size || 0);
            
            const storagePercent = stats.storage_used_pct || 0;
            document.getElementById('storage-percent').textContent = storagePercent.toFixed(1) + '%';
            document.getElementById('storage-progress').style.width = Math.min(storagePercent, 100) + '%';
            
            if (storagePercent > 80) {
                document.getElementById('storage-progress').style.backgroundColor = '#ef4444';
            } else if (storagePercent > 60) {
                document.getElementById('storage-progress').style.backgroundColor = '#f59e0b';
            }
        }

        function updateCategoriesChart(categories) {
            const ctx = document.getElementById('categories-chart').getContext('2d');
            
            if (categoriesChart) {
                categoriesChart.destroy();
            }

            const labels = Object.keys(categories);
            const data = Object.values(categories);
            const colors = [
                '#3b82f6', '#ef4444', '#10b981', '#f59e0b', '#8b5cf6',
                '#06b6d4', '#84cc16', '#f97316', '#ec4899', '#6366f1'
            ];

            categoriesChart = new Chart(ctx, {
                type: 'doughnut',
                data: {
                    labels: labels,
                    datasets: [{
                        data: data,
                        backgroundColor: colors.slice(0, labels.length),
                        borderWidth: 2,
                        borderColor: '#ffffff'
                    }]
                },
                options: {
                    responsive: true,
                    plugins: {
                        legend: {
                            position: 'right'
                        }
                    }
                }
            });
        }

        function updateTimelineChart(timeline) {
            const ctx = document.getElementById('timeline-chart').getContext('2d');
            
            if (timelineChart) {
                timelineChart.destroy();
            }

            timelineChart = new Chart(ctx, {
                type: 'line',
                data: {
                    labels: timeline.labels,
                    datasets: [{
                        label: 'Memories Created',
                        data: timeline.data,
                        borderColor: '#3b82f6',
                        backgroundColor: 'rgba(59, 130, 246, 0.1)',
                        borderWidth: 2,
                        fill: true,
                        tension: 0.4
                    }]
                },
                options: {
                    responsive: true,
                    scales: {
                        y: {
                            beginAtZero: true
                        }
                    }
                }
            });
        }

        function updateMemoriesTable(memories) {
            const tbody = document.getElementById('memories-tbody');
            tbody.innerHTML = '';
            
            memories.forEach(memory => {
                const row = tbody.insertRow();
                row.innerHTML = ` + "`" + `
                    <td>${memory.summary || memory.content.substring(0, 50) + '...'}</td>
                    <td>${memory.category || '-'}</td>
                    <td>${memory.tags ? memory.tags.join(', ') : '-'}</td>
                    <td>${new Date(memory.created_at).toLocaleDateString()}</td>
                    <td>${memory.access_count}</td>
                ` + "`" + `;
            });
        }

        async function loadDashboard() {
            try {
                document.getElementById('loading').style.display = 'block';
                document.getElementById('error').style.display = 'none';
                document.getElementById('dashboard').style.display = 'none';

                const [stats, memories, timeline] = await Promise.all([
                    fetchStats(),
                    fetchMemories(),
                    fetchTimeline()
                ]);

                updateStats(stats);
                updateCategoriesChart(stats.categories || {});
                updateTimelineChart(timeline);
                updateMemoriesTable(memories);

                document.getElementById('loading').style.display = 'none';
                document.getElementById('dashboard').style.display = 'block';

            } catch (error) {
                document.getElementById('loading').style.display = 'none';
                document.getElementById('error').style.display = 'block';
                document.getElementById('error').textContent = error.message;
            }
        }

        // Load dashboard on page load
        loadDashboard();

        // Refresh every 30 seconds
        setInterval(loadDashboard, 30000);
    </script>
</body>
</html>`

	fmt.Fprint(w, html)
}

// handleStats returns memory statistics as JSON
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := s.store.GetStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleMemories returns recent memories as JSON
func (s *Server) handleMemories(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if _, err := fmt.Sscanf(l, "%d", &limit); err != nil {
			limit = 20
		}
	}

	memories, err := s.store.List("", nil, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(memories)
}

// handleTimeline returns memory creation timeline data
func (s *Server) handleTimeline(w http.ResponseWriter, r *http.Request) {
	timeline := s.store.GetTimeline()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(timeline)
}
