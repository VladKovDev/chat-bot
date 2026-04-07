import logging
from functools import lru_cache

import pymorphy3

logger = logging.getLogger(__name__)


class Lemmatizer:
    """Wraps pymorphy3.MorphAnalyzer with per-token LRU caching.

    pymorphy3.MorphAnalyzer is thread-safe after construction and
    expensive to initialise — instantiate once and reuse.
    """

    def __init__(self, cache_size: int) -> None:
        logger.info("initialising morphological analyser", extra={"cache_size": cache_size})
        self._analyser = pymorphy3.MorphAnalyzer()
        self._lemmatize_token = lru_cache(maxsize=cache_size)(self._lemmatize_single)
        logger.info("morphological analyser ready")

    def lemmatize(self, tokens: list[str]) -> list[str]:
        return [self._lemmatize_token(token) for token in tokens]

    def _lemmatize_single(self, token: str) -> str:
        parses = self._analyser.parse(token)
        if not parses:
            return token
        return parses[0].normal_form

    @property
    def cache_info(self) -> dict[str, int]:
        info = self._lemmatize_token.cache_info()
        return {
            "hits": info.hits,
            "misses": info.misses,
            "maxsize": info.maxsize,
            "currsize": info.currsize,
        }