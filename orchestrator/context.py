import math
from dataclasses import dataclass
from typing import Any, Literal

from orchestrator.handoff import HandoffNote

DegradeStrategy = Literal["no_verify", "no_handoff", "minimal"]


@dataclass
class VerifyResult:
    returncode: int
    stdout: str
    stderr: str


TOKEN_ESTIMATE_RATIO = 4


def count_tokens(text: str) -> int:
    return math.ceil(len(text) / TOKEN_ESTIMATE_RATIO)


def build_bundle(
    milestones: list[dict[str, Any]] | None = None,
    handoff_notes: list[HandoffNote] | None = None,
    verify_results: list[VerifyResult] | None = None,
) -> str:
    parts: list[str] = []

    if milestones:
        parts.append("## Pipeline State\n")
        for ms in milestones:
            name = ms.get("name", "?")
            spec = ms.get("spec", "")
            parts.append(f"### Milestone: {name}\n\n{spec}\n")

    if handoff_notes:
        parts.append("## Handoff Notes\n")
        for n in handoff_notes:
            parts.append(f"### From: {n.source}\n\n{n.content}\n")

    if verify_results:
        parts.append("## Verify Results\n")
        for r in verify_results:
            status = "PASS" if r.returncode == 0 else "FAIL"
            parts.append(f"- {status}: {r.stdout}")
            if r.stderr:
                parts.append(f"  stderr: {r.stderr}")

    return "\n".join(parts).strip()


def exceeds_threshold(text: str, max_tokens: int = 180_000) -> bool:
    return count_tokens(text) > max_tokens


def degrade(bundle: str, strategy: DegradeStrategy) -> str:
    lines = bundle.split("\n")
    filtered: list[str] = []
    skip_section = False

    for line in lines:
        if strategy in ("no_verify", "minimal") and line.strip().startswith(
            "## Verify Results"
        ):
            skip_section = True
            continue
        if strategy in ("no_handoff", "minimal") and line.strip().startswith(
            "## Handoff Notes"
        ):
            skip_section = True
            continue
        if line.strip().startswith("## ") and not line.strip().startswith(
            "## Pipeline State"
        ):
            skip_section = False
            continue
        if line.strip().startswith("## Pipeline State"):
            skip_section = False
        if not skip_section:
            filtered.append(line)

    return "\n".join(filtered).strip()
