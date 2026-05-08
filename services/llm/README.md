# LLM Service

Microservice for dialogue classification using LLM (Ollama + Qwen).

## Features

- Intent classification
- State determination
- Action selection
- Configurable domain schema
- Automatic retries with error feedback
- Structured logging

## Installation

```bash
poetry install
```

## Configuration

Create `.env` file from `.env.example`:

```bash
cp .env.example .env
```

Edit `.env` with your settings:

```bash
OLLAMA_HOST=http://localhost:11434
OLLAMA_MODEL=qwen2.5:7b
SERVER_PORT=8001
LOG_LEVEL=DEBUG
```

## Running

### Start Ollama

```bash
ollama serve
ollama pull qwen2.5:7b
```

### Start service

```bash
uvicorn app.main:app --reload --port 8001
```

Or using poetry:

```bash
poetry run uvicorn app.main:app --reload --port 8001
```

## API

### POST /llm/config

Load domain schema.

**Request:**
```json
{
  "data": {
    "intents": ["greeting", "request_operator", "resolved"],
    "states": ["initial", "category_selection", "operator_requested"],
    "actions": ["transfer_to_operator", "ask_category"]
  }
}
```

**Response:**
```json
{
  "status": "success",
  "message": "Domain schema loaded successfully"
}
```

### POST /llm/decide

Classify dialogue and get intent, state, actions.

**Request:**
```json
{
  "data": {
    "state": "initial",
    "summary": "Пользователь спрашивает про возврат товара",
    "messages": [
      {"role": "user", "text": "Здравствуйте"},
      {"role": "assistant", "text": "Добрый день! Чем могу помочь?"},
      {"role": "user", "text": "Хочу вернуть товар"}
    ]
  }
}
```

**Response:**
```json
{
  "data": {
    "intent": "request_operator",
    "state": "operator_requested",
    "actions": ["transfer_to_operator"]
  }
}
```

### GET /health

Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "domain_loaded": true
}
```

## Testing

```bash
poetry run pytest
```

With coverage:

```bash
poetry run pytest --cov=app --cov-report=html
```

## Docker

```bash
docker build -t llm-service .
docker run -p 8001:8001 --env-file .env llm-service
```

## Project Structure

```
app/
├── api/
│   └── routes/          # API endpoints
├── core/                 # Config, logging, exceptions
├── schemas/              # Pydantic models
├── services/             # Business logic
└── prompts/              # LLM prompts
```
