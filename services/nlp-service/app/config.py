from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
        extra="ignore",
    )

    host: str = "0.0.0.0"
    port: int = 8080
    workers: int = 1
    debug: bool = False
    lemmatizer_cache_size: int = 10_000
    embedding_mode: str = Field(default="fake", pattern="^(fake|unavailable)$")
    embedding_dimension: int = Field(default=384, ge=1, le=4096)
    embedding_seed: str = "chat-bot-nlp-service"


settings = Settings()
