import pytest
from httpx import AsyncClient, ASGITransport

from app.main import app
from app.services.domain_service import DomainService


@pytest.mark.asyncio
async def test_load_config():
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        response = await client.post(
            "/llm/config",
            json={
                "data": {
                    "intents": ["greeting", "request_operator"],
                    "states": ["initial"],
                    "actions": ["transfer_to_operator"],
                }
            },
        )

        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "success"
        assert "message" in data


@pytest.mark.asyncio
async def test_health_check():
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        response = await client.get("/health")

        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "healthy"