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
