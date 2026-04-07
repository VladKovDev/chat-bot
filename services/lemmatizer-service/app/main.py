import uvicorn
from fastapi import FastAPI

from app.api.router import router
from core.lemmatizer import Lemmatizer
from app.config import settings


def create_app() -> FastAPI:
    app = FastAPI(
        title="lemmatizer-service",
        version="1.0.0",
        docs_url="/docs" if settings.debug else None,
        redoc_url=None,
    )

    # --- heavy dependency (singleton) ---
    app.state.lemmatizer = Lemmatizer(cache_size=settings.lemmatizer_cache_size)

    # --- routing ---
    app.include_router(router, prefix="/api")

    return app


# ASGI entrypoint (uvicorn)
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