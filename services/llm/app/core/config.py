from enum import StrEnum

from pydantic_settings import BaseSettings, SettingsConfigDict


class LLMProvider(StrEnum):
    """Available LLM providers."""
    OLLAMA = "ollama"
    GIGACHAT = "gigachat"


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        env_nested_delimiter="__",
        extra="ignore",
    )

    # LLM Provider selection
    llm_provider: LLMProvider = LLMProvider.OLLAMA

    # Ollama configuration
    ollama_host: str = "http://localhost:11434"
    ollama_model: str = "qwen2.5:7b"

    # GigaChat configuration
    gigachat_credentials: str = ""
    gigachat_model: str = "GigaChat"
    gigachat_base_url: str | None = None
    gigachat_scope: str = "GIGACHAT_API_PERS"
    gigachat_json_mode: bool = False  # Force JSON responses using response_format

    # Server configuration
    server_host: str = "0.0.0.0"
    server_port: int = 8001

    # LLM configuration
    llm_timeout: int = 30
    llm_max_retries: int = 3
    llm_retry_delay: float = 1.0

    # Logging configuration
    log_level: str = "DEBUG"
    log_format: str = "text"  # text or json
    log_output: str = "console"  # console or file
    log_file_path: str = "logs/llm_service.log"

    # Decision Engine configuration
    decision_engine_host: str = "http://localhost:8080"
