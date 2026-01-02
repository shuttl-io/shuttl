"""Tests for git operations module."""

from pathlib import Path

import pytest
from git import Repo

from git_syncer.config import ProjectMapping, SyncConfig
from git_syncer.git_ops import (
    GitRepository,
    filter_commit_files,
    should_include_file,
)


class TestGitRepository:
    """Tests for GitRepository wrapper."""

    def test_init_valid_repo(self, private_repo: Path):
        """Test initializing with a valid git repo."""
        git_repo = GitRepository(private_repo)
        assert git_repo.path == private_repo.resolve()

    def test_init_invalid_repo(self, temp_dir: Path):
        """Test initializing with invalid path raises error."""
        invalid_path = temp_dir / "not-a-repo"
        invalid_path.mkdir()

        with pytest.raises(ValueError, match="Not a valid git repository"):
            GitRepository(invalid_path)

    def test_get_current_branch(self, private_repo: Path):
        """Test getting current branch."""
        git_repo = GitRepository(private_repo)
        # Default branch should be master or main
        branch = git_repo.get_current_branch()
        assert branch in ["master", "main"]

    def test_get_current_commit(self, private_repo: Path):
        """Test getting current commit hash."""
        git_repo = GitRepository(private_repo)
        commit_hash = git_repo.get_current_commit()
        assert len(commit_hash) == 40  # Full SHA

    def test_get_commits_since(self, private_repo: Path, sample_project: Path):
        """Test getting commits since a specific commit."""
        git_repo = GitRepository(private_repo)

        # Get all commits
        commits = git_repo.get_commits_since()
        assert len(commits) >= 2  # Initial + sample project commit

    def test_get_commit(self, private_repo: Path):
        """Test getting a specific commit."""
        git_repo = GitRepository(private_repo)
        current_hash = git_repo.get_current_commit()

        commit_info = git_repo.get_commit(current_hash)
        assert commit_info.hash == current_hash
        assert commit_info.short_hash == current_hash[:8]

    def test_has_changes(self, private_repo: Path):
        """Test checking for uncommitted changes."""
        git_repo = GitRepository(private_repo)

        # Should have no changes initially
        assert git_repo.has_staged_changes() is False
        assert git_repo.has_unstaged_changes() is False

        # Create a new file
        (private_repo / "new_file.txt").write_text("content")
        assert git_repo.has_unstaged_changes() is True

        # Stage the file
        git_repo.stage_file("new_file.txt")
        assert git_repo.has_staged_changes() is True


class TestShouldIncludeFile:
    """Tests for file inclusion logic."""

    @pytest.fixture
    def basic_project(self):
        """Create a basic project mapping."""
        return ProjectMapping(private_path="packages/core")

    @pytest.fixture
    def basic_config(self, temp_dir: Path):
        """Create a basic config."""
        return SyncConfig(
            private_repo_path=temp_dir / "private",
            public_repo_url="git@github.com:org/public-repo.git",
        )

    def test_include_normal_file(self, basic_project, basic_config):
        """Test that normal files are included."""
        assert should_include_file("src/index.py", basic_project, basic_config) is True

    def test_exclude_env_file(self, basic_project, basic_config):
        """Test that .env files are excluded by default."""
        assert should_include_file(".env", basic_project, basic_config) is False

    def test_exclude_node_modules(self, basic_project, basic_config):
        """Test that node_modules is excluded."""
        assert (
            should_include_file("node_modules/package/index.js", basic_project, basic_config)
            is False
        )

    def test_project_exclude_patterns(self, basic_config, temp_dir):
        """Test project-specific exclude patterns."""
        project = ProjectMapping(
            private_path="packages/core",
            exclude_patterns=["*.secret", "internal/*"],
        )

        assert should_include_file("config.secret", project, basic_config) is False
        assert should_include_file("internal/data.txt", project, basic_config) is False
        assert should_include_file("public/data.txt", project, basic_config) is True

    def test_project_include_patterns(self, basic_config, temp_dir):
        """Test project-specific include patterns."""
        project = ProjectMapping(
            private_path="packages/core",
            include_patterns=["*.py", "*.md"],
        )

        assert should_include_file("src/main.py", project, basic_config) is True
        assert should_include_file("README.md", project, basic_config) is True
        assert should_include_file("config.yaml", project, basic_config) is False


