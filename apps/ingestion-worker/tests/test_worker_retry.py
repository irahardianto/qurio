import pytest
from unittest.mock import MagicMock, AsyncMock, patch
import asyncio
import json
from main import process_message


@pytest.mark.asyncio
async def test_requeue_on_timeout():
    mock_msg = MagicMock()
    mock_msg.body = b'{"type": "web", "url": "http://fail.com", "id": "1"}'
    mock_msg.attempts = 1

    # Simulate TimeoutError in handle_web_task
    with (
        patch("main.handle_web_task", side_effect=asyncio.TimeoutError("Timeout")),
        patch("main.settings") as mock_settings,
    ):
        # Configure mock settings
        mock_settings.retry_max_attempts = 3
        mock_settings.retry_initial_delay_ms = 1000
        mock_settings.retry_max_delay_ms = 60000
        mock_settings.retry_backoff_multiplier = 2

        with (
            patch("main.producer"),
            patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)),
            patch("main.get_crawler", new_callable=AsyncMock, return_value=MagicMock()),
        ):
            await process_message(mock_msg)

            # Should NOT finish, should requeue
            mock_msg.finish.assert_not_called()
            mock_msg.requeue.assert_called()

            # Check delay
            # attempt 1 -> 2^(1-1) * 1000 = 1000 ms
            args, kwargs = mock_msg.requeue.call_args
            delay = kwargs.get("delay")
            if delay is None and args:
                delay = args[0]

            assert delay == 1000


@pytest.mark.asyncio
async def test_fail_on_max_retries():
    mock_msg = MagicMock()
    mock_msg.body = b'{"type": "web", "url": "http://fail.com", "id": "1"}'
    mock_msg.attempts = 4  # Max is 3

    with (
        patch("main.handle_web_task", side_effect=asyncio.TimeoutError("Timeout")),
        patch("main.settings") as mock_settings,
    ):
        mock_settings.retry_max_attempts = 3

        with (
            patch("main.producer") as mock_producer,
            patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)),
            patch("main.get_crawler", new_callable=AsyncMock, return_value=MagicMock()),
        ):
            await process_message(mock_msg)

            # Should finish and publish failure
            mock_msg.finish.assert_called()
            mock_msg.requeue.assert_not_called()
            mock_producer.pub.assert_called()  # Publish failure


@pytest.mark.asyncio
async def test_requeue_on_crawl_timeout_ingestion_error():
    """Verify that IngestionError(ERR_CRAWL_TIMEOUT) triggers requeue with backoff."""
    from exceptions import IngestionError, ERR_CRAWL_TIMEOUT

    mock_msg = MagicMock()
    mock_msg.body = b'{"type": "web", "url": "http://fail.com", "id": "1"}'
    mock_msg.attempts = 1

    # Simulate ERR_CRAWL_TIMEOUT from handle_web_task
    crawl_err = IngestionError(ERR_CRAWL_TIMEOUT, "net::ERR_TIMED_OUT")

    with (
        patch("main.handle_web_task", side_effect=crawl_err),
        patch("main.settings") as mock_settings,
    ):
        mock_settings.retry_max_attempts = 3
        mock_settings.retry_initial_delay_ms = 1000
        mock_settings.retry_max_delay_ms = 60000
        mock_settings.retry_backoff_multiplier = 2

        with (
            patch("main.producer"),
            patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)),
            patch("main.get_crawler", new_callable=AsyncMock, return_value=MagicMock()),
        ):
            await process_message(mock_msg)

            # Should requeue with backoff, NOT finish
            mock_msg.finish.assert_not_called()
            mock_msg.requeue.assert_called()

            args, kwargs = mock_msg.requeue.call_args
            delay = kwargs.get("delay")
            if delay is None and args:
                delay = args[0]
            assert delay == 1000


# --- Additional Comprehensive Retry Tests ---


