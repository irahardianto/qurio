"""
Integration tests for NSQ message flow using testcontainers.

Tests the complete flow: publish → consume → process → result published.

The success/failure tests use a MockProducer to capture pub() calls,
because pynsq's Writer relies on Tornado's IOLoop to flush data,
which does not run under pytest-asyncio. The retry and concurrent
tests verify application-level logic (requeue, semaphore) and
do not need real NSQ delivery.
"""

import pytest
import json
import asyncio
import time
from testcontainers.core.container import DockerContainer


pytestmark = pytest.mark.integration


class MockProducer:
    """
    Captures pub() calls so tests can inspect what was published
    without depending on Tornado's IOLoop for actual delivery.
    """

    def __init__(self):
        self.published = []

    def pub(self, topic, data, callback=None):
        self.published.append(
            {"topic": topic, "data": json.loads(data.decode("utf-8"))}
        )
        if callback:
            callback(None, data)


class MockMessage:
    """Shared mock NSQ message used across tests."""

    def __init__(self, body, attempts=1):
        self.body = body
        self.attempts = attempts
        self._finished = False
        self._requeued = False

    def finish(self):
        self._finished = True

    def requeue(self, delay=0, backoff=False):
        self._requeued = True

    def touch(self):
        pass

    def enable_async(self):
        pass


@pytest.fixture(scope="module")
def nsq_containers():
    """
    Start NSQ containers (nsqlookupd and nsqd) for integration testing.
    """
    nsqlookupd = DockerContainer("nsqio/nsq:v1.2.1")
    nsqlookupd.with_command("/nsqlookupd")
    nsqlookupd.with_exposed_ports(4160, 4161)
    nsqlookupd.start()

    # Wait for nsqlookupd to be ready
    time.sleep(2)

    lookupd_http_port = nsqlookupd.get_exposed_port(4161)
    lookupd_tcp_port = nsqlookupd.get_exposed_port(4160)

    # Start nsqd
    nsqd = DockerContainer("nsqio/nsq:v1.2.1")
    nsqd.with_command(
        f"/nsqd --lookupd-tcp-address=host.docker.internal:{lookupd_tcp_port}"
    )
    nsqd.with_exposed_ports(4150, 4151)
    nsqd.with_kwargs(extra_hosts={"host.docker.internal": "host-gateway"})
    nsqd.start()

    # Wait for nsqd to be ready
    time.sleep(2)

    nsqd_tcp_port = nsqd.get_exposed_port(4150)
    nsqd_http_port = nsqd.get_exposed_port(4151)

    yield {
        "nsqd_tcp": f"localhost:{nsqd_tcp_port}",
        "nsqd_http": f"localhost:{nsqd_http_port}",
        "lookupd_http": f"localhost:{lookupd_http_port}",
    }

    # Cleanup
    nsqd.stop()
    nsqlookupd.stop()


@pytest.mark.asyncio
async def test_nsq_message_flow_success(nsq_containers):
    """
    Test complete NSQ message flow with successful processing.
    Verifies that process_message calls producer.pub with correct success payload.
    """
    from unittest.mock import AsyncMock, patch

    async def mock_handle_web_task(url, **kwargs):
        return [
            {
                "content": "Test content",
                "url": url,
                "title": "Test Title",
                "metadata": {"key": "value"},
                "links": ["http://example.com/page2"],
            }
        ]

    test_message = {
        "id": "test-123",
        "type": "web",
        "url": "http://example.com",
    }

    mock_producer = MockProducer()
    mock_msg = MockMessage(json.dumps(test_message).encode("utf-8"))

    with patch("main.handle_web_task", new_callable=AsyncMock) as mock_handler:
        mock_handler.side_effect = mock_handle_web_task

        from main import process_message
        import main

        main.producer = mock_producer

        await process_message(mock_msg)

    # Verify the handler was called
    assert mock_handler.call_count == 1

    # Verify message was finished
    assert mock_msg._finished is True

    # Verify a result was published
    assert len(mock_producer.published) == 1
    result = mock_producer.published[0]["data"]
    assert result["status"] == "success"
    assert result["source_id"] == "test-123"
    assert result["content"] == "Test content"
    assert result["title"] == "Test Title"
    assert result["url"] == "http://example.com"
    assert result["metadata"] == {"key": "value"}
    assert result["links"] == ["http://example.com/page2"]


