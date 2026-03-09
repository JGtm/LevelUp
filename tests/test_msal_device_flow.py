"""Tests unitaires pour src/utils/msal_device_flow.py."""

from __future__ import annotations

import logging
from unittest.mock import MagicMock, patch

import pytest

from src.utils.msal_device_flow import (
    DeviceCodeResult,
    DeviceFlowError,
    acquire_token_blocking,
    initiate_device_flow,
)

# ──────────────────────────────────────────────────────────────────────────────
# Fixtures
# ──────────────────────────────────────────────────────────────────────────────

_VALID_CLIENT_ID = "12345678-1234-1234-1234-123456789abc"
_FAKE_FLOW = {
    "user_code": "ABCD1234",
    "verification_uri": "https://microsoft.com/devicelogin",
    "expires_in": 900,
    "message": "Go to https://microsoft.com/devicelogin and enter ABCD1234",
}


def _make_mock_app(flow: dict | None = None, token_result: dict | None = None) -> MagicMock:
    """Construit un mock de PublicClientApplication."""
    app = MagicMock()
    app.initiate_device_flow.return_value = flow or _FAKE_FLOW
    app.acquire_token_by_device_flow.return_value = token_result or {"refresh_token": "tok_xyz"}
    return app


# ──────────────────────────────────────────────────────────────────────────────
# initiate_device_flow
# ──────────────────────────────────────────────────────────────────────────────


def test_initiate_flow_retourne_device_code_result():
    """initiate_device_flow doit retourner un DeviceCodeResult correct."""
    mock_app = _make_mock_app()
    with patch("src.utils.msal_device_flow.msal") as mock_msal:
        mock_msal.PublicClientApplication.return_value = mock_app
        result, app = initiate_device_flow(_VALID_CLIENT_ID)

    assert isinstance(result, DeviceCodeResult)
    assert result.user_code == "ABCD1234"
    assert result.verification_url == "https://microsoft.com/devicelogin"
    assert result.expires_in == 900
    assert app is mock_app


def test_initiate_flow_client_id_vide_leve_erreur():
    """Un client_id vide doit lever DeviceFlowError(code='bad_client_id')."""
    with patch("src.utils.msal_device_flow.msal"), pytest.raises(DeviceFlowError) as exc_info:
        initiate_device_flow("")

    assert exc_info.value.code == "bad_client_id"


def test_initiate_flow_erreur_dans_flow_leve_erreur():
    """Si initiate_device_flow renvoie une clé error, DeviceFlowError est levée."""
    bad_flow = dict(_FAKE_FLOW, error="invalid_client", error_description="Bad client")
    mock_app = _make_mock_app(flow=bad_flow)
    with patch("src.utils.msal_device_flow.msal") as mock_msal:
        mock_msal.PublicClientApplication.return_value = mock_app
        with pytest.raises(DeviceFlowError) as exc_info:
            initiate_device_flow(_VALID_CLIENT_ID)

    assert exc_info.value.code == "bad_client_id"


# ──────────────────────────────────────────────────────────────────────────────
# acquire_token_blocking
# ──────────────────────────────────────────────────────────────────────────────


def test_acquire_token_succes_premier_poll():
    """acquire_token_blocking doit retourner le refresh_token si la réponse est ok."""
    mock_app = _make_mock_app(token_result={"refresh_token": "tok_xyz_123"})
    token = acquire_token_blocking(mock_app, _FAKE_FLOW)
    assert token == "tok_xyz_123"
    mock_app.acquire_token_by_device_flow.assert_called_once()


def test_acquire_token_timeout_leve_erreur():
    """expired_token dans la réponse doit lever DeviceFlowError(code='timeout')."""
    mock_app = _make_mock_app(
        token_result={"error": "expired_token", "error_description": "Expired"}
    )
    with pytest.raises(DeviceFlowError) as exc_info:
        acquire_token_blocking(mock_app, _FAKE_FLOW)
    assert exc_info.value.code == "timeout"


def test_acquire_token_decline_leve_erreur():
    """authorization_declined doit lever DeviceFlowError(code='declined')."""
    mock_app = _make_mock_app(
        token_result={"error": "authorization_declined", "error_description": "User declined"}
    )
    with pytest.raises(DeviceFlowError) as exc_info:
        acquire_token_blocking(mock_app, _FAKE_FLOW)
    assert exc_info.value.code == "declined"


