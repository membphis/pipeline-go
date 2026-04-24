import pytest
import yaml
from orchestrator import spec, log


@pytest.fixture(autouse=True)
def _setup_log():
    log.setup()


@pytest.fixture
def minimal_yaml(tmp_path):
    data = {
        "project": {"name": "test-project"},
        "milestones": [
            {
                "name": "m1",
                "spec": "spec m1",
                "tasks": [{"name": "t1", "prompt": "do something"}],
            }
        ],
    }
    p = tmp_path / "project.yaml"
    with open(p, "w") as f:
        yaml.dump(data, f)
    return p


@pytest.fixture
def milestone_with_deps(tmp_path):
    data = {
        "project": {"name": "test-project"},
        "milestones": [
            {
                "name": "m1",
                "spec": "spec m1",
                "tasks": [{"name": "t1", "prompt": "do something"}],
            },
            {
                "name": "m2",
                "spec": "spec m2",
                "depends_on": ["m1"],
                "tasks": [{"name": "t2", "prompt": "do something else"}],
            },
        ],
    }
    p = tmp_path / "project.yaml"
    with open(p, "w") as f:
        yaml.dump(data, f)
    return p


def test_load_returns_project(minimal_yaml):
    project = spec.load(minimal_yaml)
    assert project["project"]["name"] == "test-project"


def test_load_includes_milestones(minimal_yaml):
    project = spec.load(minimal_yaml)
    ms = project["milestones"]
    assert len(ms) == 1
    assert ms[0]["name"] == "m1"
    assert ms[0]["spec"] == "spec m1"


def test_load_includes_tasks(minimal_yaml):
    project = spec.load(minimal_yaml)
    tasks = project["milestones"][0]["tasks"]
    assert len(tasks) == 1
    assert tasks[0]["name"] == "t1"
    assert tasks[0]["prompt"] == "do something"


def test_load_file_not_found():
    with pytest.raises(FileNotFoundError):
        spec.load("/nonexistent/project.yaml")


def test_load_invalid_yaml(tmp_path):
    p = tmp_path / "bad.yaml"
    p.write_text("{invalid: yaml: broken")
    with pytest.raises(yaml.YAMLError):
        spec.load(p)


def test_load_invalid_missing_project_key(tmp_path):
    p = tmp_path / "bad.yaml"
    yaml.dump({"milestones": []}, open(p, "w"))
    with pytest.raises(ValueError, match="missing required key"):
        spec.load(p)


def test_load_invalid_missing_milestones_key(tmp_path):
    p = tmp_path / "bad.yaml"
    yaml.dump({"project": {"name": "x"}}, open(p, "w"))
    with pytest.raises(ValueError, match="missing required key"):
        spec.load(p)


def test_get_milestone_returns_correct(minimal_yaml):
    project = spec.load(minimal_yaml)
    ms = spec.get_milestone(project, "m1")
    assert ms is not None
    assert ms["name"] == "m1"


def test_get_milestone_returns_none(minimal_yaml):
    project = spec.load(minimal_yaml)
    assert spec.get_milestone(project, "nonexistent") is None


def test_get_milestones_by_spec_yaml(tmp_path):
    data1 = {
        "project": {"name": "test-project"},
        "milestones": [
            {
                "name": "m1",
                "spec": "spec m1",
                "tasks": [{"name": "t1", "prompt": "do something"}],
            }
        ],
    }
    data2 = {
        "milestones": [
            {
                "name": "m2",
                "spec": "spec m2",
                "depends_on": ["m1"],
                "tasks": [{"name": "t2", "prompt": "do something else"}],
            }
        ],
    }
    p1 = tmp_path / "project.yaml"
    p2 = tmp_path / "project.extra.yaml"
    with open(p1, "w") as f:
        yaml.dump(data1, f)
    with open(p2, "w") as f:
        yaml.dump(data2, f)
    project = spec.load(p1, extra_specs=[p2])
    assert len(project["milestones"]) == 2


def test_preflight_success(minimal_yaml):
    project = spec.load(minimal_yaml)
    errors = spec.preflight(project)
    assert errors == []


def test_preflight_duplicate_milestone(minimal_yaml):
    project = spec.load(minimal_yaml)
    project["milestones"].append(
        {"name": "m1", "spec": "dup", "tasks": [{"name": "t2", "prompt": "p"}]}
    )
    errors = spec.preflight(project)
    assert any("duplicate milestone" in e.lower() for e in errors)


def test_preflight_missing_dep(tmp_path):
    data = {
        "project": {"name": "test-project"},
        "milestones": [
            {
                "name": "m2",
                "spec": "spec m2",
                "depends_on": ["nonexistent"],
                "tasks": [{"name": "t2", "prompt": "do something"}],
            }
        ],
    }
    p = tmp_path / "project.yaml"
    yaml.dump(data, open(p, "w"))
    project = spec.load(p)
    errors = spec.preflight(project)
    assert any("unknown dependency" in e.lower() for e in errors)


def test_preflight_empty_tasks(tmp_path):
    data = {
        "project": {"name": "test-project"},
        "milestones": [
            {"name": "m1", "spec": "spec m1", "tasks": []}
        ],
    }
    p = tmp_path / "project.yaml"
    yaml.dump(data, open(p, "w"))
    project = spec.load(p)
    errors = spec.preflight(project)
    assert any("no tasks" in e.lower() or "empty" in e.lower() for e in errors)


def test_preflight_cyclical_dependency(tmp_path):
    data = {
        "project": {"name": "test-project"},
        "milestones": [
            {
                "name": "m1",
                "spec": "spec m1",
                "depends_on": ["m2"],
                "tasks": [{"name": "t1", "prompt": "do something"}],
            },
            {
                "name": "m2",
                "spec": "spec m2",
                "depends_on": ["m1"],
                "tasks": [{"name": "t2", "prompt": "do something else"}],
            },
        ],
    }
    p = tmp_path / "project.yaml"
    yaml.dump(data, open(p, "w"))
    project = spec.load(p)
    errors = spec.preflight(project)
    assert any("cycle" in e.lower() or "circular" in e.lower() for e in errors)
