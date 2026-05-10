FROM python:3.12-slim

WORKDIR /app

ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

RUN pip install poetry poetry-core

COPY pyproject.toml poetry.lock README.md ./
RUN poetry install --only main --no-interaction --no-ansi --no-root

COPY app/ ./app/
COPY configs/ ./configs/

EXPOSE 8001

CMD ["poetry", "run", "uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8001"]