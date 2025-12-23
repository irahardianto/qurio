from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    nsq_lookupd_http: str = "nsqlookupd:4161"
    nsq_topic_ingest: str = "ingest.task"
    nsq_channel_worker: str = "worker"
    nsq_topic_result: str = "ingest.result"
    nsqd_tcp_address: str = "nsqd:4150"
    gemini_api_key: str = "" # Env: GEMINI_API_KEY

settings = Settings()
