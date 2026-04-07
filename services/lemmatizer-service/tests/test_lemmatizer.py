import pytest
from fastapi.testclient import TestClient

from app.main import app


@pytest.fixture(scope="module")
def client() -> TestClient:
    with TestClient(app) as c:
        yield c


def test_lemmatize_basic(client: TestClient) -> None:
    response = client.post("/api/lemmatize", json={"tokens": ["работает", "кабинет"]})
    assert response.status_code == 200
    data = response.json()
    assert data["lemmas"] == ["работать", "кабинет"]


def test_lemmatize_preserves_length(client: TestClient) -> None:
    tokens = ["не", "работает", "личный", "кабинет"]
    response = client.post("/api/lemmatize", json={"tokens": tokens})
    assert response.status_code == 200
    assert len(response.json()["lemmas"]) == len(tokens)


def test_lemmatize_empty_list_rejected(client: TestClient) -> None:
    response = client.post("/api/lemmatize", json={"tokens": []})
    assert response.status_code == 422


def test_health(client: TestClient) -> None:
    response = client.get("/api/health")
    assert response.status_code == 200
    assert response.json()["status"] == "ok"
    assert "cache" in response.json()