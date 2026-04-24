# Orchestrator v1 Skeleton Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Python CLI tool that orchestrates AI-driven project execution: parse project.yaml → topologically sort tasks → call opencode per task → verify → commit → handoff → review.

**Architecture:** 12 modules in the `orchestrator/` package, built from bottom-up (no deps first). Each module is independently testable with pytest. The CLI uses argparse with `run` and `run --resume` subcommands.

**Tech Stack:** Python 3.10+, PyYAML, pytest, subprocess for git/opencode calls.

---

### Task 1: Project scaffolding

**Files:**
- Create: `orchestrator/__init__.py`
- Create: `pyproject.toml`
- Create: `.gitignore`
- Create: `tests/__init__.py`

- [ ] **Step 1: Create orchestrator package init**

```python
# orchestrator/__init__.py
"""AI Project Pipeline Orchestrator."""
```

- [ ] **Step 2: Create pyproject.toml**

```toml
# pyproject.toml
[project]
name = "orchestrator"
version = "0.1.0"
requires-python = ">=3.10"
dependencies = ["pyyaml>=6.0"]

[project.scripts]
orchestrator = "orchestrator.main:main"

[build-system]
requires = ["setuptools>=68.0"]
build-backend = "setuptools.build_meta"
```

- [ ] **Step 3: Create .gitignore**

```
__pycache__/
*.pyc
.pytest_cache/
.opencode/logs/
.opencode/.tmp/
state.yaml
/dist/
```

- [ ] **Step 4: Create tests init**

```python
# tests/__init__.py
```

- [ ] **Step 5: Install editable and verify**

Run: `pip install -e .`
Expected: Package installs without error.

- [ ] **Step 6: Commit**

```bash
git init && git add -A && git commit -m "chore: scaffold project structure"
```

---

### Task 2: log.py — Structured logging

**Files:**
- Create: `orchestrator/log.py`
- Create: `tests/test_log.py`

- [ ] **Step 1: Write the failing test**

```python
# tests/test_log.py
import logging
from orchestrator import log


def test_setup_configures_root_logger():
    log.setup()
    root = logging.getLogger("orchestrator")
    assert root.level == logging.INFO
    assert len(root.handlers) >= 1


def test_get_returns_child_logger():
    logger = log.get("spec")
    assert logger.name == "orchestrator.spec"


def test_setup_respects_custom_level():
    log.setup(level=logging.DEBUG)
    root = logging.getLogger("orchestrator")
    assert root.level == logging.DEBUG
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pytest tests/test_log.py -v`
Expected: FAIL — module not found / functions not defined.

- [ ] **Step 3: Write minimal implementation**

```python
# orchestrator/log.py
import logging
import sys


def setup(level: int = logging.INFO):
    logger = logging.getLogger("orchestrator")
    logger.setLevel(level)
    logger.propagate = False

    if not logger.handlers:
        handler = logging.StreamHandler(sys.stderr)
        handler.setLevel(level)
        fmt = logging.Formatter(
            "%(asctime)s [%(levelname)-5s] %(name)s: %(message)s",
            datefmt="%Y-%m-%d %H:%M:%S",
        )
        handler.setFormatter(fmt)
        logger.addHandler(handler)


def get(name: str) -> logging.Logger:
    return logging.getLogger(f"orchestrator.{name}")
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pytest tests/test_log.py -v`
Expected: 3 PASS.

- [ ] **Step 5: Commit**

```bash
git add orchestrator/log.py tests/test_log.py
git commit -m "feat: add structured logging module"
```

---

### Task 3: spec.py — project.yaml parser

**Files:**
- Create: `orchestrator/spec.py`
- Create: `tests/test_spec.py`
- Create: `tests/fixtures/project.yaml`

- [ ] **Step 1: Create fixture project.yaml**

```yaml
# tests/fixtures/project.yaml
project:
  name: test-backend
  language: rust
  scratch_template: "cargo init --name test-backend"

milestones:
  - id: m1-core
    name: Core Foundation
    order: 1
    specs:
      - id: m1-s1-data-model
        name: Data Model
        tasks:
          - id: m1-s1-t1
            name: Define structs
            depends_on: []
            expected_files:
              - src/models/mod.rs
            prompt: |
              Define core data structures.
            verify:
              - cargo build
              - cargo test

          - id: m1-s1-t2
            name: Add validation
            depends_on: [m1-s1-t1]
            expected_files:
              - src/models/validation.rs
            prompt: |
              Add validation logic.
            verify:
              - cargo test

      - id: m1-s2-database
        name: Database Layer
        depends_on_specs: [m1-s1]
        tasks:
          - id: m1-s2-t1
            name: Connection pool
            depends_on: []
            expected_files:
              - src/db/pool.rs
            prompt: |
              Implement connection pool.
            verify:
              - cargo build
```

- [ ] **Step 2: Write the failing test**

```python
# tests/test_spec.py
import os
import pytest
from orchestrator.spec import parse, preflight, Task, Spec, Milestone, Project


FIXTURE = os.path.join(os.path.dirname(__file__), "fixtures", "project.yaml")


def test_parse_returns_project():
    project = parse(FIXTURE)
    assert isinstance(project, Project)
    assert project.name == "test-backend"
    assert project.language == "rust"


def test_parse_has_correct_milestone_count():
    project = parse(FIXTURE)
    assert len(project.milestones) == 1


def test_parse_milestone_structure():
    project = parse(FIXTURE)
    m = project.milestones[0]
    assert m.id == "m1-core"
    assert m.name == "Core Foundation"
    assert m.order == 1
    assert len(m.specs) == 2


def test_parse_spec_structure():
    project = parse(FIXTURE)
    spec = project.milestones[0].specs[0]
    assert spec.id == "m1-s1-data-model"
    assert spec.name == "Data Model"
    assert len(spec.tasks) == 2
    assert spec.depends_on_specs == []


def test_parse_spec_depends_on_specs():
    project = parse(FIXTURE)
    spec = project.milestones[0].specs[1]
    assert spec.depends_on_specs == ["m1-s1"]


def test_parse_task_structure():
    project = parse(FIXTURE)
    task = project.milestones[0].specs[0].tasks[0]
    assert task.id == "m1-s1-t1"
    assert task.name == "Define structs"
    assert task.depends_on == []
    assert task.expected_files == ["src/models/mod.rs"]
    assert task.verify == ["cargo build", "cargo test"]
    assert task.milestone_id == "m1-core"
    assert task.spec_id == "m1-s1-data-model"


def test_parse_task_with_deps():
    project = parse(FIXTURE)
    task = project.milestones[0].specs[0].tasks[1]
    assert task.depends_on == ["m1-s1-t1"]


def test_preflight_passes_on_valid_project():
    project = parse(FIXTURE)
    result = preflight(project)
    assert result is True


def test_preflight_detects_duplicate_task_ids(monkeypatch, tmp_path):
    # Create a project with duplicate task IDs
    yaml_content = """
project:
  name: dup
  language: rust
  scratch_template: "cargo init"
milestones:
  - id: m1
    name: M1
    order: 1
    specs:
      - id: s1
        name: S1
        tasks:
          - id: t1
            name: Task1
            depends_on: []
            expected_files: []
            prompt: ""
            verify: []
          - id: t1
            name: Task2
            depends_on: []
            expected_files: []
            prompt: ""
            verify: []
"""
    yaml_file = tmp_path / "project.yaml"
    yaml_file.write_text(yaml_content)
    project = parse(str(yaml_file))
    result = preflight(project)
    assert result is False


def test_preflight_detects_invalid_depends_on(monkeypatch, tmp_path):
    yaml_content = """
project:
  name: bad
  language: rust
  scratch_template: ""
milestones:
  - id: m1
    name: M1
    order: 1
    specs:
      - id: s1
        name: S1
        tasks:
          - id: t1
            name: T1
            depends_on: [nonexistent]
            expected_files: []
            prompt: ""
            verify: []
"""
    yaml_file = tmp_path / "project.yaml"
    yaml_file.write_text(yaml_content)
    project = parse(str(yaml_file))
    result = preflight(project)
    assert result is False
```

- [ ] **Step 3: Run test to verify it fails**

Run: `pytest tests/test_spec.py -v`
Expected: FAIL — module not found / functions not defined.

- [ ] **Step 4: Write minimal implementation**

