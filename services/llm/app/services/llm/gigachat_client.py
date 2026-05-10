import asyncio
from datetime import UTC, datetime, timedelta
from typing import cast

import httpx

from app.core.exceptions import LLMProviderError
from app.core.logging import get_logger
from app.services.llm.base import BaseLLMClient

logger = get_logger(__name__)


class GigaChatClient(BaseLLMClient):
    """GigaChat API client using official SDK."""

    # OAuth endpoints
    OAUTH_URL = "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
    # API base URLs
    DEFAULT_BASE_URL = "https://gigachat.devices.sberbank.ru/api/"
    BUSINESS_BASE_URL = "https://api.giga.chat/"

    # Token expiration buffer (refresh 5 minutes before actual expiration)
    TOKEN_EXPIRY_BUFFER = timedelta(minutes=5)

    def __init__(
        self,
        credentials: str,
        model: str = "GigaChat",
        timeout: int = 30,
        base_url: str | None = None,
        scope: str = "GIGACHAT_API_PERS",
        json_mode: bool = False,
    ):
        """
        Initialize GigaChat client.

        Args:
            credentials: Basic authorization key (base64 encoded)
            model: Model name to use (default: GigaChat)
            timeout: Request timeout in seconds
            base_url: API base URL (defaults to main URL)
            scope: OAuth scope (GIGACHAT_API_PERS, GIGACHAT_API_B2B, or GIGACHAT_API_CORP)
            json_mode: Force JSON responses using response_format parameter
        """
        self.credentials = credentials
        self.model = model
        self.timeout = timeout
        self.base_url = base_url or self.DEFAULT_BASE_URL
        self.scope = scope
        self.json_mode = json_mode

        # Token cache
        self._access_token: str | None = None
        self._token_expires_at: datetime | None = None
        self._token_lock = asyncio.Lock()

        # HTTP client
        self._client: httpx.AsyncClient | None = None

    async def _get_http_client(self) -> httpx.AsyncClient:
        """Get or create HTTP client."""
        if self._client is None:
            # GigaChat API uses Sber certificates, disable SSL verification for OAuth
            # but keep it for API requests
            self._client = httpx.AsyncClient(
                base_url=self.base_url,
                timeout=self.timeout,
                verify=False,  # GigaChat uses self-signed certificates
            )
        return self._client

    async def _get_access_token(self) -> str:
        """
        Get valid access token, refreshing if necessary.

        Returns:
            Valid access token

        Raises:
            LLMProviderError: If token refresh fails
        """
        async with self._token_lock:
            # Check if we have a valid token
            if self._access_token and self._token_expires_at:
                # Refresh token if it's about to expire
                if datetime.now(UTC) < (self._token_expires_at - self.TOKEN_EXPIRY_BUFFER):
                    logger.debug("Using cached access token")
                    return self._access_token

            # Need to refresh token
            logger.info("Refreshing GigaChat access token")
            await self._refresh_token()
            return self._access_token  # type: ignore

    async def _refresh_token(self) -> None:
        """
        Refresh the access token using OAuth credentials.

        Raises:
            LLMProviderError: If token refresh fails
        """
        import uuid

        headers = {
            "Content-Type": "application/x-www-form-urlencoded",
            "Accept": "application/json",
            "RqUID": str(uuid.uuid4()),
            "Authorization": f"Basic {self.credentials}",
        }

        data = {"scope": self.scope}

        try:
            # Create separate client for OAuth without SSL verification
            async with httpx.AsyncClient(verify=False, timeout=self.timeout) as oauth_client:
                response = await oauth_client.post(
                    self.OAUTH_URL,
                    headers=headers,
                    data=data,
                )
                response.raise_for_status()

            token_data = response.json()
            self._access_token = token_data["access_token"]
            expires_at = token_data.get("expires_at")

            if expires_at:
                # Parse expires_at - GigaChat returns milliseconds, convert to seconds
                # Try as milliseconds first (13 digits), then as seconds (10 digits)
                if expires_at > 1_000_000_000_000:  # Milliseconds
                    self._token_expires_at = datetime.fromtimestamp(
                        expires_at / 1000,
                        tz=UTC,
                    )
                else:  # Seconds
                    self._token_expires_at = datetime.fromtimestamp(
                        expires_at,
                        tz=UTC,
                    )
            else:
                # Default to 30 minutes if not specified
                self._token_expires_at = datetime.now(UTC) + timedelta(minutes=30)

            logger.info(
                "Access token refreshed successfully",
                expires_at=self._token_expires_at.isoformat(),
            )

        except httpx.HTTPStatusError as e:
            logger.error("HTTP error during token refresh", status_code=e.response.status_code)
            raise LLMProviderError(f"Failed to refresh token: {e.response.status_code}")

        except (KeyError, ValueError) as e:
            logger.error("Invalid token response", error=str(e))
            raise LLMProviderError(f"Invalid token response: {e}")

        except Exception as e:
            logger.error("Unexpected error during token refresh", error=str(e))
            raise LLMProviderError(f"Token refresh failed: {e}")

    async def generate(self, prompt: str) -> str:
        """
        Generate a response from GigaChat.

        Args:
            prompt: The prompt to send to the model

        Returns:
            Generated response text

        Raises:
            LLMProviderError: If generation fails
        """
        import time

        client = await self._get_http_client()
        access_token = await self._get_access_token()

        headers = {
            "Content-Type": "application/json",
            "Authorization": f"Bearer {access_token}",
        }

        payload = {
            "model": self.model,
            "messages": [
                {
                    "role": "user",
                    "content": prompt,
                }
            ],
            "temperature": 0.7,
            "max_tokens": 2000,
        }

        # Add response_format for JSON mode
        if self.json_mode:
            payload["response_format"] = {"type": "json_object"}
            logger.debug("JSON mode enabled for GigaChat request")

        try:
            logger.info("Sending request to GigaChat", model=self.model, prompt_length=len(prompt))

            logger.debug("GigaChat request prepared", prompt_length=len(prompt))

            start_time = time.time()

            response = await client.post(
                "v1/chat/completions",
                headers=headers,
                json=payload,
            )

            elapsed = time.time() - start_time
            logger.info("GigaChat response received", status_code=response.status_code, elapsed_seconds=elapsed)

            response.raise_for_status()

            result = response.json()

            # Extract response content
            content = cast(str, result["choices"][0]["message"]["content"])

            logger.info(
                "GigaChat generation successful",
                response_length=len(content),
                usage=result.get("usage"),
                total_elapsed=elapsed,
            )

            logger.debug("GigaChat response parsed", response_length=len(content))

            return content

        except httpx.TimeoutException as e:
            logger.error("GigaChat request timeout", error=str(e))
            raise LLMProviderError("Request to GigaChat timed out")

        except httpx.HTTPStatusError as e:
            logger.error("HTTP error during generation", status_code=e.response.status_code)
            raise LLMProviderError(f"GigaChat API error: {e.response.status_code}")

        except (KeyError, IndexError, TypeError) as e:
            logger.error("Invalid response format", error=str(e))
            raise LLMProviderError(f"Invalid response format: {e}")

        except Exception as e:
            logger.error("Unexpected error during generation", error=str(e), error_type=type(e).__name__)
            raise LLMProviderError(f"Generation failed: {e}")

    async def healthcheck(self) -> bool:
        """
        Check if GigaChat service is healthy.

        Returns:
            True if healthy, False otherwise
        """
        try:
            client = await self._get_http_client()
            access_token = await self._get_access_token()

            headers = {
                "Authorization": f"Bearer {access_token}",
            }

            # Try to list models as a healthcheck
            response = await client.get(
                "v1/models",
                headers=headers,
                timeout=5.0,
            )
            response.raise_for_status()

            logger.debug("GigaChat healthcheck passed")
            return True

        except Exception as e:
            logger.warning("GigaChat healthcheck failed", error=str(e))
            return False

    async def close(self) -> None:
        """Close the HTTP client."""
        if self._client:
            await self._client.aclose()
            self._client = None
            logger.debug("GigaChat HTTP client closed")