@pytest.mark.asyncio
async def test_requeue_on_err_crawl_dns():
    """Verify that ERR_CRAWL_DNS triggers requeue."""
    from exceptions import IngestionError, ERR_CRAWL_DNS

    mock_msg = MagicMock()
    mock_msg.body = b'{"type": "web", "url": "http://fail.com", "id": "1"}'
    mock_msg.attempts = 1

    crawl_err = IngestionError(ERR_CRAWL_DNS, "DNS resolution failed")

    with (
        patch("main.handle_web_task", side_effect=crawl_err),
        patch("main.settings") as mock_settings,
    ):
        mock_settings.retry_max_attempts = 3
        mock_settings.retry_initial_delay_ms = 1000
        mock_settings.retry_max_delay_ms = 60000
        mock_settings.retry_backoff_multiplier = 2

        with (
            patch("main.producer"),
            patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)),
            patch("main.get_crawler", new_callable=AsyncMock, return_value=MagicMock()),
        ):
            await process_message(mock_msg)

            mock_msg.finish.assert_not_called()
            mock_msg.requeue.assert_called()


@pytest.mark.asyncio
async def test_requeue_on_err_crawl_refused():
    """Verify that ERR_CRAWL_REFUSED triggers requeue."""
    from exceptions import IngestionError, ERR_CRAWL_REFUSED

    mock_msg = MagicMock()
    mock_msg.body = b'{"type": "web", "url": "http://fail.com", "id": "1"}'
    mock_msg.attempts = 1

    crawl_err = IngestionError(ERR_CRAWL_REFUSED, "Connection refused")

    with (
        patch("main.handle_web_task", side_effect=crawl_err),
        patch("main.settings") as mock_settings,
    ):
        mock_settings.retry_max_attempts = 3
        mock_settings.retry_initial_delay_ms = 1000
        mock_settings.retry_max_delay_ms = 60000
        mock_settings.retry_backoff_multiplier = 2

        with (
            patch("main.producer"),
            patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)),
            patch("main.get_crawler", new_callable=AsyncMock, return_value=MagicMock()),
            patch("main.restart_crawler", new_callable=AsyncMock),
        ):
            await process_message(mock_msg)

            mock_msg.finish.assert_not_called()
            mock_msg.requeue.assert_called()


@pytest.mark.asyncio
async def test_permanent_error_no_requeue_err_encrypted():
    """Verify that ERR_ENCRYPTED does NOT trigger requeue."""
    from exceptions import IngestionError, ERR_ENCRYPTED

    mock_msg = MagicMock()
    mock_msg.body = b'{"type": "file", "path": "/tmp/secret.pdf", "id": "1"}'
    mock_msg.attempts = 1

    err = IngestionError(ERR_ENCRYPTED, "File is encrypted")

    with (
        patch("main.handle_file_task", side_effect=err),
        patch("main.settings") as mock_settings,
    ):
        mock_settings.retry_max_attempts = 3

        with (
            patch("main.producer") as mock_producer,
            patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)),
        ):
            await process_message(mock_msg)

            # Should finish and publish failure, NOT requeue
            mock_msg.finish.assert_called()
            mock_msg.requeue.assert_not_called()
            mock_producer.pub.assert_called()


@pytest.mark.asyncio
async def test_permanent_error_no_requeue_err_crawl_blocked():
    """Verify that ERR_CRAWL_BLOCKED does NOT trigger requeue."""
    from exceptions import IngestionError, ERR_CRAWL_BLOCKED

    mock_msg = MagicMock()
    mock_msg.body = b'{"type": "web", "url": "http://blocked.com", "id": "1"}'
    mock_msg.attempts = 1

    err = IngestionError(ERR_CRAWL_BLOCKED, "Blocked by robots.txt")

    with (
        patch("main.handle_web_task", side_effect=err),
        patch("main.settings") as mock_settings,
    ):
        mock_settings.retry_max_attempts = 3

        with (
            patch("main.producer") as mock_producer,
            patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)),
            patch("main.get_crawler", new_callable=AsyncMock, return_value=MagicMock()),
        ):
            await process_message(mock_msg)

            mock_msg.finish.assert_called()
            mock_msg.requeue.assert_not_called()
            mock_producer.pub.assert_called()


