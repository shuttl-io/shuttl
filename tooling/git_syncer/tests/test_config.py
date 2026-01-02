"""Tests for configuration module."""

from pathlib import Path

import pytest

from git_syncer.config import (
    DEFAULT_CLONE_DIR,
    ProjectMapping,
    SyncConfig,
    SyncState,
    create_default_config,
)


class TestProjectMapping:
    """Tests for ProjectMapping."""

    def test_resolved_public_path_default(self):
        """Test that public_path defaults to private_path."""
        mapping = ProjectMapping(private_path="packages/core")
        assert mapping.resolved_public_path == "packages/core"

    def test_resolved_public_path_custom(self):
        """Test custom public_path."""
        mapping = ProjectMapping(
            private_path="packages/core",
            public_path="libs/core",
        )
        assert mapping.resolved_public_path == "libs/core"

    def test_default_enabled(self):
        """Test that projects are enabled by default."""
        mapping = ProjectMapping(private_path="packages/core")
        assert mapping.enabled is True

    def test_exclude_patterns(self):
        """Test exclude patterns."""
        mapping = ProjectMapping(
            private_path="packages/core",
            exclude_patterns=["*.secret", ".env*"],
        )
        assert len(mapping.exclude_patterns) == 2


class TestSyncConfig:
    """Tests for SyncConfig."""

    def test_create_config(self, temp_dir: Path):
        """Test creating a basic config."""
        config = SyncConfig(
            private_repo_path=temp_dir / "private",
            public_repo_url="git@github.com:org/public-repo.git",
        )
        assert config.private_branch == "main"
        assert config.public_branch == "main"

    def test_public_repo_path_from_url(self, temp_dir: Path):
        """Test that public_repo_path is derived from URL."""
        config = SyncConfig(
            private_repo_path=temp_dir / "private",
            public_repo_url="git@github.com:org/my-public-repo.git",
        )
        assert config.public_repo_path == DEFAULT_CLONE_DIR / "my-public-repo"

    def test_public_repo_path_custom(self, temp_dir: Path):
        """Test custom public_repo_clone_path."""
        custom_path = temp_dir / "custom-clone"
        config = SyncConfig(
            private_repo_path=temp_dir / "private",
            public_repo_url="git@github.com:org/repo.git",
            public_repo_clone_path=custom_path,
        )
        assert config.public_repo_path == custom_path

    def test_extract_repo_name_ssh(self, temp_dir: Path):
        """Test extracting repo name from SSH URL."""
        config = SyncConfig(
            private_repo_path=temp_dir / "private",
            public_repo_url="git@github.com:org/my-repo.git",
        )
        assert config.public_repo_path.name == "my-repo"

    def test_extract_repo_name_https(self, temp_dir: Path):
        """Test extracting repo name from HTTPS URL."""
        config = SyncConfig(
            private_repo_path=temp_dir / "private",
            public_repo_url="https://github.com/org/another-repo.git",
        )
        assert config.public_repo_path.name == "another-repo"

    def test_extract_repo_name_no_git_suffix(self, temp_dir: Path):
        """Test extracting repo name from URL without .git suffix."""
        config = SyncConfig(
            private_repo_path=temp_dir / "private",
            public_repo_url="https://github.com/org/some-repo",
        )
        assert config.public_repo_path.name == "some-repo"

    def test_yaml_roundtrip(self, temp_dir: Path):
        """Test saving and loading from YAML."""
        config = SyncConfig(
            private_repo_path=temp_dir / "private",
            public_repo_url="git@github.com:org/test-repo.git",
            projects=[
                ProjectMapping(private_path="packages/core"),
            ],
        )

        yaml_path = temp_dir / "config.yaml"
        config.to_yaml(yaml_path)

        loaded = SyncConfig.from_yaml(yaml_path)
        assert loaded.private_repo_path == config.private_repo_path
        assert loaded.public_repo_url == config.public_repo_url
        assert len(loaded.projects) == 1

    def test_get_enabled_projects(self, temp_dir: Path):
        """Test filtering enabled projects."""
        config = SyncConfig(
            private_repo_path=temp_dir / "private",
            public_repo_url="git@github.com:org/repo.git",
            projects=[
                ProjectMapping(private_path="packages/core", enabled=True),
                ProjectMapping(private_path="packages/old", enabled=False),
            ],
        )

        enabled = config.get_enabled_projects()
        assert len(enabled) == 1
        assert enabled[0].private_path == "packages/core"

    def test_global_exclude_patterns_default(self, temp_dir: Path):
        """Test default global exclude patterns."""
        config = SyncConfig(
            private_repo_path=temp_dir / "private",
            public_repo_url="git@github.com:org/repo.git",
        )
        assert ".env" in config.global_exclude_patterns
        assert "node_modules/" in config.global_exclude_patterns


class TestSyncState:
    """Tests for SyncState."""

    def test_load_nonexistent(self, temp_dir: Path):
        """Test loading from nonexistent file returns empty state."""
        state = SyncState.load(temp_dir / "nonexistent.json")
        assert state.last_private_commit is None
        assert state.last_public_commit is None

    def test_save_and_load(self, temp_dir: Path):
        """Test saving and loading state."""
        state = SyncState(
            last_private_commit="abc123",
            last_public_commit="def456",
            commit_mapping={"abc123": "def456"},
        )

        state_path = temp_dir / "state.json"
        state.save(state_path)

        loaded = SyncState.load(state_path)
        assert loaded.last_private_commit == "abc123"
        assert loaded.last_public_commit == "def456"
        assert loaded.commit_mapping == {"abc123": "def456"}


class TestCreateDefaultConfig:
    """Tests for create_default_config."""

    def test_basic_creation(self, temp_dir: Path):
        """Test basic config creation."""
        config = create_default_config(
            private_repo_path=temp_dir / "private",
            public_repo_url="git@github.com:org/repo.git",
        )
        assert config.private_repo_path == temp_dir / "private"
        assert config.public_repo_url == "git@github.com:org/repo.git"
        assert len(config.projects) == 0

    def test_with_projects(self, temp_dir: Path):
        """Test config creation with projects."""
        config = create_default_config(
            private_repo_path=temp_dir / "private",
            public_repo_url="git@github.com:org/repo.git",
            projects=["packages/core", "packages/utils"],
        )
        assert len(config.projects) == 2
        assert config.projects[0].private_path == "packages/core"

    def test_with_custom_clone_path(self, temp_dir: Path):
        """Test config creation with custom clone path."""
        clone_path = temp_dir / "my-clone"
        config = create_default_config(
            private_repo_path=temp_dir / "private",
            public_repo_url="git@github.com:org/repo.git",
            public_repo_clone_path=clone_path,
        )
        assert config.public_repo_path == clone_path
