from pydantic import BaseModel


class DomainSchema(BaseModel):
    intents: list[str]
    states: list[str]
    actions: list[str]

    model_config = {"frozen": True}