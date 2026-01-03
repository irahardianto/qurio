import pytest
from unittest.mock import MagicMock, patch
import asyncio
from concurrent.futures import Future
from handlers.file import handle_file_task, ERR_ENCRYPTED, ERR_INVALID_FORMAT, ERR_TIMEOUT, IngestionError

@pytest.mark.asyncio
async def test_handle_encrypted_pdf():
    with patch('handlers.file.converter') as mock_converter:
        mock_converter.convert.side_effect = Exception("Encrypted") # Simulating docling error
        
        with pytest.raises(Exception) as excinfo:
             await handle_file_task("/tmp/secret.pdf")
        
        assert excinfo.value.code == ERR_ENCRYPTED

@pytest.mark.asyncio
async def test_metadata_extraction():
     with patch('handlers.file.converter') as mock_converter:
        mock_doc = MagicMock()
        mock_doc.document.export_to_markdown.return_value = "# Content"
        mock_doc.document.meta.title = "Test Title"
        mock_doc.document.meta.author = "Test Author"
        mock_doc.document.num_pages = 10
        mock_converter.convert.return_value = mock_doc
        
        result = await handle_file_task("/tmp/test.pdf")
        
        assert result['metadata']['title'] == "Test Title"
        assert result['metadata']['pages'] == 10

@pytest.mark.asyncio
async def test_timeout():
    # Verify timeout handling
    with patch('handlers.file.executor') as mock_executor:
         mock_executor.submit.return_value = Future()
         with patch('asyncio.wait_for', side_effect=asyncio.TimeoutError):
             with pytest.raises(IngestionError) as exc:
                 await handle_file_task("/tmp/slow.pdf")
             assert exc.value.code == ERR_TIMEOUT

@pytest.mark.asyncio
async def test_concurrency_limit():
    # Verify semaphore existence and value
    from handlers.file import CONCURRENCY_LIMIT
    assert isinstance(CONCURRENCY_LIMIT, asyncio.Semaphore)
    # Note: In CPython asyncio.Semaphore internal value is _value, but might be different implementation.
    # Safest is to check we can acquire it.
    assert CONCURRENCY_LIMIT._value == 2
