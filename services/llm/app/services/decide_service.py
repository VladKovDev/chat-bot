import asyncio
import json

from app.core.exceptions import (
    InvalidResponseError,
    ValidationRetryExhaustedError,
)
from app.core.logging import get_logger
from app.schemas.domain import DomainSchema
from app.schemas.requests import DecideRequest
from app.schemas.responses import DecideResponse
from app.services.domain_service import DomainService
from app.services.llm_client import OllamaClient
from app.services.prompt_builder import PromptBuilder

logger = get_logger(__name__)


class DecideService:
    def __init__(
        self,
        llm_client: OllamaClient,
        domain_service: DomainService,
        prompt_builder: PromptBuilder,
        max_retries: int,
        retry_delay: float,
    ):
        self.llm_client = llm_client
        self.domain_service = domain_service
        self.prompt_builder = prompt_builder
        self.max_retries = max_retries
        self.retry_delay = retry_delay

    async def decide(self, request: DecideRequest) -> DecideResponse:
        domain = self.domain_service.get_schema()
        retry_error: str | None = None

        for attempt in range(self.max_retries):
            try:
                prompt = self.prompt_builder.build_decide_prompt(domain, request, retry_error)
                logger.debug("Built prompt for decide", attempt=attempt + 1)

                response_text = await self.llm_client.generate(prompt)
                logger.debug("Received LLM response", attempt=attempt + 1, response_length=len(response_text))

                response = self._parse_and_validate(response_text, domain)
                logger.info(
                    "Decide request processed",
                    intent=response.intent,
                    state=response.state,
                    actions_count=len(response.actions),
                )
                return response

            except InvalidResponseError as e:
                retry_error = str(e)
                logger.warning(
                    "Invalid LLM response",
                    attempt=attempt + 1,
                    error=retry_error,
                )

                if attempt < self.max_retries - 1:
                    await asyncio.sleep(self.retry_delay)

        logger.error("Validation retries exhausted")
        raise ValidationRetryExhaustedError(f"Failed after {self.max_retries} attempts")

    def _parse_and_validate(self, response_text: str, domain: DomainSchema) -> DecideResponse:
        try:
            data = json.loads(response_text)
        except json.JSONDecodeError as e:
            raise InvalidResponseError(f"Invalid JSON: {e}")

        if not isinstance(data, dict):
            raise InvalidResponseError("Response is not a JSON object")

        intent = data.get("intent")
        state = data.get("state")
        actions = data.get("actions")

        if not isinstance(intent, str):
            raise InvalidResponseError(f"Invalid intent type: {type(intent)}")
        if not isinstance(state, str):
            raise InvalidResponseError(f"Invalid state type: {type(state)}")
        if not isinstance(actions, list):
            raise InvalidResponseError(f"Invalid actions type: {type(actions)}")

        if intent not in domain.intents:
            raise InvalidResponseError(f"Intent '{intent}' not in domain schema")
        if state not in domain.states:
            raise InvalidResponseError(f"State '{state}' not in domain schema")

        for action in actions:
            if not isinstance(action, str):
                raise InvalidResponseError(f"Invalid action type: {type(action)}")
            if action not in domain.actions:
                raise InvalidResponseError(f"Action '{action}' not in domain schema")

        return DecideResponse(intent=intent, state=state, actions=actions)
