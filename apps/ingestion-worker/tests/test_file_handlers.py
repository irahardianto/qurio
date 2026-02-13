import pytest
from unittest.mock import MagicMock, patch
import asyncio
import handlers.file
from handlers.file import (
    handle_file_task,
    ERR_ENCRYPTED,
    ERR_TIMEOUT,
    IngestionError,
)


# Helper to create a done future for asyncio.wrap_future
def create_done_future(result=None, exception=None):
    f = asyncio.Future()
    if exception:
        f.set_exception(exception)
    else:
        f.set_result(result)
    return f


@pytest.mark.asyncio
async def test_handle_encrypted_pdf():
    """Test handling of encrypted PDF files."""
    # Patch the executor instance in handlers.file
    with patch.object(handlers.file, "executor") as mock_executor:
        mock_future = MagicMock()
        mock_executor.schedule.return_value = mock_future

        # Patch asyncio.wrap_future in handlers.file
        with patch("handlers.file.asyncio.wrap_future") as mock_wrap:
            mock_wrap.return_value = create_done_future(
                exception=Exception("File is password protected")
            )

            with pytest.raises(IngestionError) as excinfo:
                await handle_file_task("/tmp/secret.pdf")

            assert excinfo.value.code == ERR_ENCRYPTED


@pytest.mark.asyncio
async def test_metadata_extraction():
    """Test successful metadata extraction."""
    # Match the structure expected by the updated handler (Docling v2 style)
    expected_result = {
        "content": "# Content",
        "metadata": {
            "title": "Test Title",
            "author": "Test Author",
            "pages": 10,
            "created_at": None,
            "language": "en",
        },
    }

    with patch.object(handlers.file, "executor") as mock_executor:
        mock_future = MagicMock()
        mock_executor.schedule.return_value = mock_future

        with patch("handlers.file.asyncio.wrap_future") as mock_wrap:
            mock_wrap.return_value = create_done_future(result=expected_result)

            result = await handle_file_task("/tmp/test.pdf")

            assert result[0]["metadata"]["title"] == "Test Title"
            assert result[0]["metadata"]["pages"] == 10
            assert result[0]["content"] == "# Content"


@pytest.mark.asyncio
async def test_timeout():
    """Test timeout handling."""
    with patch.object(handlers.file, "executor") as mock_executor:
        mock_executor.schedule.return_value = MagicMock()

        with patch("handlers.file.asyncio.wrap_future") as mock_wrap:
            mock_wrap.return_value = create_done_future(
                exception=asyncio.TimeoutError()
            )

            with pytest.raises(IngestionError) as exc:
                await handle_file_task("/tmp/slow.pdf")
            assert exc.value.code == ERR_TIMEOUT


@pytest.mark.asyncio
async def test_end_to_end_pdf_simulation():
    """Simulate a full PDF upload flow."""
    simulated_worker_output = {
        "content": "# Chapter 1\nRESTful Web Services...",
        "metadata": {
            "title": "RESTful Web Services",
            "author": "Leonard Richardson",
            "created_at": "2023-01-01",
            "pages": 450,
            "language": "en",
        },
    }

    with patch.object(handlers.file, "executor") as mock_executor:
        mock_executor.schedule.return_value = MagicMock()

        with patch("handlers.file.asyncio.wrap_future") as mock_wrap:
            mock_wrap.return_value = create_done_future(result=simulated_worker_output)

            # Execute
            result = await handle_file_task("/var/lib/qurio/uploads/restful.pdf")

            # Verify structure matches backend expectations
            assert len(result) == 1
            item = result[0]
            assert "content" in item
            assert "metadata" in item
            assert item["metadata"]["title"] == "RESTful Web Services"
            assert item["metadata"]["pages"] == 450
            assert "Chapter 1" in item["content"]


@pytest.mark.asyncio
async def test_handle_file_task_returns_list_structure():
    """Test that handle_file_task returns a list with path field."""
    simulated_worker_output = {"content": "some content", "metadata": {"title": "Test"}}

    with patch.object(handlers.file, "executor") as mock_executor:
        mock_executor.schedule.return_value = MagicMock()

        with patch("handlers.file.asyncio.wrap_future") as mock_wrap:
            mock_wrap.return_value = create_done_future(result=simulated_worker_output)

            result = await handle_file_task("/path/to/file.pdf")

            assert isinstance(result, list), "Expected result to be a list"
            assert len(result) == 1
            item = result[0]
            assert item["path"] == "/path/to/file.pdf"
            assert item["url"] == "/path/to/file.pdf"
            assert item["title"] == "Test"
            assert item["links"] == []


# --- Additional Edge Cases and Error Path Tests ---


@pytest.mark.asyncio
async def test_handle_file_task_empty_content():
    """Test that empty content raises ERR_EMPTY."""
    from handlers.file import ERR_EMPTY

    simulated_worker_output = {"content": "   ", "metadata": {"title": "Empty"}}

    with patch.object(handlers.file, "executor") as mock_executor:
        mock_executor.schedule.return_value = MagicMock()

        with patch("handlers.file.asyncio.wrap_future") as mock_wrap:
            mock_wrap.return_value = create_done_future(result=simulated_worker_output)

            with pytest.raises(IngestionError) as exc:
                await handle_file_task("/path/to/empty.pdf")
            assert exc.value.code == ERR_EMPTY


