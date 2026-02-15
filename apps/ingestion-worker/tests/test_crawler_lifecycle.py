"""
Tests for crawler lifecycle functions (init_crawler, get_crawler, restart_crawler)
and browser crash detection in process_message.
"""

import pytest
import json
from unittest.mock import MagicMock, AsyncMock, patch
import main


# --- Crawler Lifecycle Tests ---


@pytest.mark.asyncio
async def test_init_crawler_success():
    """Verify init_crawler sets global CRAWLER and calls start()."""
    mock_crawler_instance = AsyncMock()

    with patch("main.AsyncWebCrawler", return_value=mock_crawler_instance):
        main.CRAWLER = None
        await main.init_crawler()

        assert main.CRAWLER is mock_crawler_instance
        mock_crawler_instance.start.assert_awaited_once()


@pytest.mark.asyncio
async def test_init_crawler_failure_propagates():
    """Verify init_crawler re-raises exceptions on start() failure."""
    mock_crawler_instance = AsyncMock()
    mock_crawler_instance.start.side_effect = RuntimeError("Browser launch failed")

    with patch("main.AsyncWebCrawler", return_value=mock_crawler_instance):
        main.CRAWLER = None

        with pytest.raises(RuntimeError, match="Browser launch failed"):
            await main.init_crawler()


@pytest.mark.asyncio
async def test_get_crawler_initializes_when_none():
    """Verify get_crawler lazily initializes CRAWLER when it is None."""
    mock_crawler_instance = AsyncMock()

    with patch("main.AsyncWebCrawler", return_value=mock_crawler_instance):
        main.CRAWLER = None
        result = await main.get_crawler()

        assert result is mock_crawler_instance
        mock_crawler_instance.start.assert_awaited_once()


@pytest.mark.asyncio
async def test_get_crawler_returns_existing():
    """Verify get_crawler returns existing CRAWLER without re-initializing."""
    existing_crawler = AsyncMock()
    main.CRAWLER = existing_crawler

    with patch("main.init_crawler") as mock_init:
        result = await main.get_crawler()

        assert result is existing_crawler
        mock_init.assert_not_awaited()

    # Cleanup
    main.CRAWLER = None


@pytest.mark.asyncio
async def test_restart_crawler_closes_and_reinits():
    """Verify restart_crawler closes old, sets None, then re-initializes."""
    old_crawler = AsyncMock()
    new_crawler = AsyncMock()

    main.CRAWLER = old_crawler

    with patch("main.init_crawler", new_callable=AsyncMock) as mock_init:
        mock_init.side_effect = lambda: setattr(main, "CRAWLER", new_crawler)

        await main.restart_crawler()

        old_crawler.close.assert_awaited_once()
        mock_init.assert_awaited_once()

    # Cleanup
    main.CRAWLER = None


@pytest.mark.asyncio
async def test_restart_crawler_ignores_close_failure():
    """Verify restart_crawler continues even if close() raises."""
    old_crawler = AsyncMock()
    old_crawler.close.side_effect = Exception("Close failed")
    new_crawler = AsyncMock()

    main.CRAWLER = old_crawler

    with patch("main.init_crawler", new_callable=AsyncMock) as mock_init:
        mock_init.side_effect = lambda: setattr(main, "CRAWLER", new_crawler)

        # Should NOT raise
        await main.restart_crawler()

        old_crawler.close.assert_awaited_once()
        mock_init.assert_awaited_once()

    # Cleanup
    main.CRAWLER = None


@pytest.mark.asyncio
async def test_restart_crawler_when_crawler_is_none():
    """Verify restart_crawler works when CRAWLER is already None."""
    main.CRAWLER = None

    with patch("main.init_crawler", new_callable=AsyncMock) as mock_init:
        await main.restart_crawler()

        # Should skip close and just call init
        mock_init.assert_awaited_once()

    # Cleanup
    main.CRAWLER = None


# --- Browser Crash Detection Tests ---


