"""
Unit tests for process_file_sync, init_worker, and handle_message.
Tests functions that were previously only tested indirectly.
"""

import pytest
import os
from unittest.mock import MagicMock, patch


# --- process_file_sync Tests ---


class TestProcessFileSync:
    """Tests for the synchronous file processing function."""

    def test_converter_not_initialized_raises(self):
        """Verify RuntimeError when converter is None."""
        import handlers.file as file_module

        original = file_module.converter
        try:
            file_module.converter = None
            with pytest.raises(RuntimeError, match="Converter not initialized"):
                file_module.process_file_sync("/tmp/test.pdf")  # nosec B108
        finally:
            file_module.converter = original

    def test_success_returns_content_and_metadata(self):
        """Verify convert→export→metadata flow returns correct dict."""
        import handlers.file as file_module

        mock_doc = MagicMock()
        mock_doc.export_to_markdown.return_value = "# Test Content\nSome text"
        mock_doc.metadata = MagicMock(
            title="Test Title",
            authors=["Author A"],
            creation_date="2024-01-01",
            language="en",
        )
        mock_doc.num_pages = 5
        mock_doc.origin = MagicMock(filename="test.pdf")

        mock_result = MagicMock()
        mock_result.document = mock_doc

        mock_converter = MagicMock()
        mock_converter.convert.return_value = mock_result

        original = file_module.converter
        try:
            file_module.converter = mock_converter
            result = file_module.process_file_sync("/tmp/test.pdf")  # nosec B108

            assert result["content"] == "# Test Content\nSome text"
            assert result["metadata"]["title"] == "Test Title"
            assert result["metadata"]["author"] == "Author A"
            assert result["metadata"]["pages"] == 5
        finally:
            file_module.converter = original

    def test_metadata_extraction_failure_uses_fallback(self):
        """Verify fallback metadata on extraction error."""
        import handlers.file as file_module

        mock_doc = MagicMock()
        mock_doc.export_to_markdown.return_value = "content"

        mock_result = MagicMock()
        mock_result.document = mock_doc

        mock_converter = MagicMock()
        mock_converter.convert.return_value = mock_result

        original = file_module.converter
        try:
            file_module.converter = mock_converter
            # Patch extract_metadata_from_doc to raise, triggering the fallback
            with patch.object(
                file_module,
                "extract_metadata_from_doc",
                side_effect=AttributeError("boom"),
            ):
                result = file_module.process_file_sync("/some/path/doc.pdf")

            assert result["content"] == "content"
            assert result["metadata"]["title"] == "doc.pdf"
            assert result["metadata"]["author"] is None
        finally:
            file_module.converter = original


# --- init_worker Tests ---


class TestInitWorker:
    """Tests for worker process initialization."""

    def test_sets_thread_env_vars(self):
        """Verify init_worker sets threading environment variables."""
        # We need to mock docling imports since they won't be available in test env
        with (
            patch.dict(
                "sys.modules",
                {
                    "docling": MagicMock(),
                    "docling.document_converter": MagicMock(),
                    "docling.datamodel": MagicMock(),
                    "docling.datamodel.pipeline_options": MagicMock(),
                    "docling.datamodel.base_models": MagicMock(),
                },
            ),
        ):
            import handlers.file as file_module

            original_converter = file_module.converter
            try:
                file_module.init_worker()

                assert os.environ.get("OMP_NUM_THREADS") == "2"
                assert os.environ.get("MKL_NUM_THREADS") == "2"
                assert os.environ.get("OPENBLAS_NUM_THREADS") == "2"
                assert os.environ.get("ONNX_NUM_THREADS") == "1"
                assert os.environ.get("OMP_THREAD_LIMIT") == "2"
            finally:
                file_module.converter = original_converter


# --- handle_message Tests ---


class TestHandleMessage:
    """Tests for the sync NSQ message callback."""

    def test_calls_enable_async(self):
        """Verify handle_message calls message.enable_async()."""
        msg = MagicMock()

        with patch("main.asyncio") as mock_asyncio:
            # Prevent create_task from actually scheduling anything
            mock_asyncio.create_task = MagicMock()
            from main import handle_message

            handle_message(msg)

            msg.enable_async.assert_called_once()

    def test_creates_task_for_process_message(self):
        """Verify handle_message creates an asyncio task."""
        msg = MagicMock()

        with patch("main.asyncio") as mock_asyncio:
            mock_asyncio.create_task = MagicMock()
            from main import handle_message

            handle_message(msg)

            mock_asyncio.create_task.assert_called_once()
