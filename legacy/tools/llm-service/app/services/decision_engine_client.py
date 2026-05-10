from types import TracebackType

import httpx

from app.core.logging import get_logger
from app.schemas.domain import DomainSchema


class DecisionEngineClient:
    """Client for fetching configuration from decision-engine service."""

    def __init__(self, base_url: str, timeout: float = 5.0) -> None:
        self._base_url = base_url.rstrip("/")
        self._timeout = timeout
        self._client: httpx.AsyncClient | None = None
        logger = get_logger(__name__)
        logger.info("DecisionEngineClient initialized", base_url=base_url)

    async def __aenter__(self) -> "DecisionEngineClient":
        self._client = httpx.AsyncClient(timeout=self._timeout)
        return self

    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc_val: BaseException | None,
        exc_tb: TracebackType | None,
    ) -> None:
        if self._client:
            await self._client.aclose()

    async def fetch_config(self) -> DomainSchema:
        """Fetch domain configuration from decision-engine.

        Returns:
            DomainSchema with intents, states, and actions

        Raises:
            httpx.HTTPError: If request fails
            ValueError: If response format is invalid
        """
        if not self._client:
            raise RuntimeError("Client not initialized. Use async context manager.")

        logger = get_logger(__name__)
        url = f"{self._base_url}/api/v1/domain/schema"

        logger.info("Fetching config from decision-engine", url=url)

        response = await self._client.get(url)
        response.raise_for_status()

        data = response.json()
        schema = DomainSchema(
            intents=data.get("intents", []),
            states=data.get("states", []),
            actions=data.get("actions", []),
        )

        logger.info(
            "Successfully fetched domain schema",
            intents_count=len(schema.intents),
            states_count=len(schema.states),
            actions_count=len(schema.actions),
        )

        return schema
