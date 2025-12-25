import asyncio
import structlog
import json
import httpx
import re
from urllib.parse import urljoin, urlparse
from crawl4ai import AsyncWebCrawler, CrawlerRunConfig, CacheMode, LLMConfig
try:
    from crawl4ai import AsyncUrlSeeder, SeedingConfig
    HAS_SEEDER = True
except ImportError:
    HAS_SEEDER = False
from crawl4ai.deep_crawling import BFSDeepCrawlStrategy
from crawl4ai.content_filter_strategy import PruningContentFilter, LLMContentFilter
from crawl4ai.deep_crawling.filters import URLPatternFilter, FilterChain
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

async def discover_urls(url: str) -> list[str]:
    """
    Discovers URLs from llms.txt and sitemap.xml.
    """
    urls = []
    parsed_url = urlparse(url)
    root_url = f"{parsed_url.scheme}://{parsed_url.netloc}"
    
    # 1. Check llms.txt at root
    llms_url = urljoin(root_url, "/llms.txt")
    try:
        async with httpx.AsyncClient(timeout=10.0) as client:
            resp = await client.get(llms_url)
            if resp.status_code == 200:
                logger.info("llms_txt_found", url=llms_url)
                # Parse markdown links: [text](link)
                links = re.findall(r'\[.*?\]\((.*?)\)', resp.text)
                for link in links:
                    full_url = urljoin(llms_url, link)
                    # Only include links from same domain
                    if urlparse(full_url).netloc == parsed_url.netloc:
                        urls.append(full_url)
    except Exception as e:
        logger.debug("llms_txt_not_found", url=llms_url, error=str(e))

    # 2. Check Sitemap
    if HAS_SEEDER:
        try:
            async with AsyncUrlSeeder() as seeder:
                # Try default sitemap
                config = SeedingConfig(source="sitemap")
                sitemap_urls = await seeder.urls(url, config)
                urls.extend([u['url'] for u in sitemap_urls])
        except Exception as e:
            logger.debug("sitemap_discovery_failed", url=url, error=str(e))
            
    return list(set(urls)) # Deduplicate

async def handle_web_task(url: str, max_depth: int = 0, exclusions: list[str] = None, api_key: str = None) -> list[dict]:
    """
    Crawls a website recursively and returns a list of dictionaries containing url and content.
    """
    logger.info("crawl_starting", url=url, depth=max_depth)
    
    # Discovery (Sitemap/llms.txt)
    seed_urls = []
    if max_depth > 0:
        seed_urls = await discover_urls(url)
        if seed_urls:
            logger.info("discovered_urls", count=len(seed_urls))
    
    if exclusions is None:
        exclusions = []
        
    # Configure Filters
    url_filter = URLPatternFilter(patterns=exclusions, reverse=True) if exclusions else None
    
    # Use passed api_key or fallback to settings
    token = api_key if api_key else app_settings.gemini_api_key
    
    llm_config = LLMConfig(
        provider="gemini/gemini-3-flash-preview", 
        api_token=token,
        temperature=1.0
    )

    llm_filter = LLMContentFilter(
        llm_config=llm_config,
        instruction=INSTRUCTION,
        chunk_token_threshold=8000
    )
    
    pruning_filter = PruningContentFilter(
        threshold=0.30, 
        min_word_threshold=5, 
        threshold_type="fixed"
    )

    # Configure Strategy
    if max_depth > 0:
        # Ensure filter_chain is always a FilterChain instance, even if empty
        filters = [url_filter] if url_filter else []
        filter_chain = FilterChain(filters)
        
        deep_crawl_strategy = BFSDeepCrawlStrategy(
            max_depth=max_depth,
            include_external=False,
            filter_chain=filter_chain
        )
    else:
        deep_crawl_strategy = None

    md_generator = DefaultMarkdownGenerator(content_filter=llm_filter)

    config = CrawlerRunConfig(
        cache_mode=CacheMode.ENABLED,
        excluded_tags=['nav', 'footer', 'aside', 'header'],
        exclude_external_links=True, # Enforce no external links
        deep_crawl_strategy=deep_crawl_strategy,
        markdown_generator=md_generator
    )
    
    # Initialize crawler
    try:
        async with AsyncWebCrawler(verbose=True) as crawler:
            if max_depth > 0:
                # Recursive crawl
                results = []
                # Calculate timeout based on depth: 60s + 60s per depth level
                recursive_timeout = 60.0 + (max_depth * 60.0)
                
                if seed_urls:
                    # If we have seed URLs, we crawl them in batch.
                    # This is more efficient than discovery-based deep crawl if we already have the list.
                    urls_to_crawl = list(set([url] + seed_urls))
                    logger.info("batch_crawling_seeds", count=len(urls_to_crawl))
                    run_results = await asyncio.wait_for(
                        crawler.arun_many(urls=urls_to_crawl, config=config),
                        timeout=recursive_timeout * 2 # Increase timeout for batch
                    )
                else:
                    # Traditional deep crawl starting from single URL
                    run_results = await asyncio.wait_for(
                        crawler.arun(url=url, config=config),
                        timeout=recursive_timeout
                    )
                
                # Verify we got a list
                if not isinstance(run_results, list):
                    run_results = [run_results]

                for result in run_results:
                    if result.success:
                        results.append({"url": result.url, "content": result.markdown})
                    else:
                        logger.error("crawl_failed", url=result.url, error=result.error_message)
                return results
            else:
                # Single page crawl
                result = await asyncio.wait_for(
                    crawler.arun(url=url, config=config),
                    timeout=60.0
                )
                if not result.success:
                    logger.error("crawl_failed", url=url, error=result.error_message)
                    raise Exception(f"Crawl failed: {result.error_message}")
                return [{"url": result.url, "content": result.markdown}]
    except asyncio.TimeoutError:
        logger.error("crawl_timeout", url=url)
        raise
    except Exception as e:
        logger.error("crawl_exception", url=url, error=str(e))
        raise
