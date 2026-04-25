from unittest.mock import patch, Mock
from orchestrator import main


def test_parse_args_defaults():
    args = main.parse_args(["--spec", "project.yaml"])
    assert args.spec == "project.yaml"
    assert args.root == "."
    assert args.branch is None
    assert args.extra_spec == []


def test_parse_args_all():
    args = main.parse_args([
        "--spec", "my.yaml",
        "--root", "/path/to/project",
        "--branch", "feature/test",
        "--extra-spec", "extra1.yaml",
        "--extra-spec", "extra2.yaml",
    ])
    assert args.spec == "my.yaml"
    assert args.root == "/path/to/project"
    assert args.branch == "feature/test"
    assert args.extra_spec == ["extra1.yaml", "extra2.yaml"]


def test_pipeline_init():
    pipeline = main.Pipeline(
        spec_path="project.yaml",
        root=".",
        extra_specs=[],
        branch=None,
    )
    assert pipeline.spec_path == "project.yaml"
    assert pipeline.root == "."


def test_compute_milestone_spec_includes_prompt():
    ms = {"name": "m1", "spec": "build the thing", "tasks": [{"name": "t1", "prompt": "do X"}]}
    result = main.compute_milestone_spec(ms, [], None)
    assert "build the thing" in result
    assert "do X" in result


def test_compute_milestone_spec_includes_state():
    ms = {"name": "m1", "tasks": [{"prompt": "p"}]}
    with patch("orchestrator.main.state.State"):
        mock_state = Mock()
        mock_state.get.return_value = "pending"
        mock_state.get_all.return_value = {"m1": "pending", "m2": "completed"}
        result = main.compute_milestone_spec(ms, [{"name": "m2"}], mock_state)
    assert "pending" in result
    assert "completed" in result


def test_compute_milestone_spec_includes_previous_handoff():
    handoff_notes = [Mock(source="m1/HANDOFF.md", content="Previous work done")]
    ms = {"name": "m2", "tasks": [{"prompt": "p"}]}
    result = main.compute_milestone_spec(ms, [], None, handoff_notes=handoff_notes)
    assert "Previous work done" in result


def test_run_pipeline_success():
    """Integration-style test for Pipeline.run() with all mocks."""
    with patch("orchestrator.main.spec") as mock_spec, \
         patch("orchestrator.main.topo") as mock_topo, \
         patch("orchestrator.main.state.State") as MockState, \
         patch("orchestrator.main.git") as mock_git, \
         patch("orchestrator.main.session") as mock_session, \
         patch("orchestrator.main.verify") as mock_verify, \
         patch("orchestrator.main.handoff") as mock_handoff, \
         patch("orchestrator.main.review") as mock_review:

        # Setup mocks
        mock_spec.load.return_value = {
            "project": {"name": "test-project"},
            "milestones": [
                {"name": "m1", "spec": "spec1", "tasks": [{"prompt": "p1"}]},
            ],
        }
        mock_spec.preflight.return_value = []
        mock_topo.sort.return_value = ["m1"]
        mock_state = Mock()
        MockState.return_value = mock_state
        mock_state.all_completed.return_value = True
        mock_state.get_all.return_value = {"m1": "completed"}
        mock_session.run.return_value = Mock(returncode=0, stdout="done", stderr="")
        mock_verify.run_verify.return_value = [Mock(returncode=0, stdout="ok", stderr="")]
        mock_handoff.collect.return_value = []
        mock_review.review_phase.return_value = Mock(returncode=0, stdout="", stderr="")
        mock_review.review_final.return_value = Mock(returncode=0, stdout="", stderr="")
        mock_git.is_clean.return_value = True

        pipeline = main.Pipeline(spec_path="p.yaml", root=".", extra_specs=[], branch=None)
        result = pipeline.run()

    assert result == 0
    mock_spec.load.assert_called_once()
    mock_spec.preflight.assert_called_once()
    mock_topo.sort.assert_called_once()
    mock_session.run.assert_called()
    mock_review.review_final.assert_called_once()


