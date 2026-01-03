# Installation

Complete installation guide for the Shuttl SDK and CLI.

---

## SDK Installation

Shuttl provides SDKs for TypeScript and Python. Choose your preferred language:

### TypeScript / JavaScript

=== "npm"

    ```bash
    npm install @shuttl-io/core
    ```

=== "pnpm"

    ```bash
    pnpm add @shuttl-io/core
    ```

=== "yarn"

    ```bash
    yarn add @shuttl-io/core
    ```

=== "bun"

    ```bash
    bun add @shuttl-io/core
    ```

**Requirements:**

- Node.js 18.0.0 or later
- TypeScript 5.0+ (recommended)

### Python

=== "pip"

    ```bash
    pip install shuttl
    ```

=== "uv"

    ```bash
    uv add shuttl
    ```

=== "poetry"

    ```bash
    poetry add shuttl
    ```

**Requirements:**

- Python 3.9 or later

---

## CLI Installation

The Shuttl CLI is a standalone binary that works on macOS, Linux, and Windows.

### Quick Install (Recommended)

```bash
curl -fsSL https://shuttl.dev/install.sh | bash
```

This script automatically:

- Detects your OS and architecture
- Downloads the correct binary
- Installs to `/usr/local/bin` (or `~/.local/bin`)
- Verifies the installation

### Manual Download

Download the appropriate binary from [GitHub Releases](https://github.com/shuttl-io/shuttl/releases):

| Platform | Architecture | Download |
|----------|--------------|----------|
| macOS | Apple Silicon (M1/M2/M3) | `shuttl-darwin-arm64` |
| macOS | Intel | `shuttl-darwin-amd64` |
| Linux | x64 | `shuttl-linux-amd64` |
| Linux | ARM64 | `shuttl-linux-arm64` |
| Windows | x64 | `shuttl-windows-amd64.exe` |
| Windows | ARM64 | `shuttl-windows-arm64.exe` |

```bash
# Example: Linux x64
curl -LO https://github.com/shuttl-io/shuttl/releases/latest/download/shuttl-linux-amd64
chmod +x shuttl-linux-amd64
sudo mv shuttl-linux-amd64 /usr/local/bin/shuttl
```

### Build from Source

If you have Go 1.21+ installed:

```bash
git clone https://github.com/shuttl-io/shuttl
cd shuttl/apps/cli
go build -o shuttl .
sudo mv shuttl /usr/local/bin/
```

### Verify Installation

```bash
shuttl version
```

Expected output:

```
Shuttl CLI v1.0.0
Built: 2025-01-03
Go: go1.21.0
```

---

## Project Setup

### Initialize a New Project

Create a project directory and initialize:

```bash
mkdir my-shuttl-project
cd my-shuttl-project
```

=== "TypeScript"

    ```bash
    npm init -y
    npm install @shuttl-io/core typescript ts-node @types/node
    npx tsc --init
    ```

=== "Python"

    ```bash
    python -m venv .venv
    source .venv/bin/activate
    pip install shuttl
    ```

### Configuration File

Create `shuttl.json` in your project root:

```json
{
    "app": "node --require ts-node/register ./src/main.ts"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `app` | string | Yes | Command to run your application |

---

## Environment Variables

Shuttl uses environment variables for sensitive configuration:

```bash
# Required: Your LLM provider API key
export OPENAI_API_KEY="sk-..."

# Optional: Custom API port
export SHUTTL_PORT="8080"

# Optional: Enable debug logging
export SHUTTL_DEBUG="true"
```

!!! tip "Using .env files"
    For local development, you can use a `.env` file with a library like `dotenv`:
    
    ```bash
    # .env
    OPENAI_API_KEY=sk-your-key-here
    ```

    Make sure to add `.env` to your `.gitignore`!

---

## IDE Setup

### VS Code

Install recommended extensions for the best experience:

```json
// .vscode/extensions.json
{
    "recommendations": [
        "ms-vscode.vscode-typescript-next",
        "esbenp.prettier-vscode",
        "bradlc.vscode-tailwindcss"
    ]
}
```

### TypeScript Configuration

Recommended `tsconfig.json` settings:

```json
{
    "compilerOptions": {
        "target": "ES2022",
        "module": "NodeNext",
        "moduleResolution": "NodeNext",
        "strict": true,
        "esModuleInterop": true,
        "skipLibCheck": true,
        "outDir": "./dist"
    },
    "include": ["src/**/*"],
    "exclude": ["node_modules"]
}
```

---

## Troubleshooting

### Common Issues

??? question "CLI: Command not found"
    Make sure the CLI is in your PATH:
    
    ```bash
    # Add to ~/.bashrc or ~/.zshrc
    export PATH="$HOME/.local/bin:$PATH"
    ```
    
    Then reload your shell: `source ~/.bashrc`

??? question "SDK: Module not found"
    Clear your package manager cache and reinstall:
    
    ```bash
    rm -rf node_modules package-lock.json
    npm install
    ```

??? question "API Key errors"
    Verify your environment variable is set:
    
    ```bash
    echo $OPENAI_API_KEY
    ```
    
    Make sure there are no extra spaces or quotes.

??? question "Port already in use"
    Another process is using the default port. Either stop it or use a different port:
    
    ```bash
    shuttl dev --port 3001
    ```

---

## Next Steps

Now that you're set up:

- [:rocket: Build your first agent](first-agent.md)
- [:book: Learn core concepts](../concepts/index.md)
- [:mag: Explore examples](../examples/index.md)

