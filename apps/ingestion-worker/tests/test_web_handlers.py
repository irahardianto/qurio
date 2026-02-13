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


# --- Additional Edge Cases and Happy Path Tests ---


@pytest.mark.asyncio
async def test_handle_web_task_llms_txt_bypass():
    """Verify that llms.txt files bypass LLM filtering."""
    from handlers.web import handle_web_task

    mock_result = MagicMock()
    mock_result.success = True
    mock_result.markdown = "# Documentation\nSome content"
    mock_result.url = "http://example.com/llms.txt"
    mock_result.links = {"internal": []}

    mock_crawler = AsyncMock()
    mock_crawler.arun.return_value = mock_result

    with patch("handlers.web.DefaultMarkdownGenerator") as MockMdGen:
        with patch("handlers.web.LLMContentFilter") as MockLLMFilter:
            await handle_web_task("http://example.com/llms.txt", crawler=mock_crawler)

            # DefaultMarkdownGenerator should be called without LLM filter
            MockMdGen.assert_called_once()
            # LLMContentFilter should NOT be instantiated
            MockLLMFilter.assert_not_called()


@pytest.mark.asyncio
async def test_handle_web_task_txt_file_bypass():
    """Verify that .txt files bypass LLM filtering."""
    from handlers.web import handle_web_task

    mock_result = MagicMock()
    mock_result.success = True
    mock_result.markdown = "Plain text content"
    mock_result.url = "http://example.com/readme.txt"
    mock_result.links = {"internal": []}

    mock_crawler = AsyncMock()
    mock_crawler.arun.return_value = mock_result

    with patch("handlers.web.DefaultMarkdownGenerator") as MockMdGen:
        with patch("handlers.web.LLMContentFilter") as MockLLMFilter:
            await handle_web_task("http://example.com/readme.txt", crawler=mock_crawler)

            MockMdGen.assert_called_once()
            MockLLMFilter.assert_not_called()


@pytest.mark.asyncio
async def test_handle_web_task_no_title():
    """Test metadata extraction when no title is found."""
    from handlers.web import handle_web_task

    mock_result = MagicMock()
    mock_result.success = True
    mock_result.markdown = "Content without heading"
    mock_result.url = "http://example.com/page"
    mock_result.links = {"internal": []}

    mock_crawler = AsyncMock()
    mock_crawler.arun.return_value = mock_result

    result = await handle_web_task("http://example.com/page", crawler=mock_crawler)

    assert result[0]["title"] == ""


@pytest.mark.asyncio
async def test_handle_web_task_no_links():
    """Test handling when no links are found."""
    from handlers.web import handle_web_task

    mock_result = MagicMock()
    mock_result.success = True
    mock_result.markdown = "# Page\nContent"
    mock_result.url = "http://example.com/page"
    mock_result.links = {}  # No links

    mock_crawler = AsyncMock()
    mock_crawler.arun.return_value = mock_result

    result = await handle_web_task("http://example.com/page", crawler=mock_crawler)

    assert result[0]["links"] == []


@pytest.mark.asyncio
async def test_handle_web_task_link_deduplication():
    """Test that duplicate links are removed."""
    from handlers.web import handle_web_task

    mock_result = MagicMock()
    mock_result.success = True
    mock_result.markdown = "# Page\nContent"
    mock_result.url = "http://example.com/page"
    mock_result.links = {
        "internal": [
            {"href": "http://example.com/page1"},
            {"href": "http://example.com/page1"},  # Duplicate
            {"href": "http://example.com/page2"},
        ]
    }

    mock_crawler = AsyncMock()
    mock_crawler.arun.return_value = mock_result

    result = await handle_web_task("http://example.com/page", crawler=mock_crawler)

    links = result[0]["links"]
    assert len(links) == 2
    assert "http://example.com/page1" in links
    assert "http://example.com/page2" in links


@pytest.mark.asyncio
async def test_handle_web_task_relative_url_resolution():
    """Test that relative URLs in markdown are resolved correctly."""
    from handlers.web import handle_web_task

    mock_result = MagicMock()
    mock_result.success = True
    # Markdown with relative link
    mock_result.markdown = "# Page\n[Link](/docs/api)"
    mock_result.url = "http://example.com/page"
    mock_result.links = {"internal": []}

    mock_crawler = AsyncMock()
    mock_crawler.arun.return_value = mock_result

    result = await handle_web_task("http://example.com/page", crawler=mock_crawler)

    links = result[0]["links"]
    assert "http://example.com/docs/api" in links


