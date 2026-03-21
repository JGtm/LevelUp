"""Tests unitaires pour src/auth/provider.py (couche d'authentification LevelUp).

Tous les tests moculent MSAL, aiohttp et SPNKr — pas d'appel réseau réel.
La persistance DuckDB est testée via une base :memory: ou un fichier tmp.
"""

from __future__ import annotations

import asyncio
import time
from pathlib import Path
from typing import Any
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from src.auth._msal import DeviceCodeInfo, DeviceFlowError
from src.auth.provider import (
    AuthRequiredError,
    DeviceCodePending,
    _halo_token_cache,
    complete_device_flow,
    get_halo_tokens,
    get_halo_tokens_or_raise,
    invalidate_token_cache,
    start_device_flow,
)
from src.data.sync._tokens import Tokens

# ---------------------------------------------------------------------------
# Fixtures helpers
# ---------------------------------------------------------------------------

FAKE_ACCESS_TOKEN = "fake_access_token_xyz"
FAKE_SPARTAN = "v4=FAKE_SPARTAN_TOKEN"
FAKE_CLEARANCE = "FAKE_CLEARANCE_TOKEN"
FAKE_GAMERTAG = "SpartanTest"
FAKE_XUID = "123456789012"


def _mock_msal_app(silent_token: str | None = FAKE_ACCESS_TOKEN) -> MagicMock:
    """Construit un mock d'app MSAL."""
    app = MagicMock()
    app.get_accounts.return_value = [{"username": "test@example.com"}] if silent_token else []
    if silent_token:
        app.acquire_token_silent.return_value = {"access_token": silent_token}
    else:
        app.acquire_token_silent.return_value = {"error": "no_account"}
    app.acquire_token_by_device_flow.return_value = {"access_token": FAKE_ACCESS_TOKEN}
    flow_dict = {
        "user_code": "ABCD1234",
        "verification_uri": "https://microsoft.com/devicelogin",
        "expires_in": 900,
        "message": "Entrez le code ABCD1234",
        "interval": 5,
    }
    app.initiate_device_flow.return_value = flow_dict
    return app


def _mock_cache(changed: bool = True) -> MagicMock:
    """Construit un mock de SerializableTokenCache."""
    cache = MagicMock()
    cache.has_state_changed = changed
    cache.serialize.return_value = '{"AccessToken": {}}'
    return cache


# ---------------------------------------------------------------------------
# Scénario 1 : cache process-level — retour immédiat sans appel réseau
# ---------------------------------------------------------------------------


def test_get_cached_tokens_hit(tmp_path: Path) -> None:
    """Le cache process retourne les tokens sans aucun appel réseau."""
    db_path = tmp_path / "stats.duckdb"
    tokens = Tokens(spartan_token=FAKE_SPARTAN, clearance_token=FAKE_CLEARANCE)
    key = str(db_path.resolve())

    # Pré-charger le cache manuellement
    _halo_token_cache[key] = (tokens, time.monotonic() + 3600)

    with (
        patch("src.auth.provider.load_msal_cache") as mock_load,
        patch("src.auth.provider.build_msal_app") as mock_build,
    ):
        result = asyncio.run(get_halo_tokens(db_path))

    assert result == tokens
    mock_load.assert_not_called()
    mock_build.assert_not_called()

    # Nettoyage
    _halo_token_cache.pop(key, None)


def test_invalidate_token_cache(tmp_path: Path) -> None:
    """invalidate_token_cache supprime l'entrée du cache process."""
    db_path = tmp_path / "stats.duckdb"
    tokens = Tokens(spartan_token=FAKE_SPARTAN, clearance_token=FAKE_CLEARANCE)
    key = str(db_path.resolve())
    _halo_token_cache[key] = (tokens, time.monotonic() + 3600)

    invalidate_token_cache(db_path)
    assert key not in _halo_token_cache


# ---------------------------------------------------------------------------
# Scénario 2 : MSAL silent réussit → tokens Halo retournés, cache sauvé
# ---------------------------------------------------------------------------


