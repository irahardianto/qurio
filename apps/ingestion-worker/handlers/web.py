import logging
import json
from crawl4ai import AsyncWebCrawler, CrawlerRunConfig, CacheMode, LLMConfig
from crawl4ai.deep_crawling import BFSDeepCrawlStrategy
from crawl4ai.content_filter_strategy import PruningContentFilter, LLMContentFilter
from crawl4ai.deep_crawling.filters import URLPatternFilter
from crawl4ai.markdown_generation_strategy import DefaultMarkdownGenerator
from config import settings as app_settings

logger = logging.getLogger(__name__)

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

async def handle_web_task(url: str, max_depth: int = 0, exclusions: list[str] = None, api_key: str = None) -> list[dict]:
    """
    Crawls a website recursively and returns a list of dictionaries containing url and content.
    """
    logger.info(f"Starting crawl for {url} with depth {max_depth}")
    
    if exclusions is None:
        exclusions = []
        
    # Configure Filters
    url_filter = URLPatternFilter(patterns=exclusions, reverse=True) if exclusions else None
    
    # Use passed api_key or fallback to settings
    token = api_key if api_key else app_settings.gemini_api_key
    
    llm_config = LLMConfig(
        provider="gemini/gemini-3-flash-preview", 
        api_token=token
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
    deep_crawl_strategy = BFSDeepCrawlStrategy(
        max_depth=max_depth,
        include_external=False
        # url_filter=url_filter # Removed due to unexpected keyword argument error
    ) if max_depth > 0 else None

    md_generator = DefaultMarkdownGenerator(content_filter=llm_filter)

    config = CrawlerRunConfig(
        cache_mode=CacheMode.ENABLED,
        excluded_tags=['nav', 'footer', 'aside', 'header'],
        exclude_external_links=True, # Enforce no external links
        deep_crawl_strategy=deep_crawl_strategy,
        markdown_generator=md_generator
    )
    
    # Initialize crawler
    async with AsyncWebCrawler(verbose=True) as crawler:
        if max_depth > 0:
            # Recursive crawl
            results = []
            run_results = await crawler.arun_many(urls=[url], config=config)
            for result in run_results:
                if result.success:
                    results.append({"url": result.url, "content": result.markdown})
                else:
                    logger.error(f"Crawl failed for {result.url}: {result.error_message}")
            return results
        else:
            # Single page crawl
            result = await crawler.arun(url=url, config=config)
            if not result.success:
                logger.error(f"Crawl failed for {url}: {result.error_message}")
                raise Exception(f"Crawl failed: {result.error_message}")
            return [{"url": result.url, "content": result.markdown}]
