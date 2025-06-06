#!/bin/bash

# MCP Memory Server Docker Setup Script

set -e

echo "🐳 Setting up MCP Memory Server with Docker..."

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo "❌ Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

# Build the containers
echo "🔨 Building Docker containers..."
docker-compose build

# Create and start the services
echo "🚀 Starting MCP Memory Server services..."
docker-compose up -d

# Wait for services to be healthy
echo "⏳ Waiting for services to be ready..."
sleep 10

# Check service status
echo "📊 Service Status:"
docker-compose ps

# Show logs
echo "📝 Recent logs:"
docker-compose logs --tail=20

echo ""
echo "✅ MCP Memory Server is now running!"
echo ""
echo "🔗 Access points:"
echo "  • Memory Reporting Dashboard: http://localhost:9000"
echo "  • MCP Server: Running in container (stdin/stdout via Docker)"
echo ""
echo "📋 Useful commands:"
echo "  • View logs: docker-compose logs -f"
echo "  • Stop services: docker-compose down"
echo "  • Restart services: docker-compose restart"
echo "  • View status: docker-compose ps"
echo ""