import uuid

import httpx
from fastapi import APIRouter, Request
from fastapi.responses import JSONResponse
from pydantic import ValidationError

from app.core.exceptions import ValidationRetryExhaustedError
from app.core.logging import get_logger
from app.schemas.requests import DecideRequest
from app.services.decide_service import DecideService

router = APIRouter()
logger = get_logger(__name__)


@router.post("/llm/decide")
async def decide(request: Request) -> JSONResponse:
    decide_service: DecideService = request.app.state.decide_service
    request_id = request.headers.get("X-Request-ID") or str(uuid.uuid4())
    try:
        body = await request.json()
    except Exception as e:
        logger.warning(
            "invalid json request body",
            request_id=request_id,
            error_type=type(e).__name__,
        )
        return public_error(
            "invalid_request",
            "Некорректный запрос. Проверьте данные и попробуйте снова.",
            request_id,
            422,
        )

    # Support both formats: {"data": {...}} and direct {...}
    # The Go client sends direct format, HTTP examples use {"data": {...}}
    if "data" in body:
        data = body.get("data")
    else:
        # Direct format from Go client
        data = body

    if not data:
        logger.warning("empty payload in request body", request_id=request_id)
        return public_error(
            "invalid_request",
            "Некорректный запрос. Проверьте данные и попробуйте снова.",
            request_id,
            422,
        )

    try:
        req = DecideRequest(**data)
    except ValidationError as e:
        logger.warning(
            "invalid request payload",
            request_id=request_id,
            validation_error_count=len(e.errors()),
        )
        return public_error(
            "invalid_request",
            "Некорректный запрос. Проверьте данные и попробуйте снова.",
            request_id,
            422,
        )

    try:
        result = await decide_service.decide(req)
        return JSONResponse(
            content={"data": result.model_dump()},
            status_code=200,
            headers={"X-Request-ID": request_id},
        )

    except httpx.HTTPError as e:
        logger.error(
            "failed to connect to decision-engine",
            request_id=request_id,
            error_type=type(e).__name__,
        )
        return public_error(
            "provider_unavailable",
            "Не удалось проверить данные. Попробуйте позже или подключим оператора.",
            request_id,
            503,
        )
    except ValidationRetryExhaustedError as e:
        logger.error(
            "validation retries exhausted",
            request_id=request_id,
            error_type=type(e).__name__,
        )
        return public_error(
            "provider_unavailable",
            "Не удалось проверить данные. Попробуйте позже или подключим оператора.",
            request_id,
            503,
        )
    except Exception as e:
        logger.error(
            "unexpected error during decide request",
            request_id=request_id,
            error_type=type(e).__name__,
        )
        return public_error(
            "internal_error",
            "Внутренняя ошибка сервиса. Попробуйте позже.",
            request_id,
            500,
        )


def public_error(code: str, message: str, request_id: str, status_code: int) -> JSONResponse:
    return JSONResponse(
        content={
            "error": {
                "code": code,
                "message": message,
                "request_id": request_id,
            }
        },
        status_code=status_code,
        headers={"X-Request-ID": request_id},
    )
