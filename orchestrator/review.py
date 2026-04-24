from dataclasses import dataclass
from typing import Any

from orchestrator import context, session, log

logger = log.get("review")


@dataclass
class ReviewResult:
    returncode: int
    stdout: str
    stderr: str


def _build_review_prompt(
    review_type: str,
    project_name: str | None = None,
    milestone: dict[str, Any] | None = None,
    handoff_notes: list[Any] | None = None,
    verify_results: list[Any] | None = None,
) -> tuple[str, str]:
    milestone_list = [milestone] if milestone else []
    bundle = context.build_bundle(
        milestones=milestone_list,
        handoff_notes=handoff_notes or [],
        verify_results=verify_results or [],
    )

    if context.exceeds_threshold(bundle):
        logger.warning("Context bundle exceeds token budget, degrading")
        degraded = False
        if verify_results:
            bundle = context.degrade(bundle, strategy="no_verify")
            degraded = True
        if handoff_notes and context.exceeds_threshold(bundle):
            bundle = context.degrade(bundle, strategy="no_handoff")
            degraded = True
        if not degraded:
            bundle = context.degrade(bundle, strategy="minimal")

    target = project_name or (milestone or {}).get("name", "unknown")
    prompt = (
        f"Review: {review_type} for {target}\n\n"
        f"## Context Bundle\n\n{bundle}\n\n"
        f"Please perform a {review_type} review and provide feedback. "
        f"Write your review to HANDOFF.md in the current directory."
    )
    return bundle, prompt


def review_phase(
    milestone: dict[str, Any],
    handoff_notes: list[Any] | None = None,
    verify_results: list[Any] | None = None,
) -> ReviewResult:
    bundle, prompt = _build_review_prompt(
        review_type="phase",
        milestone=milestone,
        handoff_notes=handoff_notes,
        verify_results=verify_results,
    )
    logger.info("starting phase review for milestone %r", milestone.get("name"))
    result = session.run(prompt)
    return ReviewResult(
        returncode=result.returncode,
        stdout=result.stdout,
        stderr=result.stderr,
    )


def review_final(
    project_name: str,
    milestones: list[dict[str, Any]],
    handoff_notes: list[Any] | None = None,
    verify_results: list[Any] | None = None,
) -> ReviewResult:
    bundle, prompt = _build_review_prompt(
        review_type="final",
        project_name=project_name,
        milestone=None,
        handoff_notes=handoff_notes,
        verify_results=verify_results,
    )
    logger.info("starting final review for project %r", project_name)
    result = session.run(prompt)
    return ReviewResult(
        returncode=result.returncode,
        stdout=result.stdout,
        stderr=result.stderr,
    )
