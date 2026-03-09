"""Tests pour src/ui/pages/setup_wizard_logic.py — Logique du wizard de config."""

from __future__ import annotations

import json
from pathlib import Path
from unittest.mock import patch

from src.ui.pages.setup_wizard_logic import (
    SetupStatus,
    create_player_profile,
    get_setup_status,
    get_sync_command,
    get_token_script_path,
    save_dc_credentials,
    validate_azure_credentials,
    validate_dc_credentials,
    validate_gamertag,
)
from src.utils.auth import AuthStatus

# =============================================================================
# SetupStatus
# =============================================================================


class TestSetupStatus:
    """Tests du dataclass SetupStatus."""

    def test_needs_setup_no_credentials(self) -> None:
        status = SetupStatus(
            auth=AuthStatus(has_client_id=False, has_client_secret=False),
            has_players=True,
            player_count=1,
        )
        assert status.needs_setup is True

    def test_needs_setup_no_players(self) -> None:
        status = SetupStatus(
            auth=AuthStatus(has_client_id=True, has_client_secret=True),
            has_players=False,
            player_count=0,
        )
        assert status.needs_setup is True

    def test_no_setup_needed(self) -> None:
        status = SetupStatus(
            auth=AuthStatus(has_client_id=True, has_client_secret=True),
            has_players=True,
            player_count=1,
        )
        assert status.needs_setup is False

    def test_current_step_1(self) -> None:
        status = SetupStatus(
            auth=AuthStatus(has_client_id=False, has_client_secret=False),
            has_players=False,
            player_count=0,
        )
        assert status.current_step == 1

    def test_current_step_2(self) -> None:
        status = SetupStatus(
            auth=AuthStatus(
                has_client_id=True,
                has_client_secret=True,
                has_refresh_token=False,
            ),
            has_players=False,
            player_count=0,
        )
        assert status.current_step == 2

    def test_current_step_3(self) -> None:
        status = SetupStatus(
            auth=AuthStatus(
                has_client_id=True,
                has_client_secret=True,
                has_refresh_token=True,
            ),
            has_players=False,
            player_count=0,
        )
        assert status.current_step == 3

    def test_current_step_0_configured(self) -> None:
        status = SetupStatus(
            auth=AuthStatus(
                has_client_id=True,
                has_client_secret=True,
                has_refresh_token=True,
            ),
            has_players=True,
            player_count=2,
        )
        assert status.current_step == 0


# =============================================================================
# validate_azure_credentials
# =============================================================================


class TestValidateAzureCredentials:
    """Tests de validate_azure_credentials()."""

    def test_valid_credentials(self) -> None:
        errors = validate_azure_credentials(
            "12345678-1234-1234-1234-123456789abc",
            "a_long_secret_value_here_123",
        )
        assert errors == []

    def test_empty_client_id(self) -> None:
        errors = validate_azure_credentials("", "secret")
        assert any("Client ID" in e for e in errors)

    def test_invalid_uuid_format(self) -> None:
        errors = validate_azure_credentials("not-a-uuid", "secret_long_enough")
        assert any("UUID" in e for e in errors)

    def test_empty_client_secret(self) -> None:
        errors = validate_azure_credentials("12345678-1234-1234-1234-123456789abc", "")
        assert any("Client Secret" in e and "requis" in e for e in errors)

    def test_short_client_secret(self) -> None:
        errors = validate_azure_credentials("12345678-1234-1234-1234-123456789abc", "short")
        assert any("trop court" in e for e in errors)

    def test_whitespace_trimmed(self) -> None:
        errors = validate_azure_credentials(
            "  12345678-1234-1234-1234-123456789abc  ",
            "  a_long_secret_value_here_123  ",
        )
        assert errors == []


# =============================================================================
# validate_gamertag
# =============================================================================


class TestValidateGamertag:
    """Tests de validate_gamertag()."""

    def test_valid_gamertag(self) -> None:
        assert validate_gamertag("MonGamertag") == []

    def test_valid_gamertag_with_spaces(self) -> None:
        assert validate_gamertag("Mon Gamertag") == []

    def test_valid_gamertag_with_dash(self) -> None:
        assert validate_gamertag("Mon-GT") == []

    def test_empty_gamertag(self) -> None:
        errors = validate_gamertag("")
        assert any("requis" in e for e in errors)

    def test_whitespace_only(self) -> None:
        errors = validate_gamertag("   ")
        assert len(errors) > 0


