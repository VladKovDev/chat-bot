import pytest
from fastapi.testclient import TestClient

from app.config import Settings
from app.main import create_app


FORBIDDEN_DECISION_FIELDS = {"intent", "state", "actions"}


@pytest.fixture()
def client() -> TestClient:
    settings = Settings(embedding_mode="fake", embedding_dimension=8)
    with TestClient(create_app(settings)) as test_client:
        yield test_client


def assert_no_decision_fields(payload: object) -> None:
    if isinstance(payload, dict):
        assert FORBIDDEN_DECISION_FIELDS.isdisjoint(payload)
        for value in payload.values():
            assert_no_decision_fields(value)
    elif isinstance(payload, list):
        for item in payload:
            assert_no_decision_fields(item)


def test_health_and_ready_report_available_model(client: TestClient) -> None:
    health = client.get("/health")
    assert health.status_code == 200
    assert health.json() == {"status": "ok"}

    ready = client.get("/ready")
    assert ready.status_code == 200
    payload = ready.json()
    assert payload["status"] == "ready"
    assert payload["model"] == "fake-hash-embedding-v1"
    assert payload["dimension"] == 8
    assert_no_decision_fields(payload)


def test_preprocess_handles_russian_text_and_lemmas(client: TestClient) -> None:
    response = client.post(
        "/api/v1/preprocess",
        json={"text": "Личный кабинет не работает!"},
    )

    assert response.status_code == 200
    payload = response.json()
    assert payload["normalized_text"] == "личный кабинет не работает"
    assert payload["tokens"] == ["личный", "кабинет", "не", "работает"]
    assert payload["lemmas"] == ["личный", "кабинет", "не", "работать"]
    assert payload["model"] in {"pymorphy3", "fallback"}
    assert_no_decision_fields(payload)


def test_embed_returns_stable_dimensioned_vector(client: TestClient) -> None:
    first = client.post("/api/v1/embed", json={"text": "Где мой платеж?"})
    second = client.post("/api/v1/embed", json={"text": "Где мой платеж?"})

    assert first.status_code == 200
    assert second.status_code == 200
    payload = first.json()
    assert payload == second.json()
    assert payload["model"] == "fake-hash-embedding-v1"
    assert payload["dimension"] == 8
    assert len(payload["embedding"]) == 8
    assert all(isinstance(value, float) for value in payload["embedding"])
    assert_no_decision_fields(payload)


def test_batch_embed_returns_one_vector_per_input(client: TestClient) -> None:
    response = client.post(
        "/api/v1/embed/batch",
        json={"texts": ["Первый вопрос", "Второй вопрос"]},
    )

    assert response.status_code == 200
    payload = response.json()
    assert payload["model"] == "fake-hash-embedding-v1"
    assert payload["dimension"] == 8
    assert len(payload["items"]) == 2
    assert [item["index"] for item in payload["items"]] == [0, 1]
    assert all(len(item["embedding"]) == 8 for item in payload["items"])
    assert payload["items"][0]["embedding"] != payload["items"][1]["embedding"]
    assert_no_decision_fields(payload)


def test_ready_and_embed_fail_when_model_is_unavailable() -> None:
    settings = Settings(embedding_mode="unavailable", embedding_dimension=8)
    with TestClient(create_app(settings)) as unavailable_client:
        ready = unavailable_client.get("/ready")
        assert ready.status_code == 503
        assert ready.json()["status"] == "unavailable"

        embed = unavailable_client.post("/api/v1/embed", json={"text": "Привет"})
        assert embed.status_code == 503
        assert embed.json()["detail"] == "embedding model unavailable"
        assert_no_decision_fields(embed.json())


@pytest.mark.parametrize(
    ("path", "body"),
    [
        ("/api/v1/preprocess", {"text": ""}),
        ("/api/v1/embed", {"text": ""}),
        ("/api/v1/embed/batch", {"texts": []}),
        ("/api/v1/preprocess", {"text": "а" * 4001}),
        ("/api/v1/embed/batch", {"texts": ["текст"] * 65}),
    ],
)
def test_contract_rejects_empty_or_unbounded_requests(
    client: TestClient,
    path: str,
    body: dict[str, object],
) -> None:
    response = client.post(path, json=body)
    assert response.status_code == 422


def test_target_runtime_does_not_mount_legacy_decide_route(client: TestClient) -> None:
    response = client.post("/llm/decide", json={})
    assert response.status_code == 404
