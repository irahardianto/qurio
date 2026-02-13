import sys
from unittest.mock import MagicMock
import types

# Create a mock package for crawl4ai
crawl4ai = types.ModuleType("crawl4ai")
sys.modules["crawl4ai"] = crawl4ai

# Mock submodules
content_filter_strategy = types.ModuleType("crawl4ai.content_filter_strategy")
sys.modules["crawl4ai.content_filter_strategy"] = content_filter_strategy
crawl4ai.content_filter_strategy = content_filter_strategy  # type: ignore[attr-defined]

markdown_generation_strategy = types.ModuleType("crawl4ai.markdown_generation_strategy")
sys.modules["crawl4ai.markdown_generation_strategy"] = markdown_generation_strategy
crawl4ai.markdown_generation_strategy = markdown_generation_strategy  # type: ignore[attr-defined]

# Populate with mocks
crawl4ai.AsyncWebCrawler = MagicMock()  # type: ignore[attr-defined]
crawl4ai.CrawlerRunConfig = MagicMock()  # type: ignore[attr-defined]
crawl4ai.CacheMode = MagicMock()  # type: ignore[attr-defined]
crawl4ai.LLMConfig = MagicMock()  # type: ignore[attr-defined]

content_filter_strategy.PruningContentFilter = MagicMock()  # type: ignore[attr-defined]
content_filter_strategy.LLMContentFilter = MagicMock()  # type: ignore[attr-defined]

markdown_generation_strategy.DefaultMarkdownGenerator = MagicMock()  # type: ignore[attr-defined]

import pytest  # noqa: E402
from unittest.mock import MagicMock, AsyncMock, ANY, patch  # noqa: E402

# Ensure we get the real module, not a mock from test_main_integration
if "handlers.web" in sys.modules:
    del sys.modules["handlers.web"]


@pytest.mark.asyncio
async def test_handle_web_task_returns_title():
    from handlers.web import handle_web_task

    # Mock crawler
    mock_crawler = AsyncMock()

    async def side_effect(url, config=None):
        res = MagicMock()
        if url.endswith("llms.txt"):
            res.success = False  # Manifest check fails
        else:
            res.success = True
            res.markdown = "# My Page Title\nSome content"
            res.url = "http://example.com"
            res.links = {"internal": []}
        return res

    mock_crawler.arun.side_effect = side_effect

    result = await handle_web_task("http://example.com", crawler=mock_crawler)

    assert isinstance(result, list)
    assert len(result) == 1
    assert "title" in result[0]
    assert result[0]["title"] == "My Page Title"


@pytest.mark.asyncio
async def test_handle_web_task_success():
    from handlers.web import handle_web_task

    # Mock crawler
    mock_crawler = AsyncMock()

    async def side_effect(url, config=None):
        res = MagicMock()
        if url.endswith("llms.txt"):
            res.success = False
        else:
            res.success = True
            res.markdown = "# Test Content"
            res.url = "http://example.com"
            res.links = {"internal": []}
        return res

    mock_crawler.arun.side_effect = side_effect

    result = await handle_web_task("http://example.com", crawler=mock_crawler)

    assert isinstance(result, list), "Expected list, got something else"
    assert len(result) == 1
    assert result[0]["content"] == "# Test Content"
    assert result[0]["url"] == "http://example.com"
    mock_crawler.arun.assert_called_with(url="http://example.com", config=ANY)


@pytest.mark.asyncio
async def test_handle_web_task_failure():
    from handlers.web import handle_web_task
    from exceptions import IngestionError

    # Mock result
    mock_result = MagicMock()
    mock_result.success = False
    mock_result.error_message = "Failed"

    # Mock crawler
    mock_crawler = AsyncMock()
    mock_crawler.arun.return_value = mock_result

    with patch("handlers.web.asyncio.sleep", new_callable=AsyncMock):
        with pytest.raises(IngestionError, match="Failed"):
            await handle_web_task("http://example.com", crawler=mock_crawler)


@pytest.mark.asyncio
async def test_handle_web_task_internal_links():
    from handlers.web import handle_web_task

    # Mock result with mixed links
    mock_result = MagicMock()
    mock_result.success = True
    mock_result.markdown = "Content"
    mock_result.url = "http://example.com/page1"
    mock_result.links = {
        "internal": [
            {"href": "http://example.com/page2"},
            {"href": "http://example.com/page1#section"},
        ],
        "external": [{"href": "http://google.com"}],
    }

    # Mock crawler
    mock_crawler = AsyncMock()
    mock_crawler.arun.return_value = mock_result

    result = await handle_web_task("http://example.com/page1", crawler=mock_crawler)

    links = result[0]["links"]
    assert "http://example.com/page2" in links
    assert "http://google.com" not in links


