import asyncio
import structlog
import re
from urllib.parse import urljoin, urlparse
from crawl4ai import AsyncWebCrawler, CrawlerRunConfig, CacheMode, LLMConfig
from crawl4ai.content_filter_strategy import LLMContentFilter
from crawl4ai.markdown_generation_strategy import DefaultMarkdownGenerator
from config import settings as app_settings

logger = structlog.get_logger(__name__)

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


async def handle_web_task(
    url: str, api_key: str | None = None, crawler=None
) -> list[dict]:
    """
    Crawls a single page and returns content and discovered internal links.
    """
    logger.info("crawl_starting", url=url)

    # Use passed api_key or fallback to settings
    token = api_key if api_key else app_settings.gemini_api_key

    results = []

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
        # Remove excluded_tags to ensure links in nav/sidebar are discovered.
        # The LLMContentFilter will handle removing them from the content.
        # excluded_tags=['nav', 'footer', 'aside', 'header'],
        exclude_external_links=True,
        markdown_generator=md_generator,
        check_robots_txt=True,
        page_timeout=app_settings.crawler_page_timeout,
    )

    # Initialize crawler
    try:
        # Use existing crawler if provided, otherwise create a new ephemeral one
        if crawler:
            # Single page crawl using shared instance
            # Use configured timeout (converted to seconds) plus a small buffer (e.g. 5s)
            outer_timeout = (app_settings.crawler_page_timeout / 1000) + 5.0

            result = await asyncio.wait_for(
                crawler.arun(url=url, config=config), timeout=outer_timeout
            )

            if not result.success:
                logger.error("crawl_failed", url=url, error=result.error_message)
                raise Exception(f"Crawl failed: {result.error_message}")

            meta = extract_web_metadata(result, url)

            logger.info(
                "crawl_completed",
                url=url,
                links_found=len(meta["links"]),
                title=meta["title"],
                path=meta["path"],
            )

            results.append(
                {
                    "url": result.url,
                    "title": meta["title"],
                    "path": meta["path"],
                    "content": result.markdown,
                    "links": meta["links"],
                }
            )
            return results

        else:
            # Ephemeral crawler (old behavior)
            async with default_crawler_factory(verbose=True) as new_crawler:
                outer_timeout = (app_settings.crawler_page_timeout / 1000) + 5.0

                result = await asyncio.wait_for(
                    new_crawler.arun(url=url, config=config), timeout=outer_timeout
                )

                if not result.success:
                    logger.error("crawl_failed", url=url, error=result.error_message)
                    raise Exception(f"Crawl failed: {result.error_message}")

                meta = extract_web_metadata(result, url)

                logger.info(
                    "crawl_completed",
                    url=url,
                    links_found=len(meta["links"]),
                    title=meta["title"],
                    path=meta["path"],
                )

                results.append(
                    {
                        "url": result.url,
                        "title": meta["title"],
                        "path": meta["path"],
                        "content": result.markdown,
                        "links": meta["links"],
                    }
                )

                return results

    except asyncio.TimeoutError:
        logger.error("crawl_timeout", url=url)
        raise
    except Exception as e:
        logger.error("crawl_exception", url=url, error=str(e))
        raise
