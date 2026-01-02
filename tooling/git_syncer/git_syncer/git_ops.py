"""
Git operations for the syncer.

Provides a wrapper around git operations using GitPython,
handling commits, diffs, file operations, and history traversal.
"""

import fnmatch
import shutil
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path

from git import Commit, Repo
from git.exc import GitCommandError, InvalidGitRepositoryError

from .config import ProjectMapping, SyncConfig


def clone_repo(url: str, path: Path, branch: str | None = None, force_reclone: bool = False) -> "GitRepository":
    """
    Clone a repository from a URL to a local path.

    Args:
        url: Git remote URL (e.g., git@github.com:org/repo.git)
        path: Local path to clone into
        branch: Branch to checkout (defaults to remote's default branch)
        force_reclone: If True, delete existing repo and re-clone

    Returns:
        GitRepository wrapper for the cloned repo
    """
    path = Path(path).resolve()

    if path.exists():
        if force_reclone:
            shutil.rmtree(path)
        else:
            # Check if it's already a valid repo with the same remote
            try:
                repo = Repo(path)
                # Verify remote matches
                origin = repo.remote("origin")
                if url in origin.urls:
                    # Already cloned, just fetch and update
                    origin.fetch()
                    if branch:
                        # Use reset instead of checkout to handle corrupted states
                        repo.git.fetch("origin", branch)
                        repo.git.reset("--hard", f"origin/{branch}")
                    return GitRepository(path)
                else:
                    # Wrong remote, re-clone
                    shutil.rmtree(path)
            except (InvalidGitRepositoryError, ValueError):
                # Not a valid repo, remove and re-clone
                shutil.rmtree(path)
            except GitCommandError as e:
                # Git command failed (possibly corrupted repo), remove and re-clone
                print(f"Git error with existing repo, re-cloning: {e}")
                shutil.rmtree(path)

    # Clone the repository
    path.parent.mkdir(parents=True, exist_ok=True)

    clone_args = {}
    if branch:
        clone_args["branch"] = branch

    Repo.clone_from(url, path, **clone_args)
    return GitRepository(path)


def ensure_public_repo(config: SyncConfig) -> "GitRepository":
    """
    Ensure the public repo is cloned and up-to-date.

    Args:
        config: Sync configuration containing public repo URL

    Returns:
        GitRepository wrapper for the public repo
    """
    return clone_repo(
        url=config.public_repo_url,
        path=config.public_repo_path,
        branch=config.public_branch,
    )


@dataclass
class FileChange:
    """Represents a single file change."""

    path: str
    change_type: str  # 'A' (added), 'M' (modified), 'D' (deleted), 'R' (renamed)
    old_path: str | None = None  # For renames


@dataclass
class CommitInfo:
    """Information about a git commit."""

    hash: str
    short_hash: str
    message: str
    author_name: str
    author_email: str
    author_date: datetime
    committer_name: str
    committer_email: str
    commit_date: datetime
    files_changed: list[FileChange]

    @classmethod
    def from_commit(cls, commit: Commit) -> "CommitInfo":
        """Create CommitInfo from a GitPython Commit object."""
        files_changed = []

        # Get the diff from parent (if exists)
        if commit.parents:
            diffs = commit.parents[0].diff(commit)
        else:
            # Initial commit - compare against empty tree
            diffs = commit.diff(None, create_patch=False)

        for diff in diffs:
            change_type = diff.change_type
            if change_type == "R":
                files_changed.append(
                    FileChange(
                        path=diff.b_path,
                        change_type=change_type,
                        old_path=diff.a_path,
                    )
                )
            else:
                files_changed.append(
                    FileChange(
                        path=diff.b_path or diff.a_path,
                        change_type=change_type,
                    )
                )

        return cls(
            hash=commit.hexsha,
            short_hash=commit.hexsha[:8],
            message=commit.message.strip(),
            author_name=commit.author.name,
            author_email=commit.author.email,
            author_date=datetime.fromtimestamp(commit.authored_date, tz=timezone.utc),
            committer_name=commit.committer.name,
            committer_email=commit.committer.email,
            commit_date=datetime.fromtimestamp(commit.committed_date, tz=timezone.utc),
            files_changed=files_changed,
        )


