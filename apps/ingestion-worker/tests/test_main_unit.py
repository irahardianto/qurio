"""
Comprehensive unit tests for main.py process_message function.
Tests cover all happy paths, error paths, and edge cases.
"""

import pytest
import json
import asyncio
from unittest.mock import MagicMock, AsyncMock, patch
from exceptions import IngestionError, ERR_ENCRYPTED, ERR_TIMEOUT, ERR_CRAWL_TIMEOUT
from main import process_message, _is_transient_error


# --- Tests for _is_transient_error function ---


def test_is_transient_error_asyncio_timeout():
    """Test that asyncio.TimeoutError is detected as transient."""
    assert _is_transient_error(asyncio.TimeoutError()) is True


def test_is_transient_error_ingestion_error_transient():
    """Test that IngestionError with transient code is detected."""
    err = IngestionError(ERR_CRAWL_TIMEOUT, "Timeout")
    assert _is_transient_error(err) is True


def test_is_transient_error_ingestion_error_permanent():
    """Test that IngestionError with permanent code is not transient."""
    err = IngestionError(ERR_ENCRYPTED, "Encrypted")
    assert _is_transient_error(err) is False


def test_is_transient_error_timeout_in_string():
    """Test that 'TIMEOUT' in error string is detected as transient."""
    err = Exception("Connection TIMEOUT occurred")
    assert _is_transient_error(err) is True


def test_is_transient_error_timed_out_in_string():
    """Test that 'TIMED_OUT' in error string is detected as transient."""
    err = Exception("Request TIMED_OUT")
    assert _is_transient_error(err) is True


def test_is_transient_error_connection_in_string():
    """Test that 'CONNECTION' in error string is detected as transient."""
    err = Exception("CONNECTION refused")
    assert _is_transient_error(err) is True


def test_is_transient_error_err_name_not_resolved():
    """Test that ERR_NAME_NOT_RESOLVED is detected as transient."""
    err = Exception("ERR_NAME_NOT_RESOLVED")
    assert _is_transient_error(err) is True


def test_is_transient_error_econnrefused():
    """Test that ECONNREFUSED is detected as transient."""
    err = Exception("ECONNREFUSED")
    assert _is_transient_error(err) is True


def test_is_transient_error_permanent():
    """Test that non-transient errors return False."""
    err = Exception("Invalid format")
    assert _is_transient_error(err) is False


# --- Tests for process_message function ---


@pytest.mark.asyncio
async def test_process_message_malformed_json():
    """Test handling of malformed JSON message."""
    mock_msg = MagicMock()
    mock_msg.body = b"not valid json"
    mock_msg.attempts = 1  # Add attempts attribute

    with patch("main.producer"):
        with patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)):
            await process_message(mock_msg)

            # Should finish the message despite error
            mock_msg.finish.assert_called()


@pytest.mark.asyncio
async def test_process_message_missing_id():
    """Test handling of message missing required 'id' field."""
    mock_msg = MagicMock()
    mock_msg.body = json.dumps({"type": "web", "url": "http://example.com"}).encode(
        "utf-8"
    )
    mock_msg.attempts = 1  # Add attempts attribute

    with patch("main.producer"):
        with patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)):
            with patch(
                "main.get_crawler", new_callable=AsyncMock, return_value=MagicMock()
            ):
                with patch(
                    "main.handle_web_task", new_callable=AsyncMock
                ) as mock_handle:
                    mock_handle.return_value = []
                    await process_message(mock_msg)

                    # Should still finish
                    mock_msg.finish.assert_called()


@pytest.mark.asyncio
async def test_process_message_missing_type():
    """Test handling of message missing 'type' field."""
    mock_msg = MagicMock()
    mock_msg.body = json.dumps({"id": "123", "url": "http://example.com"}).encode(
        "utf-8"
    )
    mock_msg.attempts = 1  # Add attempts attribute
    mock_msg.attempts = 1  # Add attempts attribute

    with patch("main.producer"):
        with patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)):
            await process_message(mock_msg)

            mock_msg.finish.assert_called()


