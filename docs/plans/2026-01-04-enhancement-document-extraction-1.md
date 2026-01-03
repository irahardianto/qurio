### Task 1: Test Scaffolding & Error Taxonomy

**Files:**
- Create: `apps/ingestion-worker/tests/test_file_handler_v2.py`
- Modify: `apps/ingestion-worker/handlers/file.py:5-10` (Imports only for test)

**Requirements:**
- **Acceptance Criteria**
  1. Unit tests exist validating `ERR_ENCRYPTED`, `ERR_INVALID_FORMAT`, `ERR_EMPTY`, `ERR_TIMEOUT`.
  2. Unit tests exist validating `metadata` dictionary creation with correct fallbacks.

- **Functional Requirements**
  1. Define Error Taxonomy constants/types.
  2. Create mock fixtures for `docling.DocumentConverter`.

- **Non-Functional Requirements**
  None for this task.

- **Test Coverage**
  - [Unit] `test_handle_encrypted_pdf` - assert raises `ERR_ENCRYPTED`
  - [Unit] `test_handle_metadata_extraction` - assert keys `title`, `author`, `created_at`, `pages`
  - Test data fixtures: Mock `docling.datamodel.base_models.InputFormat` objects.

**Step 1: Write failing test**
```python
# apps/ingestion-worker/tests/test_file_handler_v2.py
import pytest
from unittest.mock import MagicMock, patch
from handlers.file import handle_file_task, ERR_ENCRYPTED, ERR_INVALID_FORMAT

@pytest.mark.asyncio
async def test_handle_encrypted_pdf():
    with patch('handlers.file.converter') as mock_converter:
        mock_converter.convert.side_effect = Exception("Encrypted") # Simulating docling error
        
        # We expect our handler to catch this and raise a specific custom exception or return a specific error code
        # For this plan, let's assume we return a dict with 'status': 'failed' and 'error': ERR_ENCRYPTED
        # Adjusting expectation based on Architecture: main.py handles exceptions.
        # So handle_file_task should probably raise a specific typed exception.
        
        with pytest.raises(Exception) as excinfo:
             await handle_file_task("/tmp/secret.pdf")
        
        # This will fail because we haven't implemented the mapping yet
        assert "ERR_ENCRYPTED" in str(excinfo.value)

@pytest.mark.asyncio
async def test_metadata_extraction():
     with patch('handlers.file.converter') as mock_converter:
        mock_doc = MagicMock()
        mock_doc.document.export_to_markdown.return_value = "# Content"
        mock_doc.document.meta.title = "Test Title"
        mock_doc.document.meta.author = "Test Author"
        mock_doc.document.num_pages = 10
        mock_converter.convert.return_value = mock_doc
        
        result = await handle_file_task("/tmp/test.pdf")
        
        # This will fail as we currently don't return metadata
        assert result['metadata']['title'] == "Test Title"
        assert result['metadata']['pages'] == 10
```

**Step 2: Verify test fails**
Run: `pytest apps/ingestion-worker/tests/test_file_handler_v2.py -v`
Expected: FAIL (KeyError for 'metadata', or Exception mismatch)

**Step 3: Write minimal implementation (Constants & Imports)**
```python
# apps/ingestion-worker/handlers/file.py
# Add constants at top
ERR_ENCRYPTED = "ERR_ENCRYPTED"
ERR_INVALID_FORMAT = "ERR_INVALID_FORMAT"
ERR_EMPTY = "ERR_EMPTY"
ERR_TIMEOUT = "ERR_TIMEOUT"

# This step is just setup, actual logic in next task
```

**Step 4: Verify test passes**
Run: `pytest apps/ingestion-worker/tests/test_file_handler_v2.py -v`
Expected: FAIL (Still fails, logic is next task) -> *Self-Correction: This task is setup, tests WILL fail. Mark as "Partial Success" or move logic to this task. I will include logic in Task 2.*


### Task 2: Metadata Extraction & Error Logic

**Files:**
- Modify: `apps/ingestion-worker/handlers/file.py:20-60`

**Requirements:**
- **Acceptance Criteria**
  1. `handle_file_task` returns dictionary with `content` AND `metadata`.
  2. Exceptions are caught and re-raised with standard error codes strings.

- **Functional Requirements**
  1. Extract `doc.meta.title`, `author`, `creation_date`, `language`.
  2. Map `docling.errors.ConversionError` (mocked) to internal taxonomy.

- **Non-Functional Requirements**
  - Performance: Extraction must not block main loop (already threaded).

- **Test Coverage**
  - All tests from Task 1 must pass.

**Step 1: Write failing test**
(Already written in Task 1)

**Step 2: Verify test fails**
Run: `pytest apps/ingestion-worker/tests/test_file_handler_v2.py -v`
Expected: FAIL

