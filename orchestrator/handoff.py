from dataclasses import dataclass
from pathlib import Path

from orchestrator import log

logger = log.get("handoff")


@dataclass
class HandoffNote:
    source: str
    content: str


def collect(root: str) -> list[HandoffNote]:
    notes: list[HandoffNote] = []
    base = Path(root)
    if not base.exists():
        return notes
    for path in sorted(base.rglob("HANDOFF.md")):
        if path.is_file():
            notes.append(HandoffNote(
                source=str(path),
                content=path.read_text(encoding="utf-8"),
            ))
    logger.info("collected %d handoff notes from %s", len(notes), root)
    return notes


def format_notes(notes: list[HandoffNote]) -> str:
    if not notes:
        return ""
    parts: list[str] = []
    for n in notes:
        parts.append(f"## Handoff: {n.source}\n\n{n.content}")
    return "\n\n---\n\n".join(parts)
