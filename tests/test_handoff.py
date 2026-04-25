from orchestrator import handoff


def test_collect_none(tmp_path):
    notes = handoff.collect(str(tmp_path))
    assert notes == []


def test_collect_single(tmp_path):
    hf = tmp_path / "HANDOFF.md"
    hf.write_text("# Handoff: Task 1\n\nDone something.")
    notes = handoff.collect(str(tmp_path))
    assert len(notes) == 1
    assert notes[0].source == str(hf)
    assert "Done" in notes[0].content


def test_collect_multiple(tmp_path):
    (tmp_path / "sub1").mkdir()
    (tmp_path / "sub2").mkdir()
    h1 = tmp_path / "sub1" / "HANDOFF.md"
    h1.write_text("Note 1")
    h2 = tmp_path / "sub2" / "HANDOFF.md"
    h2.write_text("Note 2")
    notes = handoff.collect(str(tmp_path))
    assert len(notes) == 2


def test_collect_skips_non_handoff(tmp_path):
    (tmp_path / "README.md").write_text("readme")
    (tmp_path / "HANDOFF.md").write_text("handoff")
    notes = handoff.collect(str(tmp_path))
    assert len(notes) == 1


def test_handoff_note_dataclass():
    n = handoff.HandoffNote(source="path", content="content")
    assert n.source == "path"
    assert n.content == "content"


def test_format_notes_empty():
    result = handoff.format_notes([])
    assert result == ""


def test_format_notes_single():
    notes = [handoff.HandoffNote(source="path/to/HANDOFF.md", content="# Note\n\nBody")]
    result = handoff.format_notes(notes)
    assert "path/to/HANDOFF.md" in result
    assert "# Note" in result
    assert "Body" in result


def test_format_notes_multiple():
    notes = [
        handoff.HandoffNote(source="a/HANDOFF.md", content="First"),
        handoff.HandoffNote(source="b/HANDOFF.md", content="Second"),
    ]
    result = handoff.format_notes(notes)
    assert "a/HANDOFF.md" in result
    assert "b/HANDOFF.md" in result
    assert "First" in result
    assert "Second" in result
    assert "---" in result or "===" in result or "\n\n" in result


def test_collect_non_existent_directory():
    notes = handoff.collect("/nonexistent/path")
    assert notes == []


def test_collect_skips_directories_named_handoff(tmp_path):
    (tmp_path / "HANDOFF.md").mkdir()
    notes = handoff.collect(str(tmp_path))
    assert notes == []
