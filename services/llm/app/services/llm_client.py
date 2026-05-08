import asyncio

import ollama
from ollama import AsyncClient

from app.core.exceptions import LLMProviderError
from app.core.logging import get_logger

logger = get_logger(__name__)


class OllamaClient:
    def __init__(self, host: str, model: str, timeout: int):
        self.host = host
        self.model = model
        self.timeout = timeout
        self._client: AsyncClient | None = None

    async def _get_client(self) -> AsyncClient:
        if self._client is None:
            self._client = AsyncClient(host=self.host)
        return self._client

    async def generate(self, prompt: str) -> str:
        try:
            client = await self._get_client()
            logger.debug("Sending request to Ollama", model=self.model)

            response = await asyncio.wait_for(
                client.generate(model=self.model, prompt=prompt),
                timeout=self.timeout,
            )

            result = response.get("response", "")
            logger.debug("Received response from Ollama", response_length=len(result))

            return result

        except asyncio.TimeoutError:
            logger.error("Ollama request timeout", timeout=self.timeout)
            raise LLMProviderError(f"Request timeout after {self.timeout}s")

        except ollama.ResponseError as e:
            logger.error("Ollama response error", error=str(e))
            raise LLMProviderError(f"Ollama error: {e}")

        except Exception as e:
            logger.error("Unexpected error during Ollama request", error=str(e))
            raise LLMProviderError(f"Unexpected error: {e}")

    async def healthcheck(self) -> bool:
        try:
            client = await self._get_client()
            await asyncio.wait_for(
                client.list(),
                timeout=5.0,
            )
            logger.debug("Ollama healthcheck passed")
            return True

        except Exception as e:
            logger.warning("Ollama healthcheck failed", error=str(e))
            return False