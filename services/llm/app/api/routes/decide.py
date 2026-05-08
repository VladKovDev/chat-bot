from fastapi import APIRouter, Request, Response
from fastapi.responses import JSONResponse

from app.core.exceptions import DomainNotLoadedError, ValidationRetryExhaustedError
from app.core.logging import get_logger
from app.schemas.requests import DecideRequest
from app.services.decide_service import DecideService

router = APIRouter()
logger = get_logger(__name__)


@router.post("/llm/decide")
async def decide(request: Request, response: Response) -> JSONResponse:
    decide_service: DecideService = request.app.state.decide_service
    body = await request.json()

    data = body.get("data", {})
    req = DecideRequest(**data)

    try:
        result = await decide_service.decide(req)
        return JSONResponse(content={"data": result.model_dump()}, status_code=200)

    except DomainNotLoadedError as e:
        logger.error("Domain not loaded", error=str(e))
        return JSONResponse(
            content={"error": "Domain schema not loaded. Call /llm/config first."},
            status_code=503,
        )

    except ValidationRetryExhaustedError as e:
        logger.error("Validation retries exhausted", error=str(e))
        return JSONResponse(
            content={"error": f"Failed to get valid response from LLM: {e}"},
            status_code=500,
        )
