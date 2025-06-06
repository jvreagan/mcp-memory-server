package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: mcp-client <method> [args...]")
		fmt.Println("Methods:")
		fmt.Println("  tools/list")
		fmt.Println("  remember <content> [summary] [category] [tags]")
		fmt.Println("  recall <query> [category] [tags] [limit]")
		fmt.Println("  forget <id>")
		fmt.Println("  list_memories [category] [tags] [limit]")
		fmt.Println("  memory_stats")
		os.Exit(1)
	}

	method := os.Args[1]
	
	var req MCPRequest
	req.JSONRPC = "2.0"
	req.ID = 1

	switch method {
	case "tools/list":
		req.Method = "tools/list"
	case "remember":
		if len(os.Args) < 3 {
			fmt.Println("Usage: mcp-client remember <content> [summary] [category] [tags]")
			os.Exit(1)
		}
		req.Method = "tools/call"
		args := map[string]interface{}{
			"content": os.Args[2],
		}
		if len(os.Args) > 3 && os.Args[3] != "" {
			args["summary"] = os.Args[3]
		}
		if len(os.Args) > 4 && os.Args[4] != "" {
			args["category"] = os.Args[4]
		}
		if len(os.Args) > 5 && os.Args[5] != "" {
			args["tags"] = strings.Split(os.Args[5], ",")
		}
		req.Params = map[string]interface{}{
			"name": "remember",
			"arguments": args,
		}
	case "recall":
		if len(os.Args) < 3 {
			fmt.Println("Usage: mcp-client recall <query> [category] [tags] [limit]")
			os.Exit(1)
		}
		req.Method = "tools/call"
		args := map[string]interface{}{
			"query": os.Args[2],
		}
		if len(os.Args) > 3 && os.Args[3] != "" {
			args["category"] = os.Args[3]
		}
		if len(os.Args) > 4 && os.Args[4] != "" {
			args["tags"] = strings.Split(os.Args[4], ",")
		}
		if len(os.Args) > 5 && os.Args[5] != "" {
			args["limit"] = os.Args[5]
		}
		req.Params = map[string]interface{}{
			"name": "recall",
			"arguments": args,
		}
	case "memory_stats":
		req.Method = "tools/call"
		req.Params = map[string]interface{}{
			"name": "memory_stats",
			"arguments": map[string]interface{}{},
		}
	default:
		fmt.Printf("Unknown method: %s\n", method)
		os.Exit(1)
	}

	// Connect to the running MCP server
	cmd := exec.Command("docker", "exec", "-i", "mcp-memory-server", "./mcp-memory-server")
	
	// If Docker isn't running, fall back to local server
	if !isDockerRunning() {
		cmd = exec.Command("/Users/jamesreagan/code/mcp-memory-server/mcp-memory-server")
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Printf("Error creating stdin pipe: %v\n", err)
		os.Exit(1)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error creating stdout pipe: %v\n", err)
		os.Exit(1)
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("Error starting command: %v\n", err)
		os.Exit(1)
	}

	// Send request
	reqJSON, _ := json.Marshal(req)
	fmt.Fprintf(stdin, "%s\n", reqJSON)
	stdin.Close()

	// Read response
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "{") {
			var resp MCPResponse
			if err := json.Unmarshal([]byte(line), &resp); err == nil {
				if resp.Result != nil {
					resultJSON, _ := json.MarshalIndent(resp.Result, "", "  ")
					fmt.Println(string(resultJSON))
				}
				if resp.Error != nil {
					fmt.Printf("Error: %v\n", resp.Error)
				}
				break
			}
		}
	}

	cmd.Wait()
}

func isDockerRunning() bool {
	cmd := exec.Command("docker", "ps", "-q", "--filter", "name=mcp-memory-server")
	output, err := cmd.Output()
	return err == nil && len(strings.TrimSpace(string(output))) > 0
}