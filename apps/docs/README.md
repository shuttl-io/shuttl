# Shuttl Documentation

Public documentation website for the Shuttl AI agent framework.

Built with [MkDocs](https://www.mkdocs.org/) and the [shadcn theme](https://github.com/asiffer/mkdocs-shadcn).

## Development

### Prerequisites

- Python 3.9+
- uv (recommended) or pip

### Setup

```bash
cd apps/docs

# Install dependencies
uv sync

# Or with pip
pip install -e .
```

### Run locally

```bash
uv run mkdocs serve
```

Visit http://localhost:8000

### Build

```bash
uv run mkdocs build
```

Output is in `site/` directory.

## Structure

```
apps/docs/
├── docs/
│   ├── index.md                 # Landing page
│   ├── getting-started/
│   │   ├── quickstart.md        # Quick start guide
│   │   ├── installation.md      # Full installation guide
│   │   └── first-agent.md       # First agent tutorial
│   ├── concepts/
│   │   ├── index.md             # Architecture overview
│   │   ├── agents.md            # Agent documentation
│   │   ├── tools.md             # Tools & toolkits
│   │   ├── triggers.md          # Trigger types
│   │   ├── outcomes.md          # Outcome destinations
│   │   └── models.md            # LLM configuration
│   ├── cli/
│   │   ├── index.md             # CLI overview
│   │   └── commands.md          # Command reference
│   └── examples/
│       ├── index.md             # Examples overview
│       ├── weather-agent.md     # Weather bot example
│       └── scheduled-tasks.md   # Scheduled agent examples
├── mkdocs.yml                   # MkDocs configuration
├── pyproject.toml               # Python dependencies
└── README.md                    # This file
```

## Contributing

1. Create/edit markdown files in `docs/`
2. Preview with `uv run mkdocs serve`
3. Submit a PR

## License

MIT

