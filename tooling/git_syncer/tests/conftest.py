"""Pytest configuration and fixtures for git_syncer tests."""

import os
import tempfile
from pathlib import Path

import pytest
from git import Repo


@pytest.fixture
def temp_dir():
    """Create a temporary directory for tests."""
    with tempfile.TemporaryDirectory() as tmpdir:
        yield Path(tmpdir)


@pytest.fixture
def private_repo(temp_dir: Path):
    """Create a temporary git repository to act as private repo."""
    repo_path = temp_dir / "private"
    repo_path.mkdir()

    # Initialize git repo
    repo = Repo.init(repo_path)

    # Configure git user
    repo.config_writer().set_value("user", "name", "Test User").release()
    repo.config_writer().set_value("user", "email", "test@example.com").release()

    # Create initial commit
    readme = repo_path / "README.md"
    readme.write_text("# Private Repo")
    repo.index.add(["README.md"])
    repo.index.commit("Initial commit")

    yield repo_path


@pytest.fixture
def public_repo(temp_dir: Path):
    """Create a temporary git repository to act as public repo."""
    repo_path = temp_dir / "public"
    repo_path.mkdir()

    # Initialize git repo
    repo = Repo.init(repo_path)

    # Configure git user
    repo.config_writer().set_value("user", "name", "Test User").release()
    repo.config_writer().set_value("user", "email", "test@example.com").release()

    # Create initial commit
    readme = repo_path / "README.md"
    readme.write_text("# Public Repo")
    repo.index.add(["README.md"])
    repo.index.commit("Initial commit")

    yield repo_path


@pytest.fixture
def sample_project(private_repo: Path):
    """Create a sample project structure in the private repo."""
    project_path = private_repo / "packages" / "core"
    project_path.mkdir(parents=True)

    # Create some files
    (project_path / "src").mkdir()
    (project_path / "src" / "index.py").write_text("# Core module\n")
    (project_path / "README.md").write_text("# Core Package\n")
    (project_path / ".env").write_text("SECRET=value\n")  # Should be excluded

    # Commit the changes
    repo = Repo(private_repo)
    repo.index.add(["packages/core/src/index.py", "packages/core/README.md", "packages/core/.env"])
    repo.index.commit("Add core package")

    yield project_path