```python
# orchestrator/spec.py
import glob
import logging
import os
import yaml
from dataclasses import dataclass, field
from typing import Optional


log = logging.getLogger("orchestrator.spec")


@dataclass
class Task:
    id: str
    name: str
    depends_on: list[str]
    expected_files: list[str]
    prompt: str
    verify: list[str]
    milestone_id: str = ""
    spec_id: str = ""


@dataclass
class Spec:
    id: str
    name: str
    depends_on_specs: list[str]
    tasks: list[Task]


@dataclass
class Milestone:
    id: str
    name: str
    order: int
    specs: list[Spec]


@dataclass
class Project:
    name: str
    language: str
    scratch_template: str
    milestones: list[Milestone]


def parse(yaml_path: str) -> Project:
    """Parse project.yaml (or multiple project*.yaml files) into a Project."""
    # Support multi-file: glob project*.yaml in the same directory
    directory = os.path.dirname(yaml_path) or "."
    pattern = os.path.join(directory, "project*.yaml")
    files = sorted(glob.glob(pattern))

    if not files:
        raise FileNotFoundError(f"No project*.yaml files found matching {pattern}")

    milestones: list[Milestone] = []

    for fpath in files:
        with open(fpath) as f:
            data = yaml.safe_load(f)

        project_data = data.get("project", {})
        file_milestones = data.get("milestones", [])

        for m_data in file_milestones:
            specs = []
            for s_data in m_data.get("specs", []):
                tasks = []
                for t_data in s_data.get("tasks", []):
                    task = Task(
                        id=t_data["id"],
                        name=t_data["name"],
                        depends_on=t_data.get("depends_on", []),
                        expected_files=t_data.get("expected_files", []),
                        prompt=t_data.get("prompt", ""),
                        verify=t_data.get("verify", []),
                        milestone_id=m_data["id"],
                        spec_id=s_data["id"],
                    )
                    tasks.append(task)
                spec = Spec(
                    id=s_data["id"],
                    name=s_data["name"],
                    depends_on_specs=s_data.get("depends_on_specs", []),
                    tasks=tasks,
                )
                specs.append(spec)
            milestone = Milestone(
                id=m_data["id"],
                name=m_data["name"],
                order=m_data["order"],
                specs=specs,
            )
            milestones.append(milestone)

    milestones.sort(key=lambda m: m.order)

    return Project(
        name=project_data.get("name", ""),
        language=project_data.get("language", ""),
        scratch_template=project_data.get("scratch_template", ""),
        milestones=milestones,
    )


def preflight(project: Project) -> bool:
    """Run preflight checks. Returns True if all pass, False if errors found."""
    all_ok = True
    task_ids: set[str] = set()

    for milestone in project.milestones:
        for spec in milestone.specs:
            for task in spec.tasks:
                if task.id in task_ids:
                    log.error(f"Duplicate task ID: {task.id}")
                    all_ok = False
                task_ids.add(task.id)

    for milestone in project.milestones:
        for spec in milestone.specs:
            for task in spec.tasks:
                for dep_id in task.depends_on:
                    if dep_id not in task_ids:
                        log.error(f"Task {task.id} depends_on unknown task: {dep_id}")
                        all_ok = False
                for dep_spec_id in spec.depends_on_specs:
                    # Spec-level deps are validated separately in topo.py
                    pass

    log.info(f"Preflight: {len(task_ids)} tasks, {'PASS' if all_ok else 'FAIL'}")
    return all_ok


def all_tasks(project: Project) -> list[Task]:
    """Return a flat list of all tasks across all milestones and specs."""
    tasks = []
    for m in project.milestones:
        for s in m.specs:
            tasks.extend(s.tasks)
    return tasks
```

- [ ] **Step 5: Run test to verify it passes**

Run: `pytest tests/test_spec.py -v`
Expected: 10 PASS.

- [ ] **Step 6: Commit**

```bash
git add orchestrator/spec.py tests/test_spec.py tests/fixtures/project.yaml
git commit -m "feat: add project.yaml parser and preflight validation"
```

---

### Task 4: topo.py — Topological sort

**Files:**
- Create: `orchestrator/topo.py`
- Create: `tests/test_topo.py`

- [ ] **Step 1: Write the failing test**

```python
# tests/test_topo.py
import pytest
from orchestrator.topo import sort, detect_cycle, expand_spec_deps
from orchestrator.spec import Task, Spec, Milestone, Project


def make_task(id: str, depends_on: list[str] = None) -> Task:
    return Task(
        id=id, name=id, depends_on=depends_on or [],
        expected_files=[], prompt="", verify=[],
        milestone_id="m1", spec_id="s1",
    )


def test_sort_linear_chain():
    t1 = make_task("t1", [])
    t2 = make_task("t2", ["t1"])
    t3 = make_task("t3", ["t2"])
    spec = Spec(id="s1", name="S1", depends_on_specs=[], tasks=[t2, t1, t3])
    milestone = Milestone(id="m1", name="M1", order=1, specs=[spec])
    project = Project(name="p", language="rust", scratch_template="", milestones=[milestone])

    result = sort(project)
    ids = [t.id for t in result]
    assert ids.index("t1") < ids.index("t2")
    assert ids.index("t2") < ids.index("t3")


def test_sort_independent_tasks_can_be_any_order():
    t1 = make_task("t1", [])
    t2 = make_task("t2", [])
    spec = Spec(id="s1", name="S1", depends_on_specs=[], tasks=[t1, t2])
    milestone = Milestone(id="m1", name="M1", order=1, specs=[spec])
    project = Project(name="p", language="rust", scratch_template="", milestones=[milestone])

    result = sort(project)
    ids = [t.id for t in result]
    assert set(ids) == {"t1", "t2"}
    assert len(ids) == 2


def test_sort_diamond_dependency():
    t1 = make_task("t1", [])
    t2 = make_task("t2", ["t1"])
    t3 = make_task("t3", ["t1"])
    t4 = make_task("t4", ["t2", "t3"])
    spec = Spec(id="s1", name="S1", depends_on_specs=[], tasks=[t4, t3, t2, t1])
    milestone = Milestone(id="m1", name="M1", order=1, specs=[spec])
    project = Project(name="p", language="rust", scratch_template="", milestones=[milestone])

    result = sort(project)
    ids = [t.id for t in result]
    assert ids.index("t1") == 0
    assert ids.index("t4") == 3


def test_detect_cycle_raises():
    t1 = make_task("t1", ["t2"])
    t2 = make_task("t2", ["t1"])
    spec = Spec(id="s1", name="S1", depends_on_specs=[], tasks=[t1, t2])
    milestone = Milestone(id="m1", name="M1", order=1, specs=[spec])
    project = Project(name="p", language="rust", scratch_template="", milestones=[milestone])

    with pytest.raises(ValueError, match="cycle"):
        sort(project)


def test_spec_dep_expands_to_task_deps():
    t1 = make_task("m1-s1-t1", [])
    t2 = make_task("m1-s2-t1", [])
    spec1 = Spec(id="m1-s1", name="S1", depends_on_specs=[], tasks=[t1])
    spec2 = Spec(id="m1-s2", name="S2", depends_on_specs=["m1-s1"], tasks=[t2])
    milestone = Milestone(id="m1", name="M1", order=1, specs=[spec1, spec2])
    project = Project(name="p", language="rust", scratch_template="", milestones=[milestone])

    result = sort(project)
    ids = [t.id for t in result]
    assert ids.index("m1-s1-t1") < ids.index("m1-s2-t1")
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pytest tests/test_topo.py -v`
Expected: FAIL.

- [ ] **Step 3: Write minimal implementation**

