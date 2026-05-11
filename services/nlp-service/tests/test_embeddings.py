import math

import pytest

from core.embeddings import (
    EmbeddingConfigurationError,
    EmbeddingUnavailableError,
    Qwen3EmbeddingProvider,
)


class RecordingBackend:
    def __init__(self) -> None:
        self.calls: list[dict[str, object]] = []

    def encode(self, texts: list[str], **kwargs: object) -> list[list[float]]:
        self.calls.append({"texts": texts, **kwargs})
        return [[float(index + 1), *([0.0] * 31)] for index, _ in enumerate(texts)]


def test_qwen3_provider_embeds_batch_with_instruction_prompt_and_order() -> None:
    backend = RecordingBackend()
    provider = Qwen3EmbeddingProvider(
        model_id="Qwen/Qwen3-Embedding-0.6B",
        dimension=32,
        device="cpu",
        query_instruction="Classify support intent",
        model_loader=lambda _model_id, _device: backend,
    )

    vectors = provider.embed_batch(["первый", "второй"])

    assert len(vectors) == 2
    assert vectors[0] == [1.0, *([0.0] * 31)]
    assert vectors[1] == [1.0, *([0.0] * 31)]
    assert backend.calls == [
        {
            "texts": ["первый", "второй"],
            "truncate_dim": 32,
            "normalize_embeddings": True,
            "prompt": "<Instruct>: Classify support intent\n<Query>: ",
            "device": "cpu",
        }
    ]


def test_qwen3_provider_normalizes_vectors_after_truncation() -> None:
    class Backend:
        def encode(self, texts: list[str], **kwargs: object) -> list[list[float]]:
            return [[3.0, 4.0, 12.0, *([0.0] * 30)] for _ in texts]

    provider = Qwen3EmbeddingProvider(
        model_id="Qwen/Qwen3-Embedding-0.6B",
        dimension=32,
        device="cpu",
        query_instruction="",
        model_loader=lambda _model_id, _device: Backend(),
    )

    vector = provider.embed("текст")

    assert pytest.approx(math.sqrt(sum(value * value for value in vector))) == 1.0
    assert len(vector) == 32


def test_qwen3_provider_reports_backend_unavailable() -> None:
    provider = Qwen3EmbeddingProvider(
        model_id="Qwen/Qwen3-Embedding-0.6B",
        dimension=32,
        device="cpu",
        query_instruction="",
        model_loader=lambda _model_id, _device: (_ for _ in ()).throw(RuntimeError("missing model")),
    )

    assert provider.available is False
    with pytest.raises(EmbeddingUnavailableError):
        provider.embed("текст")


def test_qwen3_provider_rejects_dimensions_outside_model_card_range() -> None:
    with pytest.raises(EmbeddingConfigurationError):
        Qwen3EmbeddingProvider(
            model_id="Qwen/Qwen3-Embedding-0.6B",
            dimension=1025,
            device="cpu",
            query_instruction="",
        )
