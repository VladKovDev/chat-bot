from fastapi import APIRouter, Request

from app.api.schemas import HealthResponse, LemmatizeRequest, LemmatizeResponse

router = APIRouter()


@router.post("/lemmatize", response_model=LemmatizeResponse)
async def lemmatize(body: LemmatizeRequest, request: Request) -> LemmatizeResponse:
    service = request.app.state.lemmatizer

    lemmas = service.lemmatize(body.tokens)

    return LemmatizeResponse(lemmas=lemmas)


@router.get("/health", response_model=HealthResponse)
async def health(request: Request) -> HealthResponse:
    service = request.app.state.lemmatizer
    return HealthResponse(status="ok", cache=service.cache_info)