@pytest.mark.asyncio
async def test_handle_web_task_external_links_filtered():
    """Test that external links from markdown are filtered out."""
    from handlers.web import handle_web_task

    mock_result = MagicMock()
    mock_result.success = True
    mock_result.markdown = (
        "# Page\n[External](http://other.com/page)\n[Internal](/docs)"
    )
    mock_result.url = "http://example.com/page"
    mock_result.links = {"internal": []}

    mock_crawler = AsyncMock()
    mock_crawler.arun.return_value = mock_result

    result = await handle_web_task("http://example.com/page", crawler=mock_crawler)

    links = result[0]["links"]
    assert "http://other.com/page" not in links
    assert "http://example.com/docs" in links


@pytest.mark.asyncio
async def test_handle_web_task_path_extraction():
    """Test that URL path is correctly extracted."""
    from handlers.web import handle_web_task

    mock_result = MagicMock()
    mock_result.success = True
    mock_result.markdown = "# Page"
    mock_result.url = "http://example.com/docs/api/v1/users"
    mock_result.links = {"internal": []}

    mock_crawler = AsyncMock()
    mock_crawler.arun.return_value = mock_result

    result = await handle_web_task(
        "http://example.com/docs/api/v1/users", crawler=mock_crawler
    )

    assert result[0]["path"] == "docs > api > v1 > users"


@pytest.mark.asyncio
async def test_handle_web_task_empty_markdown():
    """Test handling of empty markdown content."""
    from handlers.web import handle_web_task

    mock_result = MagicMock()
    mock_result.success = True
    mock_result.markdown = ""
    mock_result.url = "http://example.com/empty"
    mock_result.links = {"internal": []}

    mock_crawler = AsyncMock()
    mock_crawler.arun.return_value = mock_result

    result = await handle_web_task("http://example.com/empty", crawler=mock_crawler)

    assert result[0]["content"] == ""
    assert result[0]["title"] == ""


@pytest.mark.asyncio
async def test_handle_web_task_without_crawler_instance():
    """Test that handle_web_task creates its own crawler when none is provided."""
    from handlers.web import handle_web_task

    mock_result = MagicMock()
    mock_result.success = True
    mock_result.markdown = "# Test"
    mock_result.url = "http://example.com"
    mock_result.links = {"internal": []}

    mock_crawler_instance = AsyncMock()
    mock_crawler_instance.arun.return_value = mock_result
    mock_crawler_instance.__aenter__.return_value = mock_crawler_instance
    mock_crawler_instance.__aexit__.return_value = None

    with patch("handlers.web.default_crawler_factory") as mock_factory:
        mock_factory.return_value = mock_crawler_instance

        result = await handle_web_task("http://example.com", crawler=None)

        # Should create new crawler
        mock_factory.assert_called_once()
        assert result[0]["content"] == "# Test"


@pytest.mark.asyncio
async def test_handle_web_task_retry_success_on_second_attempt():
    """Test that retry succeeds on second attempt after transient error."""
    from handlers.web import handle_web_task

    mock_crawler = AsyncMock()

    # First call fails with timeout, second succeeds
    call_count = 0

    async def side_effect(url, config):
        nonlocal call_count
        call_count += 1
        if call_count == 1:
            result = MagicMock()
            result.success = False
            result.error_message = "net::ERR_TIMED_OUT"
            return result
        else:
            result = MagicMock()
            result.success = True
            result.markdown = "# Success"
            result.url = url
            result.links = {"internal": []}
            return result

    mock_crawler.arun.side_effect = side_effect

    with patch("handlers.web.asyncio.sleep", new_callable=AsyncMock):
        result = await handle_web_task("http://example.com", crawler=mock_crawler)

        assert result[0]["content"] == "# Success"
        assert mock_crawler.arun.call_count == 2


@pytest.mark.asyncio
async def test_classify_crawl_error_connection_reset():
    """Test classification of connection reset errors."""
    from handlers.web import _classify_crawl_error
    from exceptions import ERR_CRAWL_REFUSED

    err = _classify_crawl_error(
        "Page.goto: net::ERR_CONNECTION_RESET at https://example.com"
    )
    assert err.code == ERR_CRAWL_REFUSED


@pytest.mark.asyncio
async def test_classify_crawl_error_connection_closed():
    """Test classification of connection closed errors."""
    from handlers.web import _classify_crawl_error
    from exceptions import ERR_CRAWL_REFUSED

    err = _classify_crawl_error(
        "Page.goto: net::ERR_CONNECTION_CLOSED at https://example.com"
    )
    assert err.code == ERR_CRAWL_REFUSED


@pytest.mark.asyncio
async def test_classify_crawl_error_forbidden():
    """Test classification of forbidden errors."""
    from handlers.web import _classify_crawl_error
    from exceptions import ERR_CRAWL_BLOCKED

    err = _classify_crawl_error("403 Forbidden")
    assert err.code == ERR_CRAWL_BLOCKED


