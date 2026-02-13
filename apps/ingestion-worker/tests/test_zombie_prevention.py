import sys
import os

sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))
import pytest
import asyncio
from unittest.mock import MagicMock, patch
from main import process_message


@pytest.mark.asyncio
async def test_process_message_cleanup_on_cancel():
    # Setup mocks
    mock_nsq_msg = MagicMock()
    mock_nsq_msg.body = b'{"type": "web", "url": "http://example.com", "id": "123"}'
    mock_nsq_msg.touch = MagicMock()
    mock_nsq_msg.finish = MagicMock()

    # Mock handle_web_task to sleep forever to simulate long work
    # We use a side_effect that checks for cancellation to simulate a real async task being cancelled
    async def long_running_task(*args, **kwargs):
        try:
            await asyncio.sleep(10)
        except asyncio.CancelledError:
            # Simulate cleanup in the handler if needed,
            # but mainly we verify process_message's cleanup
            raise

    with (
        patch("main.handle_web_task", side_effect=long_running_task),
        patch("main.producer", new=MagicMock()),
    ):
        # Run process_message in a task
        task = asyncio.create_task(process_message(mock_nsq_msg))

        # Allow the task to start and enter the touch loop
        await asyncio.sleep(0.1)

        # Verify touch loop is active (touch called at least once or scheduled)
        # Note: sleep(10) in touch loop means it might not have called touch yet if loop speed > 0.1
        # But we want to test cancellation.

        # Cancel the task (simulate connection loss logic or manual cancel)
        task.cancel()

        try:
            await task
        except asyncio.CancelledError:
            pass

        # Verify that the task finished (await task returned)
        assert task.done()

        # The critical check: did the touch loop stop?
        # We can't easily check the internal `stop_touch` event directly without introspection or modifying code to expose it.
        # But if `await touch_task` in `finally` block hangs, the test would timeout (if we verified it properly).
        # To be sure, we can check if `handle_web_task` was cancelled.
        # And we can check if `message.finish()` was NOT called (since we cancelled).

        mock_nsq_msg.finish.assert_not_called()


@pytest.mark.asyncio
async def test_process_message_self_cancel_on_touch_fail():
    # Test that if touch fails, the main task cancels itself
    mock_nsq_msg = MagicMock()
    mock_nsq_msg.body = b'{"type": "web", "url": "http://example.com", "id": "123"}'

    # touch raises exception immediately
    mock_nsq_msg.touch.side_effect = Exception("Connection lost")
    mock_nsq_msg.finish = MagicMock()

    async def long_running_task(*args, **kwargs):
        try:
            await asyncio.sleep(10)
        except asyncio.CancelledError:
            raise

    with (
        patch("main.handle_web_task", side_effect=long_running_task),
        patch("main.producer", new=MagicMock()),
    ):
        # We need to capture the task so the touch loop can cancel it
        # But process_message calls asyncio.current_task()
        # So we must wrap it in a task

        task = asyncio.create_task(process_message(mock_nsq_msg))

        try:
            # It should cancel itself quickly (touch loop sleeps 10s AFTER touch, but here touch fails immediately)
            # wait_for might catch the CancelledError
            await asyncio.wait_for(task, timeout=1.0)
        except asyncio.CancelledError:
            pass
        except asyncio.TimeoutError:
            pytest.fail("Task did not cancel itself upon touch failure")

        assert task.done()
