import argparse
import os
import sys
from typing import Any

from orchestrator import log, spec, topo, state, git, session, verify, handoff, context, review

logger = log.get("main")


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="AI Project Pipeline Orchestrator")
    parser.add_argument("--spec", "-s", default="project.yaml", help="Path to project.yaml")
    parser.add_argument("--extra-spec", action="append", default=[], help="Extra spec files to merge")
    parser.add_argument("--root", default=".", help="Project root directory")
    parser.add_argument("--branch", default=None, help="Use existing branch instead of creating milestone branches")
    return parser.parse_args(argv)


def compute_milestone_spec(
    milestone: dict[str, Any],
    all_milestones: list[dict[str, Any]],
    pipeline_state: state.State | None,
    handoff_notes: list[Any] | None = None,
) -> str:
    parts: list[str] = []

    name = milestone.get("name", "?")
    spec_text = milestone.get("spec", "")
    tasks = milestone.get("tasks", [])

    parts.append(f"# Milestone: {name}\n")
    if spec_text:
        parts.append(f"## Spec\n\n{spec_text}\n")

    if tasks:
        parts.append("## Tasks\n")
        for t in tasks:
            tname = t.get("name", "?")
            tprompt = t.get("prompt", "")
            parts.append(f"### {tname}\n\n{tprompt}\n")

    if pipeline_state:
        parts.append("## Pipeline State\n")
        for ms_name, ms_status in pipeline_state.get_all().items():
            parts.append(f"- {ms_name}: {ms_status}\n")

    if handoff_notes:
        parts.append("## Previous Handoff Notes\n")
        for n in handoff_notes:
            parts.append(f"### {n.source}\n\n{n.content}\n")

    if all_milestones:
        remaining = [m for m in all_milestones if m.get("name") != name]
        if remaining:
            parts.append("## Upcoming Milestones\n")
            for m in remaining:
                parts.append(f"- {m.get('name', '?')}: {m.get('spec', '')[:80]}...\n")

    return "\n".join(parts).strip()


class Pipeline:
    def __init__(
        self,
        spec_path: str,
        root: str,
        extra_specs: list[str],
        branch: str | None,
    ):
        self.spec_path = spec_path
        self.root = root
        self.extra_specs = extra_specs
        self.branch_override = branch

    def run(self) -> int:
        spec_path = os.path.join(self.root, self.spec_path)
        extra_paths = [os.path.join(self.root, e) for e in self.extra_specs]

        logger.info("Loading spec from %s", spec_path)
        project_spec = spec.load(spec_path, extra_specs=extra_paths)
        project_name = project_spec.get("project", {}).get("name", "unknown")
        milestones = project_spec.get("milestones", [])

        errors = spec.preflight(project_spec)
        if errors:
            logger.error("Preflight validation failed:")
            for e in errors:
                logger.error("  - %s", e)
            return 1

        ordered = topo.sort(milestones)
        logger.info("Milestone order: %s", ordered)

        pipeline_state = state.State(milestones, path=os.path.join(self.root, "state.yaml"))

        all_handoff_notes: list[Any] = []

        for ms_name in ordered:
            ms = spec.get_milestone(project_spec, ms_name)
            if ms is None:
                logger.warning("Milestone %r not found, skipping", ms_name)
                continue

            pipeline_state.set(ms_name, "in_progress")
            pipeline_state.save()

            if self.branch_override:
                branch_name = self.branch_override
                if not git.is_clean():
                    logger.warning("Working tree not clean, stashing changes")
            else:
                branch_name = f"{ms_name}-pipeline"
                git.checkout("main")
                git.create_branch(branch_name)

            logger.info("Starting milestone %r on branch %r", ms_name, branch_name)

            prompt = compute_milestone_spec(ms, milestones, pipeline_state, all_handoff_notes)
            logger.info("Running milestone %r with %d-byte prompt", ms_name, len(prompt))

            result = session.run(prompt)

            if result.returncode == 0:
                pipeline_state.set(ms_name, "completed")
            else:
                pipeline_state.set(ms_name, "failed")
                logger.warning("Milestone %r session failed (code %d)", ms_name, result.returncode)

            pipeline_state.save()

            verify_specs = ms.get("verify")
            if verify_specs:
                v_results = verify.run_verify(verify_specs)

            handoff_notes = handoff.collect(self.root)
            all_handoff_notes.extend(handoff_notes)

            review.review_phase(
                milestone=ms,
                handoff_notes=handoff_notes,
                verify_results=v_results if verify_specs else [],
            )

            if not self.branch_override:
                git.commit(f"Milestone {ms_name}")
                git.tag(f"{ms_name}-done")
                git.checkout("main")
                git.squash_merge(branch_name)

        all_v_results: list[Any] = []
        review.review_final(
            project_name=project_name,
            milestones=milestones,
            handoff_notes=all_handoff_notes,
            verify_results=all_v_results,
        )

        if pipeline_state.all_completed():
            git.tag(f"{project_name}-v1.0")
            logger.info("Pipeline completed successfully")
        else:
            logger.warning("Pipeline completed with some milestones not completed")

        return 0


def main() -> None:
    log.setup()
    args = parse_args()
    pipeline = Pipeline(
        spec_path=args.spec,
        root=args.root,
        extra_specs=args.extra_spec,
        branch=args.branch,
    )
    sys.exit(pipeline.run())


if __name__ == "__main__":
    main()
