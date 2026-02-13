"""Unit tests for the sitemap handler."""

import pytest
from unittest.mock import AsyncMock, patch, MagicMock
import httpx


VALID_SITEMAP_XML = """<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/page1</loc></url>
  <url><loc>https://example.com/page2</loc></url>
  <url><loc>https://example.com/docs/api</loc></url>
</urlset>"""

SITEMAP_INDEX_XML = """<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap><loc>https://example.com/sitemap-pages.xml</loc></sitemap>
  <sitemap><loc>https://example.com/sitemap-docs.xml</loc></sitemap>
</sitemapindex>"""

SUB_SITEMAP_PAGES = """<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/about</loc></url>
  <url><loc>https://example.com/contact</loc></url>
</urlset>"""

SUB_SITEMAP_DOCS = """<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/docs/guide</loc></url>
</urlset>"""

SITEMAP_WITH_EXTERNAL = """<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/page1</loc></url>
  <url><loc>https://other-domain.com/page2</loc></url>
  <url><loc>https://example.com/page3</loc></url>
</urlset>"""

SITEMAP_NO_NS = """<?xml version="1.0" encoding="UTF-8"?>
<urlset>
  <url><loc>https://example.com/no-ns-page</loc></url>
</urlset>"""


def _mock_response(status_code: int, text: str = "") -> httpx.Response:
    """Create a mock httpx.Response."""
    response = MagicMock(spec=httpx.Response)
    response.status_code = status_code
    response.text = text
    return response


@pytest.mark.asyncio
async def test_fetch_sitemap_urls_success():
    """Verify standard sitemap.xml is parsed correctly."""
    from handlers.sitemap import fetch_sitemap_urls_with_index

    mock_client = AsyncMock()
    mock_client.get.return_value = _mock_response(200, VALID_SITEMAP_XML)
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=None)

    with patch("handlers.sitemap.httpx.AsyncClient", return_value=mock_client):
        urls = await fetch_sitemap_urls_with_index("https://example.com")

    assert len(urls) == 3
    assert "https://example.com/page1" in urls
    assert "https://example.com/page2" in urls
    assert "https://example.com/docs/api" in urls


@pytest.mark.asyncio
async def test_fetch_sitemap_urls_404():
    """Verify 404 returns empty list (graceful fallback)."""
    from handlers.sitemap import fetch_sitemap_urls_with_index

    mock_client = AsyncMock()
    mock_client.get.return_value = _mock_response(404)
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=None)

    with patch("handlers.sitemap.httpx.AsyncClient", return_value=mock_client):
        urls = await fetch_sitemap_urls_with_index("https://example.com")

    assert urls == []


@pytest.mark.asyncio
async def test_fetch_sitemap_urls_timeout():
    """Verify timeout returns empty list (graceful fallback)."""
    from handlers.sitemap import fetch_sitemap_urls_with_index

    mock_client = AsyncMock()
    mock_client.get.side_effect = httpx.TimeoutException("Connection timed out")
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=None)

    with patch("handlers.sitemap.httpx.AsyncClient", return_value=mock_client):
        urls = await fetch_sitemap_urls_with_index("https://example.com")

    assert urls == []


@pytest.mark.asyncio
async def test_fetch_sitemap_urls_invalid_xml():
    """Verify invalid XML returns empty list (graceful fallback)."""
    from handlers.sitemap import fetch_sitemap_urls_with_index

    mock_client = AsyncMock()
    mock_client.get.return_value = _mock_response(200, "<not-valid-xml<>></broken>")
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=None)

    with patch("handlers.sitemap.httpx.AsyncClient", return_value=mock_client):
        urls = await fetch_sitemap_urls_with_index("https://example.com")

    assert urls == []


@pytest.mark.asyncio
async def test_fetch_sitemap_index():
    """Verify sitemap index is resolved by fetching sub-sitemaps."""
    from handlers.sitemap import fetch_sitemap_urls_with_index

    responses = {
        "https://example.com/sitemap.xml": _mock_response(200, SITEMAP_INDEX_XML),
        "https://example.com/sitemap-pages.xml": _mock_response(200, SUB_SITEMAP_PAGES),
        "https://example.com/sitemap-docs.xml": _mock_response(200, SUB_SITEMAP_DOCS),
    }

    mock_client = AsyncMock()
    mock_client.get.side_effect = lambda url: responses.get(url, _mock_response(404))
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=None)

    with patch("handlers.sitemap.httpx.AsyncClient", return_value=mock_client):
        urls = await fetch_sitemap_urls_with_index("https://example.com")

    assert len(urls) == 3
    assert "https://example.com/about" in urls
    assert "https://example.com/contact" in urls
    assert "https://example.com/docs/guide" in urls


@pytest.mark.asyncio
async def test_fetch_sitemap_urls_filters_external():
    """Verify only same-domain URLs are returned."""
    from handlers.sitemap import fetch_sitemap_urls_with_index

    mock_client = AsyncMock()
    mock_client.get.return_value = _mock_response(200, SITEMAP_WITH_EXTERNAL)
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=None)

    with patch("handlers.sitemap.httpx.AsyncClient", return_value=mock_client):
        urls = await fetch_sitemap_urls_with_index("https://example.com")

    assert "https://example.com/page1" in urls
    assert "https://example.com/page3" in urls
    assert "https://other-domain.com/page2" not in urls
    assert len(urls) == 2


@pytest.mark.asyncio
async def test_fetch_sitemap_urls_deduplicates():
    """Verify duplicate URLs are de-duplicated."""
    from handlers.sitemap import fetch_sitemap_urls_with_index

    duped_xml = """<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/page1</loc></url>
  <url><loc>https://example.com/page1</loc></url>
  <url><loc>https://example.com/page2</loc></url>
</urlset>"""

    mock_client = AsyncMock()
    mock_client.get.return_value = _mock_response(200, duped_xml)
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=None)

    with patch("handlers.sitemap.httpx.AsyncClient", return_value=mock_client):
        urls = await fetch_sitemap_urls_with_index("https://example.com")

    assert len(urls) == 2


@pytest.mark.asyncio
async def test_fetch_sitemap_urls_empty_response():
    """Verify empty response body returns empty list."""
    from handlers.sitemap import fetch_sitemap_urls_with_index

    mock_client = AsyncMock()
    mock_client.get.return_value = _mock_response(200, "")
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=None)

    with patch("handlers.sitemap.httpx.AsyncClient", return_value=mock_client):
        urls = await fetch_sitemap_urls_with_index("https://example.com")

    assert urls == []


@pytest.mark.asyncio
async def test_fetch_sitemap_urls_no_namespace():
    """Verify sitemaps without XML namespace still work."""
    from handlers.sitemap import fetch_sitemap_urls_with_index

    mock_client = AsyncMock()
    mock_client.get.return_value = _mock_response(200, SITEMAP_NO_NS)
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=None)

    with patch("handlers.sitemap.httpx.AsyncClient", return_value=mock_client):
        urls = await fetch_sitemap_urls_with_index("https://example.com")

    assert len(urls) == 1
    assert "https://example.com/no-ns-page" in urls


@pytest.mark.asyncio
async def test_fetch_sitemap_urls_connection_error():
    """Verify connection errors return empty list."""
    from handlers.sitemap import fetch_sitemap_urls_with_index

    mock_client = AsyncMock()
    mock_client.get.side_effect = httpx.ConnectError("Connection refused")
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=None)

    with patch("handlers.sitemap.httpx.AsyncClient", return_value=mock_client):
        urls = await fetch_sitemap_urls_with_index("https://example.com")

    assert urls == []
