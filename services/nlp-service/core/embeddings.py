import hashlib
import math


class EmbeddingUnavailableError(RuntimeError):
    pass


class FakeEmbeddingProvider:
    model_name = "fake-hash-embedding-v1"

    def __init__(self, dimension: int, seed: str) -> None:
        self.dimension = dimension
        self.seed = seed

    @property
    def available(self) -> bool:
        return True

    def embed(self, text: str) -> list[float]:
        values: list[float] = []
        counter = 0
        while len(values) < self.dimension:
            digest = hashlib.sha256(f"{self.seed}:{counter}:{text}".encode("utf-8")).digest()
            for offset in range(0, len(digest), 4):
                if len(values) >= self.dimension:
                    break
                raw = int.from_bytes(digest[offset : offset + 4], "big", signed=False)
                values.append((raw / 2**31) - 1.0)
            counter += 1

        norm = math.sqrt(sum(value * value for value in values))
        if norm == 0:
            return values
        return [round(value / norm, 8) for value in values]


class UnavailableEmbeddingProvider:
    model_name = "unavailable"

    def __init__(self, dimension: int) -> None:
        self.dimension = dimension

    @property
    def available(self) -> bool:
        return False

    def embed(self, text: str) -> list[float]:
        raise EmbeddingUnavailableError("embedding model unavailable")
