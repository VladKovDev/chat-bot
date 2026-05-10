import sys
import types

from fastapi import FastAPI
from fastapi.testclient import TestClient

sys.modules.setdefault("ollama", types.SimpleNamespace(AsyncClient=object))

from app.api.routes.decide import router
from app.core.exceptions import ValidationRetryExhaustedError


class FailingDecideService:
    async def decide(self, _request):
        raise ValidationRetryExhaustedError("raw prompt SELECT * FROM messages upstream body")


def test_decide_returns_public_error_without_internal_details():
    app = FastAPI()
    app.state.decide_service = FailingDecideService()
    app.include_router(router)
    client = TestClient(app)

    response = client.post(
        "/llm/decide",
        json={"state": "initial", "messages": []},
        headers={"X-Request-ID": "req-llm-1"},
    )

    assert response.status_code == 503
    body_text = response.text
    assert "raw prompt" not in body_text
    assert "SELECT" not in body_text
    assert "upstream body" not in body_text

    body = response.json()
    assert body["error"]["code"] == "provider_unavailable"
    assert body["error"]["request_id"] == "req-llm-1"
    assert body["error"]["message"]
