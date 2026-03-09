"""Tests unitaires pour src/ui/xbox_oauth.py — Device Code Flow & persistence.

Toutes les dépendances réseau sont mockées ; pas d'appel réel à Microsoft/Halo.
"""

from __future__ import annotations

import asyncio
import json
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

# ===========================================================================
# store_refresh_token / load_refresh_token
# ===========================================================================


def test_store_and_load_refresh_token(tmp_path):
    """Cycle écriture/lecture du refresh token dans une DB DuckDB temporaire."""
    from src.ui.xbox_oauth import load_refresh_token, store_refresh_token

    db_path = tmp_path / "stats.duckdb"
    store_refresh_token(db_path, "refresh_token_value_12345")

    loaded = load_refresh_token(db_path)
    assert loaded == "refresh_token_value_12345"


def test_load_refresh_token_db_absente(tmp_path):
    """Retourne None si le fichier DB n'existe pas."""
    from src.ui.xbox_oauth import load_refresh_token

    missing = tmp_path / "nonexistent.duckdb"
    assert load_refresh_token(missing) is None


def test_store_refresh_token_idempotent(tmp_path):
    """Écraser un token existant fonctionne sans erreur."""
    from src.ui.xbox_oauth import load_refresh_token, store_refresh_token

    db = tmp_path / "stats.duckdb"
    store_refresh_token(db, "token_v1")
    store_refresh_token(db, "token_v2")

    assert load_refresh_token(db) == "token_v2"


# ===========================================================================
# provision_player (src/app/player_provisioning.py)
# ===========================================================================


def test_create_player_db(tmp_path):
    """Crée la DB et la table sync_meta."""
    from src.app.player_provisioning import create_player_db

    db_path = create_player_db("TestPlayer", base_dir=tmp_path)

    assert db_path.exists()
    assert db_path.name == "stats.duckdb"
    assert db_path.parent.name == "TestPlayer"

    import duckdb

    conn = duckdb.connect(str(db_path))
    rows = conn.execute(
        "SELECT table_name FROM information_schema.tables WHERE table_name = 'sync_meta'"
    ).fetchall()
    conn.close()
    assert rows, "La table sync_meta doit exister"


def test_create_player_db_idempotent(tmp_path):
    """Appeler deux fois ne lève pas d'erreur."""
    from src.app.player_provisioning import create_player_db

    db1 = create_player_db("TestPlayer", base_dir=tmp_path)
    db2 = create_player_db("TestPlayer", base_dir=tmp_path)
    assert db1 == db2


def test_register_player_profile(tmp_path):
    """Enregistre un profil dans un db_profiles.json temporaire."""
    profiles_file = tmp_path / "db_profiles.json"
    profiles_file.write_text(json.dumps({"profiles": {}}), encoding="utf-8")

    db_path = tmp_path / "stats.duckdb"
    db_path.touch()

    with (
        patch("src.utils.profiles.PROFILES_PATH", str(profiles_file)),
        patch("src.utils.profiles.get_profiles_path", return_value=str(profiles_file)),
    ):
        from src.app.player_provisioning import register_player_profile

        ok = register_player_profile("TestPlayer", "1234567890", db_path)

    assert ok
    updated = json.loads(profiles_file.read_text())
    assert "TestPlayer" in updated["profiles"]
    assert updated["profiles"]["TestPlayer"]["xuid"] == "1234567890"


def test_provision_player_full(tmp_path):
    """Flux complet : DB créée + profil enregistré."""
    profiles_file = tmp_path / "db_profiles.json"
    profiles_file.write_text(json.dumps({"profiles": {}}), encoding="utf-8")

    with patch("src.utils.profiles.get_profiles_path", return_value=str(profiles_file)):
        from src.app.player_provisioning import provision_player

        db_path = provision_player("NewPlayer", "9876543210", base_dir=tmp_path)

    assert db_path.exists()
    profiles = json.loads(profiles_file.read_text())
    assert "NewPlayer" in profiles["profiles"]


# ===========================================================================
# complete_device_code_flow (Device Code Flow)
# ===========================================================================


def test_complete_device_code_flow_succes():
    """complete_device_code_flow retourne gamertag + xuid en cas de succès."""
    from src.ui.xbox_oauth import complete_device_code_flow

    with (
        patch(
            "src.ui.xbox_oauth.get_spartan_tokens_from_refresh",
            new=AsyncMock(return_value=("spartan_tok", "clearance_tok")),
        ),
        patch(
            "src.ui.xbox_oauth.resolve_player_identity",
            new=AsyncMock(return_value=("SpartanK117", "111222333")),
        ),
    ):
        result = complete_device_code_flow(
            "my_refresh_token",
            "12345678-1234-1234-1234-123456789abc",
        )

    assert result.get("gamertag") == "SpartanK117"
    assert result.get("xuid") == "111222333"
    assert result.get("refresh_token") == "my_refresh_token"
    assert "error" not in result


def test_complete_device_code_flow_erreur_api():
    """En cas d'exception API, retourne un dict avec clé 'error'."""
    from src.ui.xbox_oauth import complete_device_code_flow

    with patch(
        "src.ui.xbox_oauth.get_spartan_tokens_from_refresh",
        new=AsyncMock(side_effect=RuntimeError("xbox service unavailable")),
    ):
        result = complete_device_code_flow(
            "my_refresh_token",
            "12345678-1234-1234-1234-123456789abc",
        )

    assert "error" in result
    assert "xbox service unavailable" in result["error"]


# ===========================================================================
# get_spartan_tokens_from_refresh — couverture lignes manquantes
# ===========================================================================