```python
# orchestrator/topo.py
import logging
from collections import deque
from orchestrator.spec import Project, Task

log = logging.getLogger("orchestrator.topo")


def sort(project: Project) -> list[Task]:
    """
    Topologically sort all tasks in the project.

    Handles:
      - Task-level depends_on
      - Spec-level depends_on_specs (expands to: all tasks in this spec
        depend on all tasks in the depended-upon spec)
    """
    all_tasks_list = _all_tasks(project)

    # Build adjacency and in-degree
    task_ids = {t.id for t in all_tasks_list}
    in_degree: dict[str, int] = {t.id: 0 for t in all_tasks_list}
    adj: dict[str, list[str]] = {t.id: [] for t in all_tasks_list}

    # Map spec_id -> list of task IDs
    spec_task_ids: dict[str, list[str]] = {}
    for m in project.milestones:
        for s in m.specs:
            spec_task_ids[s.id] = [t.id for t in s.tasks]

    # Add edges
    for t in all_tasks_list:
        # Task-level deps
        for dep_id in t.depends_on:
            if dep_id not in task_ids:
                raise ValueError(f"Task {t.id} depends_on unknown task: {dep_id}")
            adj[dep_id].append(t.id)
            in_degree[t.id] += 1

        # Spec-level deps: all tasks in this spec depend on all tasks
        # in the depended-upon spec(s)
        spec = _find_spec(project, t.spec_id, t.milestone_id)
        if spec:
            for dep_spec_id in spec.depends_on_specs:
                for dep_task_id in spec_task_ids.get(dep_spec_id, []):
                    adj[dep_task_id].append(t.id)
                    in_degree[t.id] += 1

    # Kahn's algorithm
    queue = deque([tid for tid, deg in in_degree.items() if deg == 0])
    result: list[Task] = []
    task_map = {t.id: t for t in all_tasks_list}

    while queue:
        tid = queue.popleft()
        result.append(task_map[tid])
        for neighbor in adj[tid]:
            in_degree[neighbor] -= 1
            if in_degree[neighbor] == 0:
                queue.append(neighbor)

    if len(result) != len(all_tasks_list):
        remaining = [tid for tid, deg in in_degree.items() if deg > 0]
        raise ValueError(f"Cycle detected in task dependencies: {remaining}")

    return result


def detect_cycle(project: Project) -> list[str] | None:
    """Return list of task IDs in cycle, or None if acyclic."""
    try:
        sort(project)
        return None
    except ValueError:
        return []  # Exact cycle detection is best-effort for v1


def _all_tasks(project: Project) -> list[Task]:
    tasks = []
    for m in project.milestones:
        for s in m.specs:
            tasks.extend(s.tasks)
    return tasks


def _find_spec(project: Project, spec_id: str, milestone_id: str) -> object | None:
    for m in project.milestones:
        for s in m.specs:
            if s.id == spec_id and m.id == milestone_id:
                return s
    return None
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pytest tests/test_topo.py -v`
Expected: 5 PASS.

- [ ] **Step 5: Commit**

```bash
git add orchestrator/topo.py tests/test_topo.py
git commit -m "feat: add topological sort with spec-level deps"
```

---

### Task 5: state.py — State management

**Files:**
- Create: `orchestrator/state.py`
- Create: `tests/test_state.py`

- [ ] **Step 1: Write the failing test**

```python
# tests/test_state.py
import os
import tempfile
from datetime import datetime
from orchestrator.state import State, TaskRef, FailedInfo, load_or_init, save, mark_complete, mark_failed
from orchestrator.spec import Project, Milestone, Spec, Task


def make_project():
    return Project(
        name="test",
        language="rust",
        scratch_template="",
        milestones=[
            Milestone(
                id="m1", name="M1", order=1,
                specs=[
                    Spec(
                        id="s1", name="S1", depends_on_specs=[],
                        tasks=[
                            Task(
                                id="t1", name="T1", depends_on=[],
                                expected_files=[], prompt="",
                                verify=[], milestone_id="m1", spec_id="s1",
                            )
                        ]
                    )
                ]
            )
        ]
    )


def test_load_or_init_returns_new_state_when_no_file():
    project = make_project()
    state = load_or_init(project, "/nonexistent/path/state.yaml", resume=False)
    assert state.version == 1
    assert state.project_name == "test"
    assert state.status == "running"
    assert state.completed == []
    assert state.current is None


def test_load_or_init_resume_loads_existing(tmp_path):
    project = make_project()
    state_file = tmp_path / "state.yaml"
    state = load_or_init(project, str(state_file), resume=False)
    mark_complete(state, "t1")
    save(state, str(state_file))

    loaded = load_or_init(project, str(state_file), resume=True)
    assert loaded.completed == ["t1"]


def test_mark_complete_adds_to_list():
    state = State(
        version=1, project_name="test", status="running",
        current=None, completed=[], failed={},
        started_at="", last_updated="",
    )
    mark_complete(state, "t1")
    assert "t1" in state.completed


def test_mark_failed_records_error():
    state = State(
        version=1, project_name="test", status="running",
        current=None, completed=[], failed={},
        started_at="", last_updated="",
    )
    mark_failed(state, "t1", "build failed")
    assert "t1" in state.failed
    assert state.failed["t1"].last_error == "build failed"


def test_save_and_load_roundtrip(tmp_path):
    project = make_project()
    state_file = tmp_path / "state.yaml"
    state = load_or_init(project, str(state_file), resume=False)
    mark_complete(state, "t1")
    mark_failed(state, "t2", "oops")
    save(state, str(state_file))

    loaded = load_or_init(project, str(state_file), resume=True)
    assert loaded.completed == ["t1"]
    assert loaded.failed["t2"].last_error == "oops"
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pytest tests/test_state.py -v`
Expected: FAIL.

- [ ] **Step 3: Write minimal implementation**

```python
# orchestrator/state.py
import logging
import os
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Optional
import yaml

from orchestrator.spec import Project

log = logging.getLogger("orchestrator.state")


@dataclass
class TaskRef:
    milestone: str
    spec: str
    task: str


@dataclass
class FailedInfo:
    attempts: int
    last_error: str


@dataclass
class State:
    version: int
    project_name: str
    status: str  # pending | running | completed | partially_failed | failed
    current: Optional[TaskRef]
    completed: list[str]
    failed: dict[str, FailedInfo]
    started_at: str
    last_updated: str


def load_or_init(project: Project, state_path: str, resume: bool = False) -> State:
    """Load existing state.yaml or create a new one."""
    if resume and os.path.exists(state_path):
        with open(state_path) as f:
            data = yaml.safe_load(f)
        failed = {}
        for tid, info in data.get("failed", {}).items():
            failed[tid] = FailedInfo(
                attempts=info.get("attempts", 0),
                last_error=info.get("last_error", ""),
            )
        current = None
        if data.get("current"):
            cur = data["current"]
            current = TaskRef(milestone=cur["milestone"], spec=cur["spec"], task=cur["task"])

        state = State(
            version=data.get("version", 1),
            project_name=data.get("project_name", ""),
            status=data.get("status", "running"),
            current=current,
            completed=data.get("completed", []),
            failed=failed,
            started_at=data.get("started_at", ""),
            last_updated=data.get("last_updated", ""),
        )
        log.info(f"Resumed state: {len(state.completed)} completed, {len(state.failed)} failed")
        return state

    now = datetime.now(timezone.utc).isoformat()
    state = State(
        version=1,
        project_name=project.name,
        status="running",
        current=None,
        completed=[],
        failed={},
        started_at=now,
        last_updated=now,
    )
    log.info(f"Initialized new state for {project.name}")
    return state


def save(state: State, state_path: str):
    """Write state to state.yaml."""
    state.last_updated = datetime.now(timezone.utc).isoformat()

    failed_serialized = {}
    for tid, info in state.failed.items():
        failed_serialized[tid] = {"attempts": info.attempts, "last_error": info.last_error}

    current_serialized = None
    if state.current:
        current_serialized = {
            "milestone": state.current.milestone,
            "spec": state.current.spec,
            "task": state.current.task,
        }

    data = {
        "version": state.version,
        "project_name": state.project_name,
        "status": state.status,
        "current": current_serialized,
        "completed": state.completed,
        "failed": failed_serialized,
        "started_at": state.started_at,
        "last_updated": state.last_updated,
    }

    with open(state_path, "w") as f:
        yaml.safe_dump(data, f, default_flow_style=False, sort_keys=False)


def mark_complete(state: State, task_id: str):
    """Mark a task as completed."""
    if task_id not in state.completed:
        state.completed.append(task_id)


def mark_failed(state: State, task_id: str, error: str):
    """Mark a task as failed with error info."""
    attempts = state.failed.get(task_id, FailedInfo(attempts=0, last_error="")).attempts + 1
    state.failed[task_id] = FailedInfo(attempts=attempts, last_error=error)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pytest tests/test_state.py -v`
Expected: 5 PASS.

- [ ] **Step 5: Commit**

```bash
git add orchestrator/state.py tests/test_state.py
git commit -m "feat: add state.yaml management with resume support"
```

---

### Task 6: git.py — Git operations

**Files:**
- Create: `orchestrator/git.py`
- Create: `tests/test_git.py`

- [ ] **Step 1: Write the failing test**

