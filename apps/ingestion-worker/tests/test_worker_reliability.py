import asyncio
import pytest
from unittest.mock import MagicMock
from tornado.iostream import StreamClosedError
import nsq


# This is the logic we WANT to implement in main.py
# We extract it here to unit test the concept before applying it to the main application
async def robust_touch_loop(message, stop_event, cancel_callback):
    while not stop_event.is_set():
        try:
            message.touch()
        except (nsq.Error, StreamClosedError, Exception):
            # If touch fails fatally, we should cancel the processing task
            if cancel_callback:
                cancel_callback()
            return
        # Use a small sleep for testing speed, in prod this would be longer
        await asyncio.sleep(0.01)


@pytest.mark.asyncio
async def test_touch_loop_cancels_on_error():
    # Arrange
    mock_message = MagicMock()
    # Simulate a fatal error on the first touch attempt
    mock_message.touch.side_effect = StreamClosedError(
        real_error=Exception("Stream closed")
    )

    stop_event = asyncio.Event()
    cancel_called = False

    def cancel_cb():
        nonlocal cancel_called
        cancel_called = True
        stop_event.set()

    # Act
    await robust_touch_loop(mock_message, stop_event, cancel_cb)

    # Assert
    assert cancel_called is True
    assert stop_event.is_set()
