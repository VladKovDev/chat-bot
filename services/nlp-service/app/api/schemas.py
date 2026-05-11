from typing import Annotated

from pydantic import BaseModel, ConfigDict, Field, StringConstraints


TextField = Annotated[str, StringConstraints(strip_whitespace=True, min_length=1, max_length=4000)]


class StrictModel(BaseModel):
    model_config = ConfigDict(extra="forbid")


class HealthResponse(BaseModel):
    status: str


class ReadyResponse(BaseModel):
    status: str
    provider: str
    model: str
    dimension: int
    lemmatizer_model: str


class PreprocessRequest(StrictModel):
    text: TextField


class PreprocessResponse(BaseModel):
    normalized_text: str
    tokens: list[str]
    lemmas: list[str]
    model: str


class EmbedRequest(StrictModel):
    text: TextField


class EmbedResponse(BaseModel):
    embedding: list[float]
    model: str
    dimension: int


class BatchEmbedRequest(StrictModel):
    texts: list[TextField] = Field(min_length=1, max_length=64)


class BatchEmbeddingItem(BaseModel):
    index: int
    embedding: list[float]


class BatchEmbedResponse(BaseModel):
    items: list[BatchEmbeddingItem]
    model: str
    dimension: int