class GitRepository:
    """Wrapper around a git repository for sync operations."""

    def __init__(self, path: Path):
        """Initialize repository wrapper."""
        self.path = Path(path).resolve()
        try:
            self.repo = Repo(self.path)
        except InvalidGitRepositoryError as e:
            raise ValueError(f"Not a valid git repository: {self.path}") from e

    def get_current_branch(self) -> str:
        """Get the current branch name."""
        return self.repo.active_branch.name

    def get_current_commit(self) -> str:
        """Get the current HEAD commit hash."""
        return self.repo.head.commit.hexsha

    def get_commits_since(
        self,
        since_commit: str | None = None,
        branch: str | None = None,
        paths: list[str] | None = None,
    ) -> list[CommitInfo]:
        """Get commits since a given commit hash, optionally filtered by paths."""
        target_branch = branch or self.get_current_branch()

        if since_commit:
            # Get commits from since_commit to HEAD
            try:
                range_spec = f"{since_commit}..{target_branch}"
                if paths:
                    commits = list(self.repo.iter_commits(range_spec, paths=paths))
                else:
                    commits = list(self.repo.iter_commits(range_spec))
            except GitCommandError:
                # If since_commit doesn't exist, get all commits
                if paths:
                    commits = list(
                        self.repo.iter_commits(target_branch, paths=paths)
                    )
                else:
                    commits = list(self.repo.iter_commits(target_branch))
        else:
            # Get all commits
            if paths:
                commits = list(self.repo.iter_commits(target_branch, paths=paths))
            else:
                commits = list(self.repo.iter_commits(target_branch))

        # Reverse to get chronological order (oldest first)
        commits.reverse()
        return [CommitInfo.from_commit(c) for c in commits]

    def get_commit(self, commit_hash: str) -> CommitInfo:
        """Get information about a specific commit."""
        commit = self.repo.commit(commit_hash)
        return CommitInfo.from_commit(commit)

    def get_last_synced_commit(self, commit_prefix: str = "[sync]") -> str | None:
        """
        Find the last synced commit by looking for 'synced_from:' in commit messages.

        Searches recent commits for the sync prefix and extracts the source commit hash.
        Returns None if no synced commits are found.
        """
        import re

        try:
            # Look at recent commits (limit to 100 for performance)
            for commit in self.repo.iter_commits(self.get_current_branch(), max_count=100):
                message = commit.message
                # Check if this is a sync commit
                if commit_prefix in message:
                    # Look for synced_from: <hash> pattern
                    match = re.search(r"synced_from:\s*([a-f0-9]{40})", message)
                    if match:
                        return match.group(1)
        except GitCommandError:
            pass

        return None

    def get_file_content_at_commit(
        self, commit_hash: str, file_path: str
    ) -> bytes | None:
        """Get the content of a file at a specific commit."""
        try:
            commit = self.repo.commit(commit_hash)
            blob = commit.tree / file_path
            return blob.data_stream.read()
        except (KeyError, GitCommandError):
            return None

    def file_exists_at_commit(self, commit_hash: str, file_path: str) -> bool:
        """Check if a file exists at a specific commit."""
        try:
            commit = self.repo.commit(commit_hash)
            _ = commit.tree / file_path
            return True
        except KeyError:
            return False

    def checkout_file_from_commit(
        self, commit_hash: str, file_path: str, dest_path: Path
    ) -> bool:
        """Copy a file from a specific commit to a destination path."""
        content = self.get_file_content_at_commit(commit_hash, file_path)
        if content is None:
            return False

        dest_path.parent.mkdir(parents=True, exist_ok=True)
        dest_path.write_bytes(content)
        return True

    def stage_file(self, file_path: str) -> None:
        """Stage a file for commit."""
        # Use git command directly for more reliable staging
        self.repo.git.add(file_path)

    def stage_files(self, file_paths: list[str]) -> None:
        """Stage multiple files for commit."""
        if file_paths:
            # Use git command directly for more reliable staging
            self.repo.git.add(*file_paths)

    def remove_file(self, file_path: str) -> None:
        """Remove and stage deletion of a file."""
        full_path = self.path / file_path
        if full_path.exists():
            full_path.unlink()
        try:
            self.repo.git.rm("--cached", "--ignore-unmatch", file_path)
        except GitCommandError:
            pass  # File might not be tracked

    def refresh_index(self) -> None:
        """Refresh the git index to ensure it's in sync with the working tree."""
        try:
            self.repo.git.update_index("--refresh")
        except GitCommandError:
            pass  # Ignore errors from refresh

    def commit(
        self,
        message: str,
        author_name: str | None = None,
        author_email: str | None = None,
        author_date: datetime | None = None,
    ) -> str:
        """Create a commit with the staged changes."""
        # Ensure author_date has timezone info if provided
        if author_date and author_date.tzinfo is None:
            author_date = author_date.replace(tzinfo=timezone.utc)

        # Use git commit command directly for better reliability
        cmd_args = ["-m", message]

        if author_date:
            date_str = author_date.strftime("%Y-%m-%dT%H:%M:%S%z")
            cmd_args.extend(["--date", date_str])

        if author_name and author_email:
            cmd_args.extend(["--author", f"{author_name} <{author_email}>"])

        self.repo.git.commit(*cmd_args)

        # Return the new commit hash
        return self.repo.head.commit.hexsha

    def has_staged_changes(self) -> bool:
        """Check if there are staged changes."""
        try:
            # Try using git status which is more reliable
            status = self.repo.git.status("--porcelain")
            # Staged changes show as first character being non-space and non-?
            for line in status.splitlines():
                if line and len(line) >= 2:
                    index_status = line[0]
                    # A, M, D, R, C indicate staged changes
                    if index_status in "AMDRC":
                        return True
            return False
        except GitCommandError:
            # Fallback to checking if index differs from HEAD
            try:
                return len(self.repo.index.diff("HEAD")) > 0
            except GitCommandError:
                return False

    def has_unstaged_changes(self) -> bool:
        """Check if there are unstaged changes or untracked files."""
        try:
            return len(self.repo.index.diff(None)) > 0 or len(self.repo.untracked_files) > 0
        except GitCommandError:
            return False

    def has_any_changes(self) -> bool:
        """Check if there are any uncommitted changes."""
        return self.has_staged_changes() or self.has_unstaged_changes()

    def push(self, remote: str = "origin", branch: str | None = None) -> None:
        """Push to remote."""
        target_branch = branch or self.get_current_branch()
        self.repo.remote(remote).push(target_branch)

    def pull(self, remote: str = "origin", branch: str | None = None) -> None:
        """Pull from remote."""
        target_branch = branch or self.get_current_branch()
        self.repo.remote(remote).pull(target_branch)

    def fetch(self, remote: str = "origin") -> None:
        """Fetch from remote."""
        self.repo.remote(remote).fetch()

    def is_ignored(self, file_path: str) -> bool:
        """Check if a file path is ignored by .gitignore."""
        try:
            # git check-ignore returns 0 if ignored, 1 if not ignored
            self.repo.git.check_ignore(file_path)
            return True
        except GitCommandError:
            return False


