
from pydantic import BaseModel


class DecideResponse(BaseModel):
    intent: str
    state: str
    actions: list[str]


class HealthResponse(BaseModel):
    status: str
    domain_loaded: bool
