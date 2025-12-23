import pytest
from unittest.mock import MagicMock, AsyncMock, patch, ANY
from handlers.web import handle_web_task
from handlers.file import handle_file_task

@pytest.mark.asyncio
async def test_handle_web_task_success():
    # Mock result
    mock_result = MagicMock()
    mock_result.success = True
    mock_result.markdown = "# Test Content"
    mock_result.url = "http://example.com"
    
    # Mock crawler
    mock_crawler = AsyncMock()
    mock_crawler.arun.return_value = mock_result
    
    # Context manager mock
    mock_crawler_cm = AsyncMock()
    mock_crawler_cm.__aenter__.return_value = mock_crawler
    mock_crawler_cm.__aexit__.return_value = None
    
    with patch('handlers.web.AsyncWebCrawler', return_value=mock_crawler_cm) as MockCrawler:
        result = await handle_web_task("http://example.com")
        
        assert isinstance(result, list)
        assert len(result) == 1
        assert result[0]["content"] == "# Test Content"
        assert result[0]["url"] == "http://example.com"
        mock_crawler.arun.assert_called_with(url="http://example.com", config=ANY)

@pytest.mark.asyncio
async def test_handle_web_task_failure():
    # Mock result
    mock_result = MagicMock()
    mock_result.success = False
    mock_result.error_message = "Failed"
    
    # Mock crawler
    mock_crawler = AsyncMock()
    mock_crawler.arun.return_value = mock_result
    
    # Context manager mock
    mock_crawler_cm = AsyncMock()
    mock_crawler_cm.__aenter__.return_value = mock_crawler
    mock_crawler_cm.__aexit__.return_value = None
    
    with patch('handlers.web.AsyncWebCrawler', return_value=mock_crawler_cm) as MockCrawler:
        with pytest.raises(Exception, match="Crawl failed: Failed"):
            await handle_web_task("http://example.com")

@pytest.mark.asyncio
async def test_handle_file_task_success():
    # Mock converter result
    mock_result = MagicMock()
    mock_result.document.export_to_markdown.return_value = "# File Content"
    
    # Mock executor run
    # Since we can't easily mock run_in_executor with patch directly on the loop if we don't control the loop creation
    # We patch converter.convert to return the result, but since it's run in executor, we need to ensure the mock works.
    
    with patch('handlers.file.converter') as mock_converter:
        mock_converter.convert.return_value = mock_result
        
        # We need to mock the executor execution or trust that run_in_executor calls the function
        # A simpler way is to just call handle_file_task and see if it returns what we expect.
        # But we need to make sure it doesn't actually try to read a file.
        
        result = await handle_file_task("/tmp/test.pdf")
        assert isinstance(result, list)
        assert len(result) == 1
        assert result[0]["content"] == "# File Content"
        assert result[0]["url"] == "/tmp/test.pdf"
        # Verify convert was called (eventually)
        # Note: Since it runs in a thread, verifying call args might be tricky if not waited properly, 
        # but await handle_file_task waits for it.
        mock_converter.convert.assert_called_with("/tmp/test.pdf")
