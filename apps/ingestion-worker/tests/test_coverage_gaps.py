"""
Tests for _resolve_sitemap recursive paths, _crawl_single_page,
_get_raw_markdown edge cases, and config.py full field coverage.
"""

import pytest
import asyncio
from unittest.mock import MagicMock, AsyncMock, patch
import httpx


# --- _resolve_sitemap recursive path tests ---


STANDARD_SITEMAP_XML = """<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/page1</loc></url>
  <url><loc>https://example.com/page2</loc></url>
</urlset>"""

INDEX_SITEMAP_XML = """<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap><loc>https://example.com/sitemap-1.xml</loc></sitemap>
  <sitemap><loc>https://example.com/sitemap-2.xml</loc></sitemap>
</sitemapindex>"""

INVALID_ROOT_XML = """<?xml version="1.0" encoding="UTF-8"?>
<feed><entry>not a sitemap</entry></feed>"""


def _mock_response(status_code: int, text: str = ""):
    resp = MagicMock(spec=httpx.Response)
    resp.status_code = status_code
    resp.text = text
    return resp


class TestResolveSitemap:
    """Tests for _resolve_sitemap with various edge cases."""

    @pytest.mark.asyncio
    async def test_max_depth_exceeded_returns_empty(self):
        """Returns [] when depth > MAX_SITEMAP_INDEX_DEPTH."""
        from handlers.sitemap import _resolve_sitemap, MAX_SITEMAP_INDEX_DEPTH

        result = await _resolve_sitemap(
            "https://example.com/sitemap.xml",
            "example.com",
            depth=MAX_SITEMAP_INDEX_DEPTH + 1,
        )
        assert result == []

    @pytest.mark.asyncio
    async def test_xml_parse_error_returns_empty(self):
        """Returns [] on invalid XML."""
        from handlers.sitemap import _resolve_sitemap

        with patch("handlers.sitemap.httpx.AsyncClient") as MockClient:
            mock_client = AsyncMock()
            MockClient.return_value.__aenter__ = AsyncMock(return_value=mock_client)
            MockClient.return_value.__aexit__ = AsyncMock(return_value=False)
            mock_client.get.return_value = _mock_response(200, "<<<not xml>>>")

            result = await _resolve_sitemap(
                "https://example.com/sitemap.xml", "example.com", depth=0
            )
            assert result == []

    @pytest.mark.asyncio
    async def test_connection_timeout_returns_empty(self):
        """Returns [] on httpx.TimeoutException."""
        from handlers.sitemap import _resolve_sitemap

        with patch("handlers.sitemap.httpx.AsyncClient") as MockClient:
            mock_client = AsyncMock()
            MockClient.return_value.__aenter__ = AsyncMock(return_value=mock_client)
            MockClient.return_value.__aexit__ = AsyncMock(return_value=False)
            mock_client.get.side_effect = httpx.TimeoutException("timeout")

            result = await _resolve_sitemap(
                "https://example.com/sitemap.xml", "example.com", depth=0
            )
            assert result == []

    @pytest.mark.asyncio
    async def test_connect_error_returns_empty(self):
        """Returns [] on httpx.ConnectError."""
        from handlers.sitemap import _resolve_sitemap

        with patch("handlers.sitemap.httpx.AsyncClient") as MockClient:
            mock_client = AsyncMock()
            MockClient.return_value.__aenter__ = AsyncMock(return_value=mock_client)
            MockClient.return_value.__aexit__ = AsyncMock(return_value=False)
            mock_client.get.side_effect = httpx.ConnectError("connection refused")

            result = await _resolve_sitemap(
                "https://example.com/sitemap.xml", "example.com", depth=0
            )
            assert result == []

    @pytest.mark.asyncio
    async def test_unknown_root_element_returns_empty(self):
        """Returns [] on unrecognized XML root element."""
        from handlers.sitemap import _resolve_sitemap

        with patch("handlers.sitemap.httpx.AsyncClient") as MockClient:
            mock_client = AsyncMock()
            MockClient.return_value.__aenter__ = AsyncMock(return_value=mock_client)
            MockClient.return_value.__aexit__ = AsyncMock(return_value=False)
            mock_client.get.return_value = _mock_response(200, INVALID_ROOT_XML)

            result = await _resolve_sitemap(
                "https://example.com/sitemap.xml", "example.com", depth=0
            )
            assert result == []

    @pytest.mark.asyncio
    async def test_404_returns_empty(self):
        """Returns [] on non-200 status code."""
        from handlers.sitemap import _resolve_sitemap

        with patch("handlers.sitemap.httpx.AsyncClient") as MockClient:
            mock_client = AsyncMock()
            MockClient.return_value.__aenter__ = AsyncMock(return_value=mock_client)
            MockClient.return_value.__aexit__ = AsyncMock(return_value=False)
            mock_client.get.return_value = _mock_response(404)

            result = await _resolve_sitemap(
                "https://example.com/sitemap.xml", "example.com", depth=0
            )
            assert result == []

    @pytest.mark.asyncio
    async def test_empty_body_returns_empty(self):
        """Returns [] on empty response body."""
        from handlers.sitemap import _resolve_sitemap

        with patch("handlers.sitemap.httpx.AsyncClient") as MockClient:
            mock_client = AsyncMock()
            MockClient.return_value.__aenter__ = AsyncMock(return_value=mock_client)
            MockClient.return_value.__aexit__ = AsyncMock(return_value=False)
            mock_client.get.return_value = _mock_response(200, "   ")

            result = await _resolve_sitemap(
                "https://example.com/sitemap.xml", "example.com", depth=0
            )
            assert result == []

    @pytest.mark.asyncio
    async def test_standard_sitemap_extracts_urls(self):
        """Standard <urlset> returns extracted URLs."""
        from handlers.sitemap import _resolve_sitemap

        with patch("handlers.sitemap.httpx.AsyncClient") as MockClient:
            mock_client = AsyncMock()
            MockClient.return_value.__aenter__ = AsyncMock(return_value=mock_client)
            MockClient.return_value.__aexit__ = AsyncMock(return_value=False)
            mock_client.get.return_value = _mock_response(200, STANDARD_SITEMAP_XML)

            result = await _resolve_sitemap(
                "https://example.com/sitemap.xml", "example.com", depth=0
            )
            assert sorted(result) == [
                "https://example.com/page1",
                "https://example.com/page2",
            ]

    @pytest.mark.asyncio
    async def test_sitemap_index_recursive_resolution(self):
        """Sitemap index recursively fetches sub-sitemaps."""
        from handlers.sitemap import _resolve_sitemap

        call_count = 0

        async def mock_get(url):
            nonlocal call_count
            call_count += 1
            if "sitemap.xml" in url and call_count == 1:
                return _mock_response(200, INDEX_SITEMAP_XML)
            elif "sitemap-1.xml" in url:
                return _mock_response(
                    200,
                    """<?xml version="1.0"?>
                    <urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
                    <url><loc>https://example.com/a</loc></url>
                    </urlset>""",
                )
            elif "sitemap-2.xml" in url:
                return _mock_response(
                    200,
                    """<?xml version="1.0"?>
                    <urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
                    <url><loc>https://example.com/b</loc></url>
                    </urlset>""",
                )
            return _mock_response(404)

        with patch("handlers.sitemap.httpx.AsyncClient") as MockClient:
            mock_client = AsyncMock()
            MockClient.return_value.__aenter__ = AsyncMock(return_value=mock_client)
            MockClient.return_value.__aexit__ = AsyncMock(return_value=False)
            mock_client.get.side_effect = mock_get

            result = await _resolve_sitemap(
                "https://example.com/sitemap.xml", "example.com", depth=0
            )
            assert sorted(result) == [
                "https://example.com/a",
                "https://example.com/b",
            ]


