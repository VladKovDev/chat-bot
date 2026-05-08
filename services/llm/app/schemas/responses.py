from pydantic import BaseModel
from typing import List, Optional


class DecideResponse(BaseModel):
    intent: str
    state: str
    actions: List[str]


class ConfigResponse(BaseModel):
    status: str
    message: str


class HealthResponse(BaseModel):
    status: str
    domain_loaded: bool