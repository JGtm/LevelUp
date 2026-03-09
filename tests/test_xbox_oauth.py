"""Tests unitaires pour src/ui/xbox_oauth.py — Device Code Flow & persistence.

Toutes les dépendances réseau sont mockées ; pas d'appel réel à Microsoft/Halo.
"""

from __future__ import annotations

import json
from unittest.mock import AsyncMock, patch

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
