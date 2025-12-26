from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    nsq_lookupd_http: str = "nsqlookupd:4161"
    nsq_topic_ingest: str = "ingest.task"
    nsq_channel_worker: str = "worker"
    nsq_topic_result: str = "ingest.result"
    nsqd_tcp_address: str = "nsqd:4150"
    gemini_api_key: str = "" # Env: GEMINI_API_KEY
    nsq_max_in_flight: int = 10 # Env: NSQ_MAX_IN_FLIGHT

settings = Settings()
