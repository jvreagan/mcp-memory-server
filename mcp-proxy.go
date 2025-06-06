// mcp-proxy.go - Fast MCP proxy that uses HTTP API
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

type ToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

const httpAPIURL = "http://localhost:8080"

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		var req MCPRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}

		var resp MCPResponse
		resp.JSONRPC = "2.0"
		resp.ID = req.ID

		switch req.Method {
		case "initialize":
			resp.Result = map[string]interface{}{
				"protocolVersion": "0.1.0",
				"serverInfo": map[string]string{
					"name":    "mcp-memory-proxy",
					"version": "1.0.0",
				},
			}

		case "tools/list":
			resp.Result = map[string]interface{}{
				"tools": []map[string]interface{}{
					{
						"name":        "remember",
						"description": "Store information in long-term memory",
						"inputSchema": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"content":  map[string]string{"type": "string"},
								"summary":  map[string]string{"type": "string"},
								"category": map[string]string{"type": "string"},
								"tags": map[string]interface{}{
									"type": "array",
									"items": map[string]string{"type": "string"},
								},
							},
							"required": []string{"content"},
						},
					},
					{
						"name":        "recall",
						"description": "Search and retrieve memories",
						"inputSchema": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"query": map[string]string{"type": "string"},
								"limit": map[string]string{"type": "integer"},
							},
							"required": []string{"query"},
						},
					},
					{
						"name":        "memory_stats",
						"description": "Get memory usage statistics",
						"inputSchema": map[string]interface{}{
							"type":       "object",
							"properties": map[string]interface{}{},
						},
					},
				},
			}

		case "tools/call":
			params := req.Params.(map[string]interface{})
			toolName := params["name"].(string)
			args := params["arguments"].(map[string]interface{})

			switch toolName {
			case "remember":
				result, err := callRemember(args)
				if err != nil {
					resp.Error = map[string]interface{}{
						"code":    -32603,
						"message": err.Error(),
					}
				} else {
					resp.Result = result
				}

			case "recall":
				result, err := callRecall(args)
				if err != nil {
					resp.Error = map[string]interface{}{
						"code":    -32603,
						"message": err.Error(),
					}
				} else {
					resp.Result = result
				}

			case "memory_stats":
				result, err := callStats()
				if err != nil {
					resp.Error = map[string]interface{}{
						"code":    -32603,
						"message": err.Error(),
					}
				} else {
					resp.Result = result
				}

			default:
				resp.Error = map[string]interface{}{
					"code":    -32601,
					"message": "Unknown tool: " + toolName,
				}
			}

		default:
			// Ignore unknown methods
			continue
		}

		encoder.Encode(resp)
	}
}

func callRemember(args map[string]interface{}) (interface{}, error) {
	data, _ := json.Marshal(args)
	resp, err := http.Post(httpAPIURL+"/remember", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("Memory stored successfully with ID: %s", result["id"]),
			},
		},
	}, nil
}

func callRecall(args map[string]interface{}) (interface{}, error) {
	data, _ := json.Marshal(args)
	resp, err := http.Post(httpAPIURL+"/recall", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var memories []map[string]interface{}
	if err := json.Unmarshal(body, &memories); err != nil {
		return nil, err
	}

	if len(memories) == 0 {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": "No memories found matching your query.",
				},
			},
		}, nil
	}

	var text strings.Builder
	text.WriteString(fmt.Sprintf("Found %d memories:\n\n", len(memories)))

	for i, mem := range memories {
		id := mem["id"].(string)
		content := mem["content"].(string)
		summary, _ := mem["summary"].(string)
		category, _ := mem["category"].(string)
		tags, _ := mem["tags"].([]interface{})

		text.WriteString(fmt.Sprintf("%d. [%s] ", i+1, id[:8]))
		if summary != "" {
			text.WriteString(summary)
		} else {
			if len(content) > 100 {
				text.WriteString(content[:100] + "...")
			} else {
				text.WriteString(content)
			}
		}
		text.WriteString("\n")

		if category != "" {
			text.WriteString(fmt.Sprintf("   Category: %s\n", category))
		}
		if len(tags) > 0 {
			text.WriteString("   Tags: ")
			for j, tag := range tags {
				if j > 0 {
					text.WriteString(", ")
				}
				text.WriteString(tag.(string))
			}
			text.WriteString("\n")
		}
		text.WriteString("\n")
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": text.String(),
			},
		},
	}, nil
}

func callStats() (interface{}, error) {
	resp, err := http.Get(httpAPIURL + "/stats")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var stats map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}

	totalMemories := int(stats["total_memories"].(float64))
	totalSize := int(stats["total_size"].(float64))
	categories := stats["categories"].(map[string]interface{})

	var text strings.Builder
	text.WriteString(fmt.Sprintf("Memory Statistics:\n"))
	text.WriteString(fmt.Sprintf("- Total memories: %d\n", totalMemories))
	text.WriteString(fmt.Sprintf("- Total size: %.2f MB\n", float64(totalSize)/1024/1024))
	text.WriteString("\nCategories:\n")

	for cat, count := range categories {
		text.WriteString(fmt.Sprintf("- %s: %d\n", cat, int(count.(float64))))
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": text.String(),
			},
		},
	}, nil
}