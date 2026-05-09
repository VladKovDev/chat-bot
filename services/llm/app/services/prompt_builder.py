from pathlib import Path

from app.core.logging import get_logger
from app.schemas.domain import DomainSchema
from app.schemas.requests import DecideRequest

logger = get_logger(__name__)


class PromptBuilder:
    def __init__(self, prompts_dir: Path):
        self.prompts_dir = prompts_dir
        self._decide_prompt: str | None = None
        self._decide_example: str | None = None
        self._load_prompts()

    def _load_prompts(self) -> None:
        self._decide_prompt = self._load_prompt("decide")
        self._decide_example = self._load_example("decide")
        logger.info("Prompts loaded", prompts_dir=str(self.prompts_dir))

    def _load_prompt(self, endpoint_type: str) -> str:
        path = self.prompts_dir / f"{endpoint_type}_prompt.txt"
        if not path.exists():
            raise FileNotFoundError(f"Prompt file not found: {path}")
        return path.read_text(encoding="utf-8")

    def _load_example(self, endpoint_type: str) -> str:
        path = self.prompts_dir / f"{endpoint_type}_example.txt"
        if not path.exists():
            raise FileNotFoundError(f"Example file not found: {path}")
        return path.read_text(encoding="utf-8")

    def build_decide_prompt(
        self,
        domain: DomainSchema,
        request: DecideRequest,
        retry_error: str | None = None,
    ) -> str:
        if self._decide_prompt is None or self._decide_example is None:
            raise RuntimeError("Prompts not loaded")

        messages_str = self._format_messages(request.messages)
        summary_line = f"Conversation summary: {request.summary}" if request.summary else ""

        prompt = self._decide_prompt.format(
            intents="\n".join(f"  - {intent}" for intent in domain.intents),
            states="\n".join(f"  - {state}" for state in domain.states),
            actions="\n".join(f"  - {action}" for action in domain.actions),
            state=request.state,
            summary=summary_line,
            messages=messages_str,
            example=self._decide_example,
        )

        if retry_error:
            prompt = f"{prompt}\n\nPrevious attempt failed with error: {retry_error}\nPlease try again."

        return prompt

    def _format_messages(self, messages: list) -> str:
        formatted = []
        for msg in messages:
            formatted.append(f"  {msg.role}: {msg.text}")
        return "\n".join(formatted)
