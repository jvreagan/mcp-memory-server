#!/bin/bash

# Quick memory operations using the running MCP server
# This connects more efficiently than spawning new processes

MCP_SERVER_PID=$(pgrep -f "mcp-memory-server" | head -1)

if [ -z "$MCP_SERVER_PID" ]; then
    echo "❌ MCP server not running. Starting it..."
    cd /Users/jamesreagan/code/mcp-memory-server
    ./mcp-memory-server > /tmp/mcp-server.log 2>&1 &
    sleep 2
fi

case "$1" in
    "remember")
        if [ -z "$2" ]; then
            echo "Usage: $0 remember <content> [summary] [category] [tags]"
            exit 1
        fi
        CONTENT="$2"
        SUMMARY="${3:-}"
        CATEGORY="${4:-}"
        TAGS="${5:-}"
        
        JSON_ARGS="{\"content\":\"$CONTENT\""
        [ -n "$SUMMARY" ] && JSON_ARGS="$JSON_ARGS,\"summary\":\"$SUMMARY\""
        [ -n "$CATEGORY" ] && JSON_ARGS="$JSON_ARGS,\"category\":\"$CATEGORY\""
        [ -n "$TAGS" ] && JSON_ARGS="$JSON_ARGS,\"tags\":[\"$(echo $TAGS | sed 's/,/","/g')\"]"
        JSON_ARGS="$JSON_ARGS}"
        
        echo "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"remember\",\"arguments\":$JSON_ARGS}}" | timeout 5 ./mcp-memory-server 2>/dev/null | grep -o '{"jsonrpc.*' | jq -r '.result.content[0].text // .error'
        ;;
    
    "recall")
        if [ -z "$2" ]; then
            echo "Usage: $0 recall <query> [category] [tags] [limit]"
            exit 1
        fi
        QUERY="$2"
        CATEGORY="${3:-}"
        TAGS="${4:-}"
        LIMIT="${5:-10}"
        
        JSON_ARGS="{\"query\":\"$QUERY\",\"limit\":$LIMIT"
        [ -n "$CATEGORY" ] && JSON_ARGS="$JSON_ARGS,\"category\":\"$CATEGORY\""
        [ -n "$TAGS" ] && JSON_ARGS="$JSON_ARGS,\"tags\":[\"$(echo $TAGS | sed 's/,/","/g')\"]"
        JSON_ARGS="$JSON_ARGS}"
        
        echo "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"recall\",\"arguments\":$JSON_ARGS}}" | timeout 5 ./mcp-memory-server 2>/dev/null | grep -o '{"jsonrpc.*' | jq -r '.result.content[0].text // .error'
        ;;
    
    "stats")
        echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"memory_stats","arguments":{}}}' | timeout 5 ./mcp-memory-server 2>/dev/null | grep -o '{"jsonrpc.*' | jq -r '.result.content[0].text // .error'
        ;;
        
    *)
        echo "Usage: $0 {remember|recall|stats} [args...]"
        echo ""
        echo "Examples:"
        echo "  $0 remember 'Claude Code can use MCP memory server' 'MCP integration' 'development' 'claude-code,mcp'"
        echo "  $0 recall 'running servers' '' 'server,infrastructure'"
        echo "  $0 stats"
        exit 1
        ;;
esac