# --- Embedding Content Extraction Tests ---


def test_get_embedding_content_prefers_fit_markdown():
    """Verify fit_markdown (LLM-filtered) is used over raw_markdown for embedding."""
    from handlers.web import _get_embedding_content

    result = MagicMock()
    md = MagicMock()
    md.fit_markdown = "# Clean Content\nFiltered documentation"
    md.raw_markdown = "# Clean Content\nFiltered documentation\n[Nav Link](http://x.com)\nSidebar noise"
    result.markdown = md

    content = _get_embedding_content(result)

    assert content == "# Clean Content\nFiltered documentation"
    assert "Nav Link" not in content
    assert "Sidebar noise" not in content


def test_get_embedding_content_falls_back_to_raw():
    """When fit_markdown is empty, fall back to raw_markdown."""
    from handlers.web import _get_embedding_content

    result = MagicMock()
    md = MagicMock()
    md.fit_markdown = ""  # Empty — e.g. .txt files or filter produced nothing
    md.raw_markdown = "Plain text content from .txt file"
    result.markdown = md

    content = _get_embedding_content(result)

    assert content == "Plain text content from .txt file"


def test_get_embedding_content_falls_back_to_raw_whitespace():
    """When fit_markdown is only whitespace, fall back to raw_markdown."""
    from handlers.web import _get_embedding_content

    result = MagicMock()
    md = MagicMock()
    md.fit_markdown = "   \n  "  # Whitespace only
    md.raw_markdown = "Real content here"
    result.markdown = md

    content = _get_embedding_content(result)

    assert content == "Real content here"


def test_get_embedding_content_handles_plain_string():
    """Backwards compatibility: plain string markdown still works."""
    from handlers.web import _get_embedding_content

    result = MagicMock()
    result.markdown = "# Simple String Content"

    content = _get_embedding_content(result)

    assert content == "# Simple String Content"


def test_get_embedding_content_handles_none():
    """Handle None markdown gracefully."""
    from handlers.web import _get_embedding_content

    result = MagicMock()
    result.markdown = None

    content = _get_embedding_content(result)

    assert content == ""


@pytest.mark.asyncio
async def test_handle_web_task_uses_fit_markdown_for_content():
    """Integration: handle_web_task returns fit_markdown for content, not raw."""
    from handlers.web import handle_web_task

    mock_crawler = AsyncMock()

    async def side_effect(url, config=None):
        res = MagicMock()
        if url.endswith("llms.txt"):
            res.success = False
        else:
            res.success = True
            # Simulate MarkdownGenerationResult object
            md = MagicMock()
            md.fit_markdown = "# API Reference\nClean filtered content"
            md.raw_markdown = "# API Reference\nClean filtered content\n[Home](/) [About](/about)\nNav sidebar noise"
            res.markdown = md
            res.url = "http://example.com/docs/api"
            res.links = {
                "internal": [
                    {"href": "http://example.com/docs/guide"},
                ],
            }
        return res

    mock_crawler.arun.side_effect = side_effect

    result = await handle_web_task("http://example.com/docs/api", crawler=mock_crawler)

    # Content should be the FILTERED version
    assert result[0]["content"] == "# API Reference\nClean filtered content"
    assert "Nav sidebar noise" not in result[0]["content"]

    # Links should still be discovered from result.links (untouched)
    assert "http://example.com/docs/guide" in result[0]["links"]


@pytest.mark.asyncio
async def test_handle_web_task_link_discovery_uses_raw_markdown():
    """Verify link extraction from markdown uses raw content, not filtered."""
    from handlers.web import handle_web_task

    mock_crawler = AsyncMock()

    async def side_effect(url, config=None):
        res = MagicMock()
        if url.endswith("llms.txt"):
            res.success = False
        else:
            res.success = True
            # fit_markdown has NO links (because they were filtered)
            md = MagicMock()
            md.fit_markdown = "# Docs\nClean content only"
            # raw_markdown has links (for discovery)
            md.raw_markdown = "# Docs\nClean content only\n[Guide](/guide)\n[API](/api)"
            res.markdown = md
            res.url = "http://example.com/docs"
            res.links = {"internal": []}  # No DOM-parsed links
        return res

    mock_crawler.arun.side_effect = side_effect

    result = await handle_web_task("http://example.com/docs", crawler=mock_crawler)

    # Markdown regex link extraction should find links from raw markdown
    links = result[0]["links"]
    assert "http://example.com/guide" in links
    assert "http://example.com/api" in links