def should_include_file(
    file_path: str,
    project: ProjectMapping,
    config: SyncConfig,
) -> bool:
    """Determine if a file should be included in sync based on patterns."""
    file_parts = Path(file_path).parts

    # Check global excludes first
    for pattern in config.global_exclude_patterns:
        # Remove trailing slash for directory patterns
        clean_pattern = pattern.rstrip("/")
        if fnmatch.fnmatch(file_path, pattern) or fnmatch.fnmatch(file_path, clean_pattern):
            return False
        # Also check if any path component matches
        for part in file_parts:
            if fnmatch.fnmatch(part, pattern) or fnmatch.fnmatch(part, clean_pattern):
                return False

    # Check project-specific excludes
    for pattern in project.exclude_patterns:
        clean_pattern = pattern.rstrip("/")
        if fnmatch.fnmatch(file_path, pattern) or fnmatch.fnmatch(file_path, clean_pattern):
            return False
        # Check path components too
        for part in file_parts:
            if fnmatch.fnmatch(part, clean_pattern):
                return False

    # Check project-specific includes (if specified, file must match one)
    if project.include_patterns:
        for pattern in project.include_patterns:
            if fnmatch.fnmatch(file_path, pattern):
                return True
        return False

    return True


def filter_commit_files(
    commit: CommitInfo,
    project: ProjectMapping,
    config: SyncConfig,
) -> list[FileChange]:
    """Filter commit file changes to only include relevant files for a project."""
    relevant_files = []
    project_prefix = project.private_path.rstrip("/") + "/"

    for change in commit.files_changed:
        # Check if file is within the project directory
        if not change.path.startswith(project_prefix):
            continue

        # Get relative path within project
        rel_path = change.path[len(project_prefix) :]

        # Check include/exclude patterns
        if should_include_file(rel_path, project, config):
            relevant_files.append(change)

    return relevant_files


