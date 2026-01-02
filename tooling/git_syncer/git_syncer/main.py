"""
CLI entry point for git_syncer.

Provides command-line interface for syncing between private and public repos.
"""

import os
import re
from pathlib import Path
from typing import Literal

import click
from rich.console import Console

from .config import FileMapping, ProjectMapping, SyncConfig, create_default_config
from .syncer import RepoSyncer

console = Console()


def inject_token_into_url(url: str, token: str) -> str:
    """
    Inject a token into a git URL for authentication.

    Converts SSH URLs to HTTPS and adds the token.
    """
    # If already HTTPS with credentials, return as-is
    if "@github.com" in url and url.startswith("https://"):
        return url

    # Convert SSH to HTTPS: git@github.com:org/repo.git -> https://token@github.com/org/repo.git
    ssh_match = re.match(r"git@([^:]+):(.+)", url)
    if ssh_match:
        host = ssh_match.group(1)
        path = ssh_match.group(2)
        return f"https://x-access-token:{token}@{host}/{path}"

    # Already HTTPS without credentials: add token
    https_match = re.match(r"https://([^/]+)/(.+)", url)
    if https_match:
        host = https_match.group(1)
        path = https_match.group(2)
        return f"https://x-access-token:{token}@{host}/{path}"

    return url


@click.group()
@click.version_option(package_name="git_syncer")
def cli():
    """Git Syncer - Two-way sync between private and public monorepos."""
    pass


@cli.command()
@click.option(
    "--private-repo",
    "-p",
    type=click.Path(exists=True, file_okay=False, dir_okay=True, path_type=Path),
    required=True,
    help="Path to the private monorepo",
)
@click.option(
    "--public-repo-url",
    "-P",
    type=str,
    required=True,
    help="Git remote URL for the public monorepo (e.g., git@github.com:org/repo.git)",
)
@click.option(
    "--public-repo-clone-path",
    type=click.Path(file_okay=False, dir_okay=True, path_type=Path),
    default=None,
    help="Local path to clone the public repo (defaults to ~/.git_syncer/repos/<repo-name>)",
)
@click.option(
    "--project",
    "-j",
    "projects",
    multiple=True,
    help="Project (directory) path to sync (can be specified multiple times)",
)
@click.option(
    "--file",
    "-f",
    "files",
    multiple=True,
    help="Individual file path to sync (can be specified multiple times)",
)
@click.option(
    "--output",
    "-o",
    type=click.Path(dir_okay=False, path_type=Path),
    default=Path("git_syncer.yaml"),
    help="Output config file path",
)
def init(
    private_repo: Path,
    public_repo_url: str,
    public_repo_clone_path: Path | None,
    projects: tuple[str, ...],
    files: tuple[str, ...],
    output: Path,
):
    """Initialize a new sync configuration file."""
    config = create_default_config(
        private_repo_path=private_repo,
        public_repo_url=public_repo_url,
        public_repo_clone_path=public_repo_clone_path,
        projects=list(projects) if projects else None,
        files=list(files) if files else None,
    )

    config.to_yaml(output)
    console.print(f"[green]Created configuration file: {output}[/green]")
    console.print(f"  Public repo URL: {public_repo_url}")
    console.print(f"  Clone path: {config.public_repo_path}")
    if projects:
        console.print(f"  Projects: {len(projects)}")
    if files:
        console.print(f"  Files: {len(files)}")
    console.print("\nEdit this file to add more projects/files and customize settings.")


@cli.command()
@click.option(
    "--config",
    "-c",
    "config_path",
    type=click.Path(exists=True, dir_okay=False, path_type=Path),
    default=Path("git_syncer.yaml"),
    help="Path to the sync configuration file",
)
@click.option(
    "--direction",
    "-d",
    type=click.Choice(["private-to-public", "public-to-private"]),
    default="private-to-public",
    help="Direction of sync",
)
@click.option(
    "--dry-run",
    is_flag=True,
    help="Show what would be synced without making changes",
)
@click.option(
    "--full",
    is_flag=True,
    help="Perform a full sync of all files (not just commits)",
)
@click.option(
    "--auto-push",
    is_flag=True,
    help="Automatically push after sync",
)
@click.option(
    "--force-reclone",
    is_flag=True,
    help="Force re-clone of the public repository (useful if repo is corrupted)",
)
@click.option(
    "--token",
    "-t",
    envvar="GIT_SYNCER_TOKEN",
    help="GitHub PAT for authenticating to public repo (or set GIT_SYNCER_TOKEN env var)",
)
def sync(
    config_path: Path,
    direction: Literal["private-to-public", "public-to-private"],
    dry_run: bool,
    full: bool,
    auto_push: bool,
    force_reclone: bool,
    token: str | None,
):
    """Sync commits between repositories."""
    try:
        config = SyncConfig.from_yaml(config_path)
    except FileNotFoundError:
        console.print(f"[red]Config file not found: {config_path}[/red]")
        console.print("Run 'git-syncer init' to create a configuration file.")
        raise SystemExit(1)
    except Exception as e:
        console.print(f"[red]Error loading config: {e}[/red]")
        raise SystemExit(1)

    # Inject token into URL if provided
    if token:
        config.public_repo_url = inject_token_into_url(config.public_repo_url, token)

    if auto_push:
        config.auto_push = True

    syncer = RepoSyncer(config, force_reclone=force_reclone)

    if full:
        result = syncer.sync_full(direction, dry_run=dry_run)
    else:
        result = syncer.sync(direction, dry_run=dry_run)

    if not result.success:
        raise SystemExit(1)


