import sys
from unittest.mock import MagicMock
# Mock handlers.web to avoid crawl4ai dependency
mock_web = MagicMock()
sys.modules['handlers.web'] = mock_web

import pytest
import json
from unittest.mock import AsyncMock, patch
from handlers.file import IngestionError, ERR_ENCRYPTED
import main

@pytest.mark.asyncio
async def test_process_message_success():
    # Mock message
    msg = MagicMock()
    msg.body = json.dumps({
        "id": "123",
        "type": "file",
        "path": "/tmp/test.pdf"
    }).encode('utf-8')
    
    # Mock handle_file_task
    with patch('main.handle_file_task', new_callable=AsyncMock) as mock_handle:
        mock_handle.return_value = {
            "content": "test content",
            "metadata": {"title": "Test Doc"}
        }
        
        # Mock producer
        main.producer = MagicMock()
        main.producer.pub = MagicMock()
        
        await main.process_message(msg)
        
        # Verify pub called with correct payload
        args, kwargs = main.producer.pub.call_args
        payload = json.loads(args[1])
        assert payload['status'] == "success"
        assert payload['metadata']['title'] == "Test Doc"
        assert payload['content'] == "test content"

@pytest.mark.asyncio
async def test_process_message_failure():
    # Mock message
    msg = MagicMock()
    msg.body = json.dumps({
        "id": "123",
        "type": "file",
        "path": "/tmp/secret.pdf"
    }).encode('utf-8')
    
    # Mock handle_file_task to raise IngestionError
    with patch('main.handle_file_task', new_callable=AsyncMock) as mock_handle:
        mock_handle.side_effect = IngestionError(ERR_ENCRYPTED, "Encrypted")
        
        # Mock producer
        main.producer = MagicMock()
        main.producer.pub = MagicMock()
        
        await main.process_message(msg)
        
        # Verify pub called with error payload
        args, kwargs = main.producer.pub.call_args
        payload = json.loads(args[1])
        assert payload['status'] == "failed"
        assert payload['error']['code'] == ERR_ENCRYPTED
        assert payload['error']['message'] == "Encrypted"
