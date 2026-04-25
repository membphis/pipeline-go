from orchestrator import context


def test_count_tokens_empty():
    assert context.count_tokens("") == 0


def test_count_tokens_simple():
    assert context.count_tokens("hello world") == 3


def test_count_tokens_longer():
    text = "a" * 100
    assert context.count_tokens(text) == 25


def test_build_bundle_includes_milestone_context():
    ms = {"name": "m1", "spec": "Some spec text."}
    bundle = context.build_bundle(milestones=[ms])
    assert "m1" in bundle
    assert "Some spec text." in bundle


def test_build_bundle_includes_handoff_notes():
    notes = [context.HandoffNote(source="path/HANDOFF.md", content="# Note")]
    bundle = context.build_bundle(handoff_notes=notes)
    assert "path/HANDOFF.md" in bundle
    assert "# Note" in bundle


def test_build_bundle_includes_verify_results():
    results = [context.VerifyResult(returncode=0, stdout="All good", stderr="")]
    bundle = context.build_bundle(verify_results=results)
    assert "All good" in bundle


def test_build_bundle_handoff_empty():
    bundle = context.build_bundle(milestones=[{"name": "m1"}])
    assert bundle != ""


def test_build_bundle_all_empty():
    bundle = context.build_bundle()
    assert bundle == ""


def test_bundle_exceeds_threshold():
    text = "x" * 800_000
    assert context.exceeds_threshold(text, max_tokens=180_000) is True


def test_bundle_under_threshold():
    text = "x" * 100
    assert context.exceeds_threshold(text, max_tokens=180_000) is False


def test_degrade_bundle_removes_verify():
    notes = [context.HandoffNote(source="h.md", content="note")]
    results = [context.VerifyResult(returncode=0, stdout="v", stderr="")]
    bundle = context.build_bundle(
        milestones=[{"name": "m1", "spec": "spec"}],
        handoff_notes=notes,
        verify_results=results,
    )
    degraded = context.degrade(bundle, strategy="no_verify")
    assert "v" not in degraded
    assert "note" in degraded


def test_degrade_bundle_removes_handoff():
    notes = [context.HandoffNote(source="h.md", content="note")]
    results = [context.VerifyResult(returncode=0, stdout="v", stderr="")]
    bundle = context.build_bundle(
        milestones=[{"name": "m1", "spec": "spec"}],
        handoff_notes=notes,
        verify_results=results,
    )
    degraded = context.degrade(bundle, strategy="no_handoff")
    assert "v" in degraded
    assert "note" not in degraded


def test_degrade_removes_both():
    notes = [context.HandoffNote(source="h.md", content="note")]
    results = [context.VerifyResult(returncode=0, stdout="v", stderr="")]
    bundle = context.build_bundle(
        milestones=[{"name": "m1", "spec": "spec"}],
        handoff_notes=notes,
        verify_results=results,
    )
    degraded = context.degrade(bundle, strategy="minimal")
    assert "v" not in degraded
    assert "note" not in degraded
