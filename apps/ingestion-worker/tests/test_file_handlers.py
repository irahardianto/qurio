import pytest
from unittest.mock import MagicMock, patch
import asyncio
import handlers.file
from handlers.file import (
    handle_file_task,
    ERR_ENCRYPTED,
    ERR_EMPTY,
    ERR_INVALID_FORMAT,
    ERR_TIMEOUT,
    MAX_FILE_SIZE_BYTES,
    IngestionError,
    _get_executor,
)


# Helper to create a done future for asyncio.wrap_future
def create_done_future(result=None, exception=None):
    f = asyncio.Future()
    if exception:
        f.set_exception(exception)
    else:
        f.set_result(result)
    return f


# --- File Validation Tests (NEW) ---


@pytest.mark.asyncio
async def test_handle_file_task_file_not_found():
    """File not found raises ERR_INVALID_FORMAT."""
    with patch("handlers.file.os.path.isfile", return_value=False):
        with pytest.raises(IngestionError) as exc:
            await handle_file_task("/path/to/missing.pdf")
        assert exc.value.code == ERR_INVALID_FORMAT
        assert "not found" in str(exc.value).lower()


@pytest.mark.asyncio
async def test_handle_file_task_zero_byte_file():
    """Empty file (0 bytes) raises ERR_EMPTY."""
    with patch("handlers.file.os.path.isfile", return_value=True):
        with patch("handlers.file.os.path.getsize", return_value=0):
            with pytest.raises(IngestionError) as exc:
                await handle_file_task("/path/to/empty.pdf")
            assert exc.value.code == ERR_EMPTY
            assert "0 bytes" in str(exc.value)


@pytest.mark.asyncio
async def test_handle_file_task_file_too_large():
    """Oversized file raises ERR_INVALID_FORMAT."""
    with patch("handlers.file.os.path.isfile", return_value=True):
        with patch(
            "handlers.file.os.path.getsize", return_value=MAX_FILE_SIZE_BYTES + 1
        ):
            with pytest.raises(IngestionError) as exc:
                await handle_file_task("/path/to/huge.pdf")
            assert exc.value.code == ERR_INVALID_FORMAT
            assert "too large" in str(exc.value).lower()


# --- ProcessPool Recovery Test (NEW) ---


@pytest.mark.asyncio
async def test_get_executor_recreates_on_inactive():
    """Pool manager recreates the pool when it becomes inactive."""
    import pebble

    mock_pool_1 = MagicMock(spec=pebble.ProcessPool)
    mock_pool_1.active = False  # Simulate broken pool
    mock_pool_2 = MagicMock(spec=pebble.ProcessPool)
    mock_pool_2.active = True

    with patch("handlers.file.pebble.ProcessPool", side_effect=[mock_pool_2]):
        # Set module state to broken pool
        handlers.file._executor = mock_pool_1
        pool = _get_executor()
        # Should have created a new pool since mock_pool_1.active is False
        assert pool == mock_pool_2


# --- Existing Tests (updated to mock _get_executor and file validation) ---


def _patch_valid_file_and_executor(future_result=None, future_exception=None):
    """Return nested context managers for valid file + mocked executor + wrap_future."""
    mock_pool = MagicMock()
    mock_pool.schedule.return_value = MagicMock()

    return (
        patch("handlers.file.os.path.isfile", return_value=True),
        patch("handlers.file.os.path.getsize", return_value=1024),
        patch("handlers.file._get_executor", return_value=mock_pool),
        patch(
            "handlers.file.asyncio.wrap_future",
            return_value=create_done_future(
                result=future_result, exception=future_exception
            ),
        ),
    )


@pytest.mark.asyncio
async def test_handle_encrypted_pdf():
    """Test handling of encrypted PDF files."""
    p1, p2, p3, p4 = _patch_valid_file_and_executor(
        future_exception=Exception("File is password protected")
    )
    with p1, p2, p3, p4:
        with pytest.raises(IngestionError) as excinfo:
            await handle_file_task("/tmp/secret.pdf")
        assert excinfo.value.code == ERR_ENCRYPTED


@pytest.mark.asyncio
async def test_metadata_extraction():
    """Test successful metadata extraction."""
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
    p1, p2, p3, p4 = _patch_valid_file_and_executor(future_result=expected_result)
    with p1, p2, p3, p4:
        result = await handle_file_task("/tmp/test.pdf")
        assert result[0]["metadata"]["title"] == "Test Title"
        assert result[0]["metadata"]["pages"] == 10
        assert result[0]["content"] == "# Content"


