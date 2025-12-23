import asyncio
import structlog
from concurrent.futures import ThreadPoolExecutor
from docling.document_converter import DocumentConverter

logger = structlog.get_logger(__name__)

# Initialize converter globally to reuse resources
converter = DocumentConverter()
executor = ThreadPoolExecutor(max_workers=2)

async def handle_file_task(file_path: str) -> str:
    """
    Converts a document to markdown using Docling.
    Executes blocking code in a thread pool.
    """
    logger.info("conversion_starting", path=file_path)
    
    loop = asyncio.get_running_loop()
    
    try:
        # Run synchronous convert method in thread pool
        result = await asyncio.wait_for(
            loop.run_in_executor(
                executor,
                converter.convert,
                file_path
            ),
            timeout=60.0
        )
        
        return result.document.export_to_markdown()

    except asyncio.TimeoutError:
        logger.error("conversion_timeout", path=file_path)
        raise
    except Exception as e:
        logger.error("conversion_failed", path=file_path, error=str(e))
        raise Exception(f"Conversion failed: {e}")
