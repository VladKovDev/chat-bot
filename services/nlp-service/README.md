# nlp-service

The default embedding mode is `fake`, so unit and contract tests do not download model weights.

Real Qwen3 embeddings are opt-in:

```bash
pip install -e ".[qwen]"
EMBEDDING_MODE=qwen3 \
EMBEDDING_MODEL_ID=Qwen/Qwen3-Embedding-0.6B \
EMBEDDING_DIMENSION=384 \
EMBEDDING_DEVICE=cpu \
uvicorn app.main:app --host 0.0.0.0 --port 8080
```

Quick contract tests:

```bash
pytest
```
