from typing import Optional

from app.core.exceptions import DomainNotLoadedError
from app.core.logging import get_logger
from app.schemas.domain import DomainSchema

logger = get_logger(__name__)


class DomainService:
    def __init__(self):
        self._schema: Optional[DomainSchema] = None

    def load_schema(self, intents: list[str], states: list[str], actions: list[str]) -> None:
        self._schema = DomainSchema(intents=intents, states=states, actions=actions)
        logger.info(
            "Domain schema loaded",
            intents_count=len(intents),
            states_count=len(states),
            actions_count=len(actions),
        )

    def get_schema(self) -> DomainSchema:
        if self._schema is None:
            raise DomainNotLoadedError("Domain schema not loaded")
        return self._schema

    def is_loaded(self) -> bool:
        return self._schema is not None