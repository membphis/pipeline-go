import os
import yaml
from typing import Any

Spec = dict[str, Any]


def load(path: str, extra_specs: list[str] | None = None) -> Spec:
    project: Spec | None = None
    files = [path] + (extra_specs or [])
    for f in files:
        if not os.path.exists(f):
            raise FileNotFoundError(f"Spec file not found: {f}")
        with open(f) as fh:
            data: Spec = yaml.safe_load(fh)
        if data is None:
            raise ValueError(f"Empty spec file: {f}")
        if project is None:
            project = data
        else:
            project["milestones"].extend(data.get("milestones", []))
    if project is None:
        raise FileNotFoundError("No spec files provided")
    for key in ("project", "milestones"):
        if key not in project:
            raise ValueError(f"Spec missing required key: {key!r}")
    return project


def get_milestone(project: Spec, name: str) -> Spec | None:
    for ms in project.get("milestones", []):
        if ms.get("name") == name:
            return ms
    return None


def preflight(project: Spec) -> list[str]:
    errors: list[str] = []
    names = set()
    for ms in project.get("milestones", []):
        name = ms.get("name")
        if not name:
            errors.append("Milestone without name")
            continue
        if name in names:
            errors.append(f"Duplicate milestone name: {name}")
        names.add(name)
        tasks = ms.get("tasks", [])
        if not tasks:
            errors.append(f"Milestone {name!r} has no tasks")
    for ms in project.get("milestones", []):
        for dep in ms.get("depends_on", []):
            if dep not in names:
                errors.append(f"Milestone {ms['name']!r} has unknown dependency: {dep!r}")

    return errors
