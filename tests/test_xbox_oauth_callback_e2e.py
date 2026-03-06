"""Tests e2e pour le flux Xbox OAuth callback.

Vérifie le flux complet : code échangé → joueur provisionné → token stocké.
Simule le comportement de _handle_xbox_oauth_callback sans importer
streamlit_app.py (module protégé par st.runtime.exists()).

Toutes les dépendances réseau sont mockées.
"""

from __future__ import annotations

import json
from pathlib import Path
from unittest.mock import AsyncMock, patch

# =============================================================================
# Tests e2e du flux Xbox OAuth (provision complète)
# =============================================================================


class TestXboxOauthE2EFlow:
    """Tests e2e simulant le flux complet callback → provisionnement."""

    def test_flux_complet_code_to_player(self, tmp_path: Path) -> None:
        """Flux complet : code OAuth → tokens → gamertag → DB + profil."""
        from src.ui.xbox_oauth import run_xbox_oauth_callback

        with (
            patch(
                "src.ui.xbox_oauth.exchange_code_for_refresh_token",
                new=AsyncMock(return_value="refresh_e2e"),
            ),
            patch(
                "src.ui.xbox_oauth.get_spartan_tokens_from_refresh",
                new=AsyncMock(return_value=("spartan_e2e", "clearance_e2e")),
            ),
            patch(
                "src.ui.xbox_oauth.resolve_player_identity",
                new=AsyncMock(return_value=("E2ESpartan", "99887766")),
            ),
        ):
            result = run_xbox_oauth_callback(
                "auth_code_e2e",
                client_id="12345678-1234-1234-1234-123456789abc",
                client_secret="secret_value_test",  # pragma: allowlist secret
                redirect_uri="http://localhost:8501",
            )

        assert "error" not in result
        assert result["gamertag"] == "E2ESpartan"
        assert result["xuid"] == "99887766"
        assert result["refresh_token"] == "refresh_e2e"

        # Phase 2 : provisionner le joueur
        from src.app.player_provisioning import provision_player

        profiles_file = tmp_path / "db_profiles.json"
        profiles_file.write_text(json.dumps({"profiles": {}}), encoding="utf-8")

        with patch(
            "src.utils.profiles.get_profiles_path",
            return_value=str(profiles_file),
        ):
            db_path = provision_player(
                result["gamertag"],
                result["xuid"],
                base_dir=tmp_path,
            )

        assert db_path.exists()
        assert db_path.name == "stats.duckdb"

        profiles = json.loads(profiles_file.read_text())
        assert "E2ESpartan" in profiles["profiles"]
        assert profiles["profiles"]["E2ESpartan"]["xuid"] == "99887766"

        # Phase 3 : stocker le refresh token dans la DB
        from src.ui.xbox_oauth import load_refresh_token, store_refresh_token

        store_refresh_token(db_path, result["refresh_token"])
        loaded = load_refresh_token(db_path)
        assert loaded == "refresh_e2e"

    def test_flux_erreur_code_invalide(self) -> None:
        """Un code invalide retourne un dict avec 'error'."""
        from src.ui.xbox_oauth import run_xbox_oauth_callback

        with patch(
            "src.ui.xbox_oauth.exchange_code_for_refresh_token",
            new=AsyncMock(side_effect=ValueError("invalid_grant: code expired")),
        ):
            result = run_xbox_oauth_callback(
                "expired_code",
                client_id="12345678-1234-1234-1234-123456789abc",
                client_secret="secret_value",  # pragma: allowlist secret
                redirect_uri="http://localhost:8501",
            )

        assert "error" in result
        assert "invalid_grant" in result["error"]

    def test_flux_erreur_identity_resolution(self) -> None:
        """Erreur lors de la résolution de l'identité joueur."""
        from src.ui.xbox_oauth import run_xbox_oauth_callback

        with (
            patch(
                "src.ui.xbox_oauth.exchange_code_for_refresh_token",
                new=AsyncMock(return_value="valid_refresh"),
            ),
            patch(
                "src.ui.xbox_oauth.get_spartan_tokens_from_refresh",
                new=AsyncMock(return_value=("sp", "cl")),
            ),
            patch(
                "src.ui.xbox_oauth.resolve_player_identity",
                new=AsyncMock(side_effect=ValueError("API Halo indisponible")),
            ),
        ):
            result = run_xbox_oauth_callback(
                "code_ok",
                client_id="12345678-1234-1234-1234-123456789abc",
                client_secret="secret_value",  # pragma: allowlist secret
                redirect_uri="http://localhost:8501",
            )

        assert "error" in result
        assert "Halo" in result["error"]

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
# Tests CSRF
# =============================================================================


class TestCsrfProtection:
    """Tests de la protection CSRF du flux OAuth."""

    def test_state_aleatoire_unique(self) -> None:
        """Chaque appel à generate_oauth_state() produit une valeur unique."""
        from src.ui.xbox_oauth import generate_oauth_state

        states = [generate_oauth_state() for _ in range(100)]
        assert len(set(states)) == 100

    def test_state_longueur_suffisante(self) -> None:
        """Le state CSRF a au moins 32 caractères hex (128 bits)."""
        from src.ui.xbox_oauth import generate_oauth_state

        state = generate_oauth_state()
        assert len(state) >= 32

    def test_state_inclus_dans_url(self) -> None:
        """Le state est inclus dans l'URL d'autorisation."""
        from src.ui.xbox_oauth import build_xbox_auth_url

        url = build_xbox_auth_url("cid", "http://localhost:8501", "my_csrf_token")
        assert "my_csrf_token" in url


# =============================================================================
# Tests refresh token cycle complet
# =============================================================================


class TestRefreshTokenLifecycle:
    """Tests du cycle de vie complet du refresh token."""

    def test_store_load_update_cycle(self, tmp_path: Path) -> None:
        """Cycle complet : store → load → update → load."""
        from src.ui.xbox_oauth import load_refresh_token, store_refresh_token

        db = tmp_path / "stats.duckdb"

        # 1. Store initial
        store_refresh_token(db, "token_v1")
        assert load_refresh_token(db) == "token_v1"

        # 2. Update
        store_refresh_token(db, "token_v2")
        assert load_refresh_token(db) == "token_v2"

        # 3. Valeur avec espaces
        store_refresh_token(db, "  token_v3  ")
        loaded = load_refresh_token(db)
        assert loaded == "token_v3"  # Doit être trimé au load

    def test_load_db_sans_table(self, tmp_path: Path) -> None:
        """Load depuis une DB sans table sync_meta retourne None."""
        import duckdb

        db = tmp_path / "empty.duckdb"
        conn = duckdb.connect(str(db))
        conn.close()

        from src.ui.xbox_oauth import load_refresh_token

        assert load_refresh_token(db) is None