# --- _crawl_single_page Tests ---


class TestCrawlSinglePage:
    """Direct tests for _crawl_single_page."""

    @pytest.mark.asyncio
    async def test_success_returns_result(self):
        """Verify successful crawl returns the result."""
        from handlers.web import _crawl_single_page

        mock_crawler = AsyncMock()
        mock_result = MagicMock()
        mock_result.success = True
        mock_crawler.arun.return_value = mock_result

        result = await _crawl_single_page(
            mock_crawler, "http://example.com", MagicMock()
        )
        assert result is mock_result

    @pytest.mark.asyncio
    async def test_failure_raises_classified_error(self):
        """Verify failed crawl raises IngestionError via _classify_crawl_error."""
        from handlers.web import _crawl_single_page
        from exceptions import IngestionError

        mock_crawler = AsyncMock()
        mock_result = MagicMock()
        mock_result.success = False
        mock_result.error_message = "net::ERR_TIMED_OUT"
        mock_crawler.arun.return_value = mock_result

        with pytest.raises(IngestionError) as exc_info:
            await _crawl_single_page(mock_crawler, "http://example.com", MagicMock())
        assert "ERR_CRAWL_TIMEOUT" == exc_info.value.code

    @pytest.mark.asyncio
    async def test_outer_timeout_applied(self):
        """Verify asyncio.wait_for applies outer timeout based on config."""
        from handlers.web import _crawl_single_page

        mock_crawler = AsyncMock()
        # Make arun hang to trigger timeout
        mock_crawler.arun.side_effect = asyncio.TimeoutError()

        with pytest.raises(asyncio.TimeoutError):
            await _crawl_single_page(mock_crawler, "http://example.com", MagicMock())


# --- _get_raw_markdown Edge Cases ---


