from fastapi import APIRouter, HTTPException, Request
from fastapi.responses import JSONResponse

from app.api.schemas import (
    BatchEmbedRequest,
    BatchEmbedResponse,
    BatchEmbeddingItem,
    EmbedRequest,
    EmbedResponse,
    HealthResponse,
    PreprocessRequest,
    PreprocessResponse,
    ReadyResponse,
)
from core.embeddings import EmbeddingUnavailableError

root_router = APIRouter()
api_router = APIRouter(prefix="/api/v1")


@root_router.get("/health", response_model=HealthResponse)
async def health() -> HealthResponse:
    return HealthResponse(status="ok")


@root_router.get("/ready", response_model=ReadyResponse)
async def ready(request: Request) -> ReadyResponse | JSONResponse:
    embeddings = request.app.extra["embeddings"]
    preprocessor = request.app.extra["preprocessor"]
    status = "ready" if embeddings.available else "unavailable"
    response = ReadyResponse(
        status=status,
        model=embeddings.model_name,
        dimension=embeddings.dimension,
        lemmatizer_model=preprocessor.model_name,
    )
    if not embeddings.available:
        return JSONResponse(status_code=503, content=response.model_dump())
    return response


@api_router.post("/preprocess", response_model=PreprocessResponse)
async def preprocess(body: PreprocessRequest, request: Request) -> PreprocessResponse:
    result = request.app.extra["preprocessor"].preprocess(body.text)
    return PreprocessResponse(**result)


@api_router.post("/embed", response_model=EmbedResponse)
async def embed(body: EmbedRequest, request: Request) -> EmbedResponse:
    embeddings = request.app.extra["embeddings"]
    try:
        vector = embeddings.embed(body.text)
    except EmbeddingUnavailableError as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
    return EmbedResponse(
        embedding=vector,
        model=embeddings.model_name,
        dimension=embeddings.dimension,
    )


@api_router.post("/embed/batch", response_model=BatchEmbedResponse)
async def embed_batch(body: BatchEmbedRequest, request: Request) -> BatchEmbedResponse:
    embeddings = request.app.extra["embeddings"]
    try:
        items = [
            BatchEmbeddingItem(index=index, embedding=embeddings.embed(text))
            for index, text in enumerate(body.texts)
        ]
    except EmbeddingUnavailableError as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
    return BatchEmbedResponse(
        items=items,
        model=embeddings.model_name,
        dimension=embeddings.dimension,
    )
