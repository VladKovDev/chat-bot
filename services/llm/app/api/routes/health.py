from fastapi import APIRouter, Request
from fastapi.responses import JSONResponse

from app.services.domain_service import DomainService

router = APIRouter()


@router.get("/health")
async def health_check(request: Request) -> JSONResponse:
    domain_service: DomainService = request.app.state.domain_service
    return JSONResponse(
        content={"status": "healthy", "domain_loaded": domain_service.is_loaded()},
        status_code=200,
    )