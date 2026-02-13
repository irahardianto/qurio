"""
Sitemap detection and parsing for URL discovery.

Fetches and parses sitemap.xml files to extract URLs for crawling.
Handles both standard sitemaps and sitemap index files (recursive fetch).
Falls back gracefully on errors (404, timeout, invalid XML).
"""

from urllib.parse import urljoin, urlparse
from defusedxml import ElementTree

import httpx
import structlog

logger = structlog.get_logger(__name__)

# Sitemap XML namespace
SITEMAP_NS = "http://www.sitemaps.org/schemas/sitemap/0.9"

# Limits
MAX_SITEMAP_INDEX_DEPTH = 3
SITEMAP_FETCH_TIMEOUT_S = 15.0


async def fetch_sitemap_urls(base_url: str) -> list[str]:
    """
    Discover URLs from a sitemap.xml at the given base URL.

    Attempts to fetch {base_url}/sitemap.xml, parse it, and extract all <loc>
    URLs. Handles sitemap index files by recursively fetching sub-sitemaps.

    Returns an empty list on any error (404, timeout, invalid XML).
    Only returns URLs that belong to the same domain as base_url.

    Args:
        base_url: The root URL of the site (e.g. "https://docs.example.com").

    Returns:
        List of discovered URLs from the sitemap, or empty list on failure.
    """
    parsed_base = urlparse(base_url)
    base_domain = parsed_base.netloc

    sitemap_url = urljoin(base_url.rstrip("/") + "/", "sitemap.xml")

    logger.info("sitemap_check_starting", base_url=base_url, sitemap_url=sitemap_url)

    try:
        urls = await _fetch_and_parse_sitemap(sitemap_url, base_domain, depth=0)
        logger.info(
            "sitemap_check_completed",
            base_url=base_url,
            urls_found=len(urls),
        )
        return urls
    except Exception as e:
        logger.warning(
            "sitemap_check_failed",
            base_url=base_url,
            error=str(e),
        )
        return []


async def _fetch_and_parse_sitemap(
    sitemap_url: str,
    base_domain: str,
    depth: int,
) -> list[str]:
    """
    Fetch a single sitemap URL and parse its contents.

    Handles both:
    - Standard sitemaps: <urlset> containing <url><loc>...</loc></url>
    - Sitemap indexes: <sitemapindex> containing <sitemap><loc>...</loc></sitemap>

    Args:
        sitemap_url: URL of the sitemap to fetch.
        base_domain: Domain to filter URLs (only same-domain URLs returned).
        depth: Current recursion depth (prevents runaway fetching).

    Returns:
        List of discovered URLs.
    """
    if depth > MAX_SITEMAP_INDEX_DEPTH:
        logger.warning(
            "sitemap_max_depth_exceeded",
            sitemap_url=sitemap_url,
            max_depth=MAX_SITEMAP_INDEX_DEPTH,
        )
        return []

    async with httpx.AsyncClient(
        timeout=SITEMAP_FETCH_TIMEOUT_S,
        follow_redirects=True,
    ) as client:
        response = await client.get(sitemap_url)

    if response.status_code != 200:
        logger.info(
            "sitemap_not_found",
            sitemap_url=sitemap_url,
            status_code=response.status_code,
        )
        return []

    content = response.text
    if not content.strip():
        logger.info("sitemap_empty", sitemap_url=sitemap_url)
        return []

    return _parse_sitemap_xml(content, base_domain, sitemap_url, depth)


def _parse_sitemap_xml(
    xml_content: str,
    base_domain: str,
    sitemap_url: str,
    depth: int,
) -> list[str]:
    """
    Parse sitemap XML and extract URLs synchronously.

    This is a pure function that handles XML parsing. Async recursion for
    sitemap indexes is handled by the caller (_fetch_and_parse_sitemap).

    Returns a list of URLs for standard sitemaps, or triggers recursive
    fetching for sitemap index files.
    """
    try:
        root = ElementTree.fromstring(xml_content)
    except ElementTree.ParseError as e:
        logger.warning(
            "sitemap_xml_parse_error",
            sitemap_url=sitemap_url,
            error=str(e),
        )
        return []

    tag = root.tag

    # Standard sitemap: <urlset>
    if tag == f"{{{SITEMAP_NS}}}urlset" or tag == "urlset":
        return _extract_urls_from_urlset(root, base_domain)

    # Sitemap index: <sitemapindex>
    if tag == f"{{{SITEMAP_NS}}}sitemapindex" or tag == "sitemapindex":
        # We can't do async recursion from a sync function,
        # so return sub-sitemap URLs as a marker for the caller.
        # Instead, collect sub-sitemap URLs here.
        return _extract_sub_sitemap_urls(root)

    logger.warning(
        "sitemap_unknown_root_element",
        sitemap_url=sitemap_url,
        tag=tag,
    )
    return []


