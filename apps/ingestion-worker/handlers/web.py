from typing import Any

import asyncio
import structlog
import re
import time as time_mod
from urllib.parse import urljoin, urlparse
from handlers.sitemap import fetch_sitemap_urls_with_index
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

# --- LLM Content Filter Circuit Breaker ---
_llm_consecutive_failures: int = 0
_llm_circuit_open_until: float = 0.0
_LLM_CIRCUIT_THRESHOLD = 3
_LLM_CIRCUIT_COOLDOWN_S = 300.0  # 5 minutes


def _is_llm_circuit_open() -> bool:
    """Check if the LLM circuit breaker is open (too many recent failures)."""
    return time_mod.monotonic() < _llm_circuit_open_until


def _record_llm_failure() -> None:
    """Record an LLM filter failure; opens circuit after threshold."""
    global _llm_consecutive_failures, _llm_circuit_open_until
    _llm_consecutive_failures += 1
    if _llm_consecutive_failures >= _LLM_CIRCUIT_THRESHOLD:
        _llm_circuit_open_until = time_mod.monotonic() + _LLM_CIRCUIT_COOLDOWN_S
        logger.warning(
            "llm_circuit_opened",
            operation="handle_web_task",
            cooldown_s=_LLM_CIRCUIT_COOLDOWN_S,
            failures=_llm_consecutive_failures,
        )


def _record_llm_success() -> None:
    """Reset the circuit breaker after a successful LLM filter call."""
    global _llm_consecutive_failures, _llm_circuit_open_until
    _llm_consecutive_failures = 0
    _llm_circuit_open_until = 0.0


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


def _get_raw_markdown(result: Any) -> str:
    """
    Extract raw markdown string from a crawl4ai result.

    In crawl4ai v0.5+, result.markdown is a MarkdownGenerationResult object.
    This helper safely extracts the raw markdown string for use in regex-based
    link extraction and title extraction.
    """
    md = result.markdown
    if hasattr(md, "raw_markdown"):
        return md.raw_markdown or ""
    if isinstance(md, str):
        return md
    return str(md) if md else ""


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
    # Always use raw markdown for link discovery — fit_markdown may have links stripped
    raw_md = _get_raw_markdown(result)
    if raw_md:
        markdown_links = re.findall(r"\[.*?\]\((.*?)\)", raw_md)
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
    if raw_md:
        match = re.search(r"^#\s+(.+)$", raw_md, re.MULTILINE)
        if match:
            title = match.group(1).strip()

    # Extract path (breadcrumbs)
    parsed_url = urlparse(result.url)
    path_segments = [s for s in parsed_url.path.split("/") if s]
    path_str = " > ".join(path_segments)

    return {"title": title, "path": path_str, "links": internal_links}


def _get_embedding_content(result: Any) -> str:
    """
    Extract clean content for embedding from a crawl4ai result.

    In crawl4ai v0.5+, result.markdown is a MarkdownGenerationResult object
    with fit_markdown (LLM-filtered) and raw_markdown (unfiltered). We prefer
    fit_markdown because it has navigation, sidebars, and boilerplate removed.

    Falls back to raw_markdown or string form for backwards compatibility.
    """
    md = result.markdown

    # Handle MarkdownGenerationResult object (crawl4ai v0.5+)
    if hasattr(md, "fit_markdown"):
        fit = md.fit_markdown
        if fit and fit.strip():
            return fit
        # fit_markdown empty (e.g. .txt files, filter produced nothing)
        raw = getattr(md, "raw_markdown", "")
        return raw if raw else ""

    # Handle plain string (older crawl4ai or pre-filtered content)
    if isinstance(md, str):
        return md

    return str(md) if md else ""


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
    logger.info("crawl_starting", operation="handle_web_task", url=url)
    start = time_mod.monotonic()

    # Use passed api_key or fallback to settings
    token = api_key if api_key else app_settings.gemini_api_key

    # Determine if LLM filtering should be used
    is_text_file = url.endswith(".txt") or url.endswith("llms.txt")
    use_llm = not is_text_file

    # Configure Generator (Bypass LLM for .txt/llms.txt or when circuit is open)
    if not use_llm:
        md_generator = DefaultMarkdownGenerator()
        logger.info(
            "llm_bypass_enabled",
            operation="handle_web_task",
            url=url,
            reason="text_file",
        )
    elif _is_llm_circuit_open():
        md_generator = DefaultMarkdownGenerator()
        logger.info("llm_bypass_circuit_open", operation="handle_web_task", url=url)
    else:
        llm_config = LLMConfig(
            provider="gemini/gemini-3-flash-preview", api_token=token, temperature=0.0
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

            # Success — update LLM circuit breaker state
            if use_llm and not _is_llm_circuit_open():
                md = result.markdown
                if hasattr(md, "fit_markdown") and (
                    not md.fit_markdown or not md.fit_markdown.strip()
                ):
                    _record_llm_failure()
                else:
                    _record_llm_success()

            meta = extract_web_metadata(result, url)

            # Sitemap discovery: only for root URLs (seed pages)
            parsed_url = urlparse(url)
            is_root = parsed_url.path in ("", "/")
            if is_root:
                try:
                    sitemap_urls = await fetch_sitemap_urls_with_index(url)
                    if sitemap_urls:
                        # Merge sitemap URLs into discovered links
                        existing = set(meta["links"])
                        new_from_sitemap = [
                            u for u in sitemap_urls if u not in existing
                        ]
                        meta["links"].extend(new_from_sitemap)
                        logger.info(
                            "sitemap_urls_merged",
                            operation="handle_web_task",
                            url=url,
                            sitemap_count=len(new_from_sitemap),
                            total_links=len(meta["links"]),
                        )
                except Exception as e:
                    # Sitemap failure is non-blocking
                    logger.warning(
                        "sitemap_discovery_error",
                        operation="handle_web_task",
                        url=url,
                        error=str(e),
                    )

            content = _get_embedding_content(result)
            elapsed_ms = (time_mod.monotonic() - start) * 1000
            logger.info(
                "crawl_completed",
                operation="handle_web_task",
                url=url,
                links_found=len(meta["links"]),
                title=meta["title"],
                content_length=len(content),
                duration_ms=round(elapsed_ms, 1),
                attempt=attempt,
            )

            return [
                {
                    "url": result.url,
                    "title": meta["title"],
                    "path": meta["path"],
                    "content": content,
                    "links": meta["links"],
                    "metadata": {},  # Web pages have no doc-level metadata
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
                    operation="handle_web_task",
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
                    operation="handle_web_task",
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
                operation="handle_web_task",
                url=url,
                attempt=attempt,
                next_delay_s=delay,
                error=str(last_error),
            )
            await asyncio.sleep(delay)

    # All retries exhausted
    elapsed_ms = (time_mod.monotonic() - start) * 1000
    logger.error(
        "crawl_retries_exhausted",
        operation="handle_web_task",
        url=url,
        attempts=CRAWL_MAX_RETRIES + 1,
        error=str(last_error),
        duration_ms=round(elapsed_ms, 1),
    )
    raise last_error  # type: ignore[misc]
