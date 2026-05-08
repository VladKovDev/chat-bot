from fastapi import APIRouter, Request, Response
from fastapi.responses import JSONResponse

from app.core.logging import get_logger
from app.schemas.requests import ConfigRequest
from app.schemas.responses import ConfigResponse
from app.services.domain_service import DomainService

router = APIRouter()
logger = get_logger(__name__)


@router.post("/llm/config")
async def load_config(request: Request, response: Response) -> JSONResponse:
    domain_service: DomainService = request.app.state.domain_service
    body = await request.json()

    data = body.get("data", {})
    config = ConfigRequest(**data)

    domain_service.load_schema(
        intents=config.intents,
        states=config.states,
        actions=config.actions,
    )

    result = ConfigResponse(status="success", message="Domain schema loaded successfully")
    return JSONResponse(content=result.model_dump(), status_code=200)