def _extract_urls_from_urlset(root: ElementTree.Element, base_domain: str) -> list[str]:
    """Extract <loc> URLs from a standard <urlset> sitemap."""
    urls: list[str] = []

    for url_elem in root.iter():
        if url_elem.tag in (f"{{{SITEMAP_NS}}}loc", "loc"):
            loc = url_elem.text
            if loc and loc.strip():
                loc = loc.strip()
                parsed = urlparse(loc)
                # Only include URLs from the same domain
                if parsed.netloc == base_domain:
                    urls.append(loc)

    return list(set(urls))  # De-duplicate


def _extract_sub_sitemap_urls(root: ElementTree.Element) -> list[str]:
    """Extract sub-sitemap <loc> URLs from a <sitemapindex>."""
    urls: list[str] = []

    for elem in root.iter():
        if elem.tag in (f"{{{SITEMAP_NS}}}loc", "loc"):
            loc = elem.text
            if loc and loc.strip():
                urls.append(loc.strip())

    return urls


async def fetch_sitemap_urls_with_index(base_url: str) -> list[str]:
    """
    Full sitemap discovery including sitemap index resolution.

    This is the main entry point. It fetches the sitemap, and if it's a
    sitemap index, recursively fetches all sub-sitemaps.

    Args:
        base_url: The root URL of the site.

    Returns:
        All discovered page URLs from the sitemap(s).
    """
    parsed_base = urlparse(base_url)
    base_domain = parsed_base.netloc

    sitemap_url = urljoin(base_url.rstrip("/") + "/", "sitemap.xml")

    logger.info(
        "sitemap_discovery_starting", base_url=base_url, sitemap_url=sitemap_url
    )

    try:
        all_urls = await _resolve_sitemap(sitemap_url, base_domain, depth=0)

        # Final domain filter
        filtered = [u for u in all_urls if urlparse(u).netloc == base_domain]
        unique = list(set(filtered))

        logger.info(
            "sitemap_discovery_completed",
            base_url=base_url,
            total_urls=len(unique),
        )
        return unique

    except Exception as e:
        logger.warning(
            "sitemap_discovery_failed",
            base_url=base_url,
            error=str(e),
        )
        return []


async def _resolve_sitemap(
    sitemap_url: str,
    base_domain: str,
    depth: int,
) -> list[str]:
    """
    Resolve a sitemap URL, handling both standard sitemaps and indexes.

    For sitemap indexes, recursively fetches each sub-sitemap.
    """
    if depth > MAX_SITEMAP_INDEX_DEPTH:
        logger.warning(
            "sitemap_index_max_depth",
            url=sitemap_url,
            max_depth=MAX_SITEMAP_INDEX_DEPTH,
        )
        return []

    try:
        async with httpx.AsyncClient(
            timeout=SITEMAP_FETCH_TIMEOUT_S,
            follow_redirects=True,
        ) as client:
            response = await client.get(sitemap_url)
    except (httpx.TimeoutException, httpx.ConnectError) as e:
        logger.warning("sitemap_fetch_error", url=sitemap_url, error=str(e))
        return []

    if response.status_code != 200:
        logger.info(
            "sitemap_not_found",
            url=sitemap_url,
            status=response.status_code,
        )
        return []

    content = response.text
    if not content.strip():
        return []

    try:
        root = ElementTree.fromstring(content)
    except ElementTree.ParseError as e:
        logger.warning("sitemap_parse_error", url=sitemap_url, error=str(e))
        return []

    tag = root.tag

    # Standard sitemap
    if tag in (f"{{{SITEMAP_NS}}}urlset", "urlset"):
        return _extract_urls_from_urlset(root, base_domain)

    # Sitemap index — recursively fetch sub-sitemaps
    if tag in (f"{{{SITEMAP_NS}}}sitemapindex", "sitemapindex"):
        sub_urls = _extract_sub_sitemap_urls(root)
        logger.info(
            "sitemap_index_detected",
            url=sitemap_url,
            sub_sitemaps=len(sub_urls),
        )

        all_page_urls: list[str] = []
        for sub_url in sub_urls:
            pages = await _resolve_sitemap(sub_url, base_domain, depth + 1)
            all_page_urls.extend(pages)

        return all_page_urls

    logger.warning("sitemap_unknown_format", url=sitemap_url, tag=tag)
    return []