def test_run_pipeline_preflight_errors_abort():
    with patch("orchestrator.main.spec") as mock_spec:
        mock_spec.load.return_value = {"project": {"name": "t"}, "milestones": []}
        mock_spec.preflight.return_value = ["error1"]
        pipeline = main.Pipeline(spec_path="p.yaml", root=".", extra_specs=[], branch=None)
        result = pipeline.run()
    assert result == 1


def test_run_pipeline_session_failure_continues():
    """Session failure should log but continue to next milestone."""
    with patch("orchestrator.main.spec") as mock_spec, \
         patch("orchestrator.main.topo") as mock_topo, \
         patch("orchestrator.main.state.State") as MockState, \
         patch("orchestrator.main.git") as mock_git, \
         patch("orchestrator.main.session") as mock_session, \
         patch("orchestrator.main.verify") as mock_verify, \
         patch("orchestrator.main.handoff") as mock_handoff, \
         patch("orchestrator.main.review") as mock_review:

        mock_spec.load.return_value = {
            "project": {"name": "t"},
            "milestones": [
                {"name": "m1", "tasks": [{"prompt": "p1"}]},
                {"name": "m2", "tasks": [{"prompt": "p2"}]},
            ],
        }
        mock_spec.preflight.return_value = []
        mock_topo.sort.return_value = ["m1", "m2"]
        mock_state = Mock()
        MockState.return_value = mock_state
        mock_state.all_completed.return_value = False
        mock_state.get_all.return_value = {"m1": "failed", "m2": "completed"}
        mock_session.run.side_effect = [
            Mock(returncode=1, stdout="fail", stderr="error"),
            Mock(returncode=0, stdout="done", stderr=""),
        ]
        mock_verify.run_verify.return_value = [Mock(returncode=0, stdout="", stderr="")]
        mock_handoff.collect.return_value = []
        mock_review.review_phase.return_value = Mock(returncode=0, stdout="", stderr="")
        mock_review.review_final.return_value = Mock(returncode=0, stdout="", stderr="")
        mock_git.is_clean.return_value = True

        pipeline = main.Pipeline(spec_path="p.yaml", root=".", extra_specs=[], branch=None)
        result = pipeline.run()

    assert result == 1
    assert mock_session.run.call_count >= 2


def test_run_pipeline_uses_branch_override():
    with patch("orchestrator.main.spec") as mock_spec, \
         patch("orchestrator.main.topo") as mock_topo, \
         patch("orchestrator.main.state.State") as MockState, \
         patch("orchestrator.main.git") as mock_git, \
         patch("orchestrator.main.session") as mock_session, \
         patch("orchestrator.main.verify") as mock_verify, \
         patch("orchestrator.main.handoff") as mock_handoff, \
         patch("orchestrator.main.review") as mock_review:

        mock_spec.load.return_value = {
            "project": {"name": "t"},
            "milestones": [{"name": "m1", "tasks": [{"prompt": "p1"}]}],
        }
        mock_spec.preflight.return_value = []
        mock_topo.sort.return_value = ["m1"]
        mock_state = Mock()
        MockState.return_value = mock_state
        mock_state.all_completed.return_value = True
        mock_state.get_all.return_value = {"m1": "completed"}
        mock_session.run.return_value = Mock(returncode=0, stdout="done", stderr="")
        mock_verify.run_verify.return_value = [Mock(returncode=0, stdout="", stderr="")]
        mock_handoff.collect.return_value = []
        mock_review.review_phase.return_value = Mock(returncode=0, stdout="", stderr="")
        mock_review.review_final.return_value = Mock(returncode=0, stdout="", stderr="")
        mock_git.is_clean.return_value = True

        pipeline = main.Pipeline(spec_path="p.yaml", root=".", extra_specs=[], branch="existing-branch")
        pipeline.run()

        mock_git.create_branch.assert_not_called()
