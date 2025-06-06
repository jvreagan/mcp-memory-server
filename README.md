# MCP Memory Server

A Model Context Protocol (MCP) server that provides persistent memory capabilities for Large Language Models. Store concepts, code snippets, notes, and any information you want your LLM to remember across conversations.

[![Go Version](https://img.shields.io/badge/Go-1.19+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![MCP](https://img.shields.io/badge/MCP-2024--11--05-orange.svg)](https://modelcontextprotocol.io)

## Features

- 🧠 **Persistent Memory** - Information persists across LLM sessions
- 🏷️ **Smart Organization** - Categorize and tag memories for easy retrieval
- 🔍 **Natural Language Search** - Find memories using conversational queries
- 📊 **Usage Analytics** - Track which memories are accessed most frequently
- 🔒 **Local Storage** - All data stays on your machine
- ⚡ **Fast Access** - In-memory indexing for quick retrieval
- 🔧 **Easy Configuration** - Environment variable based configuration
- 🛡️ **Thread Safe** - Concurrent access with proper locking

## Quick Start

### Prerequisites

- Go 1.19 or later
- Claude Desktop application

### Installation

1. **Clone and build the server:**

```bash
git clone https://github.com/yourusername/mcp-memory-server.git
cd mcp-memory-server
go build -o mcp-memory-server cmd/server/main.go
```

2. **Configure Claude Desktop:**

Add the following to your Claude Desktop MCP configuration file:

**macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`
**Linux:** `~/.config/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "memory": {
      "command": "/absolute/path/to/your/mcp-memory-server",
      "args": [],
      "env": {
        "MCP_DATA_DIR": "/Users/yourusername/.mcp-memory"
      }
    }
  }
}
```

See `config.example.json` for a complete example with all available configuration options.

3. **Restart Claude Desktop**

The memory server will now be available in your Claude conversations!

## Usage

Once configured, you can use these commands in Claude Desktop:

### Store Information
```
Remember this: The scraping project uses Go with Playwright for browser automation. 
The architecture follows domain-driven design with repositories and services.

Category: project
Tags: go, scraping, web-automation
```

### Search Memories
```
What do you remember about Go web scraping?
```

### List by Category
```
Show me all my project memories
```

### Get Statistics
```
What are my memory usage statistics?
```

## Available Tools

The MCP server provides these tools to Claude:

| Tool | Description | Parameters |
|------|-------------|------------|
| `remember` | Store new information | `content` (required), `summary`, `category`, `tags` |
| `recall` | Search stored memories | `query` (required), `category`, `tags`, `limit` |
| `forget` | Delete a memory by ID | `id` (required) |
| `list_memories` | List all memories with filtering | `category`, `tags`, `limit` |
| `memory_stats` | Get usage statistics | None |

## Configuration

Configure the server using environment variables:

### Storage Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `MCP_DATA_DIR` | Directory for storing memory files | `~/.mcp-memory` |
| `MCP_MAX_FILE_SIZE` | Maximum size for memory files (bytes) | `104857600` (100MB) |
| `MCP_MAX_STORAGE_SIZE` | Total storage limit (bytes) | `107374182400` (100GB) |

### Async Behavior Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `MCP_ENABLE_ASYNC` | Enable asynchronous save operations | `true` |
| `MCP_QUEUE_SIZE` | Size of async save queue | `1000` |
| `MCP_WORKER_THREADS` | Number of worker threads for async saves | `2` |

### Compression Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `MCP_ENABLE_COMPRESSION` | Enable gzip compression for memory files | `true` |
| `MCP_COMPRESSION_LEVEL` | Gzip compression level (1-9, where 9 is maximum) | `6` |

### Other Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `MCP_LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` |
| `MCP_LOG_FORMAT` | Log format (json, text) | `json` |
| `MCP_MAX_RESULTS` | Maximum search results returned | `20` |
| `MCP_ENABLE_EMBEDDINGS` | Enable semantic search (future) | `false` |
| `MCP_EMBEDDING_MODEL` | OpenAI embedding model | `text-embedding-ada-002` |

## Data Storage

Memories are stored in your configured data directory:

```
~/.mcp-memory/
├── memories/           # Individual memory JSON files
├── index/             # Search indexes (future enhancement)
└── logs/              # Application logs
```

Each memory includes:
- Unique content-based ID (SHA256 hash)
- Content and optional summary
- Categories and tags for organization
- Creation and update timestamps
- Access statistics and last access time
- Custom metadata

### Performance Tuning

The server can be tuned for different use cases:

**For high-throughput scenarios:**
```json
{
  "env": {
    "MCP_ENABLE_ASYNC": "true",
    "MCP_QUEUE_SIZE": "5000",
    "MCP_WORKER_THREADS": "4",
    "MCP_COMPRESSION_LEVEL": "1"
  }
}
```

**For low-latency scenarios:**
```json
{
  "env": {
    "MCP_ENABLE_ASYNC": "false",
    "MCP_ENABLE_COMPRESSION": "false"
  }
}
```

**For storage-constrained environments:**
```json
{
  "env": {
    "MCP_ENABLE_COMPRESSION": "true",
    "MCP_COMPRESSION_LEVEL": "9",
    "MCP_MAX_STORAGE_SIZE": "1073741824"
  }
}
```

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Claude LLM    │◄──►│  MCP Protocol    │◄──►│  Memory Store   │
│                 │    │  (JSON-RPC)      │    │  (File-based)   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌──────────────────┐
                       │  Local Storage   │
                       │  ~/.mcp-memory/  │
                       └──────────────────┘
```

