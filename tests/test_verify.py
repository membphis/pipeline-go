import pytest
from unittest.mock import patch, Mock
from orchestrator import verify


def test_run_verify_single_command_success():
    with patch("subprocess.run") as mock_run:
        mock_run.return_value = Mock(returncode=0, stdout="ok\n", stderr="")
        result = verify.run_verify("echo ok")
        assert result.returncode == 0
        assert result.stdout == "ok\n"


def test_run_verify_single_command_failure():
    with patch("subprocess.run") as mock_run:
        mock_run.return_value = Mock(returncode=1, stdout="", stderr="fail")
        result = verify.run_verify("false")
        assert result.returncode == 1


def test_run_verify_list_all_succeed():
    with patch("subprocess.run") as mock_run:
        mock_run.return_value = Mock(returncode=0, stdout="", stderr="")
        results = verify.run_verify(["echo a", "echo b"])
        assert len(results) == 2
        assert all(r.returncode == 0 for r in results)


def test_run_verify_list_one_fails():
    with patch("subprocess.run") as mock_run:
        mock_run.side_effect = [
            Mock(returncode=0, stdout="", stderr=""),
            Mock(returncode=1, stdout="", stderr="fail"),
        ]
        results = verify.run_verify(["echo a", "false"])
        assert len(results) == 2
        assert results[0].returncode == 0
        assert results[1].returncode == 1


def test_run_verify_none():
    result = verify.run_verify(None)
    assert result == []


def test_run_verify_empty_list():
    result = verify.run_verify([])
    assert result == []


def test_run_verify_with_timeout():
    with patch("subprocess.run") as mock_run:
        mock_run.return_value = Mock(returncode=0, stdout="", stderr="")
        result = verify.run_verify("echo ok", timeout=30)
        assert result.returncode == 0


def test_verify_result_dataclass():
    r = verify.VerifyResult(returncode=0, stdout="out", stderr="err")
    assert r.returncode == 0
    assert r.stdout == "out"
    assert r.stderr == "err"
    assert r.success is True


def test_verify_result_success_property():
    assert verify.VerifyResult(returncode=0, stdout="", stderr="").success is True
    assert verify.VerifyResult(returncode=1, stdout="", stderr="").success is False
    assert verify.VerifyResult(returncode=-1, stdout="", stderr="").success is False


def test_all_successful_true():
    results = [
        verify.VerifyResult(returncode=0, stdout="", stderr=""),
        verify.VerifyResult(returncode=0, stdout="", stderr=""),
    ]
    assert verify.all_successful(results) is True


def test_all_successful_false():
    results = [
        verify.VerifyResult(returncode=0, stdout="", stderr=""),
        verify.VerifyResult(returncode=1, stdout="", stderr=""),
    ]
    assert verify.all_successful(results) is False
