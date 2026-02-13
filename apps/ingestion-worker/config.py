from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    nsq_lookupd_http: str = "nsqlookupd:4161"
    nsq_topic_ingest: str = "ingest.task"
    nsq_channel_worker: str = "worker"
    nsq_topic_result: str = "ingest.result"
    nsqd_tcp_address: str = "nsqd:4150"
    gemini_api_key: str = ""  # Env: GEMINI_API_KEY
    nsq_max_in_flight: int = 8  # Env: NSQ_MAX_IN_FLIGHT
    nsq_heartbeat_interval: int = 60  # Env: NSQ_HEARTBEAT_INTERVAL
    crawler_page_timeout: int = 120000  # Env: CRAWLER_PAGE_TIMEOUT
    env: str = "production"  # Env: ENV

    # Retry Logic
    retry_max_attempts: int = 3
    retry_initial_delay_ms: int = 1000
    retry_max_delay_ms: int = 60000
    retry_backoff_multiplier: int = 2


settings = Settings()
