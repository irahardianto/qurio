import pytest
from config import Settings
from unittest.mock import patch, AsyncMock, MagicMock, ANY
from importlib import reload
import sys


def test_settings_timeout_default():
    s = Settings()
    # Default is 120000ms (120s)
    assert s.crawler_page_timeout == 120000

    @pytest.mark.asyncio
    async def test_web_handler_uses_timeout():
        # Reload handlers.web to ensure we are testing a fresh module instance
        # independent of other tests that might have messed with sys.modules
        if "handlers.web" in sys.modules:
            import handlers.web

            reload(handlers.web)
        else:
            import handlers.web

        from handlers.web import handle_web_task

        # Patch settings to return a custom timeout
        with patch("handlers.web.app_settings") as mock_settings:
            mock_settings.crawler_page_timeout = 120000
            mock_settings.gemini_api_key = "fake"

            # Patch the crawler configuration
            with patch.object(handlers.web, "CrawlerRunConfig") as MockCrawlerRunConfig:
                mock_crawler = AsyncMock()
                mock_result = MagicMock()
                mock_result.success = True
                mock_result.markdown = "test"
                mock_result.url = "http://example.com"
                mock_result.links = {}
                mock_crawler.arun.return_value = mock_result

                # We mock asyncio.wait_for to avoid actual waiting if logic uses it
                async def mock_wait_for_impl(awaitable, timeout):
                    return await awaitable

                with patch(
                    "asyncio.wait_for", side_effect=mock_wait_for_impl
                ) as mock_wait:
                    await handle_web_task("http://example.com", crawler=mock_crawler)

                    # Verify wait_for was called with correct timeout logic
                    # Logic is (timeout / 1000) + 5.0
                    expected_timeout = (120000 / 1000) + 5.0
                    mock_wait.assert_called_with(ANY, timeout=expected_timeout)
                # Verify CrawlerRunConfig was called with page_timeout=120000
                assert MockCrawlerRunConfig.call_count >= 1

                found = False
                for call in MockCrawlerRunConfig.call_args_list:
                    if call.kwargs.get("page_timeout") == 120000:
                        found = True
                        break

                assert found, (
                    f"CrawlerRunConfig was not called with page_timeout=120000. Calls: {MockCrawlerRunConfig.call_args_list}"
                )