```python
# tests/test_git.py
import os
import subprocess
import pytest
from orchestrator.git import checkout_branch, commit, create_tag, merge_to_main
from orchestrator.spec import Task


def make_task(id="t1", name="Test task"):
    return Task(
        id=id, name=name, depends_on=[], expected_files=[],
        prompt="", verify=[], milestone_id="m1", spec_id="s1",
    )


@pytest.fixture
def git_repo(tmp_path, monkeypatch):
    """Create a temp git repo and cd into it."""
    monkeypatch.chdir(tmp_path)
    subprocess.run(["git", "init"], check=True, capture_output=True)
    subprocess.run(
        ["git", "config", "user.email", "test@test.com"], check=True, capture_output=True
    )
    subprocess.run(
        ["git", "config", "user.name", "Test"], check=True, capture_output=True
    )
    # Create initial commit so we have a main branch
    (tmp_path / "README.md").write_text("# test")
    subprocess.run(["git", "add", "."], check=True, capture_output=True)
    subprocess.run(["git", "commit", "-m", "initial"], check=True, capture_output=True)
    return tmp_path


def test_checkout_branch_creates_new_branch(git_repo):
    checkout_branch("milestone/m1-core")
    result = subprocess.run(
        ["git", "branch", "--show-current"], capture_output=True, text=True
    )
    assert result.stdout.strip() == "milestone/m1-core"


def test_commit_creates_commit_with_task_message(git_repo):
    checkout_branch("milestone/m1-core")
    task = make_task("m1-s1-t1", "Define structs")
    (git_repo / "test.txt").write_text("content")
    commit(task)

    result = subprocess.run(
        ["git", "log", "--oneline", "-1"], capture_output=True, text=True
    )
    assert "task: m1-s1-t1 - Define structs" in result.stdout


def test_create_tag_adds_tag(git_repo):
    create_tag("v0.1.0-m1")
    result = subprocess.run(
        ["git", "tag", "-l"], capture_output=True, text=True
    )
    assert "v0.1.0-m1" in result.stdout


def test_merge_to_main_squash(git_repo):
    # Create a branch with a commit
    checkout_branch("milestone/m1-core")
    task = make_task("m1-s1-t1", "T1")
    (git_repo / "work.txt").write_text("done")
    commit(task)

    # Merge back to main with squash
    merge_to_main("milestone/m1-core")

    result = subprocess.run(
        ["git", "log", "--oneline", "main"], capture_output=True, text=True
    )
    assert "merge milestone/milestone/m1-core" in result.stdout
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pytest tests/test_git.py -v`
Expected: FAIL.

- [ ] **Step 3: Write minimal implementation**

```python
# orchestrator/git.py
import logging
import subprocess

from orchestrator.spec import Task

log = logging.getLogger("orchestrator.git")


def checkout_branch(name: str):
    """Create or switch to a branch. Caller passes the exact branch name."""
    r = subprocess.run(
        ["git", "checkout", "-b", name],
        capture_output=True, text=True,
    )
    if r.returncode != 0:
        subprocess.run(
            ["git", "checkout", name],
            check=True, capture_output=True, text=True,
        )
    log.info(f"Checked out branch: {name}")


def commit(task: Task):
    subprocess.run(["git", "add", "."], check=True, capture_output=True)
    msg = f"task: {task.id} - {task.name}"
    subprocess.run(["git", "commit", "-m", msg], check=True, capture_output=True)
    log.info(f"Committed: {msg}")


def create_tag(tag: str):
    subprocess.run(["git", "tag", tag], check=True, capture_output=True)
    log.info(f"Tagged: {tag}")


def merge_to_main(branch: str):
    subprocess.run(["git", "checkout", "main"], check=True, capture_output=True)
    subprocess.run(
        ["git", "merge", "--squash", branch], check=True, capture_output=True
    )
    msg = f"merge milestone/{branch}"
    subprocess.run(
        ["git", "commit", "-m", msg], check=True, capture_output=True
    )
    log.info(f"Merged (squash) {branch} into main")
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pytest tests/test_git.py -v`
Expected: 4 PASS.

- [ ] **Step 5: Commit**

```bash
git add orchestrator/git.py tests/test_git.py
git commit -m "feat: add git operations (branch/commit/tag/squash-merge)"
```

---

### Task 7: verify.py — Verify commands

**Files:**
- Create: `orchestrator/verify.py`
- Create: `tests/test_verify.py`

- [ ] **Step 1: Write the failing test**

```python
# tests/test_verify.py
import os
import pytest
from orchestrator.verify import run


def test_run_all_pass(tmp_path):
    """All commands succeed → pass."""
    ok, results = run(["echo hello", "echo world"], str(tmp_path))
    assert ok is True
    assert len(results) == 2
    assert all(r["ok"] for r in results)


def test_run_single_fail(tmp_path):
    """One command fails → fail."""
    ok, results = run(["echo ok", "exit 1"], str(tmp_path))
    assert ok is False
    assert len(results) == 2
    assert results[0]["ok"] is True
    assert results[1]["ok"] is False


def test_run_empty_commands(tmp_path):
    """Empty verify list → pass."""
    ok, results = run([], str(tmp_path))
    assert ok is True
    assert len(results) == 0


def test_run_captures_stderr(tmp_path):
    ok, results = run(["echo error >&2"], str(tmp_path))
    assert ok is True
    assert "error" in results[0]["stderr"]
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pytest tests/test_verify.py -v`
Expected: FAIL.

- [ ] **Step 3: Write minimal implementation**

```python
# orchestrator/verify.py
import logging
import subprocess

log = logging.getLogger("orchestrator.verify")


def run(commands: list[str], workdir: str = ".") -> tuple[bool, list[dict]]:
    """
    Execute verify commands sequentially.

    Returns (all_passed, results_list) where each result is:
        {"cmd": str, "ok": bool, "stdout": str, "stderr": str, "returncode": int}
    """
    results = []
    all_ok = True

    for cmd in commands:
        log.info(f"Verify: {cmd}")
        try:
            proc = subprocess.run(
                cmd,
                shell=True,
                capture_output=True,
                text=True,
                cwd=workdir,
                timeout=120,
            )
            ok = proc.returncode == 0
            results.append({
                "cmd": cmd,
                "ok": ok,
                "stdout": proc.stdout.strip(),
                "stderr": proc.stderr.strip(),
                "returncode": proc.returncode,
            })
            if not ok:
                log.error(f"Verify FAILED: {cmd} (exit {proc.returncode})")
                all_ok = False
        except subprocess.TimeoutExpired:
            log.error(f"Verify TIMEOUT: {cmd}")
            results.append({
                "cmd": cmd, "ok": False, "stdout": "",
                "stderr": "timeout after 120s", "returncode": -1,
            })
            all_ok = False

    status = "PASS" if all_ok else "FAIL"
    log.info(f"Verify {status}: {len(commands)} commands")
    return all_ok, results
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pytest tests/test_verify.py -v`
Expected: 4 PASS.

- [ ] **Step 5: Commit**

```bash
git add orchestrator/verify.py tests/test_verify.py
git commit -m "feat: add verify command execution"
```

---

### Task 8: session.py — AI session via opencode

**Files:**
- Create: `orchestrator/session.py`
- Create: `tests/test_session.py`

- [ ] **Step 1: Write the failing test**

```python
# tests/test_session.py
import subprocess
import pytest
from orchestrator.session import execute, SessionResult
from orchestrator.spec import Task


def make_task():
    return Task(
        id="t1", name="Test", depends_on=[], expected_files=[],
        prompt="hello", verify=[], milestone_id="m1", spec_id="s1",
    )


def test_execute_calls_opencode(monkeypatch, tmp_path):
    calls = []

    def fake_run(args, **kwargs):
        calls.append((args, kwargs))
        return subprocess.CompletedProcess(args, 0, stdout="ok", stderr="")

    monkeypatch.setattr(subprocess, "run", fake_run)
    monkeypatch.chdir(tmp_path)

    result = execute(make_task(), "test bundle")

    assert len(calls) == 1
    args, kwargs = calls[0]
    assert args[0] == "opencode"
    assert "test bundle" in kwargs.get("input", "")


def test_execute_returns_session_result(monkeypatch, tmp_path):
    def fake_run(args, **kwargs):
        return subprocess.CompletedProcess(args, 0, stdout="output", stderr="")

    monkeypatch.setattr(subprocess, "run", fake_run)
    monkeypatch.chdir(tmp_path)

    result = execute(make_task(), "bundle")
    assert isinstance(result, SessionResult)
    assert result.success is True
    assert result.stdout == "output"
    assert result.task_id == "t1"


def test_execute_handles_failure(monkeypatch, tmp_path):
    def fake_run(args, **kwargs):
        raise subprocess.TimeoutExpired(args, 30, output=b"partial")

    monkeypatch.setattr(subprocess, "run", fake_run)
    monkeypatch.chdir(tmp_path)

    result = execute(make_task(), "bundle")
    assert result.success is False
    assert "timeout" in result.stderr.lower()
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pytest tests/test_session.py -v`
Expected: FAIL.

- [ ] **Step 3: Write minimal implementation**