# =============================================================================
# create_player_profile
# =============================================================================


class TestCreatePlayerProfile:
    """Tests de create_player_profile()."""

    def test_create_new_profile(self, tmp_path: Path) -> None:
        db_profiles = tmp_path / "db_profiles.json"
        players_dir = tmp_path / "players"

        with (
            patch("src.ui.pages.setup_wizard_logic._DB_PROFILES_PATH", db_profiles),
            patch("src.ui.pages.setup_wizard_logic.PLAYERS_DIR", players_dir),
        ):
            key = create_player_profile("TestPlayer")

        assert key == "TestPlayer"
        assert (players_dir / "TestPlayer").exists()

        data = json.loads(db_profiles.read_text(encoding="utf-8"))
        assert "TestPlayer" in data["profiles"]
        assert data["profiles"]["TestPlayer"]["waypoint_player"] == "TestPlayer"

    def test_create_profile_with_xuid(self, tmp_path: Path) -> None:
        db_profiles = tmp_path / "db_profiles.json"
        players_dir = tmp_path / "players"

        with (
            patch("src.ui.pages.setup_wizard_logic._DB_PROFILES_PATH", db_profiles),
            patch("src.ui.pages.setup_wizard_logic.PLAYERS_DIR", players_dir),
        ):
            create_player_profile("TestPlayer", xuid="xuid(12345)")

        data = json.loads(db_profiles.read_text(encoding="utf-8"))
        assert data["profiles"]["TestPlayer"]["xuid"] == "xuid(12345)"

    def test_update_existing_profile(self, tmp_path: Path) -> None:
        db_profiles = tmp_path / "db_profiles.json"
        players_dir = tmp_path / "players"
        db_profiles.write_text(
            json.dumps(
                {
                    "version": "2.1",
                    "profiles": {
                        "ExistingPlayer": {
                            "db_path": "data/players/ExistingPlayer/stats.duckdb",
                            "waypoint_player": "ExistingPlayer",
                            "custom_field": "keep_me",
                        }
                    },
                }
            ),
            encoding="utf-8",
        )

        with (
            patch("src.ui.pages.setup_wizard_logic._DB_PROFILES_PATH", db_profiles),
            patch("src.ui.pages.setup_wizard_logic.PLAYERS_DIR", players_dir),
        ):
            key = create_player_profile("ExistingPlayer")

        data = json.loads(db_profiles.read_text(encoding="utf-8"))
        profile = data["profiles"]["ExistingPlayer"]
        # Nouveau champ mergé
        assert profile["waypoint_player"] == "ExistingPlayer"
        # Ancien champ préservé
        assert profile["custom_field"] == "keep_me"
        assert key == "ExistingPlayer"


# =============================================================================
# get_sync_command
# =============================================================================


class TestGetSyncCommand:
    """Tests de get_sync_command()."""

    def test_default_command(self) -> None:
        cmd = get_sync_command("MyGT")
        assert cmd == [
            "python",
            "scripts/sync.py",
            "--add-player",
            "MyGT",
            "--full",
            "--max-matches",
            "200",
        ]

    def test_custom_max_matches(self) -> None:
        cmd = get_sync_command("MyGT", max_matches=500)
        assert "--max-matches" in cmd
        assert "500" in cmd

    def test_gamertag_trimmed(self) -> None:
        cmd = get_sync_command("  MyGT  ")
        assert "MyGT" in cmd


# =============================================================================
# get_token_script_path
# =============================================================================


class TestGetTokenScriptPath:
    """Tests de get_token_script_path()."""

    def test_returns_path(self) -> None:
        path = get_token_script_path()
        assert path.name == "spnkr_get_refresh_token.py"
        assert "scripts" in str(path)


# =============================================================================
# get_setup_status
# =============================================================================