@pytest.mark.asyncio
async def test_nsq_message_flow_failure(nsq_containers):
    """
    Test NSQ message flow with processing failure (permanent error).
    Verifies that process_message publishes a failure payload.
    """
    from unittest.mock import AsyncMock, patch
    from exceptions import IngestionError, ERR_ENCRYPTED

    async def mock_handle_file_task(path):
        raise IngestionError(ERR_ENCRYPTED, "File is encrypted")

    test_message = {
        "id": "test-456",
        "type": "file",
        "path": "/tmp/encrypted.pdf",  # nosec B108
    }

    mock_producer = MockProducer()
    mock_msg = MockMessage(json.dumps(test_message).encode("utf-8"))

    with patch("main.handle_file_task", new_callable=AsyncMock) as mock_handler:
        mock_handler.side_effect = mock_handle_file_task

        from main import process_message
        import main

        main.producer = mock_producer

        await process_message(mock_msg)

    # Verify the handler was called
    assert mock_handler.call_count == 1

    # Verify message was finished (permanent error)
    assert mock_msg._finished is True

    # Verify a failure was published
    assert len(mock_producer.published) == 1
    result = mock_producer.published[0]["data"]
    assert result["status"] == "failed"
    assert result["source_id"] == "test-456"
    assert result["code"] == ERR_ENCRYPTED
    assert "ERR_ENCRYPTED" in result["error"]


@pytest.mark.asyncio
async def test_nsq_retry_flow(nsq_containers):
    """
    Test NSQ retry flow with transient errors.
    Uses a MockMessage with custom requeue that re-invokes process_message.
    """
    from unittest.mock import AsyncMock, patch

    call_count = 0

    async def mock_handle_with_retry(url, **kwargs):
        nonlocal call_count
        call_count += 1
        if call_count < 2:
            raise asyncio.TimeoutError("Timeout on first attempt")
        return [
            {
                "content": "Success after retry",
                "url": url,
                "title": "Test",
                "metadata": {},
                "links": [],
            }
        ]

    test_message = {
        "id": "test-retry-789",
        "type": "web",
        "url": "http://example.com",
    }

    mock_producer = MockProducer()

    with patch("main.handle_web_task", new_callable=AsyncMock) as mock_handler:
        mock_handler.side_effect = mock_handle_with_retry

        from main import process_message
        import main

        main.producer = mock_producer

        class RetryMockMessage(MockMessage):
            def requeue(self, delay=0, backoff=False):
                self._requeued = True
                self.attempts += 1
                asyncio.create_task(process_message(self))

        mock_msg = RetryMockMessage(json.dumps(test_message).encode("utf-8"))

        await process_message(mock_msg)
        # Allow the retry task to complete
        await asyncio.sleep(1)

    # Verify that retry succeeded (handler called at least twice)
    assert call_count >= 2


@pytest.mark.asyncio
async def test_nsq_concurrent_messages(nsq_containers):
    """
    Test concurrent message processing with semaphore limiting.
    """
    from unittest.mock import AsyncMock, patch

    processing_count = 0
    max_concurrent = 0
    lock = asyncio.Lock()

    async def mock_slow_handler(url, **kwargs):
        nonlocal processing_count, max_concurrent
        async with lock:
            processing_count += 1
            max_concurrent = max(max_concurrent, processing_count)

        await asyncio.sleep(0.5)

        async with lock:
            processing_count -= 1

        return [
            {
                "content": "Test",
                "url": url,
                "title": "Test",
                "metadata": {},
                "links": [],
            }
        ]

    mock_producer = MockProducer()

    with patch("main.handle_web_task", new_callable=AsyncMock) as mock_handler:
        mock_handler.side_effect = mock_slow_handler

        from main import process_message
        import main

        main.WORKER_SEMAPHORE = asyncio.Semaphore(2)
        main.producer = mock_producer

        tasks = []
        for i in range(5):
            test_message = {
                "id": f"test-concurrent-{i}",
                "type": "web",
                "url": f"http://example.com/{i}",
            }
            mock_msg = MockMessage(json.dumps(test_message).encode("utf-8"))
            tasks.append(process_message(mock_msg))

        await asyncio.gather(*tasks)

    # Verify that semaphore limited concurrency
    assert max_concurrent <= 2

    # Verify all 5 results were published
    assert len(mock_producer.published) == 5


