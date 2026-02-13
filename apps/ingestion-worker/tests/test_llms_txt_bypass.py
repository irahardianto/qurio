import sys
from unittest.mock import MagicMock, AsyncMock, patch
import pytest


@pytest.fixture
def mock_crawl4ai_env():
    # Setup mocks for crawl4ai
    mock_crawl4ai = MagicMock()
    sys.modules["crawl4ai"] = mock_crawl4ai
    sys.modules["crawl4ai.content_filter_strategy"] = MagicMock()
    sys.modules["crawl4ai.markdown_generation_strategy"] = MagicMock()

    # Force reload of handlers.web to pick up these mocks
    if "handlers.web" in sys.modules:
        del sys.modules["handlers.web"]
    import handlers.web

    yield handlers.web

    # Cleanup
    if "handlers.web" in sys.modules:
        del sys.modules["handlers.web"]

    @pytest.mark.asyncio
    async def test_llms_txt_uses_default_generator(mock_crawl4ai_env):
        handlers_web = mock_crawl4ai_env
        handle_web_task = handlers_web.handle_web_task

        url = "https://example.com/llms.txt"
        mock_result = MagicMock()
        mock_result.success = True
        mock_result.markdown = "content"
        mock_result.url = url
        mock_result.links = {"internal": []}

        mock_crawler = AsyncMock()

        async def fake_arun(url, config=None):
            return mock_result

        mock_crawler.arun.side_effect = fake_arun

        # Patch DefaultMarkdownGenerator IN the reloaded module
        with patch.object(handlers_web, "DefaultMarkdownGenerator") as MockGen:
            generator_instance = MagicMock(name="generator_instance")
            MockGen.return_value = generator_instance

            await handle_web_task(url, crawler=mock_crawler)

            # Verify DefaultMarkdownGenerator was used without content_filter
            MockGen.assert_called_with()

    @pytest.mark.asyncio
    async def test_standard_page_uses_llm_filter(mock_crawl4ai_env):
        handlers_web = mock_crawl4ai_env
        handle_web_task = handlers_web.handle_web_task

        url = "https://example.com/page"
        mock_result = MagicMock()
        mock_result.success = True
        mock_result.markdown = "content"
        mock_result.url = url
        mock_result.links = {"internal": []}

        mock_crawler = AsyncMock()

        async def fake_arun(url, config=None):
            # No manifest check logic needed in mock as we removed it from handler
            return mock_result

        mock_crawler.arun.side_effect = fake_arun

        with patch.object(handlers_web, "DefaultMarkdownGenerator") as MockGen:
            generator_instance = MagicMock(name="generator_instance")
            MockGen.return_value = generator_instance

            await handle_web_task(url, crawler=mock_crawler)

            # Verify DefaultMarkdownGenerator was called with content_filter
            call_args = MockGen.call_args
            assert call_args is not None, "DefaultMarkdownGenerator should be called"
            kwargs = call_args.kwargs
            assert "content_filter" in kwargs, "Should pass content_filter"
            assert kwargs["content_filter"] is not None
