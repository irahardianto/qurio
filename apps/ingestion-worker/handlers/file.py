import asyncio
import structlog
from concurrent.futures import ThreadPoolExecutor
from docling.document_converter import DocumentConverter
import os

logger = structlog.get_logger(__name__)

# Error Taxonomy
ERR_ENCRYPTED = "ERR_ENCRYPTED"
ERR_INVALID_FORMAT = "ERR_INVALID_FORMAT"
ERR_EMPTY = "ERR_EMPTY"
ERR_TIMEOUT = "ERR_TIMEOUT"

class IngestionError(Exception):
    def __init__(self, code, message):
        self.code = code
        super().__init__(message)

# Initialize converter globally to reuse resources
converter = DocumentConverter()
executor = ThreadPoolExecutor(max_workers=2)

CONCURRENCY_LIMIT = asyncio.Semaphore(2)
TIMEOUT_SECONDS = 300

async def handle_file_task(file_path: str) -> dict:
    """
    Converts a document to markdown using Docling.
    Executes blocking code in a thread pool.
    """
    logger.info("conversion_starting", path=file_path)
    
    loop = asyncio.get_running_loop()
    
    async with CONCURRENCY_LIMIT:
        try:
            # Run synchronous convert method in thread pool
            result = await asyncio.wait_for(
                loop.run_in_executor(
                    executor,
                    converter.convert,
                    file_path
                ),
                timeout=TIMEOUT_SECONDS
            )
            
            content = result.document.export_to_markdown()
            
            if not content.strip():
                 raise IngestionError(ERR_EMPTY, "File contains no text")

            # Extract metadata
            meta = {
                "title": getattr(result.document.meta, 'title', None) or os.path.basename(file_path),
                "author": getattr(result.document.meta, 'author', None),
                "created_at": getattr(result.document.meta, 'creation_date', None),
                "pages": getattr(result.document, 'num_pages', 0),
                "language": getattr(result.document.meta, 'language', 'en'),
            }

            return {
                "content": content,
                "metadata": meta
            }

        except asyncio.TimeoutError:
            logger.error("conversion_timeout", path=file_path)
            raise IngestionError(ERR_TIMEOUT, "Processing timed out")
        except IngestionError:
            raise
        except Exception as e:
            logger.error("conversion_failed", path=file_path, error=str(e))
            msg = str(e).lower()
            if "password" in msg or "encrypted" in msg:
                 raise IngestionError(ERR_ENCRYPTED, "File is password protected")
            elif "format" in msg:
                 raise IngestionError(ERR_INVALID_FORMAT, "Invalid file format")
            else:
                 raise e