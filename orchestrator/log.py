import logging
import sys


def setup(level: int = logging.INFO):
    logger = logging.getLogger("orchestrator")
    logger.setLevel(level)
    logger.propagate = False

    if not logger.handlers:
        handler = logging.StreamHandler(sys.stderr)
        handler.setLevel(level)
        fmt = logging.Formatter(
            "%(asctime)s [%(levelname)-5s] %(name)s: %(message)s",
            datefmt="%Y-%m-%d %H:%M:%S",
        )
        handler.setFormatter(fmt)
        logger.addHandler(handler)
    else:
        for h in logger.handlers:
            h.setLevel(level)


def get(name: str) -> logging.Logger:
    return logging.getLogger(f"orchestrator.{name}")
