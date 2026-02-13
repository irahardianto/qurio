import logging
from logger import configure_logger


def test_stdlib_logging_captured(capsys):
    configure_logger()
    logging.getLogger("test_lib").warning("hello stdlib")

    captured = capsys.readouterr()
    assert '"event": "hello stdlib"' in captured.out
    assert '"logger": "test_lib"' in captured.out
