"""
Configuration handling for git_syncer.

Defines the configuration schema and provides methods for loading/saving
sync configuration from YAML files.
"""

from pathlib import Path
from typing import Literal

import yaml
from pydantic import BaseModel, Field, model_validator


# Default location for cloned public repos
DEFAULT_CLONE_DIR = Path.home() / ".git_syncer" / "repos"


class ProjectMapping(BaseModel):
    """Mapping configuration for a single project to sync."""

    # Path relative to private repo root
    private_path: str = Field(
        ..., description="Path to the project in the private monorepo"
    )
    # Path relative to public repo root (defaults to same as private_path)
    public_path: str | None = Field(
        None,
        description="Path in the public repo (defaults to same as private_path)",
    )
    # Whether to include this project in sync
    enabled: bool = Field(default=True, description="Whether to sync this project")
    # Optional list of files/patterns to exclude from sync
    exclude_patterns: list[str] = Field(
        default_factory=list,
        description="Glob patterns for files to exclude from sync",
    )
    # Optional list of files/patterns to include (overrides exclude)
    include_patterns: list[str] = Field(
        default_factory=list,
        description="Glob patterns for files to explicitly include",
    )

    @property
    def resolved_public_path(self) -> str:
        """Get the public path, defaulting to private path if not set."""
        return self.public_path or self.private_path


class SyncConfig(BaseModel):
    """Main configuration for the git syncer."""

    # Private monorepo configuration
    private_repo_path: Path = Field(
        ..., description="Path to the private monorepo root"
    )
    private_remote: str = Field(
        default="origin", description="Git remote name for the private repo"
    )
    private_branch: str = Field(
        default="main", description="Default branch in the private repo"
    )

    # Public monorepo configuration - use remote URL
    public_repo_url: str = Field(
        ..., description="Git remote URL for the public monorepo (e.g., git@github.com:org/repo.git)"
    )
    public_repo_clone_path: Path | None = Field(
        default=None,
        description="Local path where the public repo will be cloned. Defaults to ~/.git_syncer/repos/<repo-name>",
    )
    public_remote: str = Field(
        default="origin", description="Git remote name for the public repo"
    )
    public_branch: str = Field(
        default="main", description="Default branch in the public repo"
    )

    @property
    def public_repo_path(self) -> Path:
        """Get the local path for the public repo (computed from URL if not set)."""
        if self.public_repo_clone_path:
            return self.public_repo_clone_path
        # Derive repo name from URL
        repo_name = self._extract_repo_name(self.public_repo_url)
        return DEFAULT_CLONE_DIR / repo_name

    @staticmethod
    def _extract_repo_name(url: str) -> str:
        """Extract repository name from a git URL."""
        # Handle various URL formats:
        # git@github.com:org/repo.git
        # https://github.com/org/repo.git
        # https://github.com/org/repo
        name = url.rstrip("/")
        if name.endswith(".git"):
            name = name[:-4]
        # Get the last part after / or :
        if "/" in name:
            name = name.rsplit("/", 1)[-1]
        elif ":" in name:
            name = name.rsplit(":", 1)[-1]
        return name or "public-repo"

    # Projects to sync
    projects: list[ProjectMapping] = Field(
        default_factory=list, description="List of projects to synchronize"
    )

    # Global exclude patterns (applied to all projects)
    global_exclude_patterns: list[str] = Field(
        default_factory=lambda: [
            ".env",
            ".env.*",
            "*.secret",
            "*.secrets",
            ".secrets/",
            "__pycache__/",
            "*.pyc",
            ".git/",
            "node_modules/",
            ".nx/",
        ],
        description="Patterns to exclude from all projects",
    )

    # Commit message prefix for synced commits
    commit_prefix: str = Field(
        default="[sync]",
        description="Prefix to add to commit messages when syncing",
    )

    # State file to track sync progress
    state_file: Path = Field(
        default=Path(".git_syncer_state.json"),
        description="Path to the sync state file",
    )

    # Sync behavior settings
    dry_run: bool = Field(
        default=False, description="If true, don't actually make changes"
    )
    auto_push: bool = Field(
        default=False, description="Automatically push after sync"
    )
    squash_commits: bool = Field(
        default=False,
        description="Squash multiple commits into one when syncing",
    )

    @classmethod
    def from_yaml(cls, path: Path) -> "SyncConfig":
        """Load configuration from a YAML file."""
        with open(path) as f:
            data = yaml.safe_load(f)
        return cls.model_validate(data)

    def to_yaml(self, path: Path) -> None:
        """Save configuration to a YAML file."""
        with open(path, "w") as f:
            yaml.dump(
                self.model_dump(mode="json"),
                f,
                default_flow_style=False,
                sort_keys=False,
            )

    def get_enabled_projects(self) -> list[ProjectMapping]:
        """Get only the enabled projects."""
        return [p for p in self.projects if p.enabled]


class SyncState(BaseModel):
    """Tracks the state of synchronization between repos."""

    # Last synced commit hash in private repo
    last_private_commit: str | None = Field(
        None, description="Last synced commit hash from private repo"
    )
    # Last synced commit hash in public repo
    last_public_commit: str | None = Field(
        None, description="Last synced commit hash from public repo"
    )
    # Mapping of private commit hashes to public commit hashes
    commit_mapping: dict[str, str] = Field(
        default_factory=dict,
        description="Mapping of private commit hashes to public commit hashes",
    )
    # Last sync timestamp
    last_sync_timestamp: str | None = Field(
        None, description="ISO timestamp of last sync"
    )
    # Sync direction of last sync
    last_sync_direction: Literal["private-to-public", "public-to-private"] | None = (
        Field(None, description="Direction of last sync")
    )

    @classmethod
    def load(cls, path: Path) -> "SyncState":
        """Load state from a JSON file."""
        import json

        if not path.exists():
            return cls()
        with open(path) as f:
            data = json.load(f)
        return cls.model_validate(data)

    def save(self, path: Path) -> None:
        """Save state to a JSON file."""
        import json

        path.parent.mkdir(parents=True, exist_ok=True)
        with open(path, "w") as f:
            json.dump(self.model_dump(mode="json"), f, indent=2)


def create_default_config(
    private_repo_path: Path,
    public_repo_url: str,
    projects: list[str] | None = None,
    public_repo_clone_path: Path | None = None,
) -> SyncConfig:
    """Create a default configuration with sensible defaults."""
    project_mappings = []
    if projects:
        for project in projects:
            project_mappings.append(ProjectMapping(private_path=project))

    return SyncConfig(
        private_repo_path=private_repo_path,
        public_repo_url=public_repo_url,
        public_repo_clone_path=public_repo_clone_path,
        projects=project_mappings,
    )

