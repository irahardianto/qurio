import asyncio
import json
import nsq
import uvloop
import tornado.platform.asyncio
from tornado.iostream import StreamClosedError
import structlog
from config import settings
from handlers.web import handle_web_task
from handlers.file import handle_file_task, IngestionError
from logger import configure_logger
from crawl4ai import AsyncWebCrawler

# Configure logging
configure_logger()
logger = structlog.get_logger(__name__)

# Global producer
producer = None
# Global Crawler
CRAWLER = None

# Global concurrency semaphore
# We use a value slightly higher than nsq_max_in_flight to allow for buffering,
# or match it to enforce strict parallelism.
# Defaulting to 8 matches the typical core count/worker capacity.
# Initialized to a safe default (1) to support tests/imports; overwritten in main().
WORKER_SEMAPHORE = asyncio.Semaphore(1)


async def init_crawler():
    global CRAWLER
    logger.info("initializing_global_crawler")
    try:
        CRAWLER = AsyncWebCrawler(verbose=True)
        await CRAWLER.start()
        logger.info("global_crawler_started")
    except Exception as e:
        logger.error("global_crawler_init_failed", error=str(e))
        # If crawler fails to start, we should probably exit or retry
        raise


def handle_message(message):
    """
    pynsq callback. Must be sync.
    We'll schedule the async processing on the event loop.
    """
    message.enable_async()
    asyncio.create_task(process_message(message))


async def process_message(message):
    global producer
    global CRAWLER

    # Keep message alive
    stop_touch = asyncio.Event()
    current_task = asyncio.current_task()

    async def touch_loop():
        while not stop_touch.is_set():
            try:
                message.touch()
            except (nsq.Error, StreamClosedError, Exception) as e:
                logger.warning("touch_failed_connection_lost", error=str(e))
                if current_task:
                    current_task.cancel()
                return

            # Wait for stop signal or timeout (heartbeat interval)
            try:
                await asyncio.wait_for(stop_touch.wait(), timeout=10)
            except asyncio.TimeoutError:
                pass  # Continue loop

    touch_task = asyncio.create_task(touch_loop())

    try:
        data = json.loads(message.body)
        logger.info("message_received", data=data)

        source_id = data.get("id")
        task_type = data.get("type")
        results_list = []

        # Enforce global concurrency limit
        async with WORKER_SEMAPHORE:
            if task_type == "web":
                url = data.get("url")
                # exclusions = data.get('exclusions', []) # Deprecated/Unused
                api_key = data.get("gemini_api_key")
                # Pass the global crawler
                results_list = await handle_web_task(
                    url, api_key=api_key, crawler=CRAWLER
                )

            elif task_type == "file":
                file_path = data.get("path")
                results_list = await handle_file_task(file_path)

        if results_list and producer:
            for res in results_list:
                result_payload = {
                    "source_id": source_id,
                    "correlation_id": source_id,
                    "content": res["content"],
                    "metadata": res.get("metadata", {}),
                    "title": res.get("title", ""),
                    "url": res["url"],
                    "path": res.get("path", ""),
                    "status": "success",
                    "links": res.get("links", []),
                    "depth": data.get("depth", 0),
                }

                try:
                    producer.pub(
                        settings.nsq_topic_result,
                        json.dumps(result_payload).encode("utf-8"),
                        callback=lambda c, d: logger.info(
                            "result_published", source_id=source_id, url=res.get("url")
                        ),
                    )
                except Exception as e:
                    logger.error("pub_failed", source_id=source_id, error=str(e))

        elif producer:
            # Handle case where no results returned
            fail_payload = {
                "source_id": source_id,
                "correlation_id": source_id,
                "status": "failed",
                "error": "No content extracted",
                "url": data.get("url", ""),
                "content": "",
            }
            try:
                producer.pub(
                    settings.nsq_topic_result,
                    json.dumps(fail_payload).encode("utf-8"),
                    callback=lambda c, d: logger.info(
                        "failure_reported", source_id=source_id, reason="empty_results"
                    ),
                )
            except Exception as e:
                logger.error("pub_failed", source_id=source_id, error=str(e))

        try:
            message.finish()
        except Exception as e:
            logger.warning("finish_failed", error=str(e))

    except IngestionError as e:
        logger.error("ingestion_error", error=str(e), code=e.code)

        if producer and "source_id" in locals():
            error_code = e.code
            fail_payload = {
                "source_id": source_id,
                "correlation_id": source_id,
                "status": "failed",
                "code": error_code,
                "error": f"[{e.code}] {e}",
                "url": data.get("url", "") or data.get("path", ""),
                "original_payload": data,
            }
            try:
                producer.pub(
                    settings.nsq_topic_result,
                    json.dumps(fail_payload).encode("utf-8"),
                    callback=lambda c, d: logger.info(
                        "failure_reported", source_id=source_id, code=error_code
                    ),
                )
            except Exception as ex:
                logger.error("pub_failed_in_error_handler", error=str(ex))

        try:
            message.finish()
        except Exception as ex:
            logger.warning("finish_failed_in_error_handler", error=str(ex))
        return

    except asyncio.CancelledError:
        logger.warning(
            "processing_cancelled_due_to_connection_loss",
            source_id=source_id if "source_id" in locals() else "unknown",
        )
        return

    except (asyncio.TimeoutError, Exception) as e:
        # Check for transient errors
        is_transient = (
            "Timeout" in str(e)
            or "Connection" in str(e)
            or isinstance(e, asyncio.TimeoutError)
        )

        if is_transient and message.attempts <= settings.retry_max_attempts:
            # Exponential Backoff (in milliseconds)
            # attempts=1 -> 2^0 * initial
            backoff_factor = settings.retry_backoff_multiplier ** (message.attempts - 1)
            delay = min(
                settings.retry_initial_delay_ms * backoff_factor,
                settings.retry_max_delay_ms,
            )

            logger.warning(
                "task_requeue_transient_error",
                source_id=source_id if "source_id" in locals() else "unknown",
                attempt=message.attempts,
                delay_ms=delay,
                error=str(e),
            )
            try:
                message.requeue(delay=int(delay), backoff=True)
            except Exception as req_ex:
                logger.error("requeue_failed", error=str(req_ex))
                # Fallthrough to finish? No, if requeue fails, we might as well try to finish or just return
                # But safer to let it be or try to finish if requeue failed explicitly
                pass
            return

        logger.error(
            "message_processing_failed", error=str(e), attempts=message.attempts
        )

        if producer and "source_id" in locals():
            fail_payload = {
                "source_id": source_id,
                "correlation_id": source_id,
                "status": "failed",
                "error": str(e),
                "url": data.get("url", ""),
                "content": "",
                "original_payload": data,
            }
            try:
                producer.pub(
                    settings.nsq_topic_result,
                    json.dumps(fail_payload).encode("utf-8"),
                    callback=lambda c, d: logger.info(
                        "failure_reported", source_id=source_id
                    ),
                )
            except Exception as ex:
                logger.error("pub_failed_in_error_handler", error=str(ex))

        try:
            message.finish()
        except Exception as ex:
            logger.warning("finish_failed_in_error_handler", error=str(ex))

    finally:
        stop_touch.set()
        await touch_task


