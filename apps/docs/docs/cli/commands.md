# CLI Commands

Detailed reference for all Shuttl CLI commands.

---

## shuttl dev

Run agents in development mode with an interactive TUI (Terminal User Interface).

```bash
shuttl dev [flags]
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--config` | `-c` | `./shuttl.json` | Config file path |
| `--verbose` | `-v` | `false` | Enable debug output |

### Examples

```bash
# Basic usage
shuttl dev

# With debug logging
shuttl dev -v

# Specify config file
shuttl dev --config ./configs/development.json
```

### TUI Screens

The development TUI provides four interactive screens:

| Screen | Description |
|--------|-------------|
| **Agent Select** | Browse and select from available agents in your project |
| **Chat** | Interactive chat interface to converse with your selected agent |
| **Agent Debug** | Real-time view of agent state, tool calls, LLM responses, and execution flow |
| **Program Debug** | System logs, errors, and program-level debugging information |

### Features

- **Hot Reload**: Automatically reloads when files change
- **Real-time Debugging**: Watch tool calls and LLM responses as they happen
- **Multi-agent Support**: Switch between agents without restarting
- **Conversation History**: Maintain context across multiple messages

### Navigation

Use keyboard shortcuts to navigate between screens:

| Key | Action |
|-----|--------|
| `Tab` | Switch between screens |
| `Enter` | Send message / Select agent |
| `Ctrl+C` | Exit the TUI |
| `↑/↓` | Scroll through history |

---

## shuttl serve

Run agents in production mode with three operating modes.

```bash
shuttl serve [flags]
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--port` | `-p` | `8080` | HTTP server port |
| `--config` | `-c` | `./shuttl.json` | Config file path |
| `--ssl-cert` | | | Path to SSL certificate |
| `--ssl-key` | | | Path to SSL private key |
| `--agent` | | | Serve only a specific agent |
| `--trigger` | | | Serve only a specific trigger type |
| `--invoke` | | | Invoke once and exit (for cron/Lambda) |

### Mode 1: Serve All Agents

```bash
# Serve all agents on HTTP
shuttl serve

# With SSL
shuttl serve --ssl-cert ./cert.pem --ssl-key ./key.pem

# Custom port
shuttl serve --port 443
```

### Mode 2: Serve Specific Agent or Trigger

```bash
# Serve only the SupportBot agent
shuttl serve --agent SupportBot

# Serve only API triggers (no scheduled tasks)
shuttl serve --trigger api

# Serve only rate/cron triggers (worker mode)
shuttl serve --trigger rate

# Combine: specific agent with specific trigger
shuttl serve --agent SupportBot --trigger api
```

### Mode 3: Invoke and Exit

For serverless/cron deployments, invoke a trigger once and exit:

```bash
# Invoke the rate trigger for DailyReporter
shuttl serve --invoke --agent DailyReporter --trigger rate
```

Use this for AWS Lambda, Cloud Functions, or Kubernetes CronJobs.

### Differences from dev

| Feature | `dev` | `serve` |
|---------|-------|---------|
| Interface | Interactive TUI | HTTP Server |
| Hot reload | ✅ | ❌ |
| Debug logging | Optional | Minimal |
| Performance | Development | Optimized |
| Use case | Local development | Production deployment |

See the [Deployment Guide](../getting-started/deployment.md) for detailed deployment patterns.

---

## shuttl build

Build agents for deployment.

```bash
shuttl build [flags]
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `./dist` | Output directory |
| `--config` | `-c` | `./shuttl.json` | Config file path |

### Examples

```bash
# Build with defaults
shuttl build