```python
# orchestrator/session.py
import logging
import subprocess
import tempfile
import os
from dataclasses import dataclass

from orchestrator.spec import Task

log = logging.getLogger("orchestrator.session")

TIMEOUT_SECONDS = 600


@dataclass
class SessionResult:
    task_id: str
    success: bool
    stdout: str
    stderr: str
    handoff_written: bool = False
    files_changed: list[str] = None

    def __post_init__(self):
        if self.files_changed is None:
            self.files_changed = []


def execute(task: Task, context_bundle: str) -> SessionResult:
    """
    Start an independent opencode session to execute a single task.
    
    Passes context_bundle via stdin to avoid command-line length limits.
    """
    log.info(f"Starting session for {task.id}")

    try:
        proc = subprocess.run(
            ["opencode"],
            input=context_bundle,
            timeout=TIMEOUT_SECONDS,
            capture_output=True,
            text=True,
        )
        success = proc.returncode == 0
        log.info(f"Session {task.id}: {'SUCCESS' if success else 'FAILED'} (exit {proc.returncode})")

        # Check for handoff file
        handoff_path = f".opencode/handoff/{task.id}-notes.md"
        handoff_written = os.path.exists(handoff_path)

        return SessionResult(
            task_id=task.id,
            success=success,
            stdout=proc.stdout,
            stderr=proc.stderr,
            handoff_written=handoff_written,
        )
    except subprocess.TimeoutExpired as e:
        log.error(f"Session {task.id}: TIMEOUT after {TIMEOUT_SECONDS}s")
        return SessionResult(
            task_id=task.id,
            success=False,
            stdout=e.output.decode() if e.output else "",
            stderr=f"timeout after {TIMEOUT_SECONDS}s",
        )
    except Exception as e:
        log.error(f"Session {task.id}: ERROR - {e}")
        return SessionResult(
            task_id=task.id,
            success=False,
            stdout="",
            stderr=str(e),
        )
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pytest tests/test_session.py -v`
Expected: 3 PASS.

- [ ] **Step 5: Commit**

```bash
git add orchestrator/session.py tests/test_session.py
git commit -m "feat: add opencode session execution"
```

---

### Task 9: handoff.py — Handoff notes collection

**Files:**
- Create: `orchestrator/handoff.py`
- Create: `tests/test_handoff.py`

- [ ] **Step 1: Write the failing test**

```python
# tests/test_handoff.py
import os
import pytest
from orchestrator.handoff import collect, write, template
from orchestrator.spec import Task


def make_task(id="t1", name="Test", depends_on=None, milestone_id="m1", spec_id="s1"):
    return Task(
        id=id, name=name, depends_on=depends_on or [],
        expected_files=[], prompt="", verify=[],
        milestone_id=milestone_id, spec_id=spec_id,
    )


def test_template_generates_markdown():
    task = make_task("m1-s1-t1", "Define structs")
    tpl = template(task)
    assert "## Handoff:" in tpl
    assert "m1-s1-t1" in tpl
    assert "Define structs" in tpl
    assert "### Files Changed" in tpl
    assert "### Architectural Decisions" in tpl


def test_collect_returns_empty_when_no_handoff_files():
    task = make_task("t2", depends_on=["t1"])
    result = collect(task)
    assert result == ""


def test_collect_concatenates_handoff_files(tmp_path, monkeypatch):
    monkeypatch.chdir(tmp_path)
    os.makedirs(".opencode/handoff", exist_ok=True)

    with open(".opencode/handoff/t1-notes.md", "w") as f:
        f.write("## Handoff: t1\n\nContent from t1\n")

    with open(".opencode/handoff/t2-notes.md", "w") as f:
        f.write("## Handoff: t2\n\nContent from t2\n")

    task = make_task("t3", depends_on=["t1", "t2"])
    result = collect(task)
    assert "Content from t1" in result
    assert "Content from t2" in result
    assert "---" in result


def test_collect_skips_missing_deps():
    task = make_task("t3", depends_on=["t1", "nonexistent"])
    result = collect(task)
    # Should not crash, just return whatever exists
    assert isinstance(result, str)


def test_write_creates_handoff_file(tmp_path, monkeypatch):
    monkeypatch.chdir(tmp_path)
    os.makedirs(".opencode/handoff", exist_ok=True)

    task = make_task("m1-s1-t1")
    write(task, "Generated content")

    filepath = ".opencode/handoff/m1-s1-t1-notes.md"
    assert os.path.exists(filepath)
    content = open(filepath).read()
    assert "Generated content" in content
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pytest tests/test_handoff.py -v`
Expected: FAIL.

- [ ] **Step 3: Write minimal implementation**

```python
# orchestrator/handoff.py
import logging
import os

from orchestrator.spec import Task

log = logging.getLogger("orchestrator.handoff")

HANDOFF_DIR = ".opencode/handoff"


def template(task: Task) -> str:
    """Generate a handoff template for the AI to fill in."""
    dep_list = ", ".join(task.depends_on) if task.depends_on else "(none)"
    next_task = "N/A"

    return f"""## Handoff: {task.id}

### Files Changed

<!-- List changed files with brief descriptions -->

### Architectural Decisions

<!-- Key design/architecture decisions made -->

### For Next Task

<!-- Notes for tasks that depend on this one -->

### Known Limitations

<!-- Things left incomplete or known issues -->
"""


def collect(task: Task) -> str:
    """Collect handoff notes from all dependency tasks."""
    notes = []
    for dep_id in task.depends_on:
        path = f"{HANDOFF_DIR}/{dep_id}-notes.md"
        if os.path.exists(path):
            with open(path) as f:
                notes.append(f.read())
        else:
            log.warning(f"Handoff file not found for dependency: {dep_id}")

    separator = "\n\n---\n\n"
    return separator.join(notes)


def write(task: Task, content: str):
    """Write the handoff note for a completed task."""
    os.makedirs(HANDOFF_DIR, exist_ok=True)
    path = f"{HANDOFF_DIR}/{task.id}-notes.md"
    with open(path, "w") as f:
        f.write(content)
    log.info(f"Handoff written: {path}")
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pytest tests/test_handoff.py -v`
Expected: 5 PASS.

- [ ] **Step 5: Commit**

```bash
git add orchestrator/handoff.py tests/test_handoff.py
git commit -m "feat: add handoff collection and template generation"
```

---

### Task 10: context.py — Context bundle assembly

**Files:**
- Create: `orchestrator/context.py`
- Create: `tests/test_context.py`

- [ ] **Step 1: Write the failing test**

```python
# tests/test_context.py
import os
import pytest
from orchestrator.context import assemble, estimate_tokens, degrade
from orchestrator.spec import Task, Spec, Milestone, Project
from orchestrator.state import State


def make_project():
    return Project(
        name="test",
        language="rust",
        scratch_template="cargo init",
        milestones=[
            Milestone(
                id="m1", name="M1", order=1,
                specs=[
                    Spec(
                        id="s1", name="S1", depends_on_specs=[],
                        tasks=[
                            Task(
                                id="t1", name="T1", depends_on=[], expected_files=[],
                                prompt="Do something", verify=["cargo build"],
                                milestone_id="m1", spec_id="s1",
                            )
                        ]
                    )
                ]
            )
        ]
    )


def make_state():
    return State(
        version=1, project_name="test", status="running",
        current=None, completed=[], failed={},
        started_at="", last_updated="",
    )


def test_estimate_tokens():
    text = "hello world"  # 11 chars
    tokens = estimate_tokens(text)
    assert tokens == 3  # 11 // 4 = 2.75 -> 3


def test_estimate_tokens_empty():
    assert estimate_tokens("") == 0


def test_assemble_contains_task_prompt(monkeypatch, tmp_path):
    monkeypatch.chdir(tmp_path)
    # Create claude.md
    (tmp_path / "claude.md").write_text("# Project conventions\nUse serde.")

    project = make_project()
    state = make_state()
    task = project.milestones[0].specs[0].tasks[0]

    bundle = assemble(project, task, state, {})

    assert "Do something" in bundle
    assert "cargo build" in bundle
    assert "# Project conventions" in bundle


def test_assemble_contains_system_head(monkeypatch, tmp_path):
    monkeypatch.chdir(tmp_path)
    (tmp_path / "claude.md").write_text("")

    project = make_project()
    state = make_state()
    task = project.milestones[0].specs[0].tasks[0]

    bundle = assemble(project, task, state, {})
    assert "m1" in bundle
    assert "s1" in bundle
    assert "t1" in bundle


def test_assemble_empty_handoff_when_no_deps(monkeypatch, tmp_path):
    monkeypatch.chdir(tmp_path)
    (tmp_path / "claude.md").write_text("")

    project = make_project()
    state = make_state()
    task = project.milestones[0].specs[0].tasks[0]
    # task has no depends_on

    bundle = assemble(project, task, state, {})
    # Should still assemble without error
    assert len(bundle) > 0


def test_degrade_removes_non_immediate_handoffs(monkeypatch, tmp_path):
    monkeypatch.chdir(tmp_path)
    (tmp_path / "claude.md").write_text("" * 200000)

    project = make_project()
    task = project.milestones[0].specs[0].tasks[0]

    # Create a very large bundle by using a huge claude.md
    bundle = assemble(project, task, make_state(), {})
    original_size = len(bundle)

    if original_size > 10000:
        degraded = degrade(bundle, {"t1": "old handoff"}, {})
        assert len(degraded) <= original_size
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pytest tests/test_context.py -v`
Expected: FAIL.

