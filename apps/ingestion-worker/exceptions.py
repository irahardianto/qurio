class IngestionError(Exception):
    def __init__(self, code, message):
        self.code = code
        super().__init__(message)


# Error Taxonomy
ERR_ENCRYPTED = "ERR_ENCRYPTED"
ERR_INVALID_FORMAT = "ERR_INVALID_FORMAT"
ERR_EMPTY = "ERR_EMPTY"
ERR_TIMEOUT = "ERR_TIMEOUT"

# Crawl-specific errors
ERR_CRAWL_TIMEOUT = "ERR_CRAWL_TIMEOUT"
ERR_CRAWL_DNS = "ERR_CRAWL_DNS"
ERR_CRAWL_REFUSED = "ERR_CRAWL_REFUSED"
ERR_CRAWL_BLOCKED = "ERR_CRAWL_BLOCKED"

# Transient error codes (eligible for automatic retry)
TRANSIENT_ERRORS = {ERR_TIMEOUT, ERR_CRAWL_TIMEOUT, ERR_CRAWL_DNS, ERR_CRAWL_REFUSED}
