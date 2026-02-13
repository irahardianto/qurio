from typing import Any

import asyncio
import structlog
import re
from urllib.parse import urljoin, urlparse
from crawl4ai import AsyncWebCrawler, CrawlerRunConfig, CacheMode, LLMConfig
from crawl4ai.content_filter_strategy import LLMContentFilter
from crawl4ai.markdown_generation_strategy import DefaultMarkdownGenerator
from config import settings as app_settings
from exceptions import (
    IngestionError,
    ERR_CRAWL_TIMEOUT,
    ERR_CRAWL_DNS,
    ERR_CRAWL_REFUSED,
    ERR_CRAWL_BLOCKED,
    TRANSIENT_ERRORS,
)

logger = structlog.get_logger(__name__)

# Retry configuration for transient crawl errors
CRAWL_MAX_RETRIES = 2
CRAWL_INITIAL_BACKOFF_S = 2.0

INSTRUCTION = """
    Extract technical content from this software documentation page.
    
    KEEP:
    - All code examples with their comments
    - Function/method signatures and parameters
    - Configuration examples and syntax
    - Technical explanations and concepts
    - Error messages and troubleshooting steps
    - Links to related API documentation
    
    REMOVE:
    - Navigation menus and sidebars
    - Copyright and legal notices
    - Unrelated marketing content
    - "Edit this page" links
    - Cookie banners and consent forms
    
    PRESERVE:
    - Code block language annotations (```go, etc.)
    - Heading hierarchy for context
    - Inline code references
    - Numbered lists for sequential steps
"""


def _classify_crawl_error(error_message: str) -> IngestionError:
    """
    Classify a crawl4ai error message into a specific IngestionError.
    Matches against known Playwright/Chromium net error patterns.
    """
    msg_upper = error_message.upper()

    # Timeout errors (transient)
    if "TIMED_OUT" in msg_upper or "TIMEOUT" in msg_upper:
        return IngestionError(ERR_CRAWL_TIMEOUT, error_message)

    # DNS resolution errors (transient — could be temporary DNS issues)
    if "ERR_NAME_NOT_RESOLVED" in msg_upper or "DNS" in msg_upper:
        return IngestionError(ERR_CRAWL_DNS, error_message)

    # Connection errors (transient)
    if any(
        kw in msg_upper
        for kw in [
            "ERR_CONNECTION_REFUSED",
            "ERR_CONNECTION_RESET",
            "ERR_CONNECTION_CLOSED",
            "ECONNREFUSED",
            "ECONNRESET",
        ]
    ):
        return IngestionError(ERR_CRAWL_REFUSED, error_message)

    # Blocked by robots.txt or similar (permanent)
    if "ROBOTS" in msg_upper or "BLOCKED" in msg_upper or "FORBIDDEN" in msg_upper:
        return IngestionError(ERR_CRAWL_BLOCKED, error_message)

    # Default: treat as timeout (transient) to be safe — better to retry than drop
    return IngestionError(ERR_CRAWL_TIMEOUT, error_message)


def extract_web_metadata(result, url: str) -> dict:
    """
    Extracts metadata (title, path, links) from a crawl result.
    """
    # Extract internal links
    # Crawl4AI result.links is usually a dictionary with 'internal' and 'external' keys
    # containing lists of dicts (href, text, etc.)
    internal_links = []
    if result.links and "internal" in result.links:
        for link in result.links["internal"]:
            if "href" in link:
                internal_links.append(link["href"])

    # Additional Regex Extraction for Markdown (e.g. llms.txt)
    if result.markdown:
        markdown_links = re.findall(r"\[.*?\]\((.*?)\)", result.markdown)
        parsed_base = urlparse(url)
        base_domain = parsed_base.netloc

        for link in markdown_links:
            # Resolve relative URLs
            full_url = urljoin(url, link)
            # Filter internal
            if urlparse(full_url).netloc == base_domain:
                internal_links.append(full_url)

    # De-duplicate
    internal_links = list(set(internal_links))

    # Extract title (simplistic regex fallback if not in result)
    title = ""
    if result.markdown:
        match = re.search(r"^#\s+(.+)$", result.markdown, re.MULTILINE)
        if match:
            title = match.group(1).strip()

    # Extract path (breadcrumbs)
    parsed_url = urlparse(result.url)
    path_segments = [s for s in parsed_url.path.split("/") if s]
    path_str = " > ".join(path_segments)

    return {"title": title, "path": path_str, "links": internal_links}


