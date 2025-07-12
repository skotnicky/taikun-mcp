#!/bin/bash

# Manual Integration Test Script for Taikun MCP Server
# This script tests the MCP server by sending actual JSON-RPC requests

set -e

echo "🧪 Taikun MCP Server Integration Test"
echo "====================================="

# Check for required environment variables
print_env_error() {
    echo "❌ Error: Missing Taikun authentication environment variables"
    echo ""
    echo "Please set the required environment variables:"
    echo ""
    echo "  For email/password authentication:"
    echo "    export TAIKUN_EMAIL=your-email@example.com"
    echo "    export TAIKUN_PASSWORD=your-password"
    echo ""
    echo "  For access key authentication:"
    echo "    export TAIKUN_ACCESS_KEY=your-access-key"
    echo "    export TAIKUN_SECRET_KEY=your-secret-key"
    echo "    export TAIKUN_AUTH_MODE=token   # (optional, defaults to 'token')"
    echo ""
    echo "You may also specify the Taikun API host (optional):"
    echo "    export TAIKUN_API_HOST=api.taikun.cloud"
    echo ""
    echo "Or create a .env file with these variables"
    echo ""
    exit 1
}

# Check for credentials (either method is fine)
if { [ -n "$TAIKUN_EMAIL" ] && [ -n "$TAIKUN_PASSWORD" ]; } || \
   { [ -n "$TAIKUN_ACCESS_KEY" ] && [ -n "$TAIKUN_SECRET_KEY" ]; }
then
    # At least one valid authentication method is set, continue...
    :
else
    print_env_error
fi
# Set default API host if not provided
if [ -z "$TAIKUN_API_HOST" ]; then
    export TAIKUN_API_HOST="api.taikun.cloud"
fi

echo "📊 Test Configuration:"
echo "  Email: $TAIKUN_EMAIL"
echo "  API Host: $TAIKUN_API_HOST"
echo ""

# Load environment from .env if it exists
if [ -f .env ]; then
    echo "📁 Loading environment from .env file..."
    set -a
    source .env
    set +a
fi

# Build the project
echo "🔨 Building project..."
make build

if [ $? -ne 0 ]; then
    echo "❌ Build failed!"
    exit 1
fi

echo "✅ Build successful"
echo ""

echo "🚀 Ready to run integration tests..."

# Test 1: List available tools
echo "📋 Test 1: List available tools"
echo '{"jsonrpc": "2.0", "method": "tools/list", "id": 1}' | timeout 10s ./taikun-mcp
echo ""

# Test 2: Test list-projects tool
echo "🔧 Test 2: Test list-projects tool"
echo '{"jsonrpc": "2.0", "method": "tools/call", "params": {"name": "list-projects", "arguments": {"limit": 5}}, "id": 2}' | timeout 15s ./taikun-mcp
echo ""

# Test 3: Test catalog listing
echo "📚 Test 3: Test catalog listing"
echo '{"jsonrpc": "2.0", "method": "tools/call", "params": {"name": "list-catalogs", "arguments": {"limit": 5}}, "id": 3}' | timeout 15s ./taikun-mcp
echo ""

# Test 4: Test error handling with invalid arguments
echo "❌ Test 4: Test error handling"
echo '{"jsonrpc": "2.0", "method": "tools/call", "params": {"name": "create-virtual-cluster", "arguments": {"invalidParam": "test"}}, "id": 4}' | timeout 10s ./taikun-mcp
echo ""

echo "✅ Integration tests completed!"
echo ""
echo "📝 Check the following for more details:"
echo "  - Server logs: /tmp/taikun_mcp_server.log"
echo "  - Test output above"
echo ""
echo "🧪 Manual Testing:"
echo "You can manually test more scenarios by running:"
echo "  echo '{\"jsonrpc\": \"2.0\", \"method\": \"tools/list\", \"id\": 1}' | ./taikun-mcp"
echo ""
echo "📝 Example catalog app management tests:"
echo "  # Add app to catalog:"
echo "  echo '{\"jsonrpc\": \"2.0\", \"method\": \"tools/call\", \"params\": {\"name\": \"add-app-to-catalog\", \"arguments\": {\"catalogId\": 1, \"repository\": \"bitnami\", \"packageName\": \"nginx\"}}, \"id\": 10}' | ./taikun-mcp"
echo ""
echo "  # List apps in catalog:"
echo "  echo '{\"jsonrpc\": \"2.0\", \"method\": \"tools/call\", \"params\": {\"name\": \"list-catalog-apps\", \"arguments\": {\"catalogId\": 1}}, \"id\": 11}' | ./taikun-mcp"
echo ""
echo "  # Remove app from catalog:"
echo "  echo '{\"jsonrpc\": \"2.0\", \"method\": \"tools/call\", \"params\": {\"name\": \"remove-app-from-catalog\", \"arguments\": {\"catalogId\": 1, \"repository\": \"bitnami\", \"packageName\": \"nginx\"}}, \"id\": 12}' | ./taikun-mcp"
echo ""
echo "📖 Available tools to test:"
echo "  - list-projects"
echo "  - create-virtual-cluster"
echo "  - delete-virtual-cluster"
echo "  - list-virtual-clusters"
echo "  - create-catalog"
echo "  - list-catalogs"
echo "  - update-catalog"
echo "  - delete-catalog"
echo "  - bind-projects-to-catalog"
echo "  - unbind-projects-from-catalog"
echo "  - add-app-to-catalog"
echo "  - remove-app-from-catalog"
echo "  - list-catalog-apps"
echo "  - install-app"
echo "  - list-apps"
echo "  - get-app"
echo "  - update-sync-app"
echo "  - uninstall-app"