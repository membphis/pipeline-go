import pytest
from unittest.mock import Mock, patch
from orchestrator import session, log


@pytest.fixture(autouse=True)
def _setup_log():
    log.setup()


def test_build_cmd_returns_opencode_with_prompt():
    cmd = session._build_cmd("hello")
    assert cmd[0] == "opencode"
    assert cmd[1] == "hello"


def test_build_cmd_multiline():
    prompt = "line1\nline2\nline3"
    cmd = session._build_cmd(prompt)
    assert "line1\nline2\nline3" in cmd


def test_run_success():
    with patch("subprocess.Popen") as mock_popen:
        proc = Mock()
        proc.returncode = 0
        proc.communicate.return_value = ("output", "")
        mock_popen.return_value = proc

        result = session.run("test prompt")

        assert result.returncode == 0
        assert result.stdout == "output"


def test_run_failure():
    with patch("subprocess.Popen") as mock_popen:
        proc = Mock()
        proc.returncode = 1
        proc.communicate.return_value = ("", "error msg")
        mock_popen.return_value = proc

        result = session.run("test prompt")

        assert result.returncode == 1


def test_run_timeout():
    with patch("subprocess.Popen") as mock_popen:
        proc = Mock()
        proc.returncode = -9
        proc.communicate.return_value = ("", "")
        mock_popen.return_value = proc

        result = session.run("test prompt", timeout=5)

        assert result.returncode == -9


def test_session_result_has_stdout():
    with patch("subprocess.Popen") as mock_popen:
        proc = Mock()
        proc.returncode = 0
        proc.communicate.return_value = ("output text", "")
        mock_popen.return_value = proc

        result = session.run("prompt")
        assert result.stdout == "output text"
