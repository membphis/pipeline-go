from unittest.mock import patch, Mock
from orchestrator import review


def test_review_phase_collects_context():
    with patch("orchestrator.review.context.build_bundle") as mock_build:
        mock_build.return_value = "bundled context"
        with patch("orchestrator.review.session.run") as mock_run:
            mock_run.return_value = Mock(returncode=0, stdout="review output", stderr="")
            result = review.review_phase(
                milestone={"name": "m1", "spec": "spec"},
                handoff_notes=[],
                verify_results=[],
            )
    mock_build.assert_called_once()
    mock_run.assert_called_once()
    assert result.returncode == 0 or isinstance(result, review.ReviewResult)


def test_review_phase_prompt_contains_milestone():
    with patch("orchestrator.review.context.build_bundle", return_value="ctx"):
        with patch("orchestrator.review.session.run") as mock_run:
            mock_run.return_value = Mock(returncode=0, stdout="", stderr="")
            review.review_phase(
                milestone={"name": "m1", "spec": "spec"},
                handoff_notes=[],
                verify_results=[],
            )
    prompt = mock_run.call_args[0][0]
    assert "m1" in prompt
    assert "phase review" in prompt.lower()


def test_review_final_prompt_contains_project():
    with patch("orchestrator.review.context.build_bundle", return_value="ctx"):
        with patch("orchestrator.review.session.run") as mock_run:
            mock_run.return_value = Mock(returncode=0, stdout="", stderr="")
            review.review_final(
                project_name="test-project",
                milestones=[],
                handoff_notes=[],
                verify_results=[],
            )
    prompt = mock_run.call_args[0][0]
    assert "test-project" in prompt
    assert "final review" in prompt.lower()


def test_review_phase_degrades_on_overflow():
    with patch("orchestrator.review.context.build_bundle") as mock_build:
        mock_build.return_value = "very long context " * 50000
        with patch("orchestrator.review.context.exceeds_threshold") as mock_exceeds:
            mock_exceeds.return_value = True
            with patch("orchestrator.review.context.degrade") as mock_degrade:
                mock_degrade.return_value = "degraded"
                with patch("orchestrator.review.session.run") as mock_run:
                    mock_run.return_value = Mock(returncode=0, stdout="", stderr="")
                    review.review_phase(
                        milestone={"name": "m1"},
                        handoff_notes=[],
                        verify_results=[],
                    )
        mock_degrade.assert_called_once()