@pytest.mark.asyncio
async def test_handle_file_task_invalid_format():
    """Test handling of invalid format errors."""
    from handlers.file import ERR_INVALID_FORMAT

    with patch.object(handlers.file, "executor") as mock_executor:
        mock_executor.schedule.return_value = MagicMock()

        with patch("handlers.file.asyncio.wrap_future") as mock_wrap:
            mock_wrap.return_value = create_done_future(
                exception=Exception("Invalid format detected")
            )

            with pytest.raises(IngestionError) as exc:
                await handle_file_task("/path/to/invalid.pdf")
            assert exc.value.code == ERR_INVALID_FORMAT


@pytest.mark.asyncio
async def test_handle_file_task_process_expired():
    """Test handling of ProcessExpired exception."""
    import pebble

    with patch.object(handlers.file, "executor") as mock_executor:
        mock_executor.schedule.return_value = MagicMock()

        with patch("handlers.file.asyncio.wrap_future") as mock_wrap:
            mock_wrap.return_value = create_done_future(
                exception=pebble.ProcessExpired("Worker process died")
            )

            with pytest.raises(IngestionError) as exc:
                await handle_file_task("/path/to/file.pdf")
            assert exc.value.code == ERR_TIMEOUT


@pytest.mark.asyncio
async def test_handle_file_task_timeout_in_error_message():
    """Test that timeout keyword in error message raises ERR_TIMEOUT."""
    with patch.object(handlers.file, "executor") as mock_executor:
        mock_executor.schedule.return_value = MagicMock()

        with patch("handlers.file.asyncio.wrap_future") as mock_wrap:
            mock_wrap.return_value = create_done_future(
                exception=Exception("Operation timeout exceeded")
            )

            with pytest.raises(IngestionError) as exc:
                await handle_file_task("/path/to/slow.pdf")
            assert exc.value.code == ERR_TIMEOUT


@pytest.mark.asyncio
async def test_handle_file_task_metadata_with_missing_fields():
    """Test handling when metadata has missing optional fields."""
    expected_result = {
        "content": "# Content",
        "metadata": {
            "title": "Test Title",
            "author": None,  # Missing
            "pages": 0,  # Missing
            "created_at": None,  # Missing
            "language": "en",
        },
    }

    with patch.object(handlers.file, "executor") as mock_executor:
        mock_future = MagicMock()
        mock_executor.schedule.return_value = mock_future

        with patch("handlers.file.asyncio.wrap_future") as mock_wrap:
            mock_wrap.return_value = create_done_future(result=expected_result)

            result = await handle_file_task("/tmp/test.pdf")

            assert result[0]["metadata"]["title"] == "Test Title"
            assert result[0]["metadata"]["author"] is None
            assert result[0]["metadata"]["pages"] == 0


@pytest.mark.asyncio
async def test_handle_file_task_generic_exception():
    """Test that generic exceptions are re-raised."""
    with patch.object(handlers.file, "executor") as mock_executor:
        mock_executor.schedule.return_value = MagicMock()

        with patch("handlers.file.asyncio.wrap_future") as mock_wrap:
            mock_wrap.return_value = create_done_future(
                exception=RuntimeError("Unexpected error")
            )

            with pytest.raises(RuntimeError, match="Unexpected error"):
                await handle_file_task("/path/to/file.pdf")


@pytest.mark.asyncio
async def test_handle_file_task_successful_with_all_metadata():
    """Test successful processing with complete metadata."""
    expected_result = {
        "content": "# Complete Document\nFull content here",
        "metadata": {
            "title": "Complete Title",
            "author": "John Doe, Jane Smith",
            "created_at": "2024-01-01",
            "pages": 100,
            "language": "en",
        },
    }

    with patch.object(handlers.file, "executor") as mock_executor:
        mock_executor.schedule.return_value = MagicMock()

        with patch("handlers.file.asyncio.wrap_future") as mock_wrap:
            mock_wrap.return_value = create_done_future(result=expected_result)

            result = await handle_file_task("/path/to/complete.pdf")

            assert len(result) == 1
            item = result[0]
            assert item["content"] == "# Complete Document\nFull content here"
            assert item["metadata"]["title"] == "Complete Title"
            assert item["metadata"]["author"] == "John Doe, Jane Smith"
            assert item["metadata"]["created_at"] == "2024-01-01"
            assert item["metadata"]["pages"] == 100
            assert item["metadata"]["language"] == "en"
            assert item["url"] == "/path/to/complete.pdf"
            assert item["path"] == "/path/to/complete.pdf"


@pytest.mark.asyncio
async def test_handle_file_task_encrypted_with_password_keyword():
    """Test that 'password' keyword in error triggers ERR_ENCRYPTED."""
    with patch.object(handlers.file, "executor") as mock_executor:
        mock_executor.schedule.return_value = MagicMock()

        with patch("handlers.file.asyncio.wrap_future") as mock_wrap:
            mock_wrap.return_value = create_done_future(
                exception=Exception("Document requires password")
            )

            with pytest.raises(IngestionError) as exc:
                await handle_file_task("/path/to/protected.pdf")
            assert exc.value.code == ERR_ENCRYPTED


@pytest.mark.asyncio
async def test_handle_file_task_concurrent_timeout_error():
    """Test TimeoutError from concurrent.futures."""
    from concurrent.futures import TimeoutError as ConcurrentTimeoutError

    with patch.object(handlers.file, "executor") as mock_executor:
        mock_executor.schedule.return_value = MagicMock()

        with patch("handlers.file.asyncio.wrap_future") as mock_wrap:
            mock_wrap.return_value = create_done_future(
                exception=ConcurrentTimeoutError()
            )

            with pytest.raises(IngestionError) as exc:
                await handle_file_task("/path/to/large.pdf")
            assert exc.value.code == ERR_TIMEOUT
