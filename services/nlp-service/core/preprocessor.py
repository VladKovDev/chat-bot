import re
from functools import lru_cache

try:
    import pymorphy3
except ImportError:  # pragma: no cover - exercised only without optional dependency.
    pymorphy3 = None  # type: ignore[assignment]


TOKEN_RE = re.compile(r"[0-9A-Za-zА-Яа-яЁё]+")


class RussianPreprocessor:
    def __init__(self, cache_size: int) -> None:
        self._analyzer = pymorphy3.MorphAnalyzer() if pymorphy3 is not None else None
        self._lemmatize_token = lru_cache(maxsize=cache_size)(self._lemmatize_single)

    @property
    def model_name(self) -> str:
        return "pymorphy3" if self._analyzer is not None else "fallback"

    @property
    def available(self) -> bool:
        return self._analyzer is not None

    def preprocess(self, text: str) -> dict[str, object]:
        tokens = [match.group(0).lower() for match in TOKEN_RE.finditer(text)]
        lemmas = [self._lemmatize_token(token) for token in tokens]
        return {
            "normalized_text": " ".join(tokens),
            "tokens": tokens,
            "lemmas": lemmas,
            "model": self.model_name,
        }

    def _lemmatize_single(self, token: str) -> str:
        if self._analyzer is None:
            return token
        parses = self._analyzer.parse(token)
        if not parses:
            return token
        return parses[0].normal_form
