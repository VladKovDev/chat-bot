import pytest
from pathlib import Path

from app.schemas.domain import DomainSchema
from app.schemas.requests import DecideRequest, Message
from app.services.prompt_builder import PromptBuilder


@pytest.fixture
def prompts_dir(tmp_path: Path):
    prompt_content = """Test prompt
Intents:
{intents}
States:
{states}
Actions:
{actions}
State: {state}
Summary: {summary}
Messages:
{messages}
Example:
{example}"""

    example_content = "Test example output"

    prompts_dir = tmp_path / "prompts"
    prompts_dir.mkdir()
    (prompts_dir / "decide_prompt.txt").write_text(prompt_content)
    (prompts_dir / "decide_example.txt").write_text(example_content)
    return prompts_dir


def test_build_decide_prompt(prompts_dir: Path):
    builder = PromptBuilder(prompts_dir)

    domain = DomainSchema(
        intents=["greeting", "request_operator"],
        states=["initial"],
        actions=["transfer_to_operator"],
    )

    request = DecideRequest(
        state="initial",
        summary="User greets",
        messages=[Message(role="user", text="Привет")],
    )

    prompt = builder.build_decide_prompt(domain, request)

    assert "greeting" in prompt
    assert "initial" in prompt
    assert "transfer_to_operator" in prompt
    assert "User greets" in prompt
    assert "Привет" in prompt
    assert "Test example output" in prompt


def test_build_decide_prompt_with_retry(prompts_dir: Path):
    builder = PromptBuilder(prompts_dir)

    domain = DomainSchema(
        intents=["greeting"],
        states=["initial"],
        actions=[],
    )

    request = DecideRequest(
        state="initial",
        summary="Test",
        messages=[],
    )

    prompt = builder.build_decide_prompt(domain, request, retry_error="Invalid JSON")

    assert "Invalid JSON" in prompt
    assert "Previous attempt failed" in prompt