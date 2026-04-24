import subprocess
from dataclasses import dataclass
from orchestrator import log

logger = log.get("session")


@dataclass
class SessionResult:
    returncode: int
    stdout: str
    stderr: str


def _build_cmd(prompt: str) -> list[str]:
    return ["opencode", prompt]


def run(prompt: str, timeout: int | None = None) -> SessionResult:
    cmd = _build_cmd(prompt)
    logger.info("running opencode with %d-byte prompt", len(prompt))
    proc = subprocess.Popen(
        cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    stdout, stderr = proc.communicate(timeout=timeout)
    logger.info("opencode exited with code %d", proc.returncode)
    return SessionResult(
        returncode=proc.returncode,
        stdout=stdout,
        stderr=stderr,
    )
