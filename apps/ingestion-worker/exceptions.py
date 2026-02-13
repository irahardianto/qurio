class IngestionError(Exception):
    def __init__(self, code, message):
        self.code = code
        super().__init__(message)


# Error Taxonomy
ERR_ENCRYPTED = "ERR_ENCRYPTED"
ERR_INVALID_FORMAT = "ERR_INVALID_FORMAT"
ERR_EMPTY = "ERR_EMPTY"
ERR_TIMEOUT = "ERR_TIMEOUT"