## Development

### Project Structure

```
mcp-memory-server/
├── cmd/server/         # Main application entry point
├── internal/
│   ├── config/         # Configuration management
│   ├── mcp/           # MCP protocol implementation
│   └── memory/        # Memory storage and retrieval
├── pkg/logger/        # Logging utilities
├── go.mod
└── README.md
```

### Building from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/mcp-memory-server.git
cd mcp-memory-server

# Install dependencies
go mod tidy

# Build
go build -o mcp-memory-server cmd/server/main.go

# Run tests
go test ./...
```

### Testing the Server

You can test the MCP server manually using JSON-RPC:

```bash
# List available tools
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | ./mcp-memory-server

# Store a memory
echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"remember","arguments":{"content":"Test memory","category":"test"}}}' | ./mcp-memory-server
```

## Troubleshooting

**Server not starting:**
- Verify the binary has execute permissions
- Check that the data directory is writable
- Review logs in the data directory

**Claude not connecting to server:**
- Ensure the path in `claude_desktop_config.json` is absolute
- Verify the server binary exists and is executable
- Check Claude Desktop logs for connection errors
- Restart Claude Desktop after configuration changes

**Memory not persisting:**
- Check file permissions in the data directory
- Verify `MCP_DATA_DIR` environment variable is set correctly
- Look for error messages in the server logs

## Future Enhancements

- [ ] **Semantic Search** - Vector embeddings for better search relevance
- [ ] **Web Interface** - Browser-based memory management
- [ ] **Import/Export** - Backup and restore capabilities
- [ ] **Memory Expiration** - Automatic cleanup of old memories
- [ ] **Encryption** - Secure storage for sensitive information
- [ ] **Multi-user Support** - Separate memory spaces for different users
- [ ] **Advanced Search** - Boolean queries and filters

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built for the [Model Context Protocol](https://modelcontextprotocol.io)
- Designed to work seamlessly with Claude Desktop
- Inspired by the need for persistent memory in LLM conversations

## Support

If you encounter any issues or have questions:

1. Check the [troubleshooting section](#troubleshooting)
2. Search existing [GitHub issues](https://github.com/yourusername/mcp-memory-server/issues)
3. Create a new issue with detailed information about your problem

---

**Made with ❤️ for the MCP community**
