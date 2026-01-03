import sys
from unittest.mock import MagicMock
import types

# Create a mock package for crawl4ai
crawl4ai = types.ModuleType("crawl4ai")
sys.modules["crawl4ai"] = crawl4ai

# Mock submodules
content_filter_strategy = types.ModuleType("crawl4ai.content_filter_strategy")
sys.modules["crawl4ai.content_filter_strategy"] = content_filter_strategy
crawl4ai.content_filter_strategy = content_filter_strategy

markdown_generation_strategy = types.ModuleType("crawl4ai.markdown_generation_strategy")
sys.modules["crawl4ai.markdown_generation_strategy"] = markdown_generation_strategy
crawl4ai.markdown_generation_strategy = markdown_generation_strategy

# Populate with mocks
crawl4ai.AsyncWebCrawler = MagicMock()
crawl4ai.CrawlerRunConfig = MagicMock()
crawl4ai.CacheMode = MagicMock()
crawl4ai.LLMConfig = MagicMock()

content_filter_strategy.PruningContentFilter = MagicMock()
content_filter_strategy.LLMContentFilter = MagicMock()

markdown_generation_strategy.DefaultMarkdownGenerator = MagicMock()

import pytest
from unittest.mock import MagicMock, AsyncMock, patch, ANY
import asyncio
from handlers.web import handle_web_task

@pytest.mark.asyncio
async def test_handle_web_task_returns_title():
    # Mock result
    mock_result = MagicMock()
    mock_result.success = True
    mock_result.markdown = "# My Page Title\nSome content"
    mock_result.url = "http://example.com"
    mock_result.links = {'internal': []}
    
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
        assert "title" in result[0]
        # Since we use a fallback regex in our plan, we expect it to match the header
        assert result[0]["title"] == "My Page Title"

@pytest.mark.asyncio
async def test_handle_web_task_success():
    # Mock result
    mock_result = MagicMock()
    mock_result.success = True
    mock_result.markdown = "# Test Content"
    mock_result.url = "http://example.com"
    mock_result.links = {'internal': []}
    
    # Mock crawler
    mock_crawler = AsyncMock()
    mock_crawler.arun.return_value = mock_result
    
    # Context manager mock
    mock_crawler_cm = AsyncMock()
    mock_crawler_cm.__aenter__.return_value = mock_crawler
    mock_crawler_cm.__aexit__.return_value = None
    
    with patch('handlers.web.AsyncWebCrawler', return_value=mock_crawler_cm) as MockCrawler:
        result = await handle_web_task("http://example.com")
        
        # This assertion verifies the fix (it currently fails if returning dict)
        assert isinstance(result, list), "Expected list, got something else"
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
