import pytest
from orchestrator import topo


def test_empty_milestones():
    result = topo.sort([])
    assert result == []


def test_single_milestone():
    ms = [{"name": "m1", "tasks": [{"prompt": "p"}]}]
    result = topo.sort(ms)
    assert result == ["m1"]


def test_two_independent():
    ms = [
        {"name": "m1", "tasks": [{"prompt": "p"}]},
        {"name": "m2", "tasks": [{"prompt": "p"}]},
    ]
    result = topo.sort(ms)
    assert set(result) == {"m1", "m2"}
    assert len(result) == 2


def test_simple_chain():
    ms = [
        {"name": "m1", "tasks": [{"prompt": "p"}]},
        {"name": "m2", "depends_on": ["m1"], "tasks": [{"prompt": "p"}]},
        {"name": "m3", "depends_on": ["m2"], "tasks": [{"prompt": "p"}]},
    ]
    result = topo.sort(ms)
    assert result.index("m1") < result.index("m2")
    assert result.index("m2") < result.index("m3")


def test_diamond():
    ms = [
        {"name": "m1", "tasks": [{"prompt": "p"}]},
        {"name": "m2", "depends_on": ["m1"], "tasks": [{"prompt": "p"}]},
        {"name": "m3", "depends_on": ["m1"], "tasks": [{"prompt": "p"}]},
        {"name": "m4", "depends_on": ["m2", "m3"], "tasks": [{"prompt": "p"}]},
    ]
    result = topo.sort(ms)
    assert result.index("m1") < result.index("m2")
    assert result.index("m1") < result.index("m3")
    assert result.index("m2") < result.index("m4")
    assert result.index("m3") < result.index("m4")


def test_cycle_detected():
    ms = [
        {"name": "m1", "depends_on": ["m2"], "tasks": [{"prompt": "p"}]},
        {"name": "m2", "depends_on": ["m1"], "tasks": [{"prompt": "p"}]},
    ]
    with pytest.raises(topo.CycleError) as exc:
        topo.sort(ms)
    assert "m1" in str(exc.value) or "m2" in str(exc.value)


def test_self_cycle():
    ms = [
        {"name": "m1", "depends_on": ["m1"], "tasks": [{"prompt": "p"}]},
    ]
    with pytest.raises(topo.CycleError):
        topo.sort(ms)


def test_unknown_dependency_still_sorts():
    ms = [
        {"name": "m1", "depends_on": ["nonexistent"], "tasks": [{"prompt": "p"}]},
    ]
    result = topo.sort(ms)
    assert result == ["m1"]


def test_preserves_all_milestones():
    ms = [
        {"name": "m1", "tasks": [{"prompt": "p"}]},
        {"name": "m2", "tasks": [{"prompt": "p"}]},
        {"name": "m3", "tasks": [{"prompt": "p"}]},
    ]
    result = topo.sort(ms)
    assert len(result) == 3
    assert set(result) == {"m1", "m2", "m3"}
