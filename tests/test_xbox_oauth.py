"""Tests unitaires pour le flux Xbox OAuth (src/ui/xbox_oauth.py).

Toutes les dépendances réseau sont mockées ; pas d'appel réel à Microsoft/Halo.
"""

from __future__ import annotations

import json
from unittest.mock import AsyncMock, patch

import pytest

# ===========================================================================
# build_xbox_auth_url
# ===========================================================================


def test_build_xbox_auth_url_contient_client_id():
    from src.ui.xbox_oauth import build_xbox_auth_url

    url = build_xbox_auth_url("MY_CLIENT_ID", "http://localhost:8501", "mystate123")
    assert "MY_CLIENT_ID" in url


def test_build_xbox_auth_url_contient_scopes():
    from src.ui.xbox_oauth import build_xbox_auth_url

    url = build_xbox_auth_url("id", "http://localhost:8501", "s")
    assert "Xboxlive.signin" in url
    assert "Xboxlive.offline_access" in url


def test_build_xbox_auth_url_contient_state():
    from src.ui.xbox_oauth import build_xbox_auth_url

    url = build_xbox_auth_url("id", "http://localhost:8501", "csrf_token_xyz")
    assert "csrf_token_xyz" in url


def test_build_xbox_auth_url_domaine_microsoft():
    from src.ui.xbox_oauth import build_xbox_auth_url

    url = build_xbox_auth_url("id", "http://localhost:8501", "s")
    assert "login.live.com" in url


# ===========================================================================
# generate_oauth_state
# ===========================================================================


def test_generate_oauth_state_longueur():
    from src.ui.xbox_oauth import generate_oauth_state

    state = generate_oauth_state()
    # token_hex(16) → 32 hex chars
    assert len(state) == 32


def test_generate_oauth_state_aleatoire():
    from src.ui.xbox_oauth import generate_oauth_state

    states = {generate_oauth_state() for _ in range(10)}
    # Tous différents (collisions quasi-impossibles)
    assert len(states) == 10


# ===========================================================================
# exchange_code_for_refresh_token
# ===========================================================================


@pytest.mark.asyncio
async def test_exchange_code_succes():
    from src.ui.xbox_oauth import exchange_code_for_refresh_token

    mock_session = AsyncMock()
    mock_resp = AsyncMock()
    mock_resp.status = 200
    mock_resp.json = AsyncMock(return_value={"refresh_token": "my_refresh_token_abc"})
    mock_session.post.return_value = mock_resp

    token = await exchange_code_for_refresh_token(
        mock_session,
        client_id="cid",
        client_secret="csecret",  # pragma: allowlist secret
        redirect_uri="http://localhost:8501",
        code="auth_code_xyz",
    )

    assert token == "my_refresh_token_abc"
    mock_session.post.assert_called_once()
    _, kwargs = mock_session.post.call_args
    assert kwargs["data"]["grant_type"] == "authorization_code"
    assert kwargs["data"]["code"] == "auth_code_xyz"


@pytest.mark.asyncio
async def test_exchange_code_erreur_http():
    from src.ui.xbox_oauth import exchange_code_for_refresh_token

    mock_session = AsyncMock()
    mock_resp = AsyncMock()
    mock_resp.status = 400
    mock_resp.json = AsyncMock(
        return_value={"error": "invalid_grant", "error_description": "Code expiré"}
    )
    mock_session.post.return_value = mock_resp

    with pytest.raises(ValueError, match="invalid_grant"):
        await exchange_code_for_refresh_token(
            mock_session,
            client_id="cid",
            client_secret="csecret",  # pragma: allowlist secret
            redirect_uri="http://localhost:8501",
            code="bad_code",
        )


@pytest.mark.asyncio
async def test_exchange_code_sans_refresh_token():
    from src.ui.xbox_oauth import exchange_code_for_refresh_token

    mock_session = AsyncMock()
    mock_resp = AsyncMock()
    mock_resp.status = 200
    mock_resp.json = AsyncMock(return_value={"access_token": "only_access"})
    mock_session.post.return_value = mock_resp

    with pytest.raises(ValueError, match="refresh_token"):
        await exchange_code_for_refresh_token(
            mock_session,
            client_id="cid",
            client_secret="csecret",  # pragma: allowlist secret
            redirect_uri="http://localhost:8501",
            code="code_without_offline_access",
        )


# ===========================================================================
# store_refresh_token / load_refresh_token
# ===========================================================================


def test_store_and_load_refresh_token(tmp_path):
    """Cycle écriture/lecture du refresh token dans une DB DuckDB temporaire."""
    from src.ui.xbox_oauth import load_refresh_token, store_refresh_token

    db_path = tmp_path / "stats.duckdb"
    # La DB n'existe pas encore — store_refresh_token doit la créer
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
    store_refresh_token(db, "token_v2")  # Mise à jour

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

    # Vérifier sync_meta existe
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
# run_xbox_oauth_callback (synchrone, mock réseau)
# ===========================================================================


def test_run_xbox_oauth_callback_succes():
    """Mock complet du flux OAuth → résultat dict avec gamertag + xuid."""
    from src.ui.xbox_oauth import run_xbox_oauth_callback

    with (
        patch(
            "src.ui.xbox_oauth.exchange_code_for_refresh_token",
            new=AsyncMock(return_value="my_refresh"),
        ),
        patch(
            "src.ui.xbox_oauth.get_spartan_tokens_from_refresh",
            new=AsyncMock(return_value=("spartan_tok", "clearance_tok")),
        ),
        patch(
            "src.ui.xbox_oauth.resolve_player_identity",
            new=AsyncMock(return_value=("SpartanC", "123456789")),
        ),
    ):
        result = run_xbox_oauth_callback(
            "auth_code_test",
            client_id="cid",
            client_secret="csecret",  # pragma: allowlist secret
            redirect_uri="http://localhost:8501",
        )

    assert result.get("gamertag") == "SpartanC"
    assert result.get("xuid") == "123456789"
    assert result.get("refresh_token") == "my_refresh"
    assert "error" not in result


def test_run_xbox_oauth_callback_erreur():
    """En cas d'exception réseau → résultat dict avec clé 'error'."""
    from src.ui.xbox_oauth import run_xbox_oauth_callback

    with patch(
        "src.ui.xbox_oauth.exchange_code_for_refresh_token",
        new=AsyncMock(side_effect=ValueError("connection refused")),
    ):
        result = run_xbox_oauth_callback(
            "bad_code",
            client_id="cid",
            client_secret="csecret",  # pragma: allowlist secret
            redirect_uri="http://localhost:8501",
        )

    assert "error" in result
    assert "connection refused" in result["error"]
