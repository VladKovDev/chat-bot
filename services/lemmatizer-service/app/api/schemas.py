from pydantic import BaseModel, Field


class LemmatizeRequest(BaseModel):
    tokens: list[str] = Field(min_length=1)


class LemmatizeResponse(BaseModel):
    lemmas: list[str]

class HealthResponse(BaseModel):
    status: str
    cache: dict[str, int]