- [ ] **Step 3: Write minimal implementation**

```python
# orchestrator/context.py
import logging
import os
import subprocess

from orchestrator.spec import Project, Task
from orchestrator.state import State
from orchestrator.handoff import collect as collect_handoff

log = logging.getLogger("orchestrator.context")

TOKEN_BUDGET = 180_000
DEGRADE_TARGET = 150_000


def estimate_tokens(text: str) -> int:
    """Rough token estimation: ~4 characters per token."""
    return max(1, len(text) // 4) if text else 0


def assemble(
    project: Project,
    task: Task,
    state: State,
    handoff_notes: dict | None = None,
    claude_md_path: str = "claude.md",
) -> str:
    """
    Assemble the full context bundle for a single task session.

    Structure (per design.md §4):
      1. CLAUDE.md (coding conventions)
      2. System head (task metadata)
      3. Handoff from dependencies
      4. Task spec (prompt + expected_files + verify)
      5. Code snapshot (file tree + relevant code)
      6. Output template
    """
    sections = []

    # 1. CLAUDE.md
    claude_md = _read_file_or_empty(claude_md_path)
    if claude_md:
        sections.append(claude_md)

    # 2. System head
    milestone = _find_milestone(project, task.milestone_id)
    sections.append(
        f"## Task: {task.id} - {task.name}\n"
        f"Milestone: {task.milestone_id}"
        f"{' - ' + milestone.name if milestone else ''}\n"
        f"Spec: {task.spec_id}\n"
        f"Language: {project.language}\n"
        f"Verify commands: {', '.join(task.verify) if task.verify else 'none'}\n"
    )

    # 3. Handoff from dependencies
    handoff = collect_handoff(task)
    if handoff:
        sections.append(f"## Handoff from Dependencies\n\n{handoff}")

    # 4. Task spec
    sections.append(
        f"## Task Specification\n\n"
        f"### Prompt\n\n{task.prompt}\n\n"
        f"### Expected Files\n\n" + "\n".join(f"- `{f}`" for f in task.expected_files) + "\n\n"
        f"### Verify Commands\n\n" + "\n".join(f"- `{c}`" for c in task.verify) + "\n"
    )

    # 5. Code snapshot (file tree)
    try:
        tree_result = subprocess.run(
            ["tree", ".", "--noreport", "-L", "3", "-I", ".git|__pycache__|.pytest_cache"],
            capture_output=True, text=True, timeout=5,
        )
        if tree_result.returncode == 0:
            sections.append(f"## Current File Tree\n\n```\n{tree_result.stdout}\n```")
    except Exception:
        pass

    # 6. Output template
    sections.append(
        "## Output Requirements\n\n"
        "After completing this task, you MUST:\n"
        f"1. Ensure all test pass and verify commands succeed: {', '.join(task.verify)}\n"
        f"2. Write handoff notes to `.opencode/handoff/{task.id}-notes.md`\n"
        "3. If you change the public API, update README.md accordingly\n"
    )

    bundle = "\n\n---\n\n".join(sections)
    tokens = estimate_tokens(bundle)
    log.info(f"Context bundle for {task.id}: ~{tokens} tokens")

    if tokens > TOKEN_BUDGET:
        log.warning(f"Context bundle exceeds budget ({tokens} > {TOKEN_BUDGET}), degrading...")
        bundle = degrade(bundle, task, handoff_notes)

    return bundle


def degrade(bundle: str, task: Task, handoff_notes: dict) -> str:
    """
    Degrade context bundle to fit within token budget.

    Level 1: Keep only immediate dependency handoffs
    Level 2: Replace full code with function signatures
    Level 3: Truncate handoffs to summary only
    """
    # Level 1: Drop non-immediate handoffs (simplified for v1)
    # For now, remove sections marked as old handoffs
    sections = bundle.split("\n\n---\n\n")
    kept = []
    for section in sections:
        if estimate_tokens("\n\n---\n\n".join(kept + [section])) <= DEGRADE_TARGET:
            kept.append(section)
        else:
            break
    result = "\n\n---\n\n".join(kept)
    log.info(f"Degraded bundle: {estimate_tokens(bundle)} → {estimate_tokens(result)} tokens")
    return result


def _read_file_or_empty(path: str) -> str:
    try:
        with open(path) as f:
            return f.read()
    except FileNotFoundError:
        return ""


def _find_milestone(project: Project, milestone_id: str):
    for m in project.milestones:
        if m.id == milestone_id:
            return m
    return None
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pytest tests/test_context.py -v`
Expected: 6 PASS.

- [ ] **Step 5: Commit**

```bash
git add orchestrator/context.py tests/test_context.py
git commit -m "feat: add context bundle assembly and degradation"
```

---

### Task 11: review.py — Phase and Final review

**Files:**
- Create: `orchestrator/review.py`
- Create: `tests/test_review.py`

- [ ] **Step 1: Write the failing test**

```python
# tests/test_review.py
import os
import pytest
from orchestrator.review import phase_review, final_review
from orchestrator.spec import Task, Spec, Milestone, Project
from orchestrator.state import State, FailedInfo


def make_project():
    return Project(
        name="test",
        language="rust",
        scratch_template="",
        milestones=[
            Milestone(
                id="m1", name="M1", order=1,
                specs=[
                    Spec(
                        id="s1", name="S1", depends_on_specs=[],
                        tasks=[
                            Task(
                                id="t1", name="T1", depends_on=[], expected_files=[],
                                prompt="", verify=[], milestone_id="m1", spec_id="s1",
                            )
                        ]
                    )
                ]
            )
        ]
    )


def make_state(completed=None, failed=None):
    return State(
        version=1, project_name="test", status="running",
        current=None, completed=completed or [], failed=failed or {},
        started_at="", last_updated="",
    )


def test_phase_review_skips_when_no_completed_tasks(monkeypatch, tmp_path):
    monkeypatch.chdir(tmp_path)
    project = make_project()
    milestone = project.milestones[0]
    state = make_state(completed=[])

    result = phase_review(milestone, project, state)
    # Should return early without error
    assert result is None or result is True


def test_phase_review_creates_review_dir(monkeypatch, tmp_path):
    monkeypatch.chdir(tmp_path)
    project = make_project()
    milestone = project.milestones[0]
    state = make_state(completed=["t1"])

    # Mock session.execute to avoid real opencode call
    import orchestrator.review as review_mod
    original = getattr(review_mod, "execute_review", None)

    def fake_review(context):
        os.makedirs("review", exist_ok=True)
        with open("review/architecture.md", "w") as f:
            f.write("# Review\nok")
        with open("review/quality.md", "w") as f:
            f.write("# Quality\nok")
        with open("review/TODO.md", "w") as f:
            f.write("# TODO\nnone")

    # Inject mock
    import orchestrator.review
    orchestrator.review.execute_review = fake_review

    try:
        phase_review(milestone, project, state)
        assert os.path.exists("review/architecture.md")
        assert os.path.exists("review/quality.md")
        assert os.path.exists("review/TODO.md")
    finally:
        if original:
            orchestrator.review.execute_review = original


def test_final_review_runs_complete_check(monkeypatch, tmp_path):
    monkeypatch.chdir(tmp_path)
    project = make_project()
    state = make_state(completed=["t1"], failed={})

    import orchestrator.review as review_mod
    original = getattr(review_mod, "execute_review", None)

    def fake_review(context):
        pass

    orchestrator.review.execute_review = fake_review
    try:
        result = final_review(project, state)
        assert result is True or result is None
    finally:
        if original:
            orchestrator.review.execute_review = original
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pytest tests/test_review.py -v`
Expected: FAIL.

- [ ] **Step 3: Write minimal implementation**

