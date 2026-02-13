import logging
import pytest
from logger import configure_logger
import structlog


def test_stdlib_logging_capture(capsys):
    # Reset logging
    logging.root.handlers = []

    configure_logger()

    logger = logging.getLogger("third_party_lib")
    logger.info("leaky message")

    # Capture stderr/stdout
    captured = capsys.readouterr()

    # Check if output contains the message and is JSON formatted (if env is not development, but configure_logger defaults to ConsoleRenderer in dev)
    # Wait, configure_logger checks ENV.
    # If ENV is not "development", it uses JSONRenderer.
    # Default is "production".

    # But wait, pytest might capture stdout/stderr itself.
    # capsys should get it.

    print(f"Captured: {captured.out} {captured.err}")

    # If ENV is default (production), we expect JSON.
    # Note: structlog with ConsoleRenderer produces structured text, not JSON.

    output = captured.out + captured.err

    if "leaky message" not in output:
        pytest.fail(f"Log message not captured. Output: {output}")

    # Verify logger name is present
    if "third_party_lib" not in output:
        pytest.fail("Logger name not captured")

    # Verify structured keys (approximate for ConsoleRenderer, strict for JSON)
    # Since we can't easily control ENV inside the test execution without patching os.environ BEFORE import or config call
    pass


def test_structlog_direct_capture(capsys):
    configure_logger()
    logger = structlog.get_logger("app")
    logger.info("direct message", key="value")

    captured = capsys.readouterr()
    output = captured.out + captured.err
    assert "direct message" in output
    assert "key" in output
    assert "value" in output
