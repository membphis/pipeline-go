import logging
import pytest
from orchestrator import log


@pytest.fixture(autouse=True)
def _clean_logger():
    logger = logging.getLogger("orchestrator")
    logger.handlers.clear()
    logger.propagate = True
    yield
    logger.handlers.clear()
    logger.propagate = True


def test_setup_configures_root_logger():
    log.setup()
    root = logging.getLogger("orchestrator")
    assert root.level == logging.INFO
    assert len(root.handlers) >= 1


def test_get_returns_child_logger():
    logger = log.get("spec")
    assert logger.name == "orchestrator.spec"


def test_setup_respects_custom_level():
    log.setup(level=logging.DEBUG)
    root = logging.getLogger("orchestrator")
    assert root.level == logging.DEBUG
