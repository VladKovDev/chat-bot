
from app.core.exceptions import DomainNotLoadedError
from app.core.logging import get_logger
from app.schemas.domain import DomainSchema

logger = get_logger(__name__)


class DomainService:
    def __init__(self) -> None:
        self._schema: DomainSchema | None = None
        self._config_loaded: bool = False

    def load_schema(self, intents: list[str], states: list[str], actions: list[str]) -> None:
        self._schema = DomainSchema(intents=intents, states=states, actions=actions)
        self._config_loaded = True
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

    def mark_unloaded(self) -> None:
        """Mark domain schema as unloaded, triggering reload on next access."""
        self._config_loaded = False
        self._schema = None
        logger.info("Domain schema marked as unloaded")
