import hashlib
import math
import re
from collections.abc import Callable, Sequence
from typing import Any


TOKEN_RE = re.compile(r"[0-9A-Za-zА-Яа-яЁё]+")


class EmbeddingUnavailableError(RuntimeError):
    pass


class EmbeddingConfigurationError(ValueError):
    pass


class FakeEmbeddingProvider:
    provider_name = "fake"
    model_name = "fake-hash-embedding-v1"

    def __init__(self, dimension: int, seed: str) -> None:
        self.dimension = dimension
        self.seed = seed

    @property
    def available(self) -> bool:
        return True

    def embed(self, text: str) -> list[float]:
        values = [0.0] * self.dimension
        for feature, weight in self._features(text):
            digest = hashlib.sha256(f"{self.seed}:{feature}".encode("utf-8")).digest()
            index = int.from_bytes(digest[:4], "big", signed=False) % self.dimension
            sign = 1.0 if digest[4] % 2 == 0 else -1.0
            values[index] += sign * weight

        norm = math.sqrt(sum(value * value for value in values))
        if norm == 0:
            return values
        return [round(value / norm, 8) for value in values]

    def embed_batch(self, texts: Sequence[str]) -> list[list[float]]:
        return [self.embed(text) for text in texts]

    def _features(self, text: str) -> list[tuple[str, float]]:
        tokens = [match.group(0).lower() for match in TOKEN_RE.finditer(text)]
        features: list[tuple[str, float]] = []
        for token in tokens:
            features.append((f"tok:{token}", 2.0))
            for ngram in self._char_ngrams(token):
                features.append((f"ng:{ngram}", 1.0))
        for left, right in zip(tokens, tokens[1:]):
            features.append((f"bi:{left}_{right}", 1.5))
        return features

    @staticmethod
    def _char_ngrams(token: str) -> list[str]:
        if len(token) <= 3:
            return [token]
        ngrams: list[str] = []
        for size in (3, 4, 5):
            if len(token) < size:
                continue
            for index in range(0, len(token) - size + 1):
                ngrams.append(token[index : index + size])
        return ngrams


class UnavailableEmbeddingProvider:
    provider_name = "unavailable"
    model_name = "unavailable"

    def __init__(self, dimension: int) -> None:
        self.dimension = dimension

    @property
    def available(self) -> bool:
        return False

    def embed(self, text: str) -> list[float]:
        raise EmbeddingUnavailableError("embedding model unavailable")

    def embed_batch(self, texts: Sequence[str]) -> list[list[float]]:
        raise EmbeddingUnavailableError("embedding model unavailable")


class Qwen3EmbeddingProvider:
    provider_name = "qwen3"

    def __init__(
        self,
        model_id: str,
        dimension: int,
        device: str,
        query_instruction: str,
        model_loader: Callable[[str, str], Any] | None = None,
    ) -> None:
        if dimension < 32 or dimension > 1024:
            raise EmbeddingConfigurationError("qwen3 embedding_dimension must be between 32 and 1024")
        self.model_name = model_id
        self.dimension = dimension
        self.device = device
        self.query_instruction = query_instruction.strip()
        self._model_loader = model_loader or self._load_sentence_transformer
        self._model: Any | None = None
        self._load_error: Exception | None = None

    @property
    def available(self) -> bool:
        try:
            self._ensure_model()
        except EmbeddingUnavailableError:
            return False
        return True

    def embed(self, text: str) -> list[float]:
        return self.embed_batch([text])[0]

    def embed_batch(self, texts: Sequence[str]) -> list[list[float]]:
        model = self._ensure_model()
        if not texts:
            return []

        encode_kwargs: dict[str, Any] = {
            "normalize_embeddings": True,
        }
        prompt = self._query_prompt()
        if prompt:
            encode_kwargs["prompt"] = prompt
        if self.device:
            encode_kwargs["device"] = self.device

        try:
            raw_vectors = model.encode(list(texts), truncate_dim=self.dimension, **encode_kwargs)
        except TypeError:
            raw_vectors = model.encode(list(texts), **encode_kwargs)

        vectors = [self._coerce_vector(vector) for vector in raw_vectors]
        if len(vectors) != len(texts):
            raise EmbeddingUnavailableError("qwen3 embedding backend returned wrong batch size")
        return vectors

    def _ensure_model(self) -> Any:
        if self._model is not None:
            return self._model
        if self._load_error is not None:
            raise EmbeddingUnavailableError(f"qwen3 embedding model unavailable: {self._load_error}") from self._load_error
        try:
            self._model = self._model_loader(self.model_name, self.device)
        except Exception as exc:  # pragma: no cover - exercised through injected failing loader
            self._load_error = exc
            raise EmbeddingUnavailableError(f"qwen3 embedding model unavailable: {exc}") from exc
        return self._model

    @staticmethod
    def _load_sentence_transformer(model_id: str, device: str) -> Any:
        try:
            from sentence_transformers import SentenceTransformer
        except Exception as exc:  # pragma: no cover - depends on optional runtime package
            raise EmbeddingUnavailableError(
                "sentence-transformers is required for embedding_mode=qwen3"
            ) from exc

        kwargs: dict[str, Any] = {}
        if device:
            kwargs["device"] = device
        return SentenceTransformer(model_id, **kwargs)

    def _query_prompt(self) -> str:
        if not self.query_instruction:
            return ""
        return f"<Instruct>: {self.query_instruction}\n<Query>: "

    def _coerce_vector(self, vector: Any) -> list[float]:
        if hasattr(vector, "tolist"):
            vector = vector.tolist()
        values = [float(value) for value in vector]
        if len(values) < self.dimension:
            raise EmbeddingUnavailableError(
                f"qwen3 embedding backend returned {len(values)} dimensions, expected at least {self.dimension}"
            )
        values = values[: self.dimension]
        return self._normalize(values)

    @staticmethod
    def _normalize(values: list[float]) -> list[float]:
        norm = math.sqrt(sum(value * value for value in values))
        if norm == 0:
            return values
        return [round(value / norm, 8) for value in values]