@pytest.mark.asyncio
async def test_nsq_file_task_success(nsq_containers):
    """
    Test complete NSQ message flow for file task processing.
    Verifies that process_message handles file tasks and publishes correct payload.
    """
    from unittest.mock import AsyncMock, patch

    async def mock_handle_file_task(path):
        return [
            {
                "content": "# PDF Content\nExtracted text",
                "url": path,
                "path": path,
                "title": "Test Document",
                "metadata": {"author": "Test", "pages": 5},
                "links": [],
            }
        ]

    test_message = {
        "id": "file-int-1",
        "type": "file",
        "path": "/tmp/test.pdf",  # nosec B108
    }

    mock_producer = MockProducer()
    mock_msg = MockMessage(json.dumps(test_message).encode("utf-8"))

    with patch("main.handle_file_task", new_callable=AsyncMock) as mock_handler:
        mock_handler.side_effect = mock_handle_file_task

        from main import process_message
        import main

        main.producer = mock_producer

        await process_message(mock_msg)

    assert mock_handler.call_count == 1
    assert mock_msg._finished is True
    assert len(mock_producer.published) == 1
    result = mock_producer.published[0]["data"]
    assert result["status"] == "success"
    assert result["source_id"] == "file-int-1"
    assert result["title"] == "Test Document"
    assert result["metadata"]["author"] == "Test"
    assert result["metadata"]["pages"] == 5


@pytest.mark.asyncio
async def test_nsq_missing_type_field(nsq_containers):
    """
    Test NSQ message flow with missing 'type' field.
    Verifies graceful handling — no results, no crash.
    """

    test_message = {
        "id": "missing-type-1",
        # No "type" field
        "url": "http://example.com",
    }

    mock_producer = MockProducer()
    mock_msg = MockMessage(json.dumps(test_message).encode("utf-8"))

    from main import process_message
    import main

    main.producer = mock_producer

    await process_message(mock_msg)

    # Should finish without crash but publish empty/failure result
    assert mock_msg._finished is True


@pytest.mark.asyncio
async def test_nsq_multiple_results_published(nsq_containers):
    """
    Test that multiple result chunks are all published individually.
    """
    from unittest.mock import AsyncMock, patch

    async def mock_handle_web_task(url, **kwargs):
        return [
            {
                "content": "Chunk 1",
                "url": url + "/1",
                "title": "P1",
                "metadata": {},
                "links": [],
            },
            {
                "content": "Chunk 2",
                "url": url + "/2",
                "title": "P2",
                "metadata": {},
                "links": [],
            },
            {
                "content": "Chunk 3",
                "url": url + "/3",
                "title": "P3",
                "metadata": {},
                "links": [],
            },
        ]

    test_message = {
        "id": "multi-result-1",
        "type": "web",
        "url": "http://example.com",
    }

    mock_producer = MockProducer()
    mock_msg = MockMessage(json.dumps(test_message).encode("utf-8"))

    with patch("main.handle_web_task", new_callable=AsyncMock) as mock_handler:
        mock_handler.side_effect = mock_handle_web_task

        from main import process_message
        import main

        main.producer = mock_producer

        await process_message(mock_msg)

    assert mock_msg._finished is True
    assert len(mock_producer.published) == 3
    assert mock_producer.published[0]["data"]["content"] == "Chunk 1"
    assert mock_producer.published[1]["data"]["content"] == "Chunk 2"
    assert mock_producer.published[2]["data"]["content"] == "Chunk 3"
    # All should have same source_id
    for p in mock_producer.published:
        assert p["data"]["source_id"] == "multi-result-1"