@pytest.mark.asyncio
async def test_exponential_backoff_calculation():
    """Verify exponential backoff calculation for multiple attempts."""
    mock_msg = MagicMock()
    mock_msg.body = b'{"type": "web", "url": "http://fail.com", "id": "1"}'

    with (
        patch("main.handle_web_task", side_effect=asyncio.TimeoutError("Timeout")),
        patch("main.settings") as mock_settings,
    ):
        mock_settings.retry_max_attempts = 3
        mock_settings.retry_initial_delay_ms = 1000
        mock_settings.retry_max_delay_ms = 60000
        mock_settings.retry_backoff_multiplier = 2

        with (
            patch("main.producer"),
            patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)),
            patch("main.get_crawler", new_callable=AsyncMock, return_value=MagicMock()),
        ):
            # Attempt 1: delay = 2^0 * 1000 = 1000ms
            mock_msg.attempts = 1
            await process_message(mock_msg)
            args, kwargs = mock_msg.requeue.call_args
            delay = kwargs.get("delay") or args[0]
            assert delay == 1000

            # Attempt 2: delay = 2^1 * 1000 = 2000ms
            mock_msg.reset_mock()
            mock_msg.attempts = 2
            await process_message(mock_msg)
            args, kwargs = mock_msg.requeue.call_args
            delay = kwargs.get("delay") or args[0]
            assert delay == 2000

            # Attempt 3: delay = 2^2 * 1000 = 4000ms
            mock_msg.reset_mock()
            mock_msg.attempts = 3
            await process_message(mock_msg)
            args, kwargs = mock_msg.requeue.call_args
            delay = kwargs.get("delay") or args[0]
            assert delay == 4000


@pytest.mark.asyncio
async def test_max_delay_cap():
    """Verify that delay is capped at retry_max_delay_ms."""
    mock_msg = MagicMock()
    mock_msg.body = b'{"type": "web", "url": "http://fail.com", "id": "1"}'
    mock_msg.attempts = 10  # Very high attempt number

    with (
        patch("main.handle_web_task", side_effect=asyncio.TimeoutError("Timeout")),
        patch("main.settings") as mock_settings,
    ):
        mock_settings.retry_max_attempts = 15
        mock_settings.retry_initial_delay_ms = 1000
        mock_settings.retry_max_delay_ms = 10000  # Cap at 10 seconds
        mock_settings.retry_backoff_multiplier = 2

        with (
            patch("main.producer"),
            patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)),
            patch("main.get_crawler", new_callable=AsyncMock, return_value=MagicMock()),
        ):
            await process_message(mock_msg)

            args, kwargs = mock_msg.requeue.call_args
            delay = kwargs.get("delay") or args[0]
            # 2^9 * 1000 = 512000, but should be capped at 10000
            assert delay == 10000


@pytest.mark.asyncio
async def test_requeue_failure_handling():
    """Verify that requeue failures are logged but don't crash."""
    mock_msg = MagicMock()
    mock_msg.body = b'{"type": "web", "url": "http://fail.com", "id": "1"}'
    mock_msg.attempts = 1
    mock_msg.requeue.side_effect = Exception("Requeue failed")

    with (
        patch("main.handle_web_task", side_effect=asyncio.TimeoutError("Timeout")),
        patch("main.settings") as mock_settings,
    ):
        mock_settings.retry_max_attempts = 3
        mock_settings.retry_initial_delay_ms = 1000
        mock_settings.retry_max_delay_ms = 60000
        mock_settings.retry_backoff_multiplier = 2

        with (
            patch("main.producer"),
            patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)),
            patch("main.get_crawler", new_callable=AsyncMock, return_value=MagicMock()),
        ):
            # Should not raise exception
            await process_message(mock_msg)


@pytest.mark.asyncio
async def test_string_based_timeout_detection():
    """Verify that 'timeout' in error string triggers requeue."""
    mock_msg = MagicMock()
    mock_msg.body = b'{"type": "web", "url": "http://fail.com", "id": "1"}'
    mock_msg.attempts = 1

    with (
        patch(
            "main.handle_web_task", side_effect=Exception("Request timeout occurred")
        ),
        patch("main.settings") as mock_settings,
    ):
        mock_settings.retry_max_attempts = 3
        mock_settings.retry_initial_delay_ms = 1000
        mock_settings.retry_max_delay_ms = 60000
        mock_settings.retry_backoff_multiplier = 2

        with (
            patch("main.producer"),
            patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)),
            patch("main.get_crawler", new_callable=AsyncMock, return_value=MagicMock()),
        ):
            await process_message(mock_msg)

            # Should requeue due to 'timeout' in error string
            mock_msg.requeue.assert_called()
            mock_msg.finish.assert_not_called()


