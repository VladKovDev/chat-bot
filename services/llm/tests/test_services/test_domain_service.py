import pytest

from app.core.exceptions import DomainNotLoadedError
from app.services.domain_service import DomainService


def test_load_schema():
    service = DomainService()
    service.load_schema(
        intents=["greeting", "request_operator"],
        states=["initial", "operator_requested"],
        actions=["transfer_to_operator"],
    )

    schema = service.get_schema()
    assert schema.intents == ["greeting", "request_operator"]
    assert schema.states == ["initial", "operator_requested"]
    assert schema.actions == ["transfer_to_operator"]
    assert service.is_loaded()


def test_get_schema_not_loaded():
    service = DomainService()
    with pytest.raises(DomainNotLoadedError):
        service.get_schema()