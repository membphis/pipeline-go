import subprocess
from typing import Any

from orchestrator import log

logger = log.get("git")


def _run(*args: str) -> subprocess.CompletedProcess:
    return subprocess.run(
        ["git", *args],
        capture_output=True,
        text=True,
        check=False,
    )


def current_branch() -> str:
    result = _run("rev-parse", "--abbrev-ref", "HEAD")
    if result.returncode != 0:
        raise RuntimeError(f"Failed to get current branch: {result.stderr.strip()}")
    branch = result.stdout.strip()
    if branch == "HEAD":
        raise RuntimeError("detached HEAD state")
    return branch


def create_branch(name: str, base: str | None = None) -> None:
    args = ["checkout", "-b", name]
    if base:
        args.append(base)
    _run(*args)


def commit(message: str) -> None:
    _run("commit", "-m", message)


def checkout(branch: str) -> None:
    _run("checkout", branch)


def squash_merge(branch: str) -> None:
    _run("merge", "--squash", branch)
    _run("commit", "-m", f"Squash merge {branch}")


def tag(name: str, message: str | None = None) -> None:
    if message:
        _run("tag", "-a", name, "-m", message)
    else:
        _run("tag", name)


def is_clean() -> bool:
    result = _run("status", "--porcelain")
    return result.stdout.strip() == ""


def has_unpushed_commits() -> bool:
    result = _run("rev-list", "--count", "@{u}..HEAD")
    if result.returncode != 0:
        return False
    return int(result.stdout.strip()) > 0


def is_detached_head() -> bool:
    result = _run("rev-parse", "--abbrev-ref", "HEAD")
    return result.stdout.strip() == "HEAD"