@pytest.mark.asyncio
async def test_string_based_connection_detection():
    """Verify that 'connection' in error string triggers requeue."""
    mock_msg = MagicMock()
    mock_msg.body = b'{"type": "web", "url": "http://fail.com", "id": "1"}'
    mock_msg.attempts = 1

    with (
        patch(
            "main.handle_web_task", side_effect=Exception("Connection reset by peer")
        ),
        patch("main.settings") as mock_settings,
    ):
        mock_settings.retry_max_attempts = 3
        mock_settings.retry_initial_delay_ms = 1000
        mock_settings.retry_max_delay_ms = 60000
        mock_settings.retry_backoff_multiplier = 2

        with (
            patch("main.producer"),
            patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)),
            patch("main.get_crawler", new_callable=AsyncMock, return_value=MagicMock()),
        ):
            await process_message(mock_msg)

            mock_msg.requeue.assert_called()
            mock_msg.finish.assert_not_called()


@pytest.mark.asyncio
async def test_retry_with_depth_field():
    """Verify that depth field is preserved in retry."""
    mock_msg = MagicMock()
    mock_msg.body = b'{"type": "web", "url": "http://fail.com", "id": "1", "depth": 2}'
    mock_msg.attempts = 4  # Exceeded max

    with (
        patch("main.handle_web_task", side_effect=asyncio.TimeoutError("Timeout")),
        patch("main.settings") as mock_settings,
    ):
        mock_settings.retry_max_attempts = 3

        with (
            patch("main.producer") as mock_producer,
            patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)),
            patch("main.get_crawler", new_callable=AsyncMock, return_value=MagicMock()),
        ):
            await process_message(mock_msg)

            # Check that original_payload includes depth
            args, kwargs = mock_producer.pub.call_args
            payload = json.loads(args[1])
            assert payload["original_payload"]["depth"] == 2


@pytest.mark.asyncio
async def test_transient_error_with_gemini_api_key():
    """Verify that gemini_api_key is preserved in retry."""
    mock_msg = MagicMock()
    mock_msg.body = b'{"type": "web", "url": "http://fail.com", "id": "1", "gemini_api_key": "test-key"}'
    mock_msg.attempts = 4

    with (
        patch("main.handle_web_task", side_effect=asyncio.TimeoutError("Timeout")),
        patch("main.settings") as mock_settings,
    ):
        mock_settings.retry_max_attempts = 3

        with (
            patch("main.producer") as mock_producer,
            patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)),
            patch("main.get_crawler", new_callable=AsyncMock, return_value=MagicMock()),
        ):
            await process_message(mock_msg)

            args, kwargs = mock_producer.pub.call_args
            payload = json.loads(args[1])
            assert payload["original_payload"]["gemini_api_key"] == "test-key"


@pytest.mark.asyncio
async def test_ingestion_error_includes_code_in_failure():
    """Verify that IngestionError code is included in failure payload."""
    from exceptions import IngestionError, ERR_INVALID_FORMAT

    mock_msg = MagicMock()
    mock_msg.body = b'{"type": "file", "path": "/tmp/bad.pdf", "id": "1"}'
    mock_msg.attempts = 1

    err = IngestionError(ERR_INVALID_FORMAT, "Invalid format")

    with (
        patch("main.handle_file_task", side_effect=err),
        patch("main.settings") as mock_settings,
    ):
        mock_settings.retry_max_attempts = 3

        with (
            patch("main.producer") as mock_producer,
            patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)),
        ):
            await process_message(mock_msg)

            args, kwargs = mock_producer.pub.call_args
            payload = json.loads(args[1])
            assert payload["code"] == ERR_INVALID_FORMAT
            assert ERR_INVALID_FORMAT in payload["error"]
