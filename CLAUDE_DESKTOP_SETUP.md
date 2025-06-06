# Claude Desktop Setup for Docker MCP Memory Server

## Current Issue
New memories created in Claude Desktop are not appearing in the reporting dashboard because Claude Desktop isn't configured to use the Docker container.

## Solution

### 1. Update Claude Desktop Configuration

Edit your Claude Desktop config file:
**File:** `~/Library/Application Support/Claude/claude_desktop_config.json`

**Replace the current memory server configuration with:**

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

### 2. Restart Claude Desktop

Close and reopen Claude Desktop completely for the configuration to take effect.

### 3. Test the Connection

In Claude Desktop, try creating a new memory:
```
Remember: The Docker MCP setup is working correctly.
```

### 4. Verify in Dashboard

1. Open http://localhost:9000
2. The dashboard now auto-refreshes every 10 seconds
3. You should see the new memory appear within 10 seconds

## Troubleshooting

### If memories still don't appear:

1. **Check Docker containers are running:**
   ```bash
   docker-compose ps
   ```

2. **Check MCP server logs:**
   ```bash
   docker-compose logs -f mcp-memory-server
   ```

3. **Check Claude Desktop logs** for MCP connection errors

4. **Verify the volume has new files:**
   ```bash
   docker exec mcp-memory-server ls -la /app/.mcp-memory/memories/ | tail -5
   ```

5. **Manual refresh dashboard:**
   Click the "Refresh Data" button in the web interface

## Current Setup Status

- ✅ Docker containers running
- ✅ Reporting dashboard auto-refreshes every 10 seconds  
- ✅ 61 existing memories loaded
- ⚠️  Need to configure Claude Desktop to use Docker container