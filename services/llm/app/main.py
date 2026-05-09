from pathlib import Path

from fastapi import FastAPI

from app.api.routes.decide import router as decide_router
from app.api.routes.health import router as health_router
from app.core.config import LLMProvider, Settings
from app.core.logging import get_logger, setup_logging
from app.services.decide_service import DecideService
from app.services.decision_engine_client import DecisionEngineClient
from app.services.domain_service import DomainService
from app.services.llm.base import BaseLLMClient
from app.services.llm.gigachat_client import GigaChatClient
from app.services.llm.ollama_client import OllamaClient
from app.services.prompt_builder import PromptBuilder

settings = Settings()
setup_logging(settings)
logger = get_logger(__name__)

app = FastAPI(
    title="LLM Service",
    description="LLM microservice for dialogue classification",
    version="0.1.0",
)


def _create_llm_client() -> BaseLLMClient:
    """Create LLM client based on provider configuration."""
    provider = settings.llm_provider

    if provider == LLMProvider.OLLAMA:
        logger.info("Creating Ollama LLM client", model=settings.ollama_model)
        return OllamaClient(
            host=settings.ollama_host,
            model=settings.ollama_model,
            timeout=settings.llm_timeout,
        )
    elif provider == LLMProvider.GIGACHAT:
        logger.info("Creating GigaChat LLM client", model=settings.gigachat_model)
        return GigaChatClient(
            credentials=settings.gigachat_credentials,
            model=settings.gigachat_model,
            timeout=settings.llm_timeout,
            base_url=settings.gigachat_base_url,
            scope=settings.gigachat_scope,
            json_mode=settings.gigachat_json_mode,
        )
    else:
        raise ValueError(f"Unsupported LLM provider: {provider}")


@app.on_event("startup")
async def startup_event() -> None:
    logger.info("Starting LLM service", host=settings.server_host, port=settings.server_port)

    llm_client = _create_llm_client()

    domain_service = DomainService()

    # Fetch domain schema from decision-engine (non-blocking)
    logger.info("Attempting to fetch domain schema from decision-engine", url=settings.decision_engine_host)
    try:
        async with DecisionEngineClient(base_url=settings.decision_engine_host) as de_client:
            schema = await de_client.fetch_config()
            domain_service.load_schema(
                intents=schema.intents,
                states=schema.states,
                actions=schema.actions,
            )
            logger.info("Domain schema loaded successfully from decision-engine")
    except Exception as e:
        logger.warning(
            "Failed to fetch domain schema from decision-engine during startup. "
            "Service will start anyway and retry on first request.",
            error=str(e),
        )

    prompts_dir = Path(__file__).parent / "prompts"
    prompt_builder = PromptBuilder(prompts_dir=prompts_dir)

    decide_service = DecideService(
        llm_client=llm_client,
        domain_service=domain_service,
        prompt_builder=prompt_builder,
        max_retries=settings.llm_max_retries,
        retry_delay=settings.llm_retry_delay,
        decision_engine_url=settings.decision_engine_host,
    )

    app.state.llm_client = llm_client
    app.state.domain_service = domain_service
    app.state.prompt_builder = prompt_builder
    app.state.decide_service = decide_service

    app.include_router(health_router)
    app.include_router(decide_router)

    logger.info("LLM service started successfully")


@app.on_event("shutdown")
async def shutdown_event() -> None:
    logger.info("Shutting down LLM service")
