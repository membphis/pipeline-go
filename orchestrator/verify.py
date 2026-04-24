import subprocess
from dataclasses import dataclass
from typing import Any


@dataclass
class VerifyResult:
    returncode: int
    stdout: str
    stderr: str

    @property
    def success(self) -> bool:
        return self.returncode == 0


def run_verify(
    spec: str | list[str] | None,
    timeout: int | None = None,
) -> list[VerifyResult] | VerifyResult:
    if not spec:
        return []
    if isinstance(spec, str):
        return _run_one(spec, timeout)
    return [_run_one(cmd, timeout) for cmd in spec]


def _run_one(cmd: str, timeout: int | None = None) -> VerifyResult:
    proc = subprocess.run(
        cmd,
        shell=True,
        capture_output=True,
        text=True,
        timeout=timeout,
    )
    return VerifyResult(
        returncode=proc.returncode,
        stdout=proc.stdout,
        stderr=proc.stderr,
    )


def all_successful(results: list[VerifyResult]) -> bool:
    return all(r.success for r in results)