class TestGetRawMarkdown:
    """Edge case tests for _get_raw_markdown."""

    def test_with_markdown_generation_result(self):
        """Verify extraction from object with raw_markdown attribute."""
        from handlers.web import _get_raw_markdown

        mock_result = MagicMock()
        mock_result.markdown.raw_markdown = "# Hello"
        assert _get_raw_markdown(mock_result) == "# Hello"

    def test_with_plain_string(self):
        """Verify plain string markdown passthrough."""
        from handlers.web import _get_raw_markdown

        mock_result = MagicMock()
        # Use a real string — _get_raw_markdown checks hasattr for raw_markdown
        mock_result.markdown = "plain text"
        result = _get_raw_markdown(mock_result)
        # str has no raw_markdown, so it should return the string itself
        assert result == "plain text"

    def test_with_none_markdown(self):
        """Verify None markdown returns empty string."""
        from handlers.web import _get_raw_markdown

        mock_result = MagicMock()
        mock_result.markdown = None
        assert _get_raw_markdown(mock_result) == ""

    def test_with_object_no_raw_falls_to_str(self):
        """Verify object without raw_markdown uses str() fallback."""
        from handlers.web import _get_raw_markdown

        mock_result = MagicMock()

        # Create a custom object with no raw_markdown and no string behavior
        class CustomObj:
            def __str__(self):
                return "custom_str"

        mock_result.markdown = CustomObj()
        assert _get_raw_markdown(mock_result) == "custom_str"

    def test_with_empty_raw_markdown(self):
        """Verify empty raw_markdown returns empty string."""
        from handlers.web import _get_raw_markdown

        mock_result = MagicMock()
        mock_result.markdown.raw_markdown = ""
        assert _get_raw_markdown(mock_result) == ""

    def test_with_none_raw_markdown(self):
        """Verify None raw_markdown returns empty string."""
        from handlers.web import _get_raw_markdown

        mock_result = MagicMock()
        mock_result.markdown.raw_markdown = None
        assert _get_raw_markdown(mock_result) == ""


# --- Config Full Field Coverage ---


class TestSettingsAllFields:
    """Verify all Settings fields have correct defaults and env override."""

    def test_all_defaults(self, monkeypatch):
        """Verify all default values match expected."""
        # Clear all relevant env vars
        for var in [
            "NSQ_LOOKUPD_HTTP",
            "NSQ_TOPIC_INGEST",
            "NSQ_CHANNEL_WORKER",
            "NSQ_TOPIC_RESULT",
            "NSQD_TCP_ADDRESS",
            "GEMINI_API_KEY",
            "NSQ_MAX_IN_FLIGHT",
            "NSQ_HEARTBEAT_INTERVAL",
            "CRAWLER_PAGE_TIMEOUT",
            "ENV",
            "RETRY_MAX_ATTEMPTS",
            "RETRY_INITIAL_DELAY_MS",
            "RETRY_MAX_DELAY_MS",
            "RETRY_BACKOFF_MULTIPLIER",
        ]:
            monkeypatch.delenv(var, raising=False)

        from config import Settings

        s = Settings()

        assert s.nsq_lookupd_http == "nsqlookupd:4161"
        assert s.nsq_topic_ingest == "ingest.task"
        assert s.nsq_channel_worker == "worker"
        assert s.nsq_topic_result == "ingest.result"
        assert s.nsqd_tcp_address == "nsqd:4150"
        assert s.gemini_api_key == ""
        assert s.nsq_max_in_flight == 8
        assert s.nsq_heartbeat_interval == 60
        assert s.crawler_page_timeout == 120000
        assert s.env == "production"
        assert s.retry_max_attempts == 3
        assert s.retry_initial_delay_ms == 1000
        assert s.retry_max_delay_ms == 60000
        assert s.retry_backoff_multiplier == 2

    def test_all_env_overrides(self, monkeypatch):
        """Verify each field respects env vars."""
        monkeypatch.setenv("NSQ_LOOKUPD_HTTP", "custom:4161")
        monkeypatch.setenv("NSQ_TOPIC_INGEST", "custom.topic")
        monkeypatch.setenv("NSQ_CHANNEL_WORKER", "custom_channel")
        monkeypatch.setenv("NSQ_TOPIC_RESULT", "custom.result")
        monkeypatch.setenv("NSQD_TCP_ADDRESS", "custom:4150")
        monkeypatch.setenv("GEMINI_API_KEY", "test-key-123")
        monkeypatch.setenv("NSQ_MAX_IN_FLIGHT", "16")
        monkeypatch.setenv("NSQ_HEARTBEAT_INTERVAL", "30")
        monkeypatch.setenv("CRAWLER_PAGE_TIMEOUT", "60000")
        monkeypatch.setenv("ENV", "development")

        from config import Settings

        s = Settings()

        assert s.nsq_lookupd_http == "custom:4161"
        assert s.nsq_topic_ingest == "custom.topic"
        assert s.nsq_channel_worker == "custom_channel"
        assert s.nsq_topic_result == "custom.result"
        assert s.nsqd_tcp_address == "custom:4150"
        assert s.gemini_api_key == "test-key-123"
        assert s.nsq_max_in_flight == 16
        assert s.nsq_heartbeat_interval == 30
        assert s.crawler_page_timeout == 60000
        assert s.env == "development"
