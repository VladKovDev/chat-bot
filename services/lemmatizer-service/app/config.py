from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
    )

    host: str = "0.0.0.0"
    port: int = 8080
    workers: int = 1
    log_level: str = "debug"
    lemmatizer_cache_size: int = 10_000
    debug: bool = False


settings = Settings()