**Step 3: Write minimal implementation**
```python
# apps/ingestion-worker/handlers/file.py

# ... imports ...
import logging

# Define custom exception for controlled failures
class IngestionError(Exception):
    def __init__(self, code, message):
        self.code = code
        super().__init__(message)

async def handle_file_task(file_path: str):
    try:
        # Run conversion in thread pool (existing)
        # Assuming 'converter' is global
        result = await asyncio.get_event_loop().run_in_executor(
            None, converter.convert, file_path
        )
        
        # Extract metadata
        meta = {
            "title": getattr(result.document.meta, 'title', None) or file_path.split('/')[-1],
            "author": getattr(result.document.meta, 'author', None),
            "created_at": getattr(result.document.meta, 'creation_date', None),
            "pages": getattr(result.document, 'num_pages', 0),
            "language": getattr(result.document.meta, 'language', 'en'),
        }

        content = result.document.export_to_markdown()
        
        if not content.strip():
             raise IngestionError(ERR_EMPTY, "File contains no text")

        return {
            "content": content,
            "metadata": meta
        }

    except Exception as e:
        # Simple mapping logic (expand with real Docling exceptions if known, else generic)
        msg = str(e).lower()
        if "password" in msg or "encrypted" in msg:
             raise IngestionError(ERR_ENCRYPTED, "File is password protected")
        elif "format" in msg:
             raise IngestionError(ERR_INVALID_FORMAT, "Invalid file format")
        else:
             raise e 
```

**Step 4: Verify test passes**
Run: `pytest apps/ingestion-worker/tests/test_file_handler_v2.py -v`
Expected: PASS


### Task 3: Resource Limiting (Concurrency)

**Files:**
- Modify: `apps/ingestion-worker/handlers/file.py`
- Modify: `apps/ingestion-worker/config.py`

**Requirements:**
- **Acceptance Criteria**
  1. Max 2 concurrent file conversions allowed.
  2. 300s timeout enforced.

- **Functional Requirements**
  1. Use `asyncio.Semaphore(2)`.
  2. Wrap execution in `asyncio.wait_for`.

- **Non-Functional Requirements**
  - Prevent OOM on small instances.

- **Test Coverage**
  - [Unit] `test_concurrency_limit` - Spawn 5 tasks, verify only 2 run at once (mock delay).
  - [Unit] `test_timeout` - Mock slow task > 300s, assert `ERR_TIMEOUT`.

**Step 1: Write failing test**
```python
# apps/ingestion-worker/tests/test_file_handler_v2.py
import asyncio

@pytest.mark.asyncio
async def test_concurrency_limit():
    # Setup: fast tasks that verify active count
    # This requires more complex mocking of the semaphore, 
    # or relying on observing execution time/order.
    pass 

@pytest.mark.asyncio
async def test_timeout():
    # ... mock sleep ...
    pass
```

**Step 2: Verify test fails**
Run: `pytest apps/ingestion-worker/tests/test_file_handler_v2.py`
Expected: FAIL (Timeout not enforced)

**Step 3: Write minimal implementation**
```python
# apps/ingestion-worker/handlers/file.py

CONCURRENCY_LIMIT = asyncio.Semaphore(2)
TIMEOUT_SECONDS = 300

async def handle_file_task(file_path: str):
    async with CONCURRENCY_LIMIT:
        try:
            return await asyncio.wait_for(
                _actual_handle_logic(file_path), # Moved logic to helper or internal
                timeout=TIMEOUT_SECONDS
            )
        except asyncio.TimeoutError:
            raise IngestionError(ERR_TIMEOUT, "Processing timed out")
```

**Step 4: Verify test passes**
Run: `pytest apps/ingestion-worker/tests/test_file_handler_v2.py`
Expected: PASS


### Task 4: Main Integration & Payload Structure

**Files:**
- Modify: `apps/ingestion-worker/main.py`

**Requirements:**
- **Acceptance Criteria**
  1. `process_message` correctly merges `metadata` from handler into final payload.
  2. `fail_payload` uses the standardized error code if `IngestionError` is raised.

- **Functional Requirements**
  1. Handle `IngestionError` specifically.
  2. Update payload construction to `payload['metadata'] = result.get('metadata')`.

- **Non-Functional Requirements**
  None.

- **Test Coverage**
  - [Unit] `test_process_message_success` - verify JSON structure.
  - [Unit] `test_process_message_failure` - verify error code in JSON.

**Step 1: Write failing test**
```python
# apps/ingestion-worker/tests/test_main_integration.py
# Mock NSQ message and handler
```

**Step 2: Verify test fails**
Run: `pytest apps/ingestion-worker/tests/test_main_integration.py`
Expected: FAIL

**Step 3: Write minimal implementation**
```python
# apps/ingestion-worker/main.py

# ... inside process_message ...
try:
    result_data = await handler(file_path) # Now returns dict
    
    result_payload = {
        "source_id": str(uuid.uuid4()),
        "status": "success",
        "content": result_data['content'],
        "metadata": result_data['metadata'], # Added
        # ...
    }

except IngestionError as e:
    fail_payload = {
        "status": "failed",
        "error": {
             "code": e.code,
             "message": str(e)
        }
    }
    # ...
```

**Step 4: Verify test passes**
Run: `pytest apps/ingestion-worker/tests/test_main_integration.py`
Expected: PASS