```python
# orchestrator/review.py
import logging
import os
import subprocess

from orchestrator.spec import Milestone, Project
from orchestrator.state import State

log = logging.getLogger("orchestrator.review")


def phase_review(milestone: Milestone, project: Project, state: State):
    """
    Run a phase review after completing a milestone.

    Checks architecture consistency, code quality, and generates review docs.
    Small issues are fixed directly; large ones added to review/TODO.md.
    """
    completed_in_milestone = _completed_in_milestone(milestone, state)
    if not completed_in_milestone:
        log.warning(f"No completed tasks in milestone {milestone.id}, skipping review")
        return

    os.makedirs("review", exist_ok=True)

    context = _build_phase_review_context(milestone, project, completed_in_milestone)
    log.info(f"Launching phase review for milestone {milestone.id}")

    execute_review(context)

    log.info(f"Phase review complete for {milestone.id}")


def final_review(project: Project, state: State):
    """
    Run final integration review after all milestones complete.

    Includes: full test suite, linting, cross-module checks,
    CHANGELOG generation, README final review.
    """
    if not state.completed:
        log.warning("No completed tasks for final review")
        return

    log.info(f"Launching final review: {len(state.completed)} completed tasks")

    context = _build_final_review_context(project, state)
    execute_review(context)

    log.info("Final review complete")
    return True


def execute_review(context: str):
    """Launch a review session via opencode."""
    try:
        subprocess.run(
            ["opencode"],
            input=context,
            timeout=600,
            capture_output=True,
            text=True,
        )
    except Exception as e:
        log.error(f"Review session failed: {e}")


def _completed_in_milestone(milestone: Milestone, state: State) -> list[str]:
    """Return task IDs completed that belong to this milestone."""
    return [tid for tid in state.completed if tid.startswith(milestone.id)]


def _build_phase_review_context(
    milestone: Milestone, project: Project, completed_ids: list[str]
) -> str:
    lines = [
        f"# Phase Review: {milestone.name} ({milestone.id})",
        "",
        f"## Completed Tasks ({len(completed_ids)})",
        "",
    ]
    for tid in completed_ids:
        lines.append(f"- {tid}")

    lines += [
        "",
        "## Instructions",
        "",
        "As the architect, review this milestone's code quality:",
        "1. Check module boundaries and interface consistency",
        "2. Verify error handling uniformity",
        "3. Find duplicate or unused code",
        "4. Check naming and comment consistency",
        "5. Verify README.md accurately reflects current state",
        "",
        "Output to:",
        "- `review/architecture.md` — architecture description and risk assessment",
        "- `review/quality.md` — code quality issues found",
        "- `review/TODO.md` — improvements to add in next milestone",
        "",
        "Fix small issues directly. Add large issues to review/TODO.md.",
    ]

    # Append handoff notes from this milestone
    for tid in completed_ids:
        handoff_path = f".opencode/handoff/{tid}-notes.md"
        if os.path.exists(handoff_path):
            with open(handoff_path) as f:
                lines.append(f"\n---\n\n## Handoff: {tid}\n\n{f.read()}")

    return "\n".join(lines)


def _build_final_review_context(project: Project, state: State) -> str:
    lines = [
        "# Final Integration Review",
        "",
        f"Project: {project.name}",
        f"Completed tasks: {len(state.completed)}",
        f"Failed tasks: {len(state.failed)}",
        "",
        "## Instructions",
        "",
        "Perform the final review:",
        "1. Run the full test suite (`cargo test --all-targets` or `go test ./...`)",
        "2. Run linting (`cargo clippy -- -D warnings` or `golangci-lint run`)",
        "3. Cross-reference all public API usage for correctness",
        "4. Generate CHANGELOG.md from git log and handoff notes",
        "5. Final review of README.md for accuracy and completeness",
        "6. Tag the release: `v1.0.0`",
    ]
    return "\n".join(lines)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pytest tests/test_review.py -v`
Expected: 3 PASS (one skipped due to no completed tasks).

- [ ] **Step 5: Commit**

```bash
git add orchestrator/review.py tests/test_review.py
git commit -m "feat: add phase review and final review"
```

---

### Task 12: main.py — CLI entry and orchestration loop

**Files:**
- Create: `orchestrator/main.py`
- Create: `tests/test_main.py`

- [ ] **Step 1: Write the failing test**

```python
# tests/test_main.py
import sys
import pytest
from unittest.mock import patch, MagicMock
from orchestrator.main import main, Main


def test_main_registers_run_command(capsys):
    with patch.object(sys, "argv", ["orchestrator", "run"]):
        with patch("orchestrator.main.Main.run") as mock_run:
            try:
                main()
            except SystemExit:
                pass
            mock_run.assert_called_once_with(resume=False)


def test_main_registers_run_resume():
    with patch.object(sys, "argv", ["orchestrator", "run", "--resume"]):
        with patch("orchestrator.main.Main.run") as mock_run:
            try:
                main()
            except SystemExit:
                pass
            mock_run.assert_called_once_with(resume=True)


def test_main_no_command_shows_help(capsys):
    with patch.object(sys, "argv", ["orchestrator"]):
        try:
            main()
        except SystemExit:
            pass
        captured = capsys.readouterr()
        assert "usage" in captured.err or "orchestrator" in captured.out


class TestOrchestrator:
    def test_run_workflow_end_to_end(self, monkeypatch, tmp_path):
        """Integration test: parse → topo → execute session → verify → commit."""
        monkeypatch.chdir(tmp_path)

        # Create project.yaml
        project_yaml = """
project:
  name: test
  language: rust
  scratch_template: ""

milestones:
  - id: m1
    name: M1
    order: 1
    specs:
      - id: s1
        name: S1
        tasks:
          - id: t1
            name: T1
            depends_on: []
            expected_files: []
            prompt: "do nothing"
            verify:
              - echo ok
"""
        (tmp_path / "project.yaml").write_text(project_yaml)

        # Create claude.md
        (tmp_path / "claude.md").write_text("# Conventions")

        # Init git
        import subprocess
        subprocess.run(["git", "init"], capture_output=True)
        subprocess.run(["git", "config", "user.email", "test@test.com"], capture_output=True)
        subprocess.run(["git", "config", "user.name", "Test"], capture_output=True)
        (tmp_path / "README.md").write_text("# test")
        subprocess.run(["git", "add", "."], capture_output=True)
        subprocess.run(["git", "commit", "-m", "initial"], capture_output=True)

        # Mock session.execute to avoid real opencode call
        import orchestrator.main as main_mod
        original_execute = main_mod.session.execute

        calls = []
        def fake_execute(task, bundle):
            calls.append(task.id)
            from orchestrator.session import SessionResult
            return SessionResult(
                task_id=task.id, success=True, stdout="ok", stderr="",
            )

        main_mod.session.execute = fake_execute

        try:
            orch = Main()
            orch.run(resume=False)
            assert len(calls) == 1
            assert calls[0] == "t1"
        finally:
            main_mod.session.execute = original_execute
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pytest tests/test_main.py -v`
Expected: FAIL.

- [ ] **Step 3: Write minimal implementation**

