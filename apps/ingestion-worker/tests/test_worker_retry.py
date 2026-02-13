import pytest
from unittest.mock import MagicMock, patch
import asyncio
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
        ):
            await process_message(mock_msg)

            # Should finish and publish failure
            mock_msg.finish.assert_called()
            mock_msg.requeue.assert_not_called()
            mock_producer.pub.assert_called()  # Publish failure
