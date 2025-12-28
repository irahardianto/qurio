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
from unittest.mock import MagicMock, patch
import json
import main  # import the module

@pytest.mark.asyncio
async def test_process_message_success():
    # Arrange
    message = MagicMock()
    message.body = json.dumps({"type": "web", "url": "http://example.com", "id": "123"}).encode('utf-8')
    message.finish = MagicMock()
    message.requeue = MagicMock()

    # Mock handlers
    with patch('main.handle_web_task', new_callable=MagicMock) as mock_web_task:
        # Make it awaitable
        async def async_mock(*args, **kwargs):
            return [{"content": "content", "url": "http://example.com", "links": []}] # Fix: Return list
        mock_web_task.side_effect = async_mock
        
        # Mock producer
        mock_producer = MagicMock()
        main.producer = mock_producer
        
        # Act
        await main.process_message(message)

        # Assert
        message.finish.assert_called_once()
        message.requeue.assert_not_called()
        mock_producer.pub.assert_called()

@pytest.mark.asyncio
async def test_process_message_failure():
    # Arrange
    message = MagicMock()
    message.body = b"invalid json"
    message.finish = MagicMock()
    message.requeue = MagicMock()

    # Act
    await main.process_message(message)

    # Assert
    message.finish.assert_called_once()
    message.requeue.assert_not_called()