@pytest.mark.asyncio
async def test_process_message_browser_crash_restarts_crawler():
    """Verify that browser crash triggers restart_crawler."""
    msg = MagicMock()
    msg.body = json.dumps(
        {"id": "crash-1", "type": "web", "url": "http://example.com"}
    ).encode("utf-8")

    mock_crawler = AsyncMock()

    with (
        patch("main.get_crawler", new_callable=AsyncMock, return_value=mock_crawler),
        patch(
            "main.handle_web_task",
            new_callable=AsyncMock,
            side_effect=Exception("browser has been closed"),
        ),
        patch("main.restart_crawler", new_callable=AsyncMock) as mock_restart,
        patch("main.producer", new=MagicMock()),
    ):
        await main.process_message(msg)

        mock_restart.assert_awaited_once()


@pytest.mark.asyncio
async def test_process_message_target_closed_crash():
    """Verify 'target closed' error triggers restart_crawler."""
    msg = MagicMock()
    msg.body = json.dumps(
        {"id": "crash-2", "type": "web", "url": "http://example.com"}
    ).encode("utf-8")

    mock_crawler = AsyncMock()

    with (
        patch("main.get_crawler", new_callable=AsyncMock, return_value=mock_crawler),
        patch(
            "main.handle_web_task",
            new_callable=AsyncMock,
            side_effect=Exception("target closed unexpectedly"),
        ),
        patch("main.restart_crawler", new_callable=AsyncMock) as mock_restart,
        patch("main.producer", new=MagicMock()),
    ):
        await main.process_message(msg)

        mock_restart.assert_awaited_once()


@pytest.mark.asyncio
async def test_process_message_protocol_error_crash():
    """Verify 'protocol error' triggers restart_crawler."""
    msg = MagicMock()
    msg.body = json.dumps(
        {"id": "crash-3", "type": "web", "url": "http://example.com"}
    ).encode("utf-8")

    mock_crawler = AsyncMock()

    with (
        patch("main.get_crawler", new_callable=AsyncMock, return_value=mock_crawler),
        patch(
            "main.handle_web_task",
            new_callable=AsyncMock,
            side_effect=Exception("Protocol error: session closed"),
        ),
        patch("main.restart_crawler", new_callable=AsyncMock) as mock_restart,
        patch("main.producer", new=MagicMock()),
    ):
        await main.process_message(msg)

        mock_restart.assert_awaited_once()


@pytest.mark.asyncio
async def test_process_message_non_crash_error_no_restart():
    """Verify generic non-browser errors do NOT trigger restart_crawler."""
    msg = MagicMock()
    msg.body = json.dumps(
        {"id": "no-crash", "type": "web", "url": "http://example.com"}
    ).encode("utf-8")

    mock_crawler = AsyncMock()

    with (
        patch("main.get_crawler", new_callable=AsyncMock, return_value=mock_crawler),
        patch(
            "main.handle_web_task",
            new_callable=AsyncMock,
            side_effect=ValueError("Some random error"),
        ),
        patch("main.restart_crawler", new_callable=AsyncMock) as mock_restart,
        patch("main.producer", new=MagicMock()),
    ):
        await main.process_message(msg)

        mock_restart.assert_not_awaited()


@pytest.mark.asyncio
async def test_process_message_file_task_no_crash_check():
    """Verify file tasks do not trigger browser crash detection."""
    msg = MagicMock()
    msg.body = json.dumps(
        {"id": "file-1", "type": "file", "path": "/tmp/test.pdf"}  # nosec B108
    ).encode("utf-8")

    with (
        patch(
            "main.handle_file_task",
            new_callable=AsyncMock,
            side_effect=Exception("browser has been closed"),
        ),
        patch("main.restart_crawler", new_callable=AsyncMock) as mock_restart,
        patch("main.producer", new=MagicMock()),
    ):
        await main.process_message(msg)

        # File tasks should NOT trigger restart_crawler even with "browser" keyword
        mock_restart.assert_not_awaited()