@patch("src.auth.provider.save_msal_cache_if_changed")
@patch("src.auth.provider.exchange_access_token_for_halo", new_callable=AsyncMock)
@patch("src.auth.provider.build_msal_app")
@patch("src.auth.provider.load_msal_cache")
def test_get_halo_tokens_silent_success(
    mock_load_cache: Any,
    mock_build_app: Any,
    mock_exchange: AsyncMock,
    mock_save: Any,
    tmp_path: Path,
) -> None:
    """MSAL silent réussit → tokens Halo obtenus, cache process mis à jour."""
    db_path = tmp_path / "stats.duckdb"
    key = str(db_path.resolve())
    _halo_token_cache.pop(key, None)  # s'assurer que le cache est vide

    cache = _mock_cache(changed=True)
    app = _mock_msal_app(silent_token=FAKE_ACCESS_TOKEN)
    mock_load_cache.return_value = cache
    mock_build_app.return_value = app
    mock_exchange.return_value = (FAKE_SPARTAN, FAKE_CLEARANCE)

    result = asyncio.run(get_halo_tokens(db_path))

    assert result.spartan_token == FAKE_SPARTAN
    assert result.clearance_token == FAKE_CLEARANCE
    mock_exchange.assert_awaited_once()
    mock_save.assert_called_once_with(db_path, cache)
    assert key in _halo_token_cache

    _halo_token_cache.pop(key, None)


# ---------------------------------------------------------------------------
# Scénario 3 : MSAL silent échoue → AuthRequiredError (côté Streamlit)
# ---------------------------------------------------------------------------


@patch("src.auth.provider.exchange_access_token_for_halo", new_callable=AsyncMock)
@patch("src.auth.provider.build_msal_app")
@patch("src.auth.provider.load_msal_cache")
def test_get_halo_tokens_or_raise_silent_fails(
    mock_load_cache: Any,
    mock_build_app: Any,
    mock_exchange: AsyncMock,
    tmp_path: Path,
) -> None:
    """get_halo_tokens_or_raise lève AuthRequiredError si silent échoue."""
    db_path = tmp_path / "stats.duckdb"
    key = str(db_path.resolve())
    _halo_token_cache.pop(key, None)

    cache = _mock_cache(changed=False)
    app = _mock_msal_app(silent_token=None)  # pas de compte → silent échoue
    mock_load_cache.return_value = cache
    mock_build_app.return_value = app

    with pytest.raises(AuthRequiredError):
        asyncio.run(get_halo_tokens_or_raise(db_path))

    mock_exchange.assert_not_awaited()


# ---------------------------------------------------------------------------
# Scénario 4 : complete_device_flow → (gamertag, xuid) + cache persisté
# ---------------------------------------------------------------------------


@patch("src.auth.provider.resolve_player_identity", new_callable=AsyncMock)
@patch("src.auth.provider.save_msal_cache_if_changed")
@patch("src.auth.provider.exchange_access_token_for_halo", new_callable=AsyncMock)
@patch("src.auth.provider.acquire_token_by_device_flow")
@patch("src.auth.provider.build_msal_app")
@patch("src.auth.provider.load_msal_cache")
def test_complete_device_flow(  # noqa: PLR0913
    mock_load_cache: Any,
    mock_build_app: Any,
    mock_acquire: Any,
    mock_exchange: AsyncMock,
    mock_save: Any,
    mock_resolve: AsyncMock,
    tmp_path: Path,
) -> None:
    """complete_device_flow: authentifie, obtient les tokens, résout l'identité."""
    db_path = tmp_path / "stats.duckdb"
    key = str(db_path.resolve())
    _halo_token_cache.pop(key, None)

    cache = _mock_cache(changed=True)
    app = _mock_msal_app()
    mock_load_cache.return_value = cache
    mock_build_app.return_value = app
    mock_acquire.return_value = FAKE_ACCESS_TOKEN
    mock_exchange.return_value = (FAKE_SPARTAN, FAKE_CLEARANCE)
    mock_resolve.return_value = (FAKE_GAMERTAG, FAKE_XUID)

    flow_info = DeviceCodeInfo(
        user_code="ABCD1234",
        verification_url="https://microsoft.com/devicelogin",
        expires_in=900,
        message="Test",
        _flow={"user_code": "ABCD1234", "interval": 5},
    )
    pending = DeviceCodePending(
        user_code=flow_info.user_code,
        verification_url=flow_info.verification_url,
        expires_in=flow_info.expires_in,
        message=flow_info.message,
        _info=flow_info,
    )

    gamertag, xuid = asyncio.run(complete_device_flow(db_path, pending))

    assert gamertag == FAKE_GAMERTAG
    assert xuid == FAKE_XUID
    mock_save.assert_called_once_with(db_path, cache)
    mock_exchange.assert_awaited_once()
    mock_resolve.assert_awaited_once_with(FAKE_SPARTAN, FAKE_CLEARANCE)

    _halo_token_cache.pop(key, None)


