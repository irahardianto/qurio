import asyncio
import structlog
import os
import signal
import pebble
from concurrent.futures import ProcessPoolExecutor, TimeoutError
# Deferred imports for docling to ensure clean process initialization
# from docling.document_converter import DocumentConverter, PdfFormatOption
# from docling.datamodel.pipeline_options import PdfPipelineOptions
# from docling.datamodel.base_models import InputFormat

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

# Global converter variable (per process)
converter = None

def init_worker():
    """
    Initialize the converter in each worker process.
    This ensures isolation and avoids threading issues with underlying C++ libraries.
    """
    # Force limited-thread execution for all underlying libraries
    # Bumped to 2 threads per worker since we have 24 threads total and 8 workers (16 threads used + overhead)
    # This might speed up OCR slightly without freezing the system
    os.environ["OMP_NUM_THREADS"] = "2"
    os.environ["MKL_NUM_THREADS"] = "2"
    os.environ["OPENBLAS_NUM_THREADS"] = "2"
    os.environ["VECLIB_MAXIMUM_THREADS"] = "2"
    os.environ["NUMEXPR_NUM_THREADS"] = "2"
    # Additional constraints for ONNX/PyTorch to prevent thread explosion
    os.environ["ONNX_NUM_THREADS"] = "1" # Keep ONNX single-threaded as it spawns aggressively
    os.environ["OMP_THREAD_LIMIT"] = "2"
    
    # Deferred imports to prevent parent process initialization leaking into child
    from docling.document_converter import DocumentConverter, PdfFormatOption
    from docling.datamodel.pipeline_options import PdfPipelineOptions
    from docling.datamodel.base_models import InputFormat
    
    global converter
    
    # Configure Pipeline Options
    pipeline_opts = PdfPipelineOptions()
    pipeline_opts.do_ocr = True
    # Re-enable table structure with controlled resources
    pipeline_opts.do_table_structure = True
    
    # Initialize Converter with options
    converter = DocumentConverter(
        format_options={
            InputFormat.PDF: PdfFormatOption(pipeline_options=pipeline_opts)
        }
    )

def process_file_sync(file_path: str) -> dict:
    """
    Synchronous function running in a separate process.
    Performs CPU-intensive conversion and markdown export.
    """
    # Simulate progress updates (Docling doesn't expose granular callbacks yet)
    # Ideally we would call the backend webhook here, but we are in a sub-process
    # without network config or proper async loop. 
    # For now, we rely on the main loop to handle "Started" and "Finished".
    # With Docling v2, we can't easily hook into the PDF page loop without a custom pipeline class.
    # Future enhancement: subclass PdfPipeline to report page progress.
    try:
        if converter is None:
            raise RuntimeError("Converter not initialized in worker process")
            
        result = converter.convert(file_path)
        content = result.document.export_to_markdown()
        
        # Extract metadata (Standardized for Docling v2)
        try:
            # Primary Source: Docling v2 'origin' and 'metadata' attributes on Document
            # doc.origin -> filename, uri, format
            # doc.metadata -> title, authors, date, language
            doc = result.document
            
            # Title Strategy: Metadata Title > Filename > Fallback
            title = None
            if hasattr(doc, 'metadata') and doc.metadata and doc.metadata.title:
                title = doc.metadata.title
            elif hasattr(doc, 'origin') and doc.origin and doc.origin.filename:
                title = doc.origin.filename
            else:
                title = os.path.basename(file_path)

            # Author Strategy
            author = None
            if hasattr(doc, 'metadata') and doc.metadata and doc.metadata.authors:
                if isinstance(doc.metadata.authors, list):
                    author = ", ".join([str(a) for a in doc.metadata.authors])
                else:
                    author = str(doc.metadata.authors)

            # Date Strategy
            created_at = None
            if hasattr(doc, 'metadata') and doc.metadata and doc.metadata.creation_date:
                # Ensure we handle pydantic/native types correctly
                val = doc.metadata.creation_date
                if callable(val):
                     val = val()
                created_at = str(val)

            # Language Strategy
            language = 'en'
            if hasattr(doc, 'metadata') and doc.metadata and doc.metadata.language:
                language = doc.metadata.language

            # Page Count Strategy
            pages = 0
            if hasattr(doc, 'num_pages'):
                val = doc.num_pages
                if callable(val):
                    val = val()
                pages = int(val)
            elif hasattr(result, 'pages'):
                 pages = len(result.pages)

            meta = {
                "title": title,
                "author": author,
                "created_at": created_at,
                "pages": pages,
                "language": language,
            }
            # Final sanity check: ensure no values are methods
            for k, v in meta.items():
                if callable(v):
                    logger.warning("callable_metadata_found", key=k)
                    meta[k] = str(v()) if v is not None else None
        except Exception as e:
            logger.warning("metadata_extraction_failed", error=str(e))
            # Safe Fallback
            meta = {
                "title": os.path.basename(file_path),
                "author": None,
                "created_at": None,
                "pages": 0,
                "language": "en"
            }
        
        return {
            "content": content,
            "metadata": meta
        }
    except Exception as e:
        # Re-raise to be caught by main loop
        raise e

# Use Pebble ProcessPool for robust process management (timeout = kill)
# Scaled to 8 workers for high-core machines (12 cores / 24 threads)
# We rely on deferred imports in init_worker to simulate clean state and strict thread limits
executor = pebble.ProcessPool(
    max_workers=8, 
    initializer=init_worker
)

CONCURRENCY_LIMIT = asyncio.Semaphore(8)
# Increase timeout to 30 minutes to accommodate large PDF books with OCR
TIMEOUT_SECONDS = 1800

async def handle_file_task(file_path: str) -> dict:
    """
    Converts a document to markdown using Docling.
    Executes in a Pebble ProcessPool to enforce hard timeouts and kill stuck processes.
    """
    logger.info("conversion_starting", path=file_path)
    
    async with CONCURRENCY_LIMIT:
        try:
            # Schedule the task with a hard timeout managed by Pebble
            future = executor.schedule(
                process_file_sync, 
                args=(file_path,), 
                timeout=TIMEOUT_SECONDS
            )
            
            # Bridge Pebble Future to AsyncIO
            result = await asyncio.wrap_future(future)
            
            if not result["content"].strip():
                 raise IngestionError(ERR_EMPTY, "File contains no text")

            return result

        except (TimeoutError, pebble.ProcessExpired):
            logger.error("conversion_timeout_killed", path=file_path)
            raise IngestionError(ERR_TIMEOUT, "Processing timed out and worker process was terminated")
        except IngestionError:
            raise
        except Exception as e:
            # Check for wrapped exceptions
            err_msg = str(e).lower()
            if "timeout" in err_msg:
                 logger.error("conversion_timeout_exception", path=file_path)
                 raise IngestionError(ERR_TIMEOUT, "Processing timed out")

            logger.error("conversion_failed", path=file_path, error=str(e))
            if "password" in err_msg or "encrypted" in err_msg:
                 raise IngestionError(ERR_ENCRYPTED, "File is password protected")
            elif "format" in err_msg:
                 raise IngestionError(ERR_INVALID_FORMAT, "Invalid file format")
            else:
                 raise e