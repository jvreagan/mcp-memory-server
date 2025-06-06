// internal/reporting/server.go
package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"mcp-memory-server/internal/memory"
	"mcp-memory-server/pkg/logger"
)

// Store interface for memory operations
type Store interface {
	GetStats() map[string]interface{}
	List(category string, tags []string, limit int) ([]*memory.Memory, error)
	GetTimeline() map[string]interface{}
	Refresh() error
}

// Server provides a web interface for memory reporting
type Server struct {
	host   string
	port   int
	store  Store
	logger *logger.Logger
	server *http.Server
}

// NewServer creates a new reporting server
func NewServer(host string, port int, store Store, logger *logger.Logger) *Server {
	return &Server{
		host:   host,
		port:   port,
		store:  store,
		logger: logger.WithComponent("reporting_server"),
	}
}

// Start starts the reporting server
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Static routes
	mux.HandleFunc("/", s.handleDashboard)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/memories", s.handleMemories)
	mux.HandleFunc("/api/timeline", s.handleTimeline)
	mux.HandleFunc("/api/refresh", s.handleRefresh)

	address := fmt.Sprintf("%s:%d", s.host, s.port)
	s.server = &http.Server{
		Addr:    address,
		Handler: mux,
	}

	s.logger.Info("Starting reporting server", "address", address)

	// Start server in goroutine
	go func() {
		s.logger.Info("Reporting server listening", "address", address)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.WithError(err).Error("Reporting server failed to start", "address", address)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		s.logger.WithError(err).Error("Failed to shutdown reporting server gracefully")
		return err
	}

	s.logger.Info("Reporting server stopped")
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
    <title>MCP Memory Server - Reporting Dashboard</title>
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
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .refresh-btn {
            background: #3b82f6;
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 6px;
            cursor: pointer;
            font-size: 14px;
        }
        .refresh-btn:hover {
            background: #2563eb;
        }
        .refresh-btn:disabled {
            background: #9ca3af;
            cursor: not-allowed;
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
        .error {
            color: #dc2626;
            background-color: #fef2f2;
            padding: 10px;
            border-radius: 4px;
            margin: 10px 0;
        }
        .success {
            color: #065f46;
            background-color: #ecfdf5;
            padding: 10px;
            border-radius: 4px;
            margin: 10px 0;
        }
        .loading {
            text-align: center;
            padding: 40px;
            color: #6b7280;
        }
        .status-indicator {
            display: inline-block;
            width: 8px;
            height: 8px;
            border-radius: 50%;
            margin-left: 8px;
        }
        .status-online {
            background-color: #10b981;
        }
        .status-readonly {
            background-color: #f59e0b;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div>
                <h1>MCP Memory Reporting Dashboard 
                    <span class="status-indicator status-readonly" title="Read-only mode"></span>
                </h1>
                <p>Read-only view of memory server data</p>
            </div>
            <button class="refresh-btn" onclick="refreshData()" id="refresh-btn">
                Refresh Data
            </button>
        </div>

        <div id="loading" class="loading">Loading...</div>
        <div id="error" class="error" style="display: none;"></div>
        <div id="success" class="success" style="display: none;"></div>

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
                    <div class="stat-value" id="data-dir">-</div>
                    <div class="stat-label">Data Directory</div>
                </div>
            </div>

            <div class="chart-container">
                <h3>Categories Distribution</h3>
                <canvas id="categories-chart" width="400" height="200"></canvas>
            </div>

            <div class="chart-container">
                <h3>Memory Creation Timeline (Last 30 Days)</h3>
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
            const response = await fetch('/api/stats');
            if (!response.ok) throw new Error('Failed to fetch stats');
            return await response.json();
        }

        async function fetchMemories() {
            const response = await fetch('/api/memories?limit=10');
            if (!response.ok) throw new Error('Failed to fetch memories');
            return await response.json();
        }

        async function fetchTimeline() {
            const response = await fetch('/api/timeline');
            if (!response.ok) throw new Error('Failed to fetch timeline');
            return await response.json();
        }

        async function refreshData() {
            const btn = document.getElementById('refresh-btn');
            btn.disabled = true;
            btn.textContent = 'Refreshing...';
            
            try {
                const response = await fetch('/api/refresh', { method: 'POST' });
                if (!response.ok) throw new Error('Failed to refresh data');
                
                showSuccess('Data refreshed successfully');
                await loadDashboard();
            } catch (error) {
                showError('Failed to refresh data: ' + error.message);
            } finally {
                btn.disabled = false;
                btn.textContent = 'Refresh Data';
            }
        }

        function formatBytes(bytes) {
            if (bytes === 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }

        function showError(message) {
            const errorDiv = document.getElementById('error');
            errorDiv.textContent = message;
            errorDiv.style.display = 'block';
            setTimeout(() => errorDiv.style.display = 'none', 5000);
        }

        function showSuccess(message) {
            const successDiv = document.getElementById('success');
            successDiv.textContent = message;
            successDiv.style.display = 'block';
            setTimeout(() => successDiv.style.display = 'none', 3000);
        }

        function updateStats(stats) {
            document.getElementById('total-memories').textContent = stats.total_memories;
            document.getElementById('total-access').textContent = stats.total_access_count;
            document.getElementById('storage-used').textContent = formatBytes(stats.total_size || 0);
            document.getElementById('data-dir').textContent = stats.data_directory.split('/').pop();
        }

        function updateCategoriesChart(categories) {
            const ctx = document.getElementById('categories-chart').getContext('2d');
            
            if (categoriesChart) {
                categoriesChart.destroy();
            }

            const labels = Object.keys(categories);
            const data = Object.values(categories);
            
            if (labels.length === 0) {
                ctx.fillText('No categories found', 200, 100);
                return;
            }

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
                            beginAtZero: true,
                            ticks: {
                                stepSize: 1
                            }
                        }
                    }
                }
            });
        }

        function updateMemoriesTable(memories) {
            const tbody = document.getElementById('memories-tbody');
            tbody.innerHTML = '';
            
            if (memories.length === 0) {
                const row = tbody.insertRow();
                row.innerHTML = '<td colspan="5" style="text-align: center; color: #6b7280;">No memories found</td>';
                return;
            }
            
            memories.forEach(memory => {
                const row = tbody.insertRow();
                row.innerHTML = ` + "`" + `
                    <td>${memory.summary || (memory.content ? memory.content.substring(0, 50) + '...' : 'No content')}</td>
                    <td>${memory.category || '-'}</td>
                    <td>${memory.tags && memory.tags.length > 0 ? memory.tags.join(', ') : '-'}</td>
                    <td>${new Date(memory.created_at).toLocaleDateString()}</td>
                    <td>${memory.access_count || 0}</td>
                ` + "`" + `;
            });
        }

        async function loadDashboard() {
            try {
                document.getElementById('loading').style.display = 'block';
                document.getElementById('error').style.display = 'none';
                document.getElementById('dashboard').style.display = 'none';

                // Auto-refresh data from server
                await fetch('/api/refresh', { method: 'POST' });

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
                showError(error.message);
            }
        }

        // Load dashboard on page load
        loadDashboard();

        // Auto-refresh every 10 seconds for real-time updates
        setInterval(loadDashboard, 10000);
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

// handleRefresh refreshes the memory data from disk
func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.store.Refresh(); err != nil {
		s.logger.WithError(err).Error("Failed to refresh memory data")
		http.Error(w, "Failed to refresh data", http.StatusInternalServerError)
		return
	}

	s.logger.Info("Memory data refreshed")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}