@cli.command()
@click.option(
    "--config",
    "-c",
    "config_path",
    type=click.Path(exists=True, dir_okay=False, path_type=Path),
    default=Path("git_syncer.yaml"),
    help="Path to the sync configuration file",
)
@click.option(
    "--direction",
    "-d",
    type=click.Choice(["private-to-public", "public-to-private"]),
    default="private-to-public",
    help="Direction to preview",
)
def preview(
    config_path: Path,
    direction: Literal["private-to-public", "public-to-private"],
):
    """Preview pending commits to sync."""
    try:
        config = SyncConfig.from_yaml(config_path)
    except FileNotFoundError:
        console.print(f"[red]Config file not found: {config_path}[/red]")
        raise SystemExit(1)
    except Exception as e:
        console.print(f"[red]Error loading config: {e}[/red]")
        raise SystemExit(1)

    syncer = RepoSyncer(config)
    syncer.preview_sync(direction)


@cli.command()
@click.option(
    "--config",
    "-c",
    "config_path",
    type=click.Path(exists=True, dir_okay=False, path_type=Path),
    default=Path("git_syncer.yaml"),
    help="Path to the sync configuration file",
)
def status(config_path: Path):
    """Show current sync status."""
    try:
        config = SyncConfig.from_yaml(config_path)
    except FileNotFoundError:
        console.print(f"[red]Config file not found: {config_path}[/red]")
        raise SystemExit(1)
    except Exception as e:
        console.print(f"[red]Error loading config: {e}[/red]")
        raise SystemExit(1)

    syncer = RepoSyncer(config)
    syncer.status()


@cli.command()
@click.option(
    "--config",
    "-c",
    "config_path",
    type=click.Path(exists=True, dir_okay=False, path_type=Path),
    default=Path("git_syncer.yaml"),
    help="Path to the sync configuration file",
)
@click.argument("project_path")
@click.option(
    "--public-path",
    help="Path in the public repo (defaults to same as project path)",
)
@click.option(
    "--exclude",
    "-e",
    "excludes",
    multiple=True,
    help="Patterns to exclude (can be specified multiple times)",
)
def add_project(
    config_path: Path,
    project_path: str,
    public_path: str | None,
    excludes: tuple[str, ...],
):
    """Add a project to the sync configuration."""
    try:
        config = SyncConfig.from_yaml(config_path)
    except FileNotFoundError:
        console.print(f"[red]Config file not found: {config_path}[/red]")
        raise SystemExit(1)

    # Check if project already exists
    for p in config.projects:
        if p.private_path == project_path:
            console.print(f"[yellow]Project already exists: {project_path}[/yellow]")
            raise SystemExit(1)

    # Add new project
    new_project = ProjectMapping(
        private_path=project_path,
        public_path=public_path,
        exclude_patterns=list(excludes),
    )
    config.projects.append(new_project)
    config.to_yaml(config_path)

    console.print(f"[green]Added project: {project_path}[/green]")
    if public_path:
        console.print(f"  Public path: {public_path}")
    if excludes:
        console.print(f"  Excludes: {', '.join(excludes)}")


@cli.command()
@click.option(
    "--config",
    "-c",
    "config_path",
    type=click.Path(exists=True, dir_okay=False, path_type=Path),
    default=Path("git_syncer.yaml"),
    help="Path to the sync configuration file",
)
@click.argument("file_paths", nargs=-1, required=True)
@click.option(
    "--public-path",
    help="Path in the public repo (only applies when adding a single file)",
)
def add_file(
    config_path: Path,
    file_paths: tuple[str, ...],
    public_path: str | None,
):
    """Add one or more individual files to the sync configuration.

    Examples:
        git-syncer add-file README.md
        git-syncer add-file README.md LICENSE CONTRIBUTING.md
        git-syncer add-file README.md --public-path docs/README.md
    """
    if public_path and len(file_paths) > 1:
        console.print("[yellow]Warning: --public-path only applies to single file additions[/yellow]")
        public_path = None

    try:
        config = SyncConfig.from_yaml(config_path)
    except FileNotFoundError:
        console.print(f"[red]Config file not found: {config_path}[/red]")
        raise SystemExit(1)

    existing_files = {f.private_path for f in config.files}
    added_count = 0
    skipped_count = 0

    for file_path in file_paths:
        if file_path in existing_files:
            console.print(f"[yellow]Skipping (already exists): {file_path}[/yellow]")
            skipped_count += 1
            continue

        # Add new file
        new_file = FileMapping(
            private_path=file_path,
            public_path=public_path if len(file_paths) == 1 else None,
        )
        config.files.append(new_file)
        existing_files.add(file_path)
        console.print(f"[green]Added file: {file_path}[/green]")
        if public_path and len(file_paths) == 1:
            console.print(f"  Public path: {public_path}")
        added_count += 1

    if added_count > 0:
        config.to_yaml(config_path)
        console.print(f"\n[green]Added {added_count} file(s)[/green]")
    if skipped_count > 0:
        console.print(f"[yellow]Skipped {skipped_count} existing file(s)[/yellow]")


