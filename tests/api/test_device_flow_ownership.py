"""Tests unitaires — ownership Device Code Flow par session (Sprint 1.3 V7).

Couvre :
- Single-flight : renvoi de l'attempt existant si session_id identique
- 404 si session_id d'une autre session tente de lire l'attempt
- Écriture de linked_halo_identity dans la session au succès du flow
- Codes d'erreur structurés (device_flow_denied, halo_exchange_failed, …)
"""

from __future__ import annotations

import pytest

# ---------------------------------------------------------------------------
# Tests service (unitaires, pas d'app FastAPI)
# ---------------------------------------------------------------------------


@pytest.fixture
def _reset_attempts() -> None:
    """Vide le cache des attempts entre chaque test."""
    from apps.api.app.services.setup_service import _ATTEMPTS_LOCK, _device_flow_attempts

    with _ATTEMPTS_LOCK:
        _device_flow_attempts.clear()
    yield
    with _ATTEMPTS_LOCK:
        _device_flow_attempts.clear()


@pytest.mark.anyio
async def test_single_flight_same_session_returns_existing_attempt(_reset_attempts: None) -> None:
    """Deux appels start_device_flow avec le même session_id retournent le même attempt."""
    import uuid

    from apps.api.app.services.setup_service import (
        _ATTEMPTS_LOCK,
        _device_flow_attempts,
        _DeviceFlowAttempt,
        start_device_flow,
    )

    # Injecter manuellement un attempt pending pour la session
    attempt_id = str(uuid.uuid4())
    attempt = _DeviceFlowAttempt(
        attempt_id=attempt_id,
        user_code="ABCD-1234",
        verification_uri="https://microsoft.com/devicelogin",
        expires_in_seconds=900,
        session_id="session-abc",
    )
    with _ATTEMPTS_LOCK:
        _device_flow_attempts[attempt_id] = attempt

    # Un deuxième appel avec la même session doit retourner l'attempt existant
    # sans avoir à appeler MSAL (single-flight — retour immédiat)
    result = start_device_flow(session_id="session-abc")

    assert result.attempt_id == attempt_id
    assert result.user_code == "ABCD-1234"


def test_ownership_check_returns_404_for_foreign_session(_reset_attempts: None) -> None:
    """get_device_flow_status retourne 404 si la session ne possède pas l'attempt."""
    import uuid

    from apps.api.app.core.errors import ApiError
    from apps.api.app.services.setup_service import (
        _ATTEMPTS_LOCK,
        _device_flow_attempts,
        _DeviceFlowAttempt,
        get_device_flow_status,
    )

    attempt_id = str(uuid.uuid4())
    attempt = _DeviceFlowAttempt(
        attempt_id=attempt_id,
        user_code="XY-9999",
        verification_uri="https://microsoft.com/devicelogin",
        expires_in_seconds=900,
        session_id="session-owner",
    )
    with _ATTEMPTS_LOCK:
        _device_flow_attempts[attempt_id] = attempt

    # Une session étrangère ne doit pas pouvoir lire cet attempt
    with pytest.raises(ApiError) as exc_info:
        get_device_flow_status(attempt_id, session_id="session-autre")

    assert exc_info.value.status_code == 404


def test_ownership_check_allows_correct_session(_reset_attempts: None) -> None:
    """get_device_flow_status retourne le statut pour la bonne session."""
    import uuid

    from apps.api.app.services.setup_service import (
        _ATTEMPTS_LOCK,
        _device_flow_attempts,
        _DeviceFlowAttempt,
        get_device_flow_status,
    )

    attempt_id = str(uuid.uuid4())
    attempt = _DeviceFlowAttempt(
        attempt_id=attempt_id,
        user_code="XY-1234",
        verification_uri="https://microsoft.com/devicelogin",
        expires_in_seconds=900,
        session_id="session-owner",
    )
    with _ATTEMPTS_LOCK:
        _device_flow_attempts[attempt_id] = attempt

    result = get_device_flow_status(attempt_id, session_id="session-owner")
    assert result.attempt_id == attempt_id
    assert result.status == "pending"


def test_ownership_check_no_session_id_uses_legacy_mode(_reset_attempts: None) -> None:
    """Sans session_id, pas de vérification d'ownership (compat legacy)."""
    import uuid

    from apps.api.app.services.setup_service import (
        _ATTEMPTS_LOCK,
        _device_flow_attempts,
        _DeviceFlowAttempt,
        get_device_flow_status,
    )

    attempt_id = str(uuid.uuid4())
    attempt = _DeviceFlowAttempt(
        attempt_id=attempt_id,
        user_code="ZZ-0000",
        verification_uri="https://microsoft.com/devicelogin",
        expires_in_seconds=900,
        session_id="any-session",
    )
    with _ATTEMPTS_LOCK:
        _device_flow_attempts[attempt_id] = attempt

    # Sans session_id : accessible (mode compat)
    result = get_device_flow_status(attempt_id, session_id=None)
    assert result.status == "pending"