class TestGetSetupStatus:
    """Tests de get_setup_status()."""

    def test_returns_setup_status(self, tmp_path: Path) -> None:
        players_dir = tmp_path / "players"
        players_dir.mkdir()

        with (
            patch("src.ui.pages.setup_wizard_logic.PLAYERS_DIR", players_dir),
            patch("src.ui.pages.setup_wizard_logic.get_auth_status") as mock_auth,
        ):
            mock_auth.return_value = AuthStatus(
                has_client_id=True,
                has_client_secret=True,
            )
            status = get_setup_status()

        assert isinstance(status, SetupStatus)
        assert status.has_players is False
        assert status.player_count == 0

    def test_detects_players(self, tmp_path: Path) -> None:
        players_dir = tmp_path / "players"
        player1 = players_dir / "Player1"
        player1.mkdir(parents=True)
        (player1 / "stats.duckdb").write_bytes(b"")

        with (
            patch("src.ui.pages.setup_wizard_logic.PLAYERS_DIR", players_dir),
            patch("src.ui.pages.setup_wizard_logic.get_auth_status") as mock_auth,
        ):
            mock_auth.return_value = AuthStatus(
                has_client_id=True,
                has_client_secret=True,
            )
            status = get_setup_status()

        assert status.has_players is True
        assert status.player_count == 1

    def test_counts_multiple_players(self, tmp_path: Path) -> None:
        """Compte correctement plusieurs joueurs."""
        players_dir = tmp_path / "players"
        for name in ("Player1", "Player2", "Player3"):
            p = players_dir / name
            p.mkdir(parents=True)
            (p / "stats.duckdb").write_bytes(b"")

        with (
            patch("src.ui.pages.setup_wizard_logic.PLAYERS_DIR", players_dir),
            patch("src.ui.pages.setup_wizard_logic.get_auth_status") as mock_auth,
        ):
            mock_auth.return_value = AuthStatus(
                has_client_id=True,
                has_client_secret=True,
            )
            status = get_setup_status()

        assert status.player_count == 3

    def test_ignores_dirs_without_db(self, tmp_path: Path) -> None:
        """Ignore les dossiers joueurs sans stats.duckdb."""
        players_dir = tmp_path / "players"
        (players_dir / "Player1").mkdir(parents=True)  # Pas de .duckdb
        p2 = players_dir / "Player2"
        p2.mkdir(parents=True)
        (p2 / "stats.duckdb").write_bytes(b"")

        with (
            patch("src.ui.pages.setup_wizard_logic.PLAYERS_DIR", players_dir),
            patch("src.ui.pages.setup_wizard_logic.get_auth_status") as mock_auth,
        ):
            mock_auth.return_value = AuthStatus(
                has_client_id=True,
                has_client_secret=True,
            )
            status = get_setup_status()

        assert status.player_count == 1


# =============================================================================
# Edge cases create_player_profile
# =============================================================================


class TestCreatePlayerProfileEdgeCases:
    """Tests edge cases de create_player_profile."""

    def test_case_insensitive_lookup(self, tmp_path: Path) -> None:
        """Retrouve un profil existant indépendamment de la casse."""
        db_profiles = tmp_path / "db_profiles.json"
        players_dir = tmp_path / "players"
        db_profiles.write_text(
            json.dumps(
                {
                    "version": "2.1",
                    "profiles": {
                        "MySpartan": {
                            "db_path": "data/players/MySpartan/stats.duckdb",
                            "waypoint_player": "MySpartan",
                        }
                    },
                }
            ),
            encoding="utf-8",
        )

        with (
            patch("src.ui.pages.setup_wizard_logic._DB_PROFILES_PATH", db_profiles),
            patch("src.ui.pages.setup_wizard_logic.PLAYERS_DIR", players_dir),
        ):
            key = create_player_profile("MYSPARTAN")

        # Doit réutiliser la clé existante (pas créer un doublon)
        assert key == "MySpartan"
        data = json.loads(db_profiles.read_text(encoding="utf-8"))
        assert len(data["profiles"]) == 1

    def test_whitespace_trimmed(self, tmp_path: Path) -> None:
        """Le gamertag est nettoyé des espaces avant/après."""
        db_profiles = tmp_path / "db_profiles.json"
        players_dir = tmp_path / "players"

        with (
            patch("src.ui.pages.setup_wizard_logic._DB_PROFILES_PATH", db_profiles),
            patch("src.ui.pages.setup_wizard_logic.PLAYERS_DIR", players_dir),
        ):
            key = create_player_profile("  MyGT  ")

        assert key == "MyGT"
        assert (players_dir / "MyGT").exists()

    def test_creates_default_structure(self, tmp_path: Path) -> None:
        """Crée db_profiles.json avec la structure par défaut si absent."""
        db_profiles = tmp_path / "db_profiles.json"
        players_dir = tmp_path / "players"

        with (
            patch("src.ui.pages.setup_wizard_logic._DB_PROFILES_PATH", db_profiles),
            patch("src.ui.pages.setup_wizard_logic.PLAYERS_DIR", players_dir),
        ):
            create_player_profile("NewPlayer")

        data = json.loads(db_profiles.read_text(encoding="utf-8"))
        assert data["version"] == "2.1"
        assert "warehouse_path" in data
        assert "NewPlayer" in data["profiles"]


