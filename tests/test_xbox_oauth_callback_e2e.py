"""Tests e2e pour le flux Xbox OAuth (Device Code) et le cycle de vie des tokens.

Simule les flux sans import de streamlit_app.py.
Toutes les dépendances réseau sont mockées.
"""

from __future__ import annotations

import json
from pathlib import Path
from unittest.mock import AsyncMock, patch

# =============================================================================
# Provisionnement joueur (comportement idempotent)
# =============================================================================


class TestXboxOauthE2EFlow:
    """Tests e2e du provisionnement joueur."""

    def test_provisionnement_idempotent(self, tmp_path: Path) -> None:
        """Provisionner deux fois le même joueur ne crée pas de doublon."""
        profiles_file = tmp_path / "db_profiles.json"
        profiles_file.write_text(json.dumps({"profiles": {}}), encoding="utf-8")

        from src.app.player_provisioning import provision_player

        with patch(
            "src.utils.profiles.get_profiles_path",
            return_value=str(profiles_file),
        ):
            db1 = provision_player("IdempotentGT", "111", base_dir=tmp_path)
            db2 = provision_player("IdempotentGT", "111", base_dir=tmp_path)

        assert db1 == db2
        profiles = json.loads(profiles_file.read_text())
        assert len(profiles["profiles"]) == 1


# =============================================================================
# Tests Device Code e2e
# =============================================================================


class TestDeviceCodeE2E:
    """Tests e2e du flux Device Code (MSAL) : token → identité Xbox → DB."""

    def test_complete_flow_token_vers_joueur(self, tmp_path: Path) -> None:
        """Flux complet : refresh_token → gamertag + xuid → provisionnement."""
        from src.ui.xbox_oauth import complete_device_code_flow

        with (
            patch(
                "src.ui.xbox_oauth.get_spartan_tokens_from_refresh",
                new=AsyncMock(return_value=("spartan_dc", "clearance_dc")),
            ),
            patch(
                "src.ui.xbox_oauth.resolve_player_identity",
                new=AsyncMock(return_value=("DCPlayer", "77665544")),
            ),
        ):
            result = complete_device_code_flow(
                "dc_refresh_token_xyz",
                "12345678-1234-1234-1234-123456789abc",
            )

        assert "error" not in result
        assert result["gamertag"] == "DCPlayer"
        assert result["xuid"] == "77665544"
        assert result["refresh_token"] == "dc_refresh_token_xyz"

        # Provisionner et stocker le token
        profiles_file = tmp_path / "db_profiles.json"
        profiles_file.write_text(json.dumps({"profiles": {}}), encoding="utf-8")

        from src.app.player_provisioning import provision_player
        from src.ui.xbox_oauth import load_refresh_token, store_refresh_token

        with patch("src.utils.profiles.get_profiles_path", return_value=str(profiles_file)):
            db_path = provision_player(result["gamertag"], result["xuid"], base_dir=tmp_path)

        store_refresh_token(db_path, result["refresh_token"])

        assert load_refresh_token(db_path) == "dc_refresh_token_xyz"
        profiles = json.loads(profiles_file.read_text())
        assert "DCPlayer" in profiles["profiles"]

    def test_erreur_api_halo_retourne_dict_erreur(self) -> None:
        """Erreur API Halo → dict avec clé 'error' (pas d'exception propa gée)."""
        from src.ui.xbox_oauth import complete_device_code_flow

        with patch(
            "src.ui.xbox_oauth.get_spartan_tokens_from_refresh",
            new=AsyncMock(side_effect=RuntimeError("Halo API unavailable")),
        ):
            result = complete_device_code_flow(
                "some_token",
                "12345678-1234-1234-1234-123456789abc",
            )

        assert "error" in result
        assert "Halo" in result.get("error", "")


# =============================================================================
# Tests refresh token cycle complet
# =============================================================================


class TestRefreshTokenLifecycle:
    """Tests du cycle de vie complet du refresh token."""

    def test_store_load_update_cycle(self, tmp_path: Path) -> None:
        """Cycle complet : store → load → update → load."""
        from src.ui.xbox_oauth import load_refresh_token, store_refresh_token

        db = tmp_path / "stats.duckdb"

        store_refresh_token(db, "token_v1")
        assert load_refresh_token(db) == "token_v1"

        store_refresh_token(db, "token_v2")
        assert load_refresh_token(db) == "token_v2"

    def test_load_db_sans_table(self, tmp_path: Path) -> None:
        """Load depuis une DB sans table sync_meta retourne None."""
        import duckdb

        db = tmp_path / "empty.duckdb"
        conn = duckdb.connect(str(db))
        conn.close()

        from src.ui.xbox_oauth import load_refresh_token

        assert load_refresh_token(db) is None

    def test_load_db_absente_retourne_none(self, tmp_path: Path) -> None:
        """load_refresh_token ne lève pas si le fichier n'existe pas."""
        from src.ui.xbox_oauth import load_refresh_token

        missing = tmp_path / "does_not_exist.duckdb"
        assert load_refresh_token(missing) is None