@pytest.mark.asyncio
async def test_process_message_unknown_task_type():
    """Test handling of unknown task type."""
    mock_msg = MagicMock()
    mock_msg.body = json.dumps({"id": "123", "type": "unknown"}).encode("utf-8")
    mock_msg.attempts = 1  # Add attempts attribute

    with patch("main.producer"):
        with patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)):
            await process_message(mock_msg)

            # Should finish without processing
            mock_msg.finish.assert_called()


@pytest.mark.asyncio
async def test_process_message_web_task_success():
    """Test successful web task processing."""
    mock_msg = MagicMock()
    mock_msg.body = json.dumps(
        {"id": "123", "type": "web", "url": "http://example.com"}
    ).encode("utf-8")

    with patch("main.get_crawler", new_callable=AsyncMock, return_value=MagicMock()):
        with patch("main.handle_web_task", new_callable=AsyncMock) as mock_handle:
            mock_handle.return_value = [
                {
                    "content": "test",
                    "url": "http://example.com",
                    "title": "Test",
                    "metadata": {},
                    "links": [],
                }
            ]

            with patch("main.producer") as mock_producer:
                with patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)):
                    await process_message(mock_msg)

                    mock_msg.finish.assert_called()
                    mock_producer.pub.assert_called()


@pytest.mark.asyncio
async def test_process_message_file_task_success():
    """Test successful file task processing."""
    mock_msg = MagicMock()
    mock_msg.body = json.dumps(
        {"id": "123", "type": "file", "path": "/tmp/test.pdf"}
    ).encode("utf-8")

    with patch("main.handle_file_task", new_callable=AsyncMock) as mock_handle:
        mock_handle.return_value = [
            {
                "content": "test",
                "url": "/tmp/test.pdf",
                "path": "/tmp/test.pdf",
                "title": "Test",
                "metadata": {},
                "links": [],
            }
        ]

        with patch("main.producer") as mock_producer:
            with patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)):
                await process_message(mock_msg)

                mock_msg.finish.assert_called()
                mock_producer.pub.assert_called()


@pytest.mark.asyncio
async def test_process_message_empty_results():
    """Test handling when handler returns empty results."""
    mock_msg = MagicMock()
    mock_msg.body = json.dumps(
        {"id": "123", "type": "web", "url": "http://example.com"}
    ).encode("utf-8")

    with patch("main.get_crawler", new_callable=AsyncMock, return_value=MagicMock()):
        with patch("main.handle_web_task", new_callable=AsyncMock) as mock_handle:
            mock_handle.return_value = []  # Empty results

            with patch("main.producer") as mock_producer:
                with patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)):
                    await process_message(mock_msg)

                    # Should publish failure
                    args, kwargs = mock_producer.pub.call_args
                    payload = json.loads(args[1])
                    assert payload["status"] == "failed"
                    assert payload["error"] == "No content extracted"
                    mock_msg.finish.assert_called()


@pytest.mark.asyncio
async def test_process_message_producer_publish_failure():
    """Test handling when producer.pub fails."""
    mock_msg = MagicMock()
    mock_msg.body = json.dumps(
        {"id": "123", "type": "web", "url": "http://example.com"}
    ).encode("utf-8")

    with patch("main.get_crawler", new_callable=AsyncMock, return_value=MagicMock()):
        with patch("main.handle_web_task", new_callable=AsyncMock) as mock_handle:
            mock_handle.return_value = [
                {
                    "content": "test",
                    "url": "http://example.com",
                    "title": "Test",
                    "metadata": {},
                    "links": [],
                }
            ]

            with patch("main.producer") as mock_producer:
                mock_producer.pub.side_effect = Exception("Publish failed")

                with patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)):
                    await process_message(mock_msg)

                    # Should still finish
                    mock_msg.finish.assert_called()