def test_get_spartan_tokens_retourne_tuple():
    """Happy path : retourne (spartan_token, clearance_token)."""
    from src.ui.xbox_oauth import get_spartan_tokens_from_refresh

    mock_tokens = MagicMock()
    mock_tokens.spartan_token = "st_abc"
    mock_tokens.clearance_token = "ct_xyz"

    with patch(
        "src.data.sync._auth.refresh_halo_tokens",
        new=AsyncMock(return_value=mock_tokens),
    ):
        result = asyncio.run(
            get_spartan_tokens_from_refresh(
                None,
                client_id="cid",
                client_secret="",
                redirect_uri="",
                refresh_token="rt",
            )
        )

    assert result == ("st_abc", "ct_xyz")


# ===========================================================================
# resolve_player_identity — couverture lignes manquantes
# ===========================================================================


def _make_aiohttp_mocks(profile_resp: dict, profile_resp2: dict | None = None):
    """Construit les mocks aiohttp + spnkr pour resolve_player_identity."""
    mock_resp1 = AsyncMock()
    mock_resp1.json = AsyncMock(return_value=profile_resp)

    mock_resp2 = AsyncMock()
    mock_resp2.json = AsyncMock(return_value=profile_resp2 or {})

    mock_client = MagicMock()
    mock_client.profile.get_current_user = AsyncMock(return_value=mock_resp1)
    mock_client.profile.get_current_player = AsyncMock(return_value=mock_resp2)

    mock_session = AsyncMock()

    mock_aiohttp = MagicMock()
    mock_aiohttp.ClientTimeout = MagicMock(return_value=MagicMock())
    mock_aiohttp.ClientSession = MagicMock(return_value=mock_session)

    mock_spnkr = MagicMock()
    mock_spnkr.HaloInfiniteClient = MagicMock(return_value=mock_client)

    return mock_aiohttp, mock_spnkr, mock_client


def test_resolve_player_identity_via_get_current_user():
    """Résout via get_current_user (chemin principal)."""
    from src.ui.xbox_oauth import resolve_player_identity

    mock_aiohttp, mock_spnkr, mock_client = _make_aiohttp_mocks(
        {"xuid": "111", "gamertag": "SpartanHz"}
    )

    with patch.dict("sys.modules", {"aiohttp": mock_aiohttp, "spnkr": mock_spnkr}):
        result = asyncio.run(resolve_player_identity("sp_tok", "cl_tok"))

    assert result == ("SpartanHz", "111")
    mock_client.profile.get_current_user.assert_called_once()


def test_resolve_player_identity_fallback_get_current_player():
    """get_current_user échoue → fallback sur get_current_player."""
    from src.ui.xbox_oauth import resolve_player_identity

    mock_aiohttp, mock_spnkr, mock_client = _make_aiohttp_mocks(
        {"xuid": "", "gamertag": ""},  # mauvaise réponse → pas de xuid/gamertag
        {"xuid": "222", "gamertag": "SpartanFB"},  # fallback ok
    )

    with patch.dict("sys.modules", {"aiohttp": mock_aiohttp, "spnkr": mock_spnkr}):
        result = asyncio.run(resolve_player_identity("sp_tok", "cl_tok"))

    assert result == ("SpartanFB", "222")


def test_resolve_player_identity_both_fail_raises_valueerror():
    """Les deux endpoints échouent → ValueError."""
    from src.ui.xbox_oauth import resolve_player_identity

    mock_aiohttp, mock_spnkr, mock_client = _make_aiohttp_mocks({})
    mock_client.profile.get_current_user = AsyncMock(side_effect=Exception("erreur1"))
    mock_client.profile.get_current_player = AsyncMock(side_effect=Exception("erreur2"))

    with (
        patch.dict("sys.modules", {"aiohttp": mock_aiohttp, "spnkr": mock_spnkr}),
        pytest.raises(ValueError, match="gamertag"),
    ):
        asyncio.run(resolve_player_identity("sp_tok", "cl_tok"))


# ===========================================================================
# load_refresh_token — chemin exception
# ===========================================================================


def test_load_refresh_token_exception_retourne_none(tmp_path):
    """Une exception lors de la lecture retourne None (robustesse)."""
    from src.ui.xbox_oauth import load_refresh_token

    db = tmp_path / "stats.duckdb"
    db.touch()  # Le fichier existe, mais la lecture va planter

    with patch("src.utils.db.duckdb_read_only", side_effect=Exception("connexion impossible")):
        result = load_refresh_token(db)

    assert result is None


# ===========================================================================
# complete_device_code_flow — chemin ThreadPoolExecutor (RuntimeError)
# ===========================================================================


def test_complete_device_code_flow_runtime_error_bascule_executor():
    """RuntimeError sur asyncio.run → bascule sur ThreadPoolExecutor."""
    from src.ui.xbox_oauth import complete_device_code_flow

    call_count = [0]
    original_run = asyncio.run

    def _patched_run(coro):
        call_count[0] += 1
        if call_count[0] == 1:
            raise RuntimeError("This event loop is already running.")
        return original_run(coro)

    with (
        patch(
            "src.ui.xbox_oauth.get_spartan_tokens_from_refresh",
            new=AsyncMock(return_value=("sp_tok", "cl_tok")),
        ),
        patch(
            "src.ui.xbox_oauth.resolve_player_identity",
            new=AsyncMock(return_value=("GTfoo", "xuidfoo")),
        ),
        patch.object(asyncio, "run", side_effect=_patched_run),
    ):
        result = complete_device_code_flow("rt123", "client_id_abc")

    assert result.get("gamertag") == "GTfoo"
    assert result.get("xuid") == "xuidfoo"
    assert call_count[0] == 2
