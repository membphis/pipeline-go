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
    logger = logging.getLogger("orchestrator")
    assert logger.level == logging.INFO
    assert len(logger.handlers) >= 1
    for h in logger.handlers:
        assert h.level == logging.INFO


def test_get_returns_child_logger():
    logger = log.get("spec")
    assert logger.name == "orchestrator.spec"


def test_setup_respects_custom_level():
    log.setup(level=logging.DEBUG)
    logger = logging.getLogger("orchestrator")
    assert logger.level == logging.DEBUG
    for h in logger.handlers:
        assert h.level == logging.DEBUG


def test_setup_updates_handler_level_on_recall():
    log.setup(level=logging.INFO)
    log.setup(level=logging.DEBUG)
    logger = logging.getLogger("orchestrator")
    assert logger.level == logging.DEBUG
    for h in logger.handlers:
        assert h.level == logging.DEBUG