@pytest.mark.asyncio
async def test_process_message_finish_failure():
    """Test handling when message.finish() fails."""
    mock_msg = MagicMock()
    mock_msg.body = json.dumps(
        {"id": "123", "type": "web", "url": "http://example.com"}
    ).encode("utf-8")
    mock_msg.finish.side_effect = Exception("Finish failed")

    with patch("main.get_crawler", new_callable=AsyncMock, return_value=MagicMock()):
        with patch("main.handle_web_task", new_callable=AsyncMock) as mock_handle:
            mock_handle.return_value = [
                {
                    "content": "test",
                    "url": "http://example.com",
                    "title": "Test",
                    "metadata": {},
                    "links": [],
                }
            ]

            with patch("main.producer"):
                with patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)):
                    # Should not raise exception
                    await process_message(mock_msg)


@pytest.mark.asyncio
async def test_process_message_cancelled_error():
    """Test handling of CancelledError."""
    mock_msg = MagicMock()
    mock_msg.body = json.dumps(
        {"id": "123", "type": "web", "url": "http://example.com"}
    ).encode("utf-8")

    with patch("main.get_crawler", new_callable=AsyncMock, return_value=MagicMock()):
        with patch("main.handle_web_task", new_callable=AsyncMock) as mock_handle:
            mock_handle.side_effect = asyncio.CancelledError()

            with patch("main.producer"):
                with patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)):
                    await process_message(mock_msg)

                    # Should not finish or requeue
                    mock_msg.finish.assert_not_called()
                    mock_msg.requeue.assert_not_called()


@pytest.mark.asyncio
async def test_process_message_touch_failure():
    """Test that touch failures are logged but don't crash processing."""
    from tornado.iostream import StreamClosedError

    mock_msg = MagicMock()
    mock_msg.body = json.dumps(
        {"id": "123", "type": "web", "url": "http://example.com"}
    ).encode("utf-8")
    mock_msg.touch.side_effect = StreamClosedError()

    with patch("main.get_crawler", new_callable=AsyncMock, return_value=MagicMock()):
        with patch("main.handle_web_task", new_callable=AsyncMock) as mock_handle:
            mock_handle.return_value = [
                {
                    "content": "test",
                    "url": "http://example.com",
                    "title": "Test",
                    "metadata": {},
                    "links": [],
                }
            ]

            with patch("main.producer"):
                with patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)):
                    # Processing should be cancelled due to touch failure
                    await process_message(mock_msg)


@pytest.mark.asyncio
async def test_process_message_includes_original_payload_on_failure():
    """Test that failure messages include original payload for retry."""
    mock_msg = MagicMock()
    mock_msg.attempts = 4  # Exceeded max retries
    original_data = {
        "id": "123",
        "type": "web",
        "url": "http://example.com",
        "depth": 2,
    }
    mock_msg.body = json.dumps(original_data).encode("utf-8")

    with patch("main.get_crawler", new_callable=AsyncMock, return_value=MagicMock()):
        with patch("main.handle_web_task", new_callable=AsyncMock) as mock_handle:
            mock_handle.side_effect = IngestionError(ERR_TIMEOUT, "Timeout")

            with patch("main.producer") as mock_producer:
                with patch("main.settings") as mock_settings:
                    mock_settings.retry_max_attempts = 3

                    with patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)):
                        await process_message(mock_msg)

                        # Check that original_payload is included
                        args, kwargs = mock_producer.pub.call_args
                        payload = json.loads(args[1])
                        assert "original_payload" in payload
                        assert payload["original_payload"] == original_data
                        mock_msg.finish.assert_called()