class TestFilterCommitFiles:
    """Tests for filtering commit files."""

    def test_filter_to_project(self, private_repo: Path, sample_project: Path):
        """Test filtering commit files to a specific project."""
        git_repo = GitRepository(private_repo)
        commits = git_repo.get_commits_since()

        project = ProjectMapping(private_path="packages/core")
        config = SyncConfig(
            private_repo_path=private_repo,
            public_repo_url="git@github.com:org/public-repo.git",
        )

        # Find the commit that added the sample project
        for commit in commits:
            filtered = filter_commit_files(commit, project, config)
            if filtered:
                # Should only include files from packages/core
                for f in filtered:
                    assert f.path.startswith("packages/core/")
                break


class TestTagOperations:
    """Tests for git tag operations."""

    def test_get_tags_for_commit(self, private_repo: Path):
        """Test getting tags for a specific commit."""
        git_repo = GitRepository(private_repo)
        current_commit = git_repo.get_current_commit()

        # Initially no tags
        tags = git_repo.get_tags_for_commit(current_commit)
        assert tags == []

        # Create a tag
        git_repo.create_tag("v1.0.0", current_commit)

        # Now should have one tag
        tags = git_repo.get_tags_for_commit(current_commit)
        assert "v1.0.0" in tags

    def test_create_tag(self, private_repo: Path):
        """Test creating a tag."""
        git_repo = GitRepository(private_repo)
        current_commit = git_repo.get_current_commit()

        # Create a simple tag
        result = git_repo.create_tag("test-tag", current_commit)
        assert result is True

        # Tag should exist
        assert git_repo.tag_exists("test-tag") is True

    def test_create_annotated_tag(self, private_repo: Path):
        """Test creating an annotated tag."""
        git_repo = GitRepository(private_repo)
        current_commit = git_repo.get_current_commit()

        # Create an annotated tag
        result = git_repo.create_tag(
            "annotated-tag",
            current_commit,
            message="Release v1.0.0",
        )
        assert result is True

        # Tag should exist
        assert git_repo.tag_exists("annotated-tag") is True

    def test_tag_exists(self, private_repo: Path):
        """Test checking if a tag exists."""
        git_repo = GitRepository(private_repo)
        current_commit = git_repo.get_current_commit()

        # Tag doesn't exist initially
        assert git_repo.tag_exists("nonexistent-tag") is False

        # Create tag
        git_repo.create_tag("existing-tag", current_commit)

        # Now it should exist
        assert git_repo.tag_exists("existing-tag") is True

    def test_get_all_tags(self, private_repo: Path):
        """Test getting all tags."""
        git_repo = GitRepository(private_repo)
        current_commit = git_repo.get_current_commit()

        # Create some tags
        git_repo.create_tag("tag-a", current_commit)
        git_repo.create_tag("tag-b", current_commit)

        all_tags = git_repo.get_all_tags()
        assert "tag-a" in all_tags
        assert "tag-b" in all_tags
        assert all_tags["tag-a"] == current_commit
        assert all_tags["tag-b"] == current_commit

    def test_create_duplicate_tag_fails(self, private_repo: Path):
        """Test that creating a duplicate tag fails."""
        git_repo = GitRepository(private_repo)
        current_commit = git_repo.get_current_commit()

        # Create tag
        result = git_repo.create_tag("dup-tag", current_commit)
        assert result is True

        # Creating same tag again should fail
        result = git_repo.create_tag("dup-tag", current_commit)
        assert result is False

    def test_force_create_tag(self, private_repo: Path):
        """Test force-creating a tag over an existing one."""
        git_repo = GitRepository(private_repo)
        current_commit = git_repo.get_current_commit()

        # Create initial tag
        git_repo.create_tag("force-tag", current_commit)

        # Create a new commit
        (private_repo / "force_test.txt").write_text("test")
        git_repo.stage_file("force_test.txt")
        new_commit = git_repo.commit("Test commit for force tag")

        # Force create same tag at new commit
        result = git_repo.create_tag("force-tag", new_commit, force=True)
        assert result is True

        # Tag should now point to new commit
        all_tags = git_repo.get_all_tags()
        assert all_tags["force-tag"] == new_commit

