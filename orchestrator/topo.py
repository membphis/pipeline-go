from typing import Any

Spec = dict[str, Any]


class CycleError(ValueError):
    """Raised when a cycle is detected in milestone dependencies."""

    def __init__(self, cycle: list[str]):
        self.cycle = cycle
        super().__init__(f"Circular dependency detected: {' -> '.join(cycle)}")


def sort(milestones: list[Spec]) -> list[str]:
    graph: dict[str, set[str]] = {}
    all_names: set[str] = set()

    for ms in milestones:
        name = ms.get("name")
        if not name:
            continue
        all_names.add(name)
        deps = set(ms.get("depends_on", []))
        graph[name] = deps

    visited: set[str] = set()
    in_progress: set[str] = set()
    order: list[str] = []

    def _visit(node: str) -> None:
        if node in in_progress:
            cycle_path: list[str] = []
            for n in order:
                cycle_path.append(n)
                if n == node:
                    break
            cycle_path.append(node)
            raise CycleError(cycle_path)
        if node in visited:
            return
        in_progress.add(node)
        for dep in graph.get(node, set()):
            if dep in all_names:
                _visit(dep)
        in_progress.remove(node)
        visited.add(node)
        order.append(node)

    for name in sorted(all_names):
        _visit(name)

    return order