# =============================================================================
# save_azure_credentials
# =============================================================================


# =============================================================================
# validate_dc_credentials
# =============================================================================


class TestValidateDcCredentials:
    """Tests de validate_dc_credentials() — public client, pas de secret."""

    def test_client_id_valide(self) -> None:
        errors = validate_dc_credentials("12345678-1234-1234-1234-123456789abc")
        assert errors == []

    def test_client_id_vide(self) -> None:
        errors = validate_dc_credentials("")
        assert any("requis" in e or "Client ID" in e for e in errors)

    def test_client_id_format_invalide(self) -> None:
        errors = validate_dc_credentials("not-a-uuid")
        assert any("UUID" in e for e in errors)

    def test_whitespace_trimme(self) -> None:
        """Espaces autour d'un UUID valide → aucune erreur."""
        errors = validate_dc_credentials("  12345678-1234-1234-1234-123456789abc  ")
        assert errors == []

    def test_pas_de_secret_requis(self) -> None:
        """validate_dc_credentials ne demande qu'un seul argument (pas de secret)."""
        import inspect

        sig = inspect.signature(validate_dc_credentials)
        assert len(sig.parameters) == 1


# =============================================================================
# save_dc_credentials
# =============================================================================


class TestSaveDcCredentials:
    """Tests de save_dc_credentials() — sauvegarde client_id sans secret."""

    def test_updates_environ(self, tmp_path: Path) -> None:
        """Vérifie que os.environ est mis à jour et write_env_local appelé."""
        import os

        with patch("src.ui.pages.setup_wizard_logic.write_env_local") as mock_write:
            save_dc_credentials("  12345678-1234-1234-1234-123456789abc  ")

        assert os.environ.get("SPNKR_AZURE_CLIENT_ID") == "12345678-1234-1234-1234-123456789abc"
        mock_write.assert_called_once_with(
            {"SPNKR_AZURE_CLIENT_ID": "12345678-1234-1234-1234-123456789abc"}
        )

    def test_secret_non_ecrit(self, tmp_path: Path) -> None:
        """save_dc_credentials n'écrit jamais SPNKR_AZURE_CLIENT_SECRET."""
        import os

        os.environ.pop("SPNKR_AZURE_CLIENT_SECRET", None)

        with patch("src.ui.pages.setup_wizard_logic.write_env_local") as mock_write:
            save_dc_credentials("12345678-1234-1234-1234-123456789abc")

        written_keys = list(mock_write.call_args[0][0].keys())
        assert "SPNKR_AZURE_CLIENT_SECRET" not in written_keys


# =============================================================================
# save_azure_credentials
# =============================================================================


class TestSaveAzureCredentials:
    """Tests de save_azure_credentials."""

    def test_updates_environ(self, tmp_path: Path) -> None:
        """Vérifie que os.environ est mis à jour après sauvegarde."""
        import os

        env_file = tmp_path / ".env.local"

        with (
            patch("src.utils.auth._env_local_path", return_value=env_file),
            patch("src.ui.pages.setup_wizard_logic.write_env_local") as _mock_write,
        ):
            from src.ui.pages.setup_wizard_logic import save_azure_credentials

            save_azure_credentials(
                "  12345678-1234-1234-1234-123456789abc  ",
                "  my_secret  ",
                "  https://localhost  ",
            )

        # Vérifie que les valeurs sont trimées dans os.environ
        assert os.environ.get("SPNKR_AZURE_CLIENT_ID") == "12345678-1234-1234-1234-123456789abc"
        assert os.environ.get("SPNKR_AZURE_CLIENT_SECRET") == "my_secret"
        assert os.environ.get("SPNKR_AZURE_REDIRECT_URI") == "https://localhost"
