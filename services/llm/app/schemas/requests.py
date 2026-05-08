from pydantic import BaseModel, Field
from typing import List


class Message(BaseModel):
    role: str
    text: str


class ConfigRequest(BaseModel):
    intents: List[str]
    states: List[str]
    actions: List[str]


class DecideRequest(BaseModel):
    state: str
    summary: str
    messages: List[Message]