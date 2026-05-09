from abc import ABC, abstractmethod


class BaseLLMClient(ABC):
    """Abstract base class for LLM clients."""

    @abstractmethod
    async def generate(self, prompt: str) -> str:
        """Generate a response from the LLM."""
        pass

    @abstractmethod
    async def healthcheck(self) -> bool:
        """Check if the LLM service is healthy."""
        pass
