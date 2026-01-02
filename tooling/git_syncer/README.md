# Git Syncer

A two-way sync tool for synchronizing specific projects between a private monorepo and a public open-source monorepo while preserving git commit history and messages.

## Features

- **Two-way sync**: Sync commits from private to public or public to private
- **Remote URL support**: Specify public repo as a git URL - automatically cloned and managed
- **Selective project sync**: Only sync specified projects/packages
- **Commit preservation**: Maintains original commit messages, authors, and dates
- **Tag syncing**: Automatically syncs tags pointing to synced commits
- **Pattern-based filtering**: Exclude sensitive files with glob patterns
- **State tracking**: Tracks sync progress via commit messages
- **Dry-run mode**: Preview changes before applying them
- **Full sync mode**: Initial sync or re-sync all files

## Installation

```bash
cd tooling/git_syncer
uv sync
```

## Quick Start

### 1. Initialize Configuration

```bash
git-syncer init \
  --private-repo /path/to/private-monorepo \
  --public-repo-url git@github.com:your-org/public-repo.git \
  --project packages/core \
  --project packages/utils \
  --output git_syncer.yaml
```

The public repo will be automatically cloned to `~/.git_syncer/repos/<repo-name>`. You can override this with `--public-repo-clone-path`.

### 2. Add More Projects

```bash
git-syncer add-project packages/api --exclude "*.secrets.yaml"
```

### 3. Preview Sync

```bash
git-syncer preview --direction private-to-public
```

### 4. Perform Sync

```bash
# Sync new commits
git-syncer sync --direction private-to-public

# Full sync (all files, creates single commit)
git-syncer sync --direction private-to-public --full

# Dry run (preview without changes)
git-syncer sync --direction private-to-public --dry-run
```

### 5. Check Status

```bash
git-syncer status
```

## Configuration

The configuration file (`git_syncer.yaml`) controls sync behavior:

```yaml
# Private repository (local path)
private_repo_path: /path/to/private-monorepo
private_remote: origin
private_branch: main

# Public repository (remote URL - will be cloned automatically)
public_repo_url: git@github.com:your-org/public-repo.git
public_repo_clone_path: null  # Optional: defaults to ~/.git_syncer/repos/<repo-name>
public_remote: origin
public_branch: main

# Projects to sync
projects:
  - private_path: packages/core
    public_path: packages/core  # Optional, defaults to private_path
    enabled: true
    exclude_patterns:
      - "*.secret"
      - ".env*"
    include_patterns: []  # If set, only these patterns are included

  - private_path: apps/api
    public_path: apps/api
    enabled: true

# Global exclude patterns (applied to all projects)
global_exclude_patterns:
  - ".env"
  - ".env.*"
  - "*.secret"
  - "__pycache__/"
  - "node_modules/"
  - ".nx/"

# Sync behavior
commit_prefix: "[sync]"  # Prefix added to synced commit messages
dry_run: false
auto_push: false
squash_commits: false  # If true, squash multiple commits into one
sync_tags: true  # Sync tags pointing to synced commits
```

## CLI Commands

### `init`
Create a new configuration file.

```bash
git-syncer init \
  -p /path/to/private/repo \
  -P git@github.com:org/public-repo.git \
  -j packages/core \
  -o config.yaml
```

Options:
- `-p, --private-repo`: Path to the local private monorepo
- `-P, --public-repo-url`: Git remote URL for the public repo
- `--public-repo-clone-path`: Optional local path to clone the public repo
- `-j, --project`: Project paths to sync (can be repeated)
- `-o, --output`: Output config file path

### `sync`
Perform the synchronization.

```bash
git-syncer sync -c config.yaml -d private-to-public [--dry-run] [--full] [--auto-push]
```

Options:
- `-c, --config`: Path to config file (default: `git_syncer.yaml`)
- `-d, --direction`: `private-to-public` or `public-to-private`
- `--dry-run`: Preview without making changes
- `--full`: Sync all files (not just new commits)
- `--auto-push`: Push after successful sync

### `preview`
Preview pending commits without syncing.

```bash
git-syncer preview -c config.yaml -d private-to-public
```

### `status`
Show current sync status.

```bash
git-syncer status -c config.yaml
```

### `add-project`
Add a project to the configuration.

```bash
git-syncer add-project packages/new-pkg --public-path libs/new-pkg -e "*.secret"
```

