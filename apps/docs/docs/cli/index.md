# CLI Reference

The Shuttl CLI provides tools for developing, testing, and deploying AI agents.

---

## Installation

### Quick Install

```bash
curl -fsSL https://shuttl.dev/install.sh | bash
```

### Manual Download

Download from [GitHub Releases](https://github.com/shuttl-io/shuttl/releases):

| Platform | Download |
|----------|----------|
| macOS (Apple Silicon) | `shuttl-darwin-arm64` |
| macOS (Intel) | `shuttl-darwin-amd64` |
| Linux (x64) | `shuttl-linux-amd64` |
| Linux (ARM64) | `shuttl-linux-arm64` |
| Windows (x64) | `shuttl-windows-amd64.exe` |

### Build from Source

```bash
git clone https://github.com/shuttl-io/shuttl
cd shuttl/apps/cli
go build -o shuttl .
```

### Verify Installation

```bash
shuttl version
```

---

## Configuration

### shuttl.json

The CLI looks for a `shuttl.json` file in the current directory or parent directories.

```json
{
    "app": "node --require ts-node/register ./src/main.ts"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `app` | `string` | Yes | Command to run your application |

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SHUTTL_PORT` | HTTP server port (for `serve` command) | `8080` |
| `SHUTTL_DEBUG` | Enable debug logging | `false` |
| `OPENAI_API_KEY` | OpenAI API key | - |

---

## Global Flags

These flags work with all commands:

| Flag | Short | Description |
|------|-------|-------------|
| `--verbose` | `-v` | Enable debug logging |
| `--config` | `-c` | Path to shuttl.json |
| `--help` | `-h` | Show help |

---

## Commands Overview

| Command | Description |
|---------|-------------|
| `dev` | Run agents in development mode |
| `serve` | Run agents in production mode |
| `build` | Build agents for deployment |
| `generate` | Generate code and configs |
| `login` | Authenticate with Shuttl Cloud |
| `version` | Show version info |

---

## Development Workflow

```bash
# 1. Create your agent
vim src/agent.ts

# 2. Start the development TUI
shuttl dev

# 3. Test your agent using the interactive Chat screen
#    (Select agent → Chat → Type your message)

# 4. Build for production
shuttl build

# 5. Deploy
shuttl deploy
```

---

## Next Steps

- [:book: Command Reference](commands.md) - Detailed command documentation
- [:rocket: Quick Start](../getting-started/quickstart.md) - Get started tutorial
- [:mag: Examples](../examples/index.md) - Real-world usage

