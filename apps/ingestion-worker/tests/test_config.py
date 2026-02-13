import os
from config import Settings


def test_topic_override():
    os.environ["NSQ_TOPIC_INGEST"] = "ingest.test.topic"
    settings = Settings()
    assert settings.nsq_topic_ingest == "ingest.test.topic"
    del os.environ["NSQ_TOPIC_INGEST"]