### `remove-project`
Remove a project from the configuration.

```bash
git-syncer remove-project packages/old-pkg
```

### `list-projects`
List all configured projects.

```bash
git-syncer list-projects -c config.yaml
```

### `reset-state`
Reset sync state (for re-syncing from scratch).

```bash
git-syncer reset-state -c config.yaml
```

## NX Integration

The package includes NX targets for easy integration:

```bash
# Sync to public repo
nx run git_syncer:sync-to-public

# Sync from public repo
nx run git_syncer:sync-from-public

# Run CLI directly
nx run git_syncer:run -- status
```

## How It Works

1. **Auto-Clone**: The public repo is automatically cloned from the URL when first needed
2. **Commit Tracking**: The syncer tracks which commits have been synced via `synced_from:` trailer in commit messages
3. **Selective Sync**: Only files within configured projects are synced
4. **Pattern Filtering**: Files matching exclude patterns are never synced
5. **Commit Recreation**: Each source commit is recreated in the destination with:
   - Original commit message (with configurable prefix)
   - Original author information
   - Original timestamp
   - A `synced_from: <source-commit-hash>` trailer for traceability
6. **Tag Syncing**: After commits are synced, tags pointing to source commits are created on the corresponding destination commits

## Tag Syncing

The syncer automatically syncs tags when `sync_tags: true` (the default). When a commit is synced:

1. The syncer checks if the source commit has any tags pointing to it
2. For each tag, it creates the same tag on the destination commit
3. If `auto_push` is enabled, tags are pushed along with commits

### Tag Syncing Behavior

- **Lightweight tags**: Created as-is
- **Annotated tags**: Created as lightweight (original annotation not preserved)
- **Existing tags**: Skipped (won't overwrite existing tags in destination)

### Disable Tag Syncing

To disable tag syncing, set in your config:

```yaml
sync_tags: false
```

## GitHub Actions Usage

To use git-syncer in GitHub Actions, you need to provide a Personal Access Token (PAT) for authentication to the public repository.

### 1. Create a PAT

1. Go to GitHub Settings → Developer settings → Personal access tokens → Fine-grained tokens
2. Create a token with `contents: write` permission for your public repo
3. Add it as a repository secret named `PUBLIC_REPO_PAT`

### 2. GitHub Actions Workflow

```yaml
name: Sync to Public Repo

on:
  push:
    branches: [main]

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history needed for sync

      - name: Setup Python
        uses: actions/setup-python@v5
        with:
          python-version: '3.12'

      - name: Install uv
        uses: astral-sh/setup-uv@v4

      - name: Install git-syncer
        run: |
          cd tooling/git_syncer
          uv sync

      - name: Configure Git
        run: |
          git config --global user.name "GitHub Actions Bot"
          git config --global user.email "actions@github.com"

      - name: Sync to public repo
        env:
          GIT_SYNCER_TOKEN: ${{ secrets.PUBLIC_REPO_PAT }}
        run: |
          cd tooling/git_syncer
          uv run git-syncer sync --direction private-to-public --auto-push
```

### Authentication Options

You can provide the token in two ways:

1. **Environment variable** (recommended for CI):
   ```bash
   export GIT_SYNCER_TOKEN=ghp_xxxxx
   git-syncer sync --direction private-to-public
   ```

2. **Command line option**:
   ```bash
   git-syncer sync --direction private-to-public --token ghp_xxxxx
   ```

The token automatically converts SSH URLs (`git@github.com:org/repo.git`) to authenticated HTTPS URLs.

## Best Practices

1. **Initial Sync**: Use `--full` for the first sync to copy all files
2. **Regular Sync**: Run without `--full` to only sync new commits
3. **Sensitive Files**: Always add sensitive patterns to `global_exclude_patterns`
4. **Testing**: Use `--dry-run` before actual sync
5. **CI/CD**: Add `--auto-push` for automated pipelines
6. **Tokens**: Use `GIT_SYNCER_TOKEN` env var in CI to avoid exposing tokens in logs

## Troubleshooting

### "No commits to sync"
- Check that there are new commits in the source repo
- Verify project paths are correct
- Use `reset-state` if state is corrupted

### Commits not being picked up
- Ensure commits touch files within configured project paths
- Check exclude patterns aren't filtering all changes

### Merge conflicts
- Resolve conflicts in destination repo manually
- Then run sync again

## License

Internal tool - see repository license.

