import subprocess
import shutil
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

    if not shutil.which("opencode"):
        raise FileNotFoundError(
            "opencode executable not found on $PATH. "
            "Install it via: pip install opencode"
        )

    proc = subprocess.Popen(
        cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    try:
        stdout, stderr = proc.communicate(timeout=timeout)
    except subprocess.TimeoutExpired:
        proc.kill()
        stdout, stderr = proc.communicate()
        logger.warning("opencode timed out after %ds (code %d)", timeout or 0, proc.returncode)
        return SessionResult(returncode=-9, stdout=stdout, stderr=stderr)

    logger.info("opencode exited with code %d", proc.returncode)
    return SessionResult(
        returncode=proc.returncode,
        stdout=stdout,
        stderr=stderr,
    )