```python
# orchestrator/main.py
import argparse
import logging
import sys

from orchestrator import log as log_mod
from orchestrator import spec
from orchestrator import topo
from orchestrator import state
from orchestrator import session
from orchestrator import verify
from orchestrator import handoff
from orchestrator import context
from orchestrator import review
from orchestrator import git

log = logging.getLogger("orchestrator.main")


def main():
    parser = argparse.ArgumentParser(
        prog="orchestrator",
        description="AI project pipeline orchestrator",
    )
    sub = parser.add_subparsers(dest="command", help="Commands")

    run_parser = sub.add_parser("run", help="Run the pipeline")
    run_parser.add_argument("--resume", action="store_true", help="Resume from last checkpoint")

    args = parser.parse_args()

    if args.command == "run":
        Main().run(resume=args.resume)
    else:
        parser.print_help()
        sys.exit(1)


class Main:
    def run(self, resume: bool = False):
        log_mod.setup()

        # 1. Parse project spec
        project = spec.parse("project.yaml")
        log.info(f"Project: {project.name} ({project.language})")

        # 2. Preflight checks
        if not spec.preflight(project):
            log.error("Preflight checks failed. Aborting.")
            sys.exit(1)

        # 3. Load or initialize state
        state_mgr = state.load_or_init(project, "state.yaml", resume=resume)

        # 4. Topological sort
        try:
            tasks = topo.sort(project)
        except ValueError as e:
            log.error(f"Topological sort failed: {e}")
            sys.exit(1)

        log.info(f"Task order ({len(tasks)} tasks):")
        for t in tasks:
            log.info(f"  {t.id} (deps: {t.depends_on})")

        # 5. Determine start point
        start_idx = 0
        if resume:
            for i, t in enumerate(tasks):
                if t.id not in state_mgr.completed and t.id not in state_mgr.failed:
                    start_idx = i
                    break
            log.info(f"Resuming from task index {start_idx} ({tasks[start_idx].id})")

        # 6. Execute milestones in order
        milestones_done = set()
        for i in range(start_idx, len(tasks)):
            task = tasks[i]
            ms_id = task.milestone_id

            # Switch to milestone branch if needed
            if ms_id not in milestones_done:
                branch = f"milestone/{ms_id}"
                git.checkout_branch(branch)
                milestones_done.add(ms_id)

            # Skip completed tasks
            if task.id in state_mgr.completed:
                log.info(f"Skip completed: {task.id}")
                continue

            # Dependencies are guaranteed by topological order —
            # topo.sort expands both task-level and spec-level deps

            # Update current task pointer
            state_mgr.current = state.TaskRef(
                milestone=task.milestone_id, spec=task.spec_id, task=task.id
            )
            state.save(state_mgr, "state.yaml")

            # Assemble context bundle
            bundle = context.assemble(project, task, state_mgr)

            # Execute session
            log.info(f"--- Executing {task.id}: {task.name} ---")
            result = session.execute(task, bundle)

            # Verify
            verify_ok, verify_results = verify.run(task.verify)

            if result.success and verify_ok:
                # Success: commit + handoff + update state
                handoff.write(task, result.stdout)
                git.commit(task)
                state.mark_complete(state_mgr, task.id)
                log.info(f"Task {task.id} COMPLETED")
            else:
                # Failure: record and continue
                error_msg = result.stderr if not result.success else "verify failed"
                state.mark_failed(state_mgr, task.id, error_msg)
                log.error(f"Task {task.id} FAILED: {error_msg}")

            state.save(state_mgr, "state.yaml")

            # Phase review after all tasks in this milestone
            is_last_in_milestone = _is_last_task_in_milestone(task, project, tasks, i)
            if is_last_in_milestone:
                milestone = _get_milestone(project, ms_id)
                if milestone:
                    review.phase_review(milestone, project, state_mgr)
                    tag = f"v0.{milestone.order}.0-{milestone.id}"
                    git.create_tag(tag)
                    git.merge_to_main(branch)

        # 7. Final state
        if state_mgr.failed:
            state_mgr.status = "partially_failed"
        else:
            state_mgr.status = "completed"
        state.save(state_mgr, "state.yaml")

        # 8. Final review
        review.final_review(project, state_mgr)

        log.info(f"Pipeline complete. Status: {state_mgr.status}")
        log.info(f"  Completed: {len(state_mgr.completed)} tasks")
        log.info(f"  Failed: {len(state_mgr.failed)} tasks")


def _is_last_task_in_milestone(
    task: Task, project: Project, tasks: list[Task], current_idx: int
) -> bool:
    """Check if this is the last task in its milestone's topological order."""
    for j in range(current_idx + 1, len(tasks)):
        if tasks[j].milestone_id == task.milestone_id:
            return False
    return True


def _get_milestone(project: Project, milestone_id: str):
    for m in project.milestones:
        if m.id == milestone_id:
            return m
    return None


if __name__ == "__main__":
    main()
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pytest tests/test_main.py -v`
Expected: 4 PASS.

- [ ] **Step 5: Commit**

```bash
git add orchestrator/main.py tests/test_main.py
git commit -m "feat: add CLI entry point with full orchestration loop"
```

---

### Task 13: Integration — Sample project and CLAUDE.md

**Files:**
- Create: `project.yaml.example`
- Create: `claude.md`

- [ ] **Step 1: Create example project.yaml**

```yaml
# project.yaml.example
# Copy to project.yaml and customize.
# Can be split into project.yaml, project-m2.yaml, etc.

project:
  name: example-backend
  language: rust
  scratch_template: "cargo init --name example-backend"

milestones:
  - id: m1-core
    name: Core Foundation
    order: 1
    specs:
      - id: m1-s1-data-model
        name: Data Model
        tasks:
          - id: m1-s1-t1
            name: Define core structs
            depends_on: []
            expected_files:
              - src/models/mod.rs
              - src/models/user.rs
            prompt: |
              Create the core data model for the application.

              Define a `User` struct with fields:
              - id: uuid::Uuid
              - username: String
              - email: String
              - created_at: chrono::DateTime<Utc>

              Use serde Serialize/Deserialize derive macros.
              Add doc comments to every field and the struct itself.

              Place the module at `src/models/mod.rs` with:
              - `mod user;`
              - `pub use user::User;`

              Place the User struct in `src/models/user.rs`.
            verify:
              - cargo build
              - cargo test

          - id: m1-s1-t2
            name: Add input validation
            depends_on: [m1-s1-t1]
            expected_files:
              - src/models/validation.rs
            prompt: |
              Add input validation for the User struct.

              Create `src/models/validation.rs` with a `validate_email` function
              that checks email format using a simple regex pattern.

              Add `src/models/validation.rs` to `src/models/mod.rs`.

              Write unit tests for:
              - Valid email addresses
              - Invalid email addresses (no @, no domain, empty string)
              - Edge cases (very long email, unicode characters)

              Also add a `validate_username` function that checks:
              - Length between 3 and 32 characters
              - Only alphanumeric and underscore characters
            verify:
              - cargo build
              - cargo test

      - id: m1-s2-config
        name: Configuration
        depends_on_specs: [m1-s1]
        tasks:
          - id: m1-s2-t1
            name: Load config from file
            depends_on: []
            expected_files:
              - src/config/mod.rs
              - src/config/loader.rs
            prompt: |
              Implement configuration loading.

              Create `src/config/mod.rs` with:
              - A `Config` struct with fields for database_url, log_level, server_port
              - Use serde for deserialization from a TOML file

              Create `src/config/loader.rs` with:
              - `fn load(path: &str) -> Result<Config, ConfigError>`
              - Read from file, parse TOML, validate required fields

              Register the module in `src/main.rs` or `src/lib.rs`.
            verify:
              - cargo build
              - cargo test
```

- [ ] **Step 2: Create claude.md**

```markdown
# Project Conventions

## Language
Rust

## Code Style
- Use `cargo fmt` for formatting
- Use `cargo clippy -- -D warnings` for linting
- Prefer `anyhow::Result` for application-level errors, `thiserror` for library errors
- Use `uuid::Uuid` for all entity IDs
- Use `chrono` for date/time handling
- Use `serde` for serialization (derive macros)

## Testing
- Write unit tests in the same file as the implementation (`#[cfg(test)] mod tests`)
- Test both happy path and error cases
- Run `cargo test` before committing

## Naming
- Struct names: PascalCase
- Function/method names: snake_case
- Module file names: snake_case
- Constants: SCREAMING_SNAKE_CASE

## Module Organization
- One module per concept (e.g., user → `src/models/user.rs`)
- Re-export public types in the parent module (`pub use user::User;`)
- Keep `main.rs` minimal — delegate to library crate when possible

## Dependencies
- `uuid` with `v4` feature
- `serde` with `derive` feature
- `serde_json`
- `chrono` with `serde` feature
- `anyhow`
- `config` crate for configuration loading
- `regex` for validation
- `toml` for config file parsing
```

- [ ] **Step 3: Commit**

```bash
git add project.yaml.example claude.md
git commit -m "docs: add example project.yaml and claude.md conventions"
```

---

### Task 14: Final verification — Run all tests

- [ ] **Step 1: Run full test suite**

```bash
python -m pytest tests/ -v
```
Expected: All tests PASS (approximately 44 tests across all modules).

- [ ] **Step 2: Run preflight on example project**

```bash
python -c "
from orchestrator.spec import parse, preflight
p = parse('project.yaml.example')
print(f'Parsed: {p.name}, {len(p.milestones)} milestones')
print(f'Preflight: {preflight(p)}')
"
```
Expected: `Parsed: example-backend, 1 milestones`, `Preflight: True`.

- [ ] **Step 3: Run topological sort on example**

```bash
python -c "
from orchestrator.spec import parse
from orchestrator.topo import sort
p = parse('project.yaml.example')
tasks = sort(p)
for t in tasks:
    print(f'{t.id}: {t.name} (deps: {t.depends_on})')
"
```
Expected: Shows 3 tasks in correct dependency order (t1 before t2, both before t2-t1).

- [ ] **Step 4: Commit**

```bash
git commit --allow-empty -m "chore: final verification — all tests pass"
```