def default_crawler_factory(config=None, **kwargs):
    return AsyncWebCrawler(config=config, **kwargs)


async def _crawl_single_page(crawler: Any, url: str, config: Any) -> Any:
    """
    Execute a single crawl attempt with outer timeout.
    Returns the crawl result on success, raises IngestionError on failure.
    """
    outer_timeout = (app_settings.crawler_page_timeout / 1000) + 5.0

    result = await asyncio.wait_for(
        crawler.arun(url=url, config=config), timeout=outer_timeout
    )

    if not result.success:
        raise _classify_crawl_error(result.error_message)

    return result


async def handle_web_task(
    url: str, api_key: str | None = None, crawler=None
) -> list[dict]:
    """
    Crawls a single page and returns content and discovered internal links.
    Retries transient crawl errors with exponential backoff before escalating.
    """
    logger.info("crawl_starting", url=url)

    # Use passed api_key or fallback to settings
    token = api_key if api_key else app_settings.gemini_api_key

    # Configure Generator (Bypass LLM for .txt/llms.txt)
    if url.endswith(".txt") or url.endswith("llms.txt"):
        md_generator = DefaultMarkdownGenerator()
        logger.info("llm_bypass_enabled", url=url, reason="text_file")
    else:
        llm_config = LLMConfig(
            provider="gemini/gemini-3-flash-preview", api_token=token, temperature=1.0
        )

        llm_filter = LLMContentFilter(
            llm_config=llm_config, instruction=INSTRUCTION, chunk_token_threshold=8000
        )

        md_generator = DefaultMarkdownGenerator(content_filter=llm_filter)

    config = CrawlerRunConfig(
        cache_mode=CacheMode.ENABLED,
        exclude_external_links=True,
        markdown_generator=md_generator,
        check_robots_txt=True,
        page_timeout=app_settings.crawler_page_timeout,
    )

    last_error: Exception | None = None

    for attempt in range(1, CRAWL_MAX_RETRIES + 2):  # 1 initial + CRAWL_MAX_RETRIES
        try:
            if crawler:
                result = await _crawl_single_page(crawler, url, config)
            else:
                async with default_crawler_factory(verbose=True) as new_crawler:
                    result = await _crawl_single_page(new_crawler, url, config)

            # Success
            meta = extract_web_metadata(result, url)

            logger.info(
                "crawl_completed",
                url=url,
                links_found=len(meta["links"]),
                title=meta["title"],
                path=meta["path"],
                attempt=attempt,
            )

            return [
                {
                    "url": result.url,
                    "title": meta["title"],
                    "path": meta["path"],
                    "content": result.markdown,
                    "links": meta["links"],
                }
            ]

        except asyncio.TimeoutError:
            last_error = IngestionError(
                ERR_CRAWL_TIMEOUT,
                f"Crawl timed out after {app_settings.crawler_page_timeout}ms",
            )
        except IngestionError as e:
            last_error = e
            # Permanent errors: don't retry
            if e.code not in TRANSIENT_ERRORS:
                logger.error(
                    "crawl_permanent_error",
                    url=url,
                    error=str(e),
                    code=e.code,
                    attempt=attempt,
                )
                raise
        except Exception as e:
            # Unexpected error — classify it
            last_error = _classify_crawl_error(str(e))
            if last_error.code not in TRANSIENT_ERRORS:
                logger.error(
                    "crawl_permanent_error",
                    url=url,
                    error=str(e),
                    attempt=attempt,
                )
                raise last_error

        # Transient error: retry with backoff (unless this was the last attempt)
        if attempt <= CRAWL_MAX_RETRIES:
            delay = CRAWL_INITIAL_BACKOFF_S * (2 ** (attempt - 1))
            logger.warning(
                "crawl_transient_retry",
                url=url,
                attempt=attempt,
                next_delay_s=delay,
                error=str(last_error),
            )
            await asyncio.sleep(delay)

    # All retries exhausted
    logger.error(
        "crawl_retries_exhausted",
        url=url,
        attempts=CRAWL_MAX_RETRIES + 1,
        error=str(last_error),
    )
    raise last_error  # type: ignore[misc]