@pytest.mark.asyncio
async def test_handle_web_task_auth_precedence():
    from handlers.web import handle_web_task

    mock_result = MagicMock()
    mock_result.success = True
    mock_result.markdown = ""
    mock_result.url = "http://example.com"
    mock_result.links = {}

    mock_crawler = AsyncMock()
    mock_crawler.arun.return_value = mock_result

    with patch("handlers.web.LLMConfig") as MockLLMConfig:
        await handle_web_task(
            "http://example.com", api_key="custom-key", crawler=mock_crawler
        )

        # Verify LLMConfig initialized with custom key
        MockLLMConfig.assert_called_with(
            provider="gemini/gemini-3-flash-preview",
            api_token="custom-key",
            temperature=1.0,
        )


# --- Error Classification Tests ---


def test_classify_crawl_error_timeout():
    from handlers.web import _classify_crawl_error
    from exceptions import ERR_CRAWL_TIMEOUT

    err = _classify_crawl_error(
        "Failed on navigating ACS-GOTO: Page.goto: net::ERR_TIMED_OUT at https://example.com"
    )
    assert err.code == ERR_CRAWL_TIMEOUT


def test_classify_crawl_error_dns():
    from handlers.web import _classify_crawl_error
    from exceptions import ERR_CRAWL_DNS

    err = _classify_crawl_error(
        "Page.goto: net::ERR_NAME_NOT_RESOLVED at https://example.com"
    )
    assert err.code == ERR_CRAWL_DNS


def test_classify_crawl_error_connection_refused():
    from handlers.web import _classify_crawl_error
    from exceptions import ERR_CRAWL_REFUSED

    err = _classify_crawl_error(
        "Page.goto: net::ERR_CONNECTION_REFUSED at https://example.com"
    )
    assert err.code == ERR_CRAWL_REFUSED


def test_classify_crawl_error_blocked():
    from handlers.web import _classify_crawl_error
    from exceptions import ERR_CRAWL_BLOCKED

    err = _classify_crawl_error("blocked by robots.txt")
    assert err.code == ERR_CRAWL_BLOCKED


def test_classify_crawl_error_unknown_defaults_to_timeout():
    from handlers.web import _classify_crawl_error
    from exceptions import ERR_CRAWL_TIMEOUT

    # Unknown errors default to transient (timeout) for safety
    err = _classify_crawl_error("some unknown error")
    assert err.code == ERR_CRAWL_TIMEOUT


@pytest.mark.asyncio
async def test_handle_web_task_crawl_timeout_raises_ingestion_error():
    """Verify that net::ERR_TIMED_OUT from crawl4ai raises IngestionError, not generic Exception."""
    from handlers.web import handle_web_task
    from exceptions import IngestionError, ERR_CRAWL_TIMEOUT

    mock_result = MagicMock()
    mock_result.success = False
    mock_result.error_message = (
        "Failed on navigating ACS-GOTO: Page.goto: net::ERR_TIMED_OUT"
    )

    mock_crawler = AsyncMock()
    mock_crawler.arun.return_value = mock_result

    with patch("handlers.web.asyncio.sleep", new_callable=AsyncMock):
        with pytest.raises(IngestionError) as exc_info:
            await handle_web_task("http://example.com", crawler=mock_crawler)

        assert exc_info.value.code == ERR_CRAWL_TIMEOUT


@pytest.mark.asyncio
async def test_handle_web_task_retries_transient_errors():
    """Verify that transient errors trigger application-level retries."""
    from handlers.web import handle_web_task, CRAWL_MAX_RETRIES

    # Fail with timeout on all attempts
    mock_result = MagicMock()
    mock_result.success = False
    mock_result.error_message = "net::ERR_TIMED_OUT"

    mock_crawler = AsyncMock()
    mock_crawler.arun.return_value = mock_result

    with patch("handlers.web.asyncio.sleep", new_callable=AsyncMock) as mock_sleep:
        with pytest.raises(Exception):
            await handle_web_task("http://example.com", crawler=mock_crawler)

        # Should have been called CRAWL_MAX_RETRIES + 1 times total
        assert mock_crawler.arun.call_count == CRAWL_MAX_RETRIES + 1
        # Sleep called CRAWL_MAX_RETRIES times (between retries)
        assert mock_sleep.call_count == CRAWL_MAX_RETRIES


@pytest.mark.asyncio
async def test_handle_web_task_permanent_error_no_retry():
    """Verify that permanent errors (e.g., robots.txt blocked) are NOT retried."""
    from handlers.web import handle_web_task
    from exceptions import IngestionError, ERR_CRAWL_BLOCKED

    mock_result = MagicMock()
    mock_result.success = False
    mock_result.error_message = "blocked by robots.txt"

    mock_crawler = AsyncMock()
    mock_crawler.arun.return_value = mock_result

    with patch("handlers.web.asyncio.sleep", new_callable=AsyncMock) as mock_sleep:
        with pytest.raises(IngestionError) as exc_info:
            await handle_web_task("http://example.com", crawler=mock_crawler)

        assert exc_info.value.code == ERR_CRAWL_BLOCKED
        # No retries — should only crawl once
        assert mock_crawler.arun.call_count == 1
        mock_sleep.assert_not_called()
