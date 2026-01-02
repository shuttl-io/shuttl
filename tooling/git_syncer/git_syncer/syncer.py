"""
Main syncer logic for two-way synchronization between repositories.

This module handles the core sync operations, applying commits from one
repository to another while maintaining commit history and messages.
"""

import shutil
from dataclasses import dataclass
from pathlib import Path
from typing import Literal

from rich.console import Console
from rich.progress import Progress, SpinnerColumn, TextColumn
from rich.table import Table

from .config import SyncConfig
from .git_ops import (
    CommitInfo,
    GitRepository,
    clone_repo,
    copy_project_files,
    filter_commit_files,
    sync_file_deletion,
)

console = Console()

SyncDirection = Literal["private-to-public", "public-to-private"]


@dataclass
class SyncResult:
    """Result of a sync operation."""

    success: bool
    commits_synced: int
    files_changed: int
    errors: list[str]
    warnings: list[str]
    commit_mappings: dict[str, str]  # source_hash -> dest_hash


class RepoSyncer:
    """Handles synchronization between private and public repositories."""

    def __init__(self, config: SyncConfig, force_reclone: bool = False):
        """Initialize the syncer with configuration."""
        self.config = config
        self.private_repo: GitRepository | None = None
        self.public_repo: GitRepository | None = None
        self.force_reclone = force_reclone

    def _init_repos(self) -> tuple[GitRepository, GitRepository]:
        """Initialize repository wrappers, cloning public repo if needed."""
        if self.private_repo is None:
            self.private_repo = GitRepository(self.config.private_repo_path)
        if self.public_repo is None:
            action = "Re-cloning" if self.force_reclone else "Ensuring"
            console.print(f"[dim]{action} public repo from {self.config.public_repo_url}...[/dim]")
            self.public_repo = clone_repo(
                url=self.config.public_repo_url,
                path=self.config.public_repo_path,
                branch=self.config.public_branch,
                force_reclone=self.force_reclone,
            )
            console.print(f"[dim]Public repo ready at {self.config.public_repo_path}[/dim]")
        return self.private_repo, self.public_repo

    def _get_last_synced_commit(self, direction: SyncDirection) -> str | None:
        """Get the last synced commit by reading from destination repo's commit messages."""
        private_repo, public_repo = self._init_repos()

        if direction == "private-to-public":
            # Look in public repo for synced_from: to find last private commit synced
            return public_repo.get_last_synced_commit(self.config.commit_prefix)
        else:
            # Look in private repo for synced_from: to find last public commit synced
            return private_repo.get_last_synced_commit(self.config.commit_prefix)

    def get_pending_commits(
        self,
        direction: SyncDirection,
    ) -> list[CommitInfo]:
        """Get commits that need to be synced."""
        private_repo, public_repo = self._init_repos()

        # Get the last synced commit from the destination repo's commit messages
        last_synced = self._get_last_synced_commit(direction)

        if last_synced:
            console.print(f"[dim]Last synced commit: {last_synced[:8]}[/dim]")
        else:
            console.print("[dim]No previous sync found - will sync all matching commits[/dim]")

        # Get paths to filter commits by (projects + individual files)
        if direction == "private-to-public":
            source_repo = private_repo
            sync_paths = [p.private_path for p in self.config.get_enabled_projects()]
            sync_paths += [f.private_path for f in self.config.get_enabled_files()]
        else:
            source_repo = public_repo
            # For public-to-private, use public paths
            sync_paths = [
                p.resolved_public_path for p in self.config.get_enabled_projects()
            ]
            sync_paths += [
                f.resolved_public_path for f in self.config.get_enabled_files()
            ]

        return source_repo.get_commits_since(
            since_commit=last_synced,
            paths=sync_paths if sync_paths else None,
        )

    def preview_sync(self, direction: SyncDirection) -> None:
        """Preview what would be synced without making changes."""
        console.print(f"\n[bold]Preview: {direction}[/bold]\n")

        pending_commits = self.get_pending_commits(direction)

        if not pending_commits:
            console.print("[green]No commits to sync.[/green]")
            return

        # Show commits table
        table = Table(title=f"Pending Commits ({len(pending_commits)})")
        table.add_column("Hash", style="cyan", width=10)
        table.add_column("Date", style="green", width=20)
        table.add_column("Author", style="yellow", width=25)
        table.add_column("Message", style="white")

        for commit in pending_commits[:20]:  # Show first 20
            message = commit.message.split("\n")[0][:60]
            if len(commit.message.split("\n")[0]) > 60:
                message += "..."
            table.add_row(
                commit.short_hash,
                commit.author_date.strftime("%Y-%m-%d %H:%M"),
                commit.author_name,
                message,
            )

        if len(pending_commits) > 20:
            table.add_row("...", "...", "...", f"[dim]({len(pending_commits) - 20} more commits)[/dim]")

        console.print(table)

        # Show affected projects
        console.print("\n[bold]Affected Projects:[/bold]")
        for project in self.config.get_enabled_projects():
            files_affected = 0
            for commit in pending_commits:
                files = filter_commit_files(commit, project, self.config)
                files_affected += len(files)
            if files_affected > 0:
                console.print(f"  • {project.private_path} ({files_affected} file changes)")

    def sync(
        self,
        direction: SyncDirection,
        dry_run: bool | None = None,
    ) -> SyncResult:
        """
        Perform the synchronization.

        Args:
            direction: Which way to sync
            dry_run: If True, don't make actual changes (overrides config)

        Returns:
            SyncResult with details about what was synced
        """
        dry_run = dry_run if dry_run is not None else self.config.dry_run
        private_repo, public_repo = self._init_repos()

        result = SyncResult(
            success=True,
            commits_synced=0,
            files_changed=0,
            errors=[],
            warnings=[],
            commit_mappings={},
        )

        # Determine source and destination repos
        if direction == "private-to-public":
            source_repo = private_repo
            dest_repo = public_repo
        else:
            source_repo = public_repo
            dest_repo = private_repo

        # Get pending commits
        pending_commits = self.get_pending_commits(direction)

        if not pending_commits:
            console.print("[green]No commits to sync.[/green]")
            return result

        console.print(f"\n[bold]Syncing {len(pending_commits)} commits ({direction})...[/bold]\n")

        if dry_run:
            console.print("[yellow]DRY RUN - No changes will be made[/yellow]\n")

        last_successful_commit: CommitInfo | None = None

        with Progress(
            SpinnerColumn(),
            TextColumn("[progress.description]{task.description}"),
            console=console,
        ) as progress:
            task = progress.add_task("Syncing commits...", total=len(pending_commits))

            for commit in pending_commits:
                progress.update(task, description=f"Syncing {commit.short_hash}...")

                try:
                    new_hash = self._sync_single_commit(
                        commit=commit,
                        direction=direction,
                        source_repo=source_repo,
                        dest_repo=dest_repo,
                        dry_run=dry_run,
                    )

                    if new_hash:
                        result.commit_mappings[commit.hash] = new_hash
                        result.commits_synced += 1
                        result.files_changed += len(commit.files_changed)
                        last_successful_commit = commit

                except Exception as e:
                    error_msg = f"Error syncing {commit.short_hash}: {e}"
                    result.errors.append(error_msg)
                    console.print(f"[red]{error_msg}[/red]")
                    result.success = False
                    break

                progress.advance(task)

        # State is now tracked via synced_from: in commit messages, no state file needed

        # Push if configured
        if not dry_run and self.config.auto_push and result.commits_synced > 0:
            console.print("\n[bold]Pushing changes...[/bold]")
            try:
                if direction == "private-to-public":
                    dest_repo.push(
                        self.config.public_remote,
                        self.config.public_branch,
                    )
                else:
                    dest_repo.push(
                        self.config.private_remote,
                        self.config.private_branch,
                    )
                console.print("[green]Push successful.[/green]")
            except Exception as e:
                result.warnings.append(f"Push failed: {e}")
                console.print(f"[yellow]Push failed: {e}[/yellow]")

        # Print summary
        self._print_summary(result, dry_run)

        return result

    def _sync_single_commit(
        self,
        commit: CommitInfo,
        direction: SyncDirection,
        source_repo: GitRepository,
        dest_repo: GitRepository,
        dry_run: bool,
    ) -> str | None:
        """Sync a single commit from source to destination."""
        files_to_stage: list[str] = []
        files_to_remove: list[str] = []

        # Process project (directory) changes
        for project in self.config.get_enabled_projects():
            # Get files relevant to this project
            relevant_files = filter_commit_files(commit, project, self.config)

            if not relevant_files:
                continue

            for file_change in relevant_files:
                if file_change.change_type == "D":
                    # Handle deletion
                    deleted = sync_file_deletion(dest_repo, file_change.path, project)
                    if deleted:
                        files_to_remove.append(deleted)
                else:
                    # Handle addition or modification
                    # Calculate paths
                    private_prefix = project.private_path.rstrip("/") + "/"
                    rel_path = file_change.path[len(private_prefix):]
                    dest_file_path = f"{project.resolved_public_path}/{rel_path}"

                    if not dry_run:
                        # Copy file content from source commit
                        content = source_repo.get_file_content_at_commit(
                            commit.hash, file_change.path
                        )
                        if content is not None:
                            full_dest = dest_repo.path / dest_file_path
                            full_dest.parent.mkdir(parents=True, exist_ok=True)
                            full_dest.write_bytes(content)
                            files_to_stage.append(dest_file_path)

        # Process individual file changes
        for file_mapping in self.config.get_enabled_files():
            # Check if this commit touches this file
            for file_change in commit.files_changed:
                if file_change.path != file_mapping.private_path:
                    continue

                if file_change.change_type == "D":
                    # Handle deletion
                    dest_file_path = file_mapping.resolved_public_path
                    full_dest = dest_repo.path / dest_file_path
                    if full_dest.exists():
                        if not dry_run:
                            full_dest.unlink()
                        files_to_remove.append(dest_file_path)
                else:
                    # Handle addition or modification
                    dest_file_path = file_mapping.resolved_public_path

                    if not dry_run:
                        content = source_repo.get_file_content_at_commit(
                            commit.hash, file_change.path
                        )
                        if content is not None:
                            full_dest = dest_repo.path / dest_file_path
                            full_dest.parent.mkdir(parents=True, exist_ok=True)
                            full_dest.write_bytes(content)
                            files_to_stage.append(dest_file_path)

        if not files_to_stage and not files_to_remove:
            return None  # No relevant changes

        if dry_run:
            console.print(f"  [dim]Would sync {commit.short_hash}: {len(files_to_stage)} files[/dim]")
            return "dry-run"

        # Stage files
        if files_to_stage:
            dest_repo.stage_files(files_to_stage)
        for file_path in files_to_remove:
            try:
                dest_repo.remove_file(file_path)
            except Exception:
                pass  # File might already be removed

        # Create commit with original message and synced_from trailer
        commit_message = f"{self.config.commit_prefix} {commit.message}\n\nsynced_from: {commit.hash}"

        # Only commit if we have files staged
        if not files_to_stage and not files_to_remove:
            return None

        try:
            new_hash = dest_repo.commit(
                message=commit_message,
                author_name=commit.author_name,
                author_email=commit.author_email,
                author_date=commit.author_date,
            )
            console.print(f"  [green]✓[/green] {commit.short_hash} → {new_hash[:8]}")
            return new_hash
        except Exception as e:
            # If commit fails (e.g., nothing to commit), log and continue
            if "nothing to commit" in str(e).lower():
                return None
            raise

    def _print_summary(self, result: SyncResult, dry_run: bool) -> None:
        """Print sync summary."""
        console.print("\n[bold]Sync Summary:[/bold]")
        prefix = "[DRY RUN] " if dry_run else ""

        if result.success:
            console.print(f"  [green]✓ {prefix}Synced {result.commits_synced} commits[/green]")
        else:
            console.print(f"  [red]✗ {prefix}Sync failed[/red]")

        console.print(f"  Files changed: {result.files_changed}")

        if result.errors:
            console.print(f"  [red]Errors: {len(result.errors)}[/red]")
            for error in result.errors:
                console.print(f"    • {error}")

        if result.warnings:
            console.print(f"  [yellow]Warnings: {len(result.warnings)}[/yellow]")
            for warning in result.warnings:
                console.print(f"    • {warning}")

    def sync_full(self, direction: SyncDirection, dry_run: bool = False) -> SyncResult:
        """
        Perform a full sync of all project files (not just commits).

        This is useful for initial sync or when repos are out of sync.
        """
        private_repo, public_repo = self._init_repos()

        result = SyncResult(
            success=True,
            commits_synced=0,
            files_changed=0,
            errors=[],
            warnings=[],
            commit_mappings={},
        )

        if direction == "private-to-public":
            source_repo = private_repo
            dest_repo = public_repo
        else:
            source_repo = public_repo
            dest_repo = private_repo

        console.print(f"\n[bold]Full sync ({direction})...[/bold]\n")

        if dry_run:
            console.print("[yellow]DRY RUN - No changes will be made[/yellow]\n")

        all_copied_files: list[str] = []

        # Sync projects (directories)
        for project in self.config.get_enabled_projects():
            console.print(f"  Syncing project [cyan]{project.private_path}[/cyan]...")

            if not dry_run:
                # Clear destination directory first
                dest_path = dest_repo.path / project.resolved_public_path
                if dest_path.exists():
                    shutil.rmtree(dest_path)

                # Copy all files
                copied = copy_project_files(
                    source_repo=source_repo,
                    dest_repo=dest_repo,
                    project=project,
                    config=self.config,
                )
                all_copied_files.extend(copied)
                console.print(f"    [green]✓ Copied {len(copied)} files[/green]")
            else:
                source_path = source_repo.path / project.private_path
                if source_path.exists():
                    file_count = sum(1 for _ in source_path.rglob("*") if _.is_file())
                    console.print(f"    [dim]Would copy ~{file_count} files[/dim]")

        # Sync individual files
        for file_mapping in self.config.get_enabled_files():
            console.print(f"  Syncing file [cyan]{file_mapping.private_path}[/cyan]...")

            source_file = source_repo.path / file_mapping.private_path
            dest_file = dest_repo.path / file_mapping.resolved_public_path

            if not dry_run:
                if source_file.exists():
                    dest_file.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copy2(source_file, dest_file)
                    all_copied_files.append(file_mapping.resolved_public_path)
                    console.print(f"    [green]✓ Copied[/green]")
                else:
                    console.print(f"    [yellow]⚠ Source file not found[/yellow]")
            else:
                if source_file.exists():
                    console.print(f"    [dim]Would copy file[/dim]")
                else:
                    console.print(f"    [dim]Source file not found[/dim]")

        if not dry_run and all_copied_files:
            # Stage and commit with synced_from trailer
            dest_repo.stage_files(all_copied_files)
            if dest_repo.has_staged_changes():
                source_commit = source_repo.get_current_commit()
                commit_message = (
                    f"{self.config.commit_prefix} Full sync from "
                    f"{'private' if direction == 'private-to-public' else 'public'} repo\n\n"
                    f"synced_from: {source_commit}"
                )
                new_hash = dest_repo.commit(commit_message)
                result.commits_synced = 1
                result.files_changed = len(all_copied_files)
                console.print(f"\n  [green]✓ Created commit {new_hash[:8]}[/green]")
                console.print(f"  [dim]synced_from: {source_commit[:8]}[/dim]")

        self._print_summary(result, dry_run)
        return result

    def status(self) -> None:
        """Print current sync status."""
        console.print("\n[bold]Git Syncer Status[/bold]\n")

        # Config info
        console.print("[bold]Configuration:[/bold]")
        console.print(f"  Private repo: {self.config.private_repo_path}")
        console.print(f"  Public repo URL: {self.config.public_repo_url}")
        console.print(f"  Public repo path: {self.config.public_repo_path}")

        enabled_projects = self.config.get_enabled_projects()
        enabled_files = self.config.get_enabled_files()

        console.print(f"  Projects: {len(enabled_projects)}")
        for project in enabled_projects:
            console.print(f"    • {project.private_path} → {project.resolved_public_path}")

        console.print(f"  Files: {len(enabled_files)}")
        for file in enabled_files:
            console.print(f"    • {file.private_path} → {file.resolved_public_path}")

        # Sync state from commit messages
        console.print("\n[bold]Sync State (from commit messages):[/bold]")
        try:
            last_synced_to_public = self._get_last_synced_commit("private-to-public")
            if last_synced_to_public:
                console.print(f"  Last synced to public: {last_synced_to_public[:8]}")
            else:
                console.print("  Last synced to public: Never")
        except Exception as e:
            console.print(f"  [red]Error reading public repo: {e}[/red]")

        try:
            last_synced_to_private = self._get_last_synced_commit("public-to-private")
            if last_synced_to_private:
                console.print(f"  Last synced to private: {last_synced_to_private[:8]}")
            else:
                console.print("  Last synced to private: Never")
        except Exception as e:
            console.print(f"  [red]Error reading private repo: {e}[/red]")

        # Pending commits
        console.print("\n[bold]Pending Commits:[/bold]")
        try:
            private_pending = self.get_pending_commits("private-to-public")
            console.print(f"  Private → Public: {len(private_pending)} commits")
        except Exception as e:
            console.print(f"  [red]Error checking private repo: {e}[/red]")

        try:
            public_pending = self.get_pending_commits("public-to-private")
            console.print(f"  Public → Private: {len(public_pending)} commits")
        except Exception as e:
            console.print(f"  [red]Error checking public repo: {e}[/red]")

