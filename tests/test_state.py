import pytest
import yaml
from orchestrator import state


def test_new_state_is_pending_for_all():
    ms = [{"name": "m1"}, {"name": "m2"}]
    st = state.State(ms)
    assert st.get("m1") == "pending"
    assert st.get("m2") == "pending"


def test_set_milestone_status():
    ms = [{"name": "m1"}]
    st = state.State(ms)
    st.set("m1", "completed")
    assert st.get("m1") == "completed"


def test_set_invalid_status():
    ms = [{"name": "m1"}]
    st = state.State(ms)
    with pytest.raises(ValueError):
        st.set("m1", "invalid_status")


def test_unknown_milestone():
    ms = [{"name": "m1"}]
    st = state.State(ms)
    with pytest.raises(KeyError):
        st.get("nonexistent")


def test_get_all_statuses():
    ms = [{"name": "m1"}, {"name": "m2"}]
    st = state.State(ms)
    st.set("m1", "completed")
    all_st = st.get_all()
    assert all_st["m1"] == "completed"
    assert all_st["m2"] == "pending"


def test_is_completed():
    ms = [{"name": "m1"}]
    st = state.State(ms)
    assert not st.is_completed("m1")
    st.set("m1", "completed")
    assert st.is_completed("m1")


def test_all_completed():
    ms = [{"name": "m1"}, {"name": "m2"}]
    st = state.State(ms)
    assert not st.all_completed()
    st.set("m1", "completed")
    assert not st.all_completed()
    st.set("m2", "completed")
    assert st.all_completed()


def test_save_and_load(tmp_path):
    ms = [{"name": "m1"}]
    st = state.State(ms)
    st.set("m1", "completed")
    p = tmp_path / "state.yaml"
    st.save(str(p))

    st2 = state.State(ms, path=str(p))
    assert st2.get("m1") == "completed"


def test_load_empty_file(tmp_path):
    p = tmp_path / "state.yaml"
    p.write_text("")
    ms = [{"name": "m1"}]
    st = state.State(ms, path=str(p))
    assert st.get("m1") == "pending"


def test_load_corrupted_file(tmp_path):
    p = tmp_path / "state.yaml"
    p.write_text("{broken: yaml: ...")
    ms = [{"name": "m1"}]
    st = state.State(ms, path=str(p))
    assert st.get("m1") == "pending"


def test_timestamps(tmp_path):
    ms = [{"name": "m1"}]
    st = state.State(ms, path=str(tmp_path / "state.yaml"))
    import time
    st.set("m1", "completed")
    saved = st._data["milestones"]["m1"]
    assert "timestamp" in saved
    assert saved["status"] == "completed"
