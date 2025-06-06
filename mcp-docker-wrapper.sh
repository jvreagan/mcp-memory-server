#!/bin/bash
# MCP wrapper that uses Docker container

# Pass stdin to docker exec and get stdout
docker exec -i mcp-memory-server ./mcp-memory-server