from pathlib import Path

from fastapi import FastAPI

from app.api.routes.decide import router as decide_router
from app.api.routes.health import router as health_router
from app.core.config import Settings
from app.core.logging import get_logger, setup_logging
from app.services.decide_service import DecideService
from app.services.decision_engine_client import DecisionEngineClient
from app.services.domain_service import DomainService
from app.services.llm_client import OllamaClient
from app.services.prompt_builder import PromptBuilder

settings = Settings()
setup_logging(settings)
logger = get_logger(__name__)

app = FastAPI(
    title="LLM Service",
    description="LLM microservice for dialogue classification",
    version="0.1.0",
)


@app.on_event("startup")
async def startup_event():
    logger.info("Starting LLM service", host=settings.server_host, port=settings.server_port)

    llm_client = OllamaClient(
        host=settings.ollama_host,
        model=settings.ollama_model,
        timeout=settings.llm_timeout,
    )

    domain_service = DomainService()

    # Fetch domain schema from decision-engine
    logger.info("Fetching domain schema from decision-engine", url=settings.decision_engine_host)
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
        logger.error("Failed to fetch domain schema from decision-engine", error=str(e))
        raise RuntimeError(f"Failed to initialize domain schema: {e}") from e

    prompts_dir = Path(__file__).parent / "prompts"
    prompt_builder = PromptBuilder(prompts_dir=prompts_dir)

    decide_service = DecideService(
        llm_client=llm_client,
        domain_service=domain_service,
        prompt_builder=prompt_builder,
        max_retries=settings.llm_max_retries,
        retry_delay=settings.llm_retry_delay,
    )

    app.state.llm_client = llm_client
    app.state.domain_service = domain_service
    app.state.prompt_builder = prompt_builder
    app.state.decide_service = decide_service

    app.include_router(health_router)
    app.include_router(decide_router)

    logger.info("LLM service started successfully")


@app.on_event("shutdown")
async def shutdown_event():
    logger.info("Shutting down LLM service")