def test_refresh_token_absent_dans_reponse_leve_erreur():
    """Une réponse sans refresh_token ni error doit lever DeviceFlowError(code='no_refresh_token')."""
    mock_app = _make_mock_app(token_result={"access_token": "at_only"})
    with pytest.raises(DeviceFlowError) as exc_info:
        acquire_token_blocking(mock_app, _FAKE_FLOW)
    assert exc_info.value.code == "no_refresh_token"


# ──────────────────────────────────────────────────────────────────────────────
# msal absent
# ──────────────────────────────────────────────────────────────────────────────


def test_msal_absent_leve_erreur_claire():
    """Si msal n'est pas installé, DeviceFlowError(code='msal_unavailable') doit être levée."""
    with patch("src.utils.msal_device_flow.msal", None), pytest.raises(DeviceFlowError) as exc_info:
        initiate_device_flow(_VALID_CLIENT_ID)
    assert exc_info.value.code == "msal_unavailable"


# ──────────────────────────────────────────────────────────────────────────────
# Logging
# ──────────────────────────────────────────────────────────────────────────────


def test_logging_info_sur_succes(caplog):
    """Un succès doit logger INFO avec le user_code et la longueur du token."""
    mock_app = _make_mock_app(token_result={"refresh_token": "tok_log_test"})
    with caplog.at_level(logging.INFO, logger="src.utils.msal_device_flow"):
        acquire_token_blocking(mock_app, _FAKE_FLOW)

    messages = " ".join(caplog.messages)
    assert "refresh_token" in messages.lower() or "token" in messages.lower()


def test_logging_warning_sur_timeout(caplog):
    """Un timeout doit logger WARNING avec le user_code."""
    mock_app = _make_mock_app(
        token_result={"error": "expired_token", "error_description": "Expired"}
    )
    with (
        caplog.at_level(logging.WARNING, logger="src.utils.msal_device_flow"),
        pytest.raises(DeviceFlowError),
    ):
        acquire_token_blocking(mock_app, _FAKE_FLOW)

    messages = " ".join(caplog.messages)
    assert "timeout" in messages.lower() or "expir" in messages.lower()


# ────────────────────────────────────────────────────────────────────────────────
# Cas additionnels : code_expired, unknown, DeviceFlowError.detail, logging init
# ────────────────────────────────────────────────────────────────────────────────


def test_code_expired_leve_timeout():
    """code_expired (alt. de expired_token) doit aussi lever code='timeout'."""
    mock_app = _make_mock_app(
        token_result={"error": "code_expired", "error_description": "Code expired"}
    )
    with pytest.raises(DeviceFlowError) as exc_info:
        acquire_token_blocking(mock_app, _FAKE_FLOW)
    assert exc_info.value.code == "timeout"


def test_unknown_error_leve_erreur_generique():
    """Une erreur MSAL inconnue doit lever DeviceFlowError(code='unknown')."""
    mock_app = _make_mock_app(
        token_result={"error": "some_unknown_msal_error", "error_description": "?"}
    )
    with pytest.raises(DeviceFlowError) as exc_info:
        acquire_token_blocking(mock_app, _FAKE_FLOW)
    assert exc_info.value.code == "unknown"
    assert "some_unknown_msal_error" in exc_info.value.detail


def test_device_flow_error_attributs():
    """DeviceFlowError expose .code et .detail (contrat public)."""
    err = DeviceFlowError("declined", "L'utilisateur a refusé.")
    assert err.code == "declined"
    assert err.detail == "L'utilisateur a refusé."
    assert str(err) == "L'utilisateur a refusé."


def test_logging_error_sur_init_failure(caplog):
    """Un échec d'initiation (error dans le flow) doit logger ERROR."""
    bad_flow = dict(_FAKE_FLOW, error="invalid_client", error_description="Bad")
    mock_app = _make_mock_app(flow=bad_flow)
    with (
        patch("src.utils.msal_device_flow.msal") as mock_msal,
        caplog.at_level(logging.ERROR, logger="src.utils.msal_device_flow"),
        pytest.raises(DeviceFlowError),
    ):
        mock_msal.PublicClientApplication.return_value = mock_app
        initiate_device_flow(_VALID_CLIENT_ID)

    assert any("invalid_client" in m or "échec" in m.lower() for m in caplog.messages)
