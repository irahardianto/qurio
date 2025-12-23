import asyncio
import logging
import json
import nsq
import uvloop
import tornado.platform.asyncio
from config import settings
from handlers.web import handle_web_task
from handlers.file import handle_file_task

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# Global producer
producer = None

def handle_message(message):
    """
    pynsq callback. Must be sync.
    We'll schedule the async processing on the event loop.
    """
    message.enable_async()
    asyncio.create_task(process_message(message))

async def process_message(message):
    global producer
    
    # Keep message alive
    stop_touch = asyncio.Event()
    async def touch_loop():
        while not stop_touch.is_set():
            message.touch()
            await asyncio.sleep(30)
            
    touch_task = asyncio.create_task(touch_loop())

    try:
        data = json.loads(message.body)
        logger.info(f"Received message: {data}")
        
        result_content = None
        source_id = data.get('id')
        task_type = data.get('type')
        results_list = []
        
        if task_type == 'web':
            url = data.get('url')
            max_depth = data.get('max_depth', 0)
            exclusions = data.get('exclusions', [])
            api_key = data.get('gemini_api_key')
            # Returns list of dicts: [{"url": "...", "content": "..."}]
            results_list = await handle_web_task(url, max_depth=max_depth, exclusions=exclusions, api_key=api_key)
        
        elif task_type == 'file':
            file_path = data.get('path')
            content = await handle_file_task(file_path)
            # Use path as URL for files
            results_list = [{"url": file_path, "content": content}]
            
        if results_list and producer:
            for res in results_list:
                result_payload = {
                    "source_id": source_id,
                    "content": res['content'],
                    "url": res['url']
                }
                
                producer.pub(
                    settings.nsq_topic_result,
                    json.dumps(result_payload).encode('utf-8'),
                    callback=lambda c, d: logger.info(f"Published result for {source_id} - {res.get('url')}")
                )
            
        message.finish()
        
    except Exception as e:
        logger.error(f"Error processing message: {e}")
        # message.requeue() # pynsq handles requeue on timeout if not finished? 
        # Or explicit requeue:
        message.requeue(delay=10)
    finally:
        stop_touch.set()
        await touch_task

def main():
    logger.info("Ingestion Worker Starting...")
    
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
    reader = nsq.Reader(
        message_handler=handle_message,
        nsqd_tcp_addresses=[settings.nsqd_tcp_address],
        lookupd_http_addresses=[settings.nsq_lookupd_http],
        topic=settings.nsq_topic_ingest,
        channel=settings.nsq_channel_worker,
        max_in_flight=1
    )
    
    # Create Producer (Writer)
    # nsq.Writer connects to nsqd_tcp_addresses
    global producer
    producer = nsq.Writer([settings.nsqd_tcp_address])
    
    logger.info("NSQ Reader and Writer initialized")
    
    # Run the loop
    loop.run_forever()

if __name__ == "__main__":
    main()
