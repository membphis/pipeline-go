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
    result = _run(*args)
    if result.returncode != 0:
        raise RuntimeError(f"Failed to create branch {name!r}: {result.stderr.strip()}")


def commit(message: str) -> None:
    result = _run("commit", "-m", message)
    if result.returncode != 0:
        raise RuntimeError(f"Failed to commit: {result.stderr.strip()}")


def checkout(branch: str) -> None:
    result = _run("checkout", branch)
    if result.returncode != 0:
        raise RuntimeError(f"Failed to checkout {branch!r}: {result.stderr.strip()}")


def squash_merge(branch: str) -> None:
    result = _run("merge", "--squash", branch)
    if result.returncode != 0:
        raise RuntimeError(f"Failed to squash-merge {branch!r}: {result.stderr.strip()}")
    result = _run("commit", "-m", f"Squash merge {branch}")
    if result.returncode != 0:
        raise RuntimeError(f"Failed to commit squash-merge: {result.stderr.strip()}")


def tag(name: str, message: str | None = None) -> None:
    if message:
        result = _run("tag", "-a", name, "-m", message)
    else:
        result = _run("tag", name)
    if result.returncode != 0:
        raise RuntimeError(f"Failed to tag {name!r}: {result.stderr.strip()}")


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