@pytest.mark.asyncio
async def test_timeout():
    """Test timeout handling."""
    p1, p2, p3, p4 = _patch_valid_file_and_executor(
        future_exception=asyncio.TimeoutError()
    )
    with p1, p2, p3, p4:
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
    p1, p2, p3, p4 = _patch_valid_file_and_executor(
        future_result=simulated_worker_output
    )
    with p1, p2, p3, p4:
        result = await handle_file_task("/var/lib/qurio/uploads/restful.pdf")
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
    p1, p2, p3, p4 = _patch_valid_file_and_executor(
        future_result=simulated_worker_output
    )
    with p1, p2, p3, p4:
        result = await handle_file_task("/path/to/file.pdf")
        assert isinstance(result, list), "Expected result to be a list"
        assert len(result) == 1
        item = result[0]
        assert item["path"] == "/path/to/file.pdf"
        assert item["url"] == "/path/to/file.pdf"
        assert item["title"] == "Test"
        assert item["links"] == []


@pytest.mark.asyncio
async def test_handle_file_task_empty_content():
    """Test that empty content raises ERR_EMPTY."""
    simulated_worker_output = {"content": "   ", "metadata": {"title": "Empty"}}
    p1, p2, p3, p4 = _patch_valid_file_and_executor(
        future_result=simulated_worker_output
    )
    with p1, p2, p3, p4:
        with pytest.raises(IngestionError) as exc:
            await handle_file_task("/path/to/empty.pdf")
        assert exc.value.code == ERR_EMPTY


@pytest.mark.asyncio
async def test_handle_file_task_invalid_format():
    """Test handling of invalid format errors."""
    p1, p2, p3, p4 = _patch_valid_file_and_executor(
        future_exception=Exception("Invalid format detected")
    )
    with p1, p2, p3, p4:
        with pytest.raises(IngestionError) as exc:
            await handle_file_task("/path/to/invalid.pdf")
        assert exc.value.code == ERR_INVALID_FORMAT


@pytest.mark.asyncio
async def test_handle_file_task_process_expired():
    """Test handling of ProcessExpired exception."""
    import pebble

    p1, p2, p3, p4 = _patch_valid_file_and_executor(
        future_exception=pebble.ProcessExpired("Worker process died")
    )
    with p1, p2, p3, p4:
        with pytest.raises(IngestionError) as exc:
            await handle_file_task("/path/to/file.pdf")
        assert exc.value.code == ERR_TIMEOUT


@pytest.mark.asyncio
async def test_handle_file_task_timeout_in_error_message():
    """Test that timeout keyword in error message raises ERR_TIMEOUT."""
    p1, p2, p3, p4 = _patch_valid_file_and_executor(
        future_exception=Exception("Operation timeout exceeded")
    )
    with p1, p2, p3, p4:
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
            "author": None,
            "pages": 0,
            "created_at": None,
            "language": "en",
        },
    }
    p1, p2, p3, p4 = _patch_valid_file_and_executor(future_result=expected_result)
    with p1, p2, p3, p4:
        result = await handle_file_task("/tmp/test.pdf")
        assert result[0]["metadata"]["title"] == "Test Title"
        assert result[0]["metadata"]["author"] is None
        assert result[0]["metadata"]["pages"] == 0


@pytest.mark.asyncio
async def test_handle_file_task_generic_exception():
    """Test that generic exceptions are re-raised."""
    p1, p2, p3, p4 = _patch_valid_file_and_executor(
        future_exception=RuntimeError("Unexpected error")
    )
    with p1, p2, p3, p4:
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
    p1, p2, p3, p4 = _patch_valid_file_and_executor(future_result=expected_result)
    with p1, p2, p3, p4:
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
    p1, p2, p3, p4 = _patch_valid_file_and_executor(
        future_exception=Exception("Document requires password")
    )
    with p1, p2, p3, p4:
        with pytest.raises(IngestionError) as exc:
            await handle_file_task("/path/to/protected.pdf")
        assert exc.value.code == ERR_ENCRYPTED


@pytest.mark.asyncio
async def test_handle_file_task_concurrent_timeout_error():
    """Test TimeoutError from concurrent.futures."""
    from concurrent.futures import TimeoutError as ConcurrentTimeoutError

    p1, p2, p3, p4 = _patch_valid_file_and_executor(
        future_exception=ConcurrentTimeoutError()
    )
    with p1, p2, p3, p4:
        with pytest.raises(IngestionError) as exc:
            await handle_file_task("/path/to/large.pdf")
        assert exc.value.code == ERR_TIMEOUT