def main():
    logger.info("worker_starting")

    # Configure uvloop
    uvloop.install()

    # Explicitly create and set the event loop for Python 3.10+ compat
    loop = asyncio.new_event_loop()
    asyncio.set_event_loop(loop)

    # Enable asyncio integration for Tornado
    # Tornado 6.1+ uses asyncio by default, but pynsq might rely on IOLoop.current()
    # which needs to be bridged if not fully native yet.
    # However, newer Tornado versions just wrap asyncio.
    # AsyncIOMainLoop().install() is technically deprecated but might be needed if pynsq assumes global IOLoop.
    tornado.platform.asyncio.AsyncIOMainLoop().install()

    # Create Consumer (Reader)
    # nsq.Reader connects immediately
    reader = nsq.Reader(  # noqa: F841 — Reader runs as side-effect of construction
        message_handler=handle_message,
        nsqd_tcp_addresses=[settings.nsqd_tcp_address],
        lookupd_http_addresses=[settings.nsq_lookupd_http],
        topic=settings.nsq_topic_ingest,
        channel=settings.nsq_channel_worker,
        max_in_flight=settings.nsq_max_in_flight,
        heartbeat_interval=60,
    )

    # Create Producer (Writer)
    # nsq.Writer connects to nsqd_tcp_addresses
    global producer
    producer = nsq.Writer([settings.nsqd_tcp_address])

    # Initialize semaphore with configured concurrency
    global WORKER_SEMAPHORE
    WORKER_SEMAPHORE = asyncio.Semaphore(settings.nsq_max_in_flight)

    logger.info("nsq_initialized", max_in_flight=settings.nsq_max_in_flight)

    # Initialize Global Crawler
    loop.run_until_complete(init_crawler())

    # Run the loop
    loop.run_forever()


if __name__ == "__main__":
    main()
