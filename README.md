# Cloudera Cloud Factory MCP Server

A Model Context Protocol (MCP) server that provides tools for managing Cloudera Cloud Factory resources (formerly Taikun), including projects, virtual clusters, catalogs, and applications.

Note: The repository and binary name remain `taikun-mcp` for compatibility.

[![Release](https://img.shields.io/github/v/release/itera-io/taikun-mcp)](https://github.com/itera-io/taikun-mcp/releases)
[![CI](https://github.com/itera-io/taikun-mcp/workflows/CI/badge.svg)](https://github.com/itera-io/taikun-mcp/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/itera-io/taikun-mcp)](https://goreportcard.com/report/github.com/itera-io/taikun-mcp)

## Installation

### Option 1: Download Pre-built Binaries (Recommended)

Download the latest release for your platform from the [releases page](https://github.com/itera-io/taikun-mcp/releases).

#### Linux (x86_64)
```bash
curl -L https://github.com/itera-io/taikun-mcp/releases/latest/download/taikun-mcp_Linux_x86_64.tar.gz | tar xz
sudo mv taikun-mcp /usr/local/bin/
```

#### macOS (Intel)
```bash
curl -L https://github.com/itera-io/taikun-mcp/releases/latest/download/taikun-mcp_Darwin_x86_64.tar.gz | tar xz
sudo mv taikun-mcp /usr/local/bin/
```

#### macOS (Apple Silicon)
```bash
curl -L https://github.com/itera-io/taikun-mcp/releases/latest/download/taikun-mcp_Darwin_arm64.tar.gz | tar xz
sudo mv taikun-mcp /usr/local/bin/
```

#### Windows (PowerShell)
```powershell
Invoke-WebRequest -Uri "https://github.com/itera-io/taikun-mcp/releases/latest/download/taikun-mcp_Windows_x86_64.zip" -OutFile "taikun-mcp.zip"
Expand-Archive -Path "taikun-mcp.zip" -DestinationPath .
# Move taikun-mcp.exe to your PATH
```

### Option 2: Build from Source

#### Prerequisites
- Go 1.24 or later
- Cloudera Cloud Factory account with API access

```bash
git clone https://github.com/itera-io/taikun-mcp
cd taikun-mcp
go build -o taikun-mcp
```

### Option 3: Using Go Install

```bash
go install github.com/itera-io/taikun-mcp@latest
```

## Configuration

The server supports multiple authentication methods with the Cloudera Cloud Factory API. Choose one of the following options (legacy `TAIKUN_*` environment variables are kept for compatibility):

### Option 1: Access Key/Secret Key Authentication (Recommended)

```bash
export TAIKUN_ACCESS_KEY="your-access-key"
export TAIKUN_SECRET_KEY="your-secret-key"
export TAIKUN_AUTH_MODE="token"  # Optional, defaults to "token"
export TAIKUN_API_HOST="api.taikun.cloud"  # Optional, defaults to api.taikun.cloud
```

### Option 2: Email/Password Authentication

```bash
export TAIKUN_EMAIL="your-email@example.com"
export TAIKUN_PASSWORD="your-password"
export TAIKUN_API_HOST="api.taikun.cloud"  # Optional, defaults to api.taikun.cloud
```


### Environment File

You can also create a `.env` file with your preferred authentication method:

**For Access Key/Secret Key:**
```bash
TAIKUN_ACCESS_KEY=your-access-key
TAIKUN_SECRET_KEY=your-secret-key
TAIKUN_AUTH_MODE=token
TAIKUN_API_HOST=api.taikun.cloud
```

**For Email/Password:**
```bash
TAIKUN_EMAIL=your-email@example.com
TAIKUN_PASSWORD=your-password
TAIKUN_API_HOST=api.taikun.cloud
```

## Usage

### Starting the Server

```bash
./taikun-mcp
```

The server will start and listen for MCP requests via stdio transport.

### Connecting from Claude Desktop

Add this configuration to your Claude Desktop config using your preferred authentication method:

**For Access Key/Secret Key Authentication:**
```json
{
  "mcpServers": {
    "cloudera-cloud-factory": {
      "command": "/path/to/taikun-mcp",
      "env": {
        "TAIKUN_ACCESS_KEY": "your-access-key",
        "TAIKUN_SECRET_KEY": "your-secret-key",
        "TAIKUN_AUTH_MODE": "token"
      }
    }
  }
}
```

**For Email/Password Authentication:**
```json
{
  "mcpServers": {
    "cloudera-cloud-factory": {
      "command": "/path/to/taikun-mcp",
      "env": {
        "TAIKUN_EMAIL": "your-email@example.com",
        "TAIKUN_PASSWORD": "your-password"
      }
    }
  }
}
```

## Support

For issues and questions:
- Create an issue in this repository
- Check the [Cloudera Cloud Factory documentation](https://docs.taikun.cloud/)
- Review the [MCP specification](https://modelcontextprotocol.io/)