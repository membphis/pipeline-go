import pytest
from unittest.mock import patch, Mock
from orchestrator import git


def test_current_branch():
    with patch("subprocess.run") as mock_run:
        mock_run.return_value = Mock(returncode=0, stdout="feature-branch\n", stderr="")
        result = git.current_branch()
        assert result == "feature-branch"


def test_current_branch_detached_head():
    with patch("subprocess.run") as mock_run:
        mock_run.return_value = Mock(returncode=0, stdout="HEAD\n", stderr="")
        with pytest.raises(RuntimeError, match="detached"):
            git.current_branch()


def test_create_branch():
    with patch("subprocess.run") as mock_run:
        mock_run.return_value = Mock(returncode=0, stdout="", stderr="")
        git.create_branch("feature/my-feature")
        cmd = mock_run.call_args[0][0]
        assert "checkout" in cmd
        assert "-b" in cmd
        assert "feature/my-feature" in cmd


def test_create_branch_from_base():
    with patch("subprocess.run") as mock_run:
        mock_run.return_value = Mock(returncode=0, stdout="", stderr="")
        git.create_branch("feature/my-feature", base="main")
        cmd = mock_run.call_args[0][0]
        assert cmd[:4] == ["git", "checkout", "-b", "feature/my-feature"]


def test_commit():
    with patch("subprocess.run") as mock_run:
        mock_run.return_value = Mock(returncode=0, stdout="", stderr="")
        git.commit("feat: add something")
        cmd = mock_run.call_args[0][0]
        assert cmd[:3] == ["git", "commit", "-m"]


def test_checkout():
    with patch("subprocess.run") as mock_run:
        mock_run.return_value = Mock(returncode=0, stdout="", stderr="")
        git.checkout("main")
        mock_run.assert_called_once()
        cmd = mock_run.call_args[0][0]
        assert cmd == ["git", "checkout", "main"]


def test_squash_merge():
    with patch("subprocess.run") as mock_run:
        mock_run.return_value = Mock(returncode=0, stdout="", stderr="")
        git.squash_merge("feature/my-feature")
        calls = mock_run.call_args_list
        merge_cmd = calls[0][0][0]
        assert "merge" in merge_cmd
        assert "--squash" in merge_cmd
        assert "feature/my-feature" in merge_cmd
        assert "git" in merge_cmd


def test_tag():
    with patch("subprocess.run") as mock_run:
        mock_run.return_value = Mock(returncode=0, stdout="", stderr="")
        git.tag("v1.0.0")
        cmd = mock_run.call_args[0][0]
        assert cmd[:3] == ["git", "tag", "v1.0.0"]


def test_tag_with_message():
    with patch("subprocess.run") as mock_run:
        mock_run.return_value = Mock(returncode=0, stdout="", stderr="")
        git.tag("v1.0.0", message="Release v1.0.0")
        cmd = mock_run.call_args[0][0]
        assert "--message" in cmd or "-m" in cmd


def test_is_clean_returns_true():
    with patch("subprocess.run") as mock_run:
        mock_run.return_value = Mock(returncode=0, stdout="", stderr="")
        assert git.is_clean() is True


def test_is_clean_returns_false():
    with patch("subprocess.run") as mock_run:
        mock_run.return_value = Mock(returncode=0, stdout=" M foo.py\n", stderr="")
        assert git.is_clean() is False


def test_has_unpushed_commits():
    with patch("subprocess.run") as mock_run:
        mock_run.return_value = Mock(returncode=0, stdout="1\n", stderr="")
        assert git.has_unpushed_commits() is True


def test_has_unpushed_commits_false():
    with patch("subprocess.run") as mock_run:
        mock_run.return_value = Mock(returncode=0, stdout="0\n", stderr="")
        assert git.has_unpushed_commits() is False


def test_is_detached_head():
    with patch("subprocess.run") as mock_run:
        mock_run.return_value = Mock(returncode=0, stdout="HEAD\n", stderr="")
        assert git.is_detached_head() is True


def test_is_detached_head_false():
    with patch("subprocess.run") as mock_run:
        mock_run.return_value = Mock(returncode=0, stdout="main\n", stderr="")
        assert git.is_detached_head() is False


def test_git_command_failure_raises():
    with patch("subprocess.run") as mock_run:
        mock_run.side_effect = RuntimeError("git: command not found")
        with pytest.raises(RuntimeError):
            git.current_branch()
