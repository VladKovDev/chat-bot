
from pydantic import BaseModel


class Message(BaseModel):
    role: str
    text: str


class DecideRequest(BaseModel):
    state: str
    summary: str
    messages: list[Message]
