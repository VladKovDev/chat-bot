import logging
import sys
from pathlib import Path

import structlog

from app.core.config import Settings


def setup_logging(settings: Settings) -> None:
    """Configure structlog based on settings."""
    log_level = getattr(logging, settings.log_level.upper(), logging.DEBUG)
    log_dir = Path(settings.log_file_path).parent
    log_dir.mkdir(parents=True, exist_ok=True)

    # Configure standard logging
    handlers: list[logging.Handler] = []
    if settings.log_output == "console":
        handlers.append(logging.StreamHandler(sys.stdout))
    elif settings.log_output == "file":
        handlers.append(logging.FileHandler(settings.log_file_path))

    logging.basicConfig(
        format="%(message)s",
        level=log_level,
        handlers=handlers,
    )

    # Configure processors
    processors = [
        structlog.contextvars.merge_contextvars,
        structlog.stdlib.add_log_level,
        structlog.stdlib.add_logger_name,
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.StackInfoRenderer(),
        structlog.processors.format_exc_info,
    ]

    if settings.log_format == "json":
        processors.append(structlog.processors.JSONRenderer())
    else:
        processors.append(structlog.dev.ConsoleRenderer(colors=False))

    structlog.configure(
        processors=processors,
        wrapper_class=structlog.stdlib.BoundLogger,
        context_class=dict,
        logger_factory=structlog.stdlib.LoggerFactory(),
        cache_logger_on_first_use=True,
    )


def get_logger(name: str | None = None) -> structlog.stdlib.BoundLogger:
    """Get a structured logger instance."""
    return structlog.get_logger(name)