# ---------------------------------------------------------------------------
# Scénario 5 : cache tourné → save_msal_cache_if_changed appelé exactement 1 fois
# ---------------------------------------------------------------------------


@patch("src.auth.provider.save_msal_cache_if_changed")
@patch("src.auth.provider.exchange_access_token_for_halo", new_callable=AsyncMock)
@patch("src.auth.provider.build_msal_app")
@patch("src.auth.provider.load_msal_cache")
def test_msal_cache_saved_once_when_changed(
    mock_load_cache: Any,
    mock_build_app: Any,
    mock_exchange: AsyncMock,
    mock_save: Any,
    tmp_path: Path,
) -> None:
    """save_msal_cache_if_changed est appelé exactement une fois si has_state_changed."""
    db_path = tmp_path / "stats.duckdb"
    key = str(db_path.resolve())
    _halo_token_cache.pop(key, None)

    cache = _mock_cache(changed=True)
    app = _mock_msal_app(silent_token=FAKE_ACCESS_TOKEN)
    mock_load_cache.return_value = cache
    mock_build_app.return_value = app
    mock_exchange.return_value = (FAKE_SPARTAN, FAKE_CLEARANCE)

    asyncio.run(get_halo_tokens(db_path))

    mock_save.assert_called_once_with(db_path, cache)

    _halo_token_cache.pop(key, None)


# ---------------------------------------------------------------------------
# Scénario 6 : start_device_flow retourne DeviceCodePending valide
# ---------------------------------------------------------------------------


@patch("src.auth.provider.initiate_device_flow")
@patch("src.auth.provider.build_msal_app")
@patch("src.auth.provider.load_msal_cache")
def test_start_device_flow_returns_pending(
    mock_load_cache: Any,
    mock_build_app: Any,
    mock_initiate: Any,
    tmp_path: Path,
) -> None:
    """start_device_flow retourne un DeviceCodePending avec les bonnes infos."""
    db_path = tmp_path / "stats.duckdb"
    cache = _mock_cache()
    app = _mock_msal_app()
    mock_load_cache.return_value = cache
    mock_build_app.return_value = app

    flow_info = DeviceCodeInfo(
        user_code="ZZZZ9999",
        verification_url="https://microsoft.com/devicelogin",
        expires_in=900,
        message="Entrez le code ZZZZ9999",
        _flow={"user_code": "ZZZZ9999"},
    )
    mock_initiate.return_value = flow_info

    pending = start_device_flow(db_path)

    assert isinstance(pending, DeviceCodePending)
    assert pending.user_code == "ZZZZ9999"
    assert "devicelogin" in pending.verification_url
    assert pending._info is flow_info


# ---------------------------------------------------------------------------
# Scénario 7 : DeviceFlowError propagée correctement
# ---------------------------------------------------------------------------


@patch("src.auth.provider.build_msal_app")
@patch("src.auth.provider.load_msal_cache")
def test_start_device_flow_propagates_error(
    mock_load_cache: Any,
    mock_build_app: Any,
    tmp_path: Path,
) -> None:
    """DeviceFlowError de l'initiation se propage sans modification."""
    db_path = tmp_path / "stats.duckdb"
    cache = _mock_cache()
    app = _mock_msal_app()
    mock_load_cache.return_value = cache
    mock_build_app.return_value = app
    app.initiate_device_flow.return_value = {
        "error": "invalid_client",
        "error_description": "App non trouvée",
    }

    with pytest.raises(DeviceFlowError) as exc_info:
        start_device_flow(db_path)

    assert exc_info.value.code == "bad_client_id"
