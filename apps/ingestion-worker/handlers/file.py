import asyncio
import structlog
import os
import pebble
from concurrent.futures import TimeoutError
# Deferred imports for docling to ensure clean process initialization
# from docling.document_converter import DocumentConverter, PdfFormatOption
# from docling.datamodel.pipeline_options import PdfPipelineOptions
# from docling.datamodel.base_models import InputFormat

from exceptions import (
    IngestionError,
    ERR_ENCRYPTED,
    ERR_INVALID_FORMAT,
    ERR_EMPTY,
    ERR_TIMEOUT,
)

logger = structlog.get_logger(__name__)

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
    os.environ["ONNX_NUM_THREADS"] = (
        "1"  # Keep ONNX single-threaded as it spawns aggressively
    )
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


def extract_metadata_from_doc(doc, result, file_path: str) -> dict:
    """
    Extracts standardized metadata from a Docling document object.
    """

    def unwrap(val):
        if callable(val):
            return val()
        return val

    # Title Strategy: Metadata Title > Filename > Fallback
    title = None
    if hasattr(doc, "metadata") and doc.metadata:
        raw_title = unwrap(doc.metadata.title)
        if raw_title:
            title = raw_title

    if not title:
        if hasattr(doc, "origin") and doc.origin:
            raw_filename = unwrap(doc.origin.filename)
            if raw_filename:
                title = raw_filename

    if not title:
        title = os.path.basename(file_path)

    # Author Strategy
    author = None
    if hasattr(doc, "metadata") and doc.metadata:
        authors = unwrap(doc.metadata.authors)
        if authors:
            if isinstance(authors, list):
                # unwrapping elements if list itself is not callable but elements might be?
                # Docling usually doesn't have callable elements in a list, but let's be safe
                clean_authors = [str(unwrap(a)) for a in authors]
                author = ", ".join(clean_authors)
            else:
                author = str(authors)

    # Date Strategy
    created_at = None
    if hasattr(doc, "metadata") and doc.metadata:
        val = unwrap(doc.metadata.creation_date)
        if val:
            created_at = str(val)

    # Language Strategy
    language = "en"
    if hasattr(doc, "metadata") and doc.metadata:
        val = unwrap(doc.metadata.language)
        if val:
            language = val

    # Page Count Strategy
    pages = 0
    # doc.num_pages might be None, or a callable returning int, or int
    if hasattr(doc, "num_pages"):
        val = unwrap(doc.num_pages)
        if val is not None:
            pages = int(val)

    # Fallback to result.pages if doc.num_pages failed
    if pages == 0 and hasattr(result, "pages"):
        val = unwrap(result.pages)
        if val is not None:
            pages = len(val)

    meta = {
        "title": title,
        "author": author,
        "created_at": created_at,
        "pages": pages,
        "language": language,
    }

    return meta


def process_file_sync(file_path: str) -> dict:
    """
    Synchronous function running in a separate process.
    Performs CPU-intensive conversion and markdown export.
    """
    try:
        if converter is None:
            raise RuntimeError("Converter not initialized in worker process")

        result = converter.convert(file_path)
        content = result.document.export_to_markdown()

        # Extract metadata (Standardized for Docling v2)
        try:
            meta = extract_metadata_from_doc(result.document, result, file_path)
        except Exception as e:
            logger.warning("metadata_extraction_failed", error=str(e))
            # Safe Fallback
            meta = {
                "title": os.path.basename(file_path),
                "author": None,
                "created_at": None,
                "pages": 0,
                "language": "en",
            }

        return {"content": content, "metadata": meta}
    except Exception as e:
        # Re-raise to be caught by main loop
        raise e


# Use Pebble ProcessPool for robust process management (timeout = kill)
# Scaled to 8 workers for high-core machines (12 cores / 24 threads)
# We rely on deferred imports in init_worker to simulate clean state and strict thread limits
executor = pebble.ProcessPool(max_workers=8, initializer=init_worker)

# Increase timeout to 30 minutes to accommodate large PDF books with OCR
TIMEOUT_SECONDS = 1800


async def handle_file_task(file_path: str) -> list[dict]:
    """
    Converts a document to markdown using Docling.
    Executes in a Pebble ProcessPool to enforce hard timeouts and kill stuck processes.
    """
    logger.info("conversion_starting", path=file_path)

    try:
        # Schedule the task with a hard timeout managed by Pebble
        future = executor.schedule(
            process_file_sync, args=[file_path], timeout=TIMEOUT_SECONDS
        )

        # Bridge Pebble Future to AsyncIO
        result = await asyncio.wrap_future(future)

        if not result["content"].strip():
            raise IngestionError(ERR_EMPTY, "File contains no text")

        return [
            {
                "content": result["content"],
                "metadata": result["metadata"],
                "url": file_path,
                "path": file_path,
                "title": result["metadata"].get("title", ""),
                "links": [],
            }
        ]

    except (TimeoutError, pebble.ProcessExpired):
        logger.error("conversion_timeout_killed", path=file_path)
        raise IngestionError(
            ERR_TIMEOUT, "Processing timed out and worker process was terminated"
        )
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
