import subprocess
import pytest
from unittest.mock import Mock, patch
from orchestrator import session, log


@pytest.fixture(autouse=True)
def _setup_log():
    log.setup()


@pytest.fixture(autouse=True)
def _mock_opencode_path():
    with patch("shutil.which", return_value="/usr/bin/opencode"):
        yield


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


def test_run_timeout_kills_process():
    with patch("subprocess.Popen") as mock_popen:
        proc = Mock()
        # communicate raises TimeoutExpired on first call
        proc.communicate.side_effect = [
            subprocess.TimeoutExpired(cmd="opencode", timeout=5, output="partial", stderr=""),
            ("partial", ""),  # second call after kill
        ]
        proc.returncode = -9
        mock_popen.return_value = proc

        result = session.run("test prompt", timeout=5)

        assert result.returncode == -9
        proc.kill.assert_called_once()


def test_session_result_has_stdout():
    with patch("subprocess.Popen") as mock_popen:
        proc = Mock()
        proc.returncode = 0
        proc.communicate.return_value = ("output text", "")
        mock_popen.return_value = proc

        result = session.run("prompt")
        assert result.stdout == "output text"


def test_run_opencode_not_found():
    with patch("shutil.which", return_value=None):
        with pytest.raises(FileNotFoundError, match="opencode"):
            session.run("prompt")
