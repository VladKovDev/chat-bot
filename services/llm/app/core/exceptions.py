class LLMServiceError(Exception):
    """Base exception for LLM service."""

    pass


class DomainNotLoadedError(LLMServiceError):
    """Raised when domain schema is not loaded."""

    pass


class LLMProviderError(LLMServiceError):
    """Raised when LLM provider fails."""

    pass


class ValidationRetryExhaustedError(LLMServiceError):
    """Raised when validation retries are exhausted."""

    pass


class InvalidResponseError(LLMServiceError):
    """Raised when LLM response is invalid."""

    pass
