import httpx
from fastapi import APIRouter, Request, Response
from fastapi.responses import JSONResponse
from pydantic import ValidationError

from app.core.exceptions import ValidationRetryExhaustedError
from app.core.logging import get_logger
from app.schemas.requests import DecideRequest
from app.services.decide_service import DecideService

router = APIRouter()
logger = get_logger(__name__)


@router.post("/llm/decide")
async def decide(request: Request, response: Response) -> JSONResponse:
    decide_service: DecideService = request.app.state.decide_service
    body = await request.json()

    # Support both formats: {"data": {...}} and direct {...}
    # The Go client sends direct format, HTTP examples use {"data": {...}}
    if "data" in body:
        data = body.get("data")
    else:
        # Direct format from Go client
        data = body

    if not data:
        logger.warning("Empty payload in request body", body=body)
        return JSONResponse(
            content={
                "error": "Empty payload. Expected {'state': str, 'messages': [...]}"
            },
            status_code=422,
        )

    try:
        req = DecideRequest(**data)
    except ValidationError as e:
        logger.warning("Invalid request payload", errors=e.errors(), received_data=data)
        error_details = [
            f"  - {err['loc'][0] if err['loc'] else 'field'}: {err['msg']}" for err in e.errors()
        ]
        return JSONResponse(
            content={
                "error": "Invalid request payload. Required fields: state, messages",
                "details": error_details,
            },
            status_code=422,
        )

    try:
        result = await decide_service.decide(req)
        return JSONResponse(content={"data": result.model_dump()}, status_code=200)

    except httpx.HTTPError as e:
        logger.error("Failed to connect to decision-engine", error=str(e))
        return JSONResponse(
            content={
                "error": "Failed to fetch domain configuration from decision-engine. "
                "Please ensure the decision-engine service is running."
            },
            status_code=503,
        )
    except ValidationRetryExhaustedError as e:
        logger.error("Validation retries exhausted", error=str(e))
        return JSONResponse(
            content={"error": f"Failed to get valid response from LLM: {e}"},
            status_code=500,
        )
    except Exception as e:
        logger.error("Unexpected error during decide request", error=str(e))
        return JSONResponse(
            content={"error": f"Internal server error: {e}"},
            status_code=500,
        )
