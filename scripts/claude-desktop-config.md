# Claude Desktop Configuration for Dockerized MCP Memory Server

## For macOS/Linux

Add this configuration to your Claude Desktop config file:

**Location:** `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS)

```json
{
  "mcpServers": {
    "memory": {
      "command": "docker",
      "args": [
        "exec", 
        "-i", 
        "mcp-memory-server", 
        "./mcp-memory-server"
      ],
      "env": {
        "MCP_DATA_DIR": "/app/.mcp-memory"
      }
    }
  }
}
```

## Alternative: Direct Docker Run

If you prefer to run the MCP server directly with Docker:

```json
{
  "mcpServers": {
    "memory": {
      "command": "docker",
      "args": [
        "run",
        "--rm",
        "-i",
        "--volume", "mcp-memory-server_memory-data:/app/.mcp-memory",
        "--name", "mcp-memory-server-instance",
        "mcp-memory-server_mcp-memory-server"
      ]
    }
  }
}
```

## Setup Steps

1. **Start the services:**
   ```bash
   ./scripts/setup.sh
   ```

2. **Update Claude Desktop config** with one of the configurations above

3. **Restart Claude Desktop** to pick up the new configuration

4. **Test the connection** by asking Claude to remember something

5. **View the dashboard** at http://localhost:9000

## Troubleshooting

- **Check if containers are running:** `docker-compose ps`
- **View logs:** `docker-compose logs -f mcp-memory-server`
- **Restart services:** `docker-compose restart`
- **Check Claude Desktop logs** for MCP connection issues