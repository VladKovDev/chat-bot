import uvicorn
from fastapi import FastAPI

from app.api.router import api_router, root_router
from app.config import Settings, settings
from core.embeddings import FakeEmbeddingProvider, Qwen3EmbeddingProvider, UnavailableEmbeddingProvider
from core.preprocessor import RussianPreprocessor


def create_app(app_settings: Settings | None = None) -> FastAPI:
    current_settings = app_settings or settings
    app = FastAPI(
        title="nlp-service",
        version="1.0.0",
        docs_url="/docs" if current_settings.debug else None,
        redoc_url=None,
    )

    app.extra["preprocessor"] = RussianPreprocessor(cache_size=current_settings.lemmatizer_cache_size)
    if current_settings.embedding_mode == "fake":
        app.extra["embeddings"] = FakeEmbeddingProvider(
            dimension=current_settings.embedding_dimension,
            seed=current_settings.embedding_seed,
        )
    elif current_settings.embedding_mode == "unavailable":
        app.extra["embeddings"] = UnavailableEmbeddingProvider(
            dimension=current_settings.embedding_dimension,
        )
    else:
        app.extra["embeddings"] = Qwen3EmbeddingProvider(
            model_id=current_settings.embedding_model_id,
            dimension=current_settings.embedding_dimension,
            device=current_settings.embedding_device,
            query_instruction=current_settings.embedding_query_instruction,
        )

    app.include_router(root_router)
    app.include_router(api_router)
    return app


app = create_app()


if __name__ == "__main__":
    uvicorn.run(
        "app.main:app",
        host=settings.host,
        port=settings.port,
        workers=settings.workers,
        log_config=None,
        access_log=False,
    )