@cli.command()
@click.option(
    "--config",
    "-c",
    "config_path",
    type=click.Path(exists=True, dir_okay=False, path_type=Path),
    default=Path("git_syncer.yaml"),
    help="Path to the sync configuration file",
)
@click.argument("file_path")
def remove_file(config_path: Path, file_path: str):
    """Remove an individual file from the sync configuration."""
    try:
        config = SyncConfig.from_yaml(config_path)
    except FileNotFoundError:
        console.print(f"[red]Config file not found: {config_path}[/red]")
        raise SystemExit(1)

    # Find and remove file
    original_len = len(config.files)
    config.files = [f for f in config.files if f.private_path != file_path]

    if len(config.files) == original_len:
        console.print(f"[yellow]File not found: {file_path}[/yellow]")
        raise SystemExit(1)

    config.to_yaml(config_path)
    console.print(f"[green]Removed file: {file_path}[/green]")


@cli.command()
@click.option(
    "--config",
    "-c",
    "config_path",
    type=click.Path(exists=True, dir_okay=False, path_type=Path),
    default=Path("git_syncer.yaml"),
    help="Path to the sync configuration file",
)
@click.argument("project_path")
def remove_project(config_path: Path, project_path: str):
    """Remove a project from the sync configuration."""
    try:
        config = SyncConfig.from_yaml(config_path)
    except FileNotFoundError:
        console.print(f"[red]Config file not found: {config_path}[/red]")
        raise SystemExit(1)

    # Find and remove project
    original_len = len(config.projects)
    config.projects = [p for p in config.projects if p.private_path != project_path]

    if len(config.projects) == original_len:
        console.print(f"[yellow]Project not found: {project_path}[/yellow]")
        raise SystemExit(1)

    config.to_yaml(config_path)
    console.print(f"[green]Removed project: {project_path}[/green]")


@cli.command(name="list")
@click.option(
    "--config",
    "-c",
    "config_path",
    type=click.Path(exists=True, dir_okay=False, path_type=Path),
    default=Path("git_syncer.yaml"),
    help="Path to the sync configuration file",
)
def list_items(config_path: Path):
    """List all configured projects and files."""
    try:
        config = SyncConfig.from_yaml(config_path)
    except FileNotFoundError:
        console.print(f"[red]Config file not found: {config_path}[/red]")
        raise SystemExit(1)

    if not config.projects and not config.files:
        console.print("[yellow]No projects or files configured.[/yellow]")
        return

    if config.projects:
        console.print("\n[bold]Configured Projects (directories):[/bold]\n")
        for project in config.projects:
            status = "[green]enabled[/green]" if project.enabled else "[red]disabled[/red]"
            console.print(f"  • {project.private_path} ({status})")
            if project.public_path:
                console.print(f"      Public path: {project.public_path}")
            if project.exclude_patterns:
                console.print(f"      Excludes: {', '.join(project.exclude_patterns)}")

    if config.files:
        console.print("\n[bold]Configured Files:[/bold]\n")
        for file in config.files:
            status = "[green]enabled[/green]" if file.enabled else "[red]disabled[/red]"
            console.print(f"  • {file.private_path} ({status})")
            if file.public_path:
                console.print(f"      Public path: {file.public_path}")


@cli.command()
def reset_state():
    """Information about resetting sync state.

    Sync state is now tracked via 'synced_from:' lines in commit messages
    in the destination repository. To re-sync from scratch:

    1. Use --full flag: git-syncer sync --direction private-to-public --full

    This will sync all current files regardless of commit history.
    """
    console.print("\n[bold]Sync State Information[/bold]\n")
    console.print("Sync state is now tracked via [cyan]synced_from:[/cyan] lines in commit messages.")
    console.print("The syncer reads the last synced commit from the destination repo's git history.\n")
    console.print("[bold]To re-sync from scratch:[/bold]")
    console.print("  git-syncer sync --direction private-to-public --full\n")
    console.print("This will sync all current files regardless of previous sync history.")


if __name__ == "__main__":
    cli()