@pytest.mark.asyncio
async def test_process_message_concurrent_semaphore():
    """Test that semaphore limits concurrent processing."""
    mock_msg1 = MagicMock()
    mock_msg1.body = json.dumps(
        {"id": "1", "type": "web", "url": "http://example.com"}
    ).encode("utf-8")

    mock_msg2 = MagicMock()
    mock_msg2.body = json.dumps(
        {"id": "2", "type": "web", "url": "http://example.com"}
    ).encode("utf-8")

    semaphore = asyncio.Semaphore(1)
    processing_count = 0
    max_concurrent = 0

    async def slow_handler(*args, **kwargs):
        nonlocal processing_count, max_concurrent
        processing_count += 1
        max_concurrent = max(max_concurrent, processing_count)
        await asyncio.sleep(0.1)
        processing_count -= 1
        return [
            {
                "content": "test",
                "url": "http://example.com",
                "title": "Test",
                "metadata": {},
                "links": [],
            }
        ]

    with patch("main.get_crawler", new_callable=AsyncMock, return_value=MagicMock()):
        with patch("main.handle_web_task", new_callable=AsyncMock) as mock_handle:
            mock_handle.side_effect = slow_handler

            with patch("main.producer"):
                with patch("main.WORKER_SEMAPHORE", semaphore):
                    # Process both messages concurrently
                    await asyncio.gather(
                        process_message(mock_msg1), process_message(mock_msg2)
                    )

                    # Semaphore should have limited to 1 concurrent
                    assert max_concurrent == 1


# --- New Tests: API Key Redaction, Duration Logging ---


@pytest.mark.asyncio
async def test_process_message_redacts_api_key():
    """API key should NOT appear in logged payload."""

    mock_msg = MagicMock()
    mock_msg.body = json.dumps(
        {
            "id": "123",
            "type": "web",
            "url": "http://example.com",
            "gemini_api_key": "sk-SECRET-KEY-12345",
        }
    ).encode("utf-8")

    with patch("main.get_crawler", new_callable=AsyncMock, return_value=MagicMock()):
        with patch("main.handle_web_task", new_callable=AsyncMock) as mock_handle:
            mock_handle.return_value = [
                {
                    "content": "test",
                    "url": "http://example.com",
                    "title": "Test",
                    "metadata": {},
                    "links": [],
                }
            ]

            with patch("main.producer"):
                with patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)):
                    with patch("main.logger") as mock_logger:
                        await process_message(mock_msg)

                        # Find the "message_received" log call
                        for call_obj in mock_logger.info.call_args_list:
                            if call_obj.args and call_obj.args[0] == "message_received":
                                logged_data = call_obj.kwargs.get("data", {})
                                assert "gemini_api_key" not in logged_data, (
                                    "API key must be redacted from logged data"
                                )
                                break


@pytest.mark.asyncio
async def test_process_message_logs_duration():
    """Completion log should include duration_ms field."""
    mock_msg = MagicMock()
    mock_msg.body = json.dumps(
        {"id": "123", "type": "file", "path": "/tmp/test.pdf"}
    ).encode("utf-8")

    with patch("main.handle_file_task", new_callable=AsyncMock) as mock_handle:
        mock_handle.return_value = [
            {
                "content": "test",
                "url": "/tmp/test.pdf",
                "path": "/tmp/test.pdf",
                "title": "Test",
                "metadata": {},
                "links": [],
            }
        ]

        with patch("main.producer"):
            with patch("main.WORKER_SEMAPHORE", asyncio.Semaphore(1)):
                with patch("main.logger") as mock_logger:
                    await process_message(mock_msg)

                    # Find "message_processed" log call
                    found_duration = False
                    for call_obj in mock_logger.info.call_args_list:
                        if call_obj.args and call_obj.args[0] == "message_processed":
                            assert "duration_ms" in call_obj.kwargs, (
                                "message_processed log must include duration_ms"
                            )
                            assert isinstance(call_obj.kwargs["duration_ms"], float)
                            found_duration = True
                            break
                    assert found_duration, "message_processed log event not found"