# Custom output directory
shuttl build --output ./build
```

### Output

Creates a deployable bundle with a full manifest:

```
dist/
├── manifest.json    # Complete deployment manifest
├── agents.json      # Agent definitions
├── triggers.json    # Trigger configurations
├── outcomes.json    # Outcome configurations
└── bundle.js        # Compiled code (if applicable)
```

The `manifest.json` contains all trigger and outcome configurations, helping you determine which infrastructure to provision for self-hosting. See the [Deployment Guide](../getting-started/deployment.md) for details.

---

## shuttl generate

Generate a new Shuttl AI agent project with all necessary scaffolding.

```bash
shuttl generate <name> --language <lang> [flags]
```

### Flags

| Flag | Short | Required | Description |
|------|-------|----------|-------------|
| `--language` | `-l` | Yes | Programming language for the project |
| `--skip-install` | | No | Skip installing dependencies |

### Supported Languages

| Language | Aliases | Description |
|----------|---------|-------------|
| `typescript` | `ts` | TypeScript/Node.js project |
| `python` | `py` | Python project with uv/pip |
| `go` | | Go project with modules |
| `java` | | Java project with Maven |
| `csharp` | `cs` | C# project with .NET |

### Examples

```bash
# Create a TypeScript project
shuttl generate my-agent --language typescript

# Create a Python project
shuttl generate my-agent -l python

# Create a Go project without installing dependencies
shuttl generate my-agent -l go --skip-install
```

### What Gets Generated

The command creates a complete project structure:

=== "TypeScript"

    ```
    my-agent/
    ├── src/
    │   └── agent.ts         # Basic agent example
    ├── package.json         # Dependencies
    ├── tsconfig.json        # TypeScript config
    └── shuttl.json          # Shuttl configuration
    ```

=== "Python"

    ```
    my-agent/
    ├── src/
    │   └── agent.py         # Basic agent example
    ├── pyproject.toml       # Dependencies
    └── shuttl.json          # Shuttl configuration
    ```

=== "Go"

    ```
    my-agent/
    ├── main.go              # Entry point
    ├── agent.go             # Basic agent example
    ├── go.mod               # Go modules
    └── shuttl.json          # Shuttl configuration
    ```

---

## shuttl login

Authenticate with Shuttl Cloud (for deployment features).

```bash
shuttl login [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--token` | Use API token instead of browser auth |

### Examples

```bash
# Interactive login (opens browser)
shuttl login

# Token-based login (for CI/CD)
shuttl login --token $SHUTTL_TOKEN
```

### Authentication Flow

1. Opens browser to `https://shuttl.dev/auth`
2. Authenticate with GitHub/Google
3. Token stored in `~/.shuttl/credentials`

---

## shuttl version

Display version and build information.

```bash
shuttl version [flags]
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--short` | `-s` | Print only version number |

### Examples

```bash
# Full version info
shuttl version
# Output:
# Shuttl CLI v1.0.0
# Built: 2025-01-03
# Go: go1.21.0
# Commit: abc1234

# Version number only
shuttl version --short
# Output:
# 1.0.0
```

---

## Environment Reference

### Required Variables

| Variable | Description |
|----------|-------------|
| `OPENAI_API_KEY` | OpenAI API key for GPT models |

### Optional Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SHUTTL_PORT` | `8080` | Server port |
| `SHUTTL_DEBUG` | `false` | Enable debug mode |
| `SHUTTL_CONFIG` | `./shuttl.json` | Default config path |
| `SLACK_BOT_TOKEN` | - | For Slack outcomes |

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Configuration error |
| 3 | Authentication error |
| 127 | Command not found |

---

## Troubleshooting

### Common Issues

??? question "Port already in use"
    ```bash
    # Find what's using the port
    lsof -i :8080
    
    # Use a different port
    shuttl dev --port 3001
    ```

??? question "Config file not found"
    ```bash
    # Check current directory
    ls -la shuttl.json
    
    # Specify path explicitly
    shuttl dev --config ./path/to/shuttl.json
    ```

??? question "Permission denied on binary"
    ```bash
    chmod +x ./shuttl
    ```

??? question "Command not found after install"
    Add to your PATH:
    ```bash
    export PATH="$HOME/.local/bin:$PATH"
    ```

---

## Next Steps

- [:rocket: Quick Start](../getting-started/quickstart.md) - Get started with Shuttl
- [:brain: Core Concepts](../concepts/index.md) - Learn the architecture
- [:mag: Examples](../examples/index.md) - Real-world usage

