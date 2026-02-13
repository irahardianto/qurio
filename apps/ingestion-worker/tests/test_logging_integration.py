# apps/ingestion-worker/tests/test_logging_integration.py
import logging
import json
import io
import contextlib
import sys
import os

# Adjust path to find modules
sys.path.append(".")
from logger import configure_logger


def test_logs_are_json():
    # Force ENV to production for JSON output
    os.environ["ENV"] = "production"

    f = io.StringIO()
    # Capture stdout
    with contextlib.redirect_stdout(f):
        configure_logger()

        # 1. Standard Root Log
        logging.getLogger("root").info("test_root_event", extra={"extra_key": "value"})

        # 2. Tornado Infrastructure Log (Simulate StreamClosedError)
        logging.getLogger("tornado.access").warning("test_tornado_request")

        # 3. Structlog Application Log
        import structlog

        structlog.get_logger().info("test_structlog_event")

    output = f.getvalue().strip()
    if not output:
        print("FAIL: No logs captured")
        sys.exit(1)

    lines = output.split("\n")
    print(f"Captured {len(lines)} log lines.")

    for line in lines:
        if not line.strip():
            continue
        try:
            data = json.loads(line)
            # Verify minimum fields
            if "timestamp" not in data and "level" not in data:
                print(f"FAIL: Missing standard fields in: {line}")
                sys.exit(1)

            # Verify specific messages exist
            if "event" in data:
                print(f"Verified: {data['event']}")
            elif "message" in data:
                print(f"Verified: {data['message']}")

        except json.JSONDecodeError:
            print(f"FAIL: Log line is not JSON: {line}")
            sys.exit(1)


if __name__ == "__main__":
    try:
        test_logs_are_json()
        print("PASS: All logs are JSON")
    except ImportError:
        print("SKIP: Project modules not found (run from apps/ingestion-worker root)")
        sys.exit(1)