def copy_project_files(
    source_repo: GitRepository,
    dest_repo: GitRepository,
    project: ProjectMapping,
    config: SyncConfig,
    commit_hash: str | None = None,
) -> list[str]:
    """
    Copy files from source project directory to destination.

    If commit_hash is provided, copies files as they exist at that commit.
    Otherwise, copies files from the current working directory.

    Returns list of files that were copied.
    """
    source_path = source_repo.path / project.private_path
    dest_path = dest_repo.path / project.resolved_public_path
    copied_files = []

    if commit_hash:
        # Copy from specific commit
        commit = source_repo.repo.commit(commit_hash)
        try:
            tree = commit.tree / project.private_path
        except KeyError:
            # Directory doesn't exist at this commit
            return copied_files

        def copy_tree(tree_obj, rel_path: str = ""):
            for item in tree_obj:
                item_rel_path = f"{rel_path}/{item.name}" if rel_path else item.name
                full_source_path = f"{project.private_path}/{item_rel_path}"

                if item.type == "tree":
                    copy_tree(item, item_rel_path)
                elif item.type == "blob":
                    if should_include_file(item_rel_path, project, config):
                        dest_file = dest_path / item_rel_path
                        dest_file.parent.mkdir(parents=True, exist_ok=True)
                        dest_file.write_bytes(item.data_stream.read())
                        dest_rel = f"{project.resolved_public_path}/{item_rel_path}"
                        copied_files.append(dest_rel)

        copy_tree(tree)
    else:
        # Copy from working directory
        if not source_path.exists():
            return copied_files

        for source_file in source_path.rglob("*"):
            if not source_file.is_file():
                continue

            rel_path = source_file.relative_to(source_path)
            full_rel_path = f"{project.private_path}/{rel_path}"

            # Skip if gitignored
            if source_repo.is_ignored(full_rel_path):
                continue

            if should_include_file(str(rel_path), project, config):
                dest_file = dest_path / rel_path
                dest_file.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(source_file, dest_file)
                dest_rel = f"{project.resolved_public_path}/{rel_path}"
                copied_files.append(dest_rel)

    return copied_files


def sync_file_deletion(
    dest_repo: GitRepository,
    file_path: str,
    project: ProjectMapping,
) -> str | None:
    """
    Handle file deletion in sync.

    Returns the path of the deleted file if it was deleted, None otherwise.
    """
    # Convert from private path to public path
    private_prefix = project.private_path.rstrip("/") + "/"
    if not file_path.startswith(private_prefix):
        return None

    rel_path = file_path[len(private_prefix) :]
    public_file = f"{project.resolved_public_path}/{rel_path}"

    full_path = dest_repo.path / public_file
    if full_path.exists():
        full_path.unlink()
        return public_file

    return None

