# Shuttl CLI

Command-line interface for deploying and managing Shuttl AI agents.

## Installation

```bash
# Build from source
nx run cli:build

# The binary will be available at apps/cli/dist/shuttl
```

## Configuration

The CLI looks for a `shuttl.json` file in the current directory or parent directories. This file defines your Shuttl application.

### shuttl.json

```json
{
  "app": "./path/to/your/app"
}
```

| Field | Type   | Required | Description                        |
|-------|--------|----------|------------------------------------|
| `app` | string | Yes      | Path to the app to run             |

## Usage

### Get Help

```bash
./dist/shuttl --help
```

### Check Version

```bash
# Full version info
./dist/shuttl version

# Short version only
./dist/shuttl version --short
```

### Development Mode

Run the app defined in `shuttl.json` in development mode:

```bash
# Run the app (uses shuttl.json in current or parent directory)
./dist/shuttl dev

# Specify port
./dist/shuttl dev --port 8080

# Use a specific config file
./dist/shuttl dev --config ./path/to/shuttl.json
```

## Development

### Build

```bash
nx run cli:build
```

### Run Tests

```bash
nx run cli:test
```

### Format Code

```bash
nx run cli:fmt
```

### Lint

```bash
nx run cli:lint
```

### Update Dependencies

```bash
nx run cli:tidy
```

## Project Structure

```
apps/cli/
├── main.go           # Entry point
├── cmd/
│   ├── root.go       # Root command and global flags
│   ├── dev.go        # Dev command
│   └── version.go    # Version command
├── config/
│   └── config.go     # Configuration file loading
├── go.mod            # Go module definition
├── project.json      # Nx project configuration
└── README.md         # This file
```

## Adding New Commands

1. Create a new file in `cmd/` (e.g., `cmd/deploy.go`)
2. Define your command using Cobra
3. Register it in `init()` with `rootCmd.AddCommand(yourCmd)`

Example:

```go
package cmd

import "github.com/spf13/cobra"

var deployCmd = &cobra.Command{
    Use:   "deploy [agent]",
    Short: "Deploy an agent to production",
    Run: func(cmd *cobra.Command, args []string) {
        // Implementation
    },
}

func init() {
    rootCmd.AddCommand(deployCmd)
}
```

