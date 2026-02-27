"""Tests pour le mécanisme de tokens per-player (v5.3).

Couvre :
- api_client.py : _normalize_gamertag_for_env, get_player_token_env_key,
  get_tokens_for_player
- engine.py : sync_career_rank skip silencieux quand token absent,
  _save_career_rank avec spartan_id
- migrations.py : add_spartan_id_to_career_progression
- profile_api_tokens.py : _normalize_gamertag_for_env, get_tokens avec gamertag
"""

from __future__ import annotations

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

# =============================================================================
# api_client.py — helpers normalize / key / get_tokens_for_player
# =============================================================================


class TestNormalizeGamertag:
    """Tests pour _normalize_gamertag_for_env."""

    def test_simple_gamertag(self):
        from src.data.sync.api_client import _normalize_gamertag_for_env

        assert _normalize_gamertag_for_env("SpartanC") == "SPARTANC"

    def test_gamertag_with_spaces(self):
        from src.data.sync.api_client import _normalize_gamertag_for_env

        assert _normalize_gamertag_for_env("Mon GT 2") == "MON_GT_2"

    def test_gamertag_with_special_chars(self):
        from src.data.sync.api_client import _normalize_gamertag_for_env

        assert _normalize_gamertag_for_env("Spartan#42") == "SPARTAN_42"

    def test_gamertag_already_upper(self):
        from src.data.sync.api_client import _normalize_gamertag_for_env

        assert _normalize_gamertag_for_env("SPARTANC") == "SPARTANC"

    def test_gamertag_with_leading_trailing_spaces(self):
        from src.data.sync.api_client import _normalize_gamertag_for_env

        assert _normalize_gamertag_for_env("  SpartanC  ") == "SPARTANC"

    def test_gamertag_with_hyphens(self):
        from src.data.sync.api_client import _normalize_gamertag_for_env

        assert _normalize_gamertag_for_env("My-Tag") == "MY_TAG"

    def test_gamertag_numbers(self):
        from src.data.sync.api_client import _normalize_gamertag_for_env

        assert _normalize_gamertag_for_env("Player123") == "PLAYER123"


class TestGetPlayerTokenEnvKey:
    """Tests pour get_player_token_env_key."""

    def test_simple_gamertag(self):
        from src.data.sync.api_client import get_player_token_env_key

        assert get_player_token_env_key("SpartanC") == "SPNKR_OAUTH_REFRESH_TOKEN_SPARTANC"

    def test_gamertag_with_spaces(self):
        from src.data.sync.api_client import get_player_token_env_key

        assert get_player_token_env_key("Mon GT 2") == "SPNKR_OAUTH_REFRESH_TOKEN_MON_GT_2"

    def test_gamertag_with_special_chars(self):
        from src.data.sync.api_client import get_player_token_env_key

        assert get_player_token_env_key("Spartan#1") == "SPNKR_OAUTH_REFRESH_TOKEN_SPARTAN_1"


class TestGetTokensForPlayer:
    """Tests pour get_tokens_for_player (async)."""

    @pytest.mark.asyncio
    async def test_returns_none_when_env_var_absent(self, monkeypatch):
        """Retourne None si la variable joueur est absente de l'env."""
        from src.data.sync.api_client import get_tokens_for_player

        monkeypatch.setattr("src.data.sync.api_client._load_dotenv_if_present", lambda: None)
        monkeypatch.delenv("SPNKR_OAUTH_REFRESH_TOKEN_SPARTANC", raising=False)

        result = await get_tokens_for_player("SpartanC")
        assert result is None

    @pytest.mark.asyncio
    async def test_returns_none_when_env_var_empty(self, monkeypatch):
        """Retourne None si la variable joueur est vide."""
        from src.data.sync.api_client import get_tokens_for_player

        monkeypatch.setattr("src.data.sync.api_client._load_dotenv_if_present", lambda: None)
        monkeypatch.setenv("SPNKR_OAUTH_REFRESH_TOKEN_SPARTANC", "")

        result = await get_tokens_for_player("SpartanC")
        assert result is None

        monkeypatch.delenv("SPNKR_OAUTH_REFRESH_TOKEN_SPARTANC", raising=False)

    @pytest.mark.asyncio
    async def test_returns_none_when_azure_creds_missing(self, monkeypatch):
        """Retourne None + warning si token joueur présent mais Azure creds absents."""
        from src.data.sync.api_client import get_tokens_for_player

        monkeypatch.setattr("src.data.sync.api_client._load_dotenv_if_present", lambda: None)
        monkeypatch.setenv("SPNKR_OAUTH_REFRESH_TOKEN_SPARTANC", "fake_refresh_token")
        monkeypatch.delenv("SPNKR_AZURE_CLIENT_ID", raising=False)
        monkeypatch.delenv("SPNKR_AZURE_CLIENT_SECRET", raising=False)

        result = await get_tokens_for_player("SpartanC")
        assert result is None

        monkeypatch.delenv("SPNKR_OAUTH_REFRESH_TOKEN_SPARTANC", raising=False)

    @pytest.mark.asyncio
    async def test_returns_tokens_when_all_present(self, monkeypatch):
        """Retourne Tokens quand token joueur + Azure creds sont présents."""
        from src.data.sync.api_client import Tokens, get_tokens_for_player

        monkeypatch.setattr("src.data.sync.api_client._load_dotenv_if_present", lambda: None)
        monkeypatch.setenv("SPNKR_OAUTH_REFRESH_TOKEN_SPARTANC", "player_refresh_token")
        monkeypatch.setenv("SPNKR_AZURE_CLIENT_ID", "client_id")
        monkeypatch.setenv("SPNKR_AZURE_CLIENT_SECRET", "client_secret")

        expected_tokens = Tokens(spartan_token="spartan_123", clearance_token="clearance_abc")
        with patch(
            "src.data.sync.api_client._get_tokens_via_oauth",
            new=AsyncMock(return_value=expected_tokens),
        ) as mock_oauth:
            result = await get_tokens_for_player("SpartanC")

        assert result == expected_tokens
        mock_oauth.assert_awaited_once_with(
            "client_id",
            "client_secret",
            "https://localhost",
            "player_refresh_token",
        )

        monkeypatch.delenv("SPNKR_OAUTH_REFRESH_TOKEN_SPARTANC", raising=False)
        monkeypatch.delenv("SPNKR_AZURE_CLIENT_ID", raising=False)
        monkeypatch.delenv("SPNKR_AZURE_CLIENT_SECRET", raising=False)

    @pytest.mark.asyncio
    async def test_uses_custom_redirect_uri(self, monkeypatch):
        """Utilise SPNKR_AZURE_REDIRECT_URI si défini."""
        from src.data.sync.api_client import Tokens, get_tokens_for_player

        monkeypatch.setattr("src.data.sync.api_client._load_dotenv_if_present", lambda: None)
        monkeypatch.setenv("SPNKR_OAUTH_REFRESH_TOKEN_SPARTANC", "player_token")
        monkeypatch.setenv("SPNKR_AZURE_CLIENT_ID", "cid")
        monkeypatch.setenv("SPNKR_AZURE_CLIENT_SECRET", "csecret")
        monkeypatch.setenv("SPNKR_AZURE_REDIRECT_URI", "https://custom.redirect")

        fake_tokens = Tokens(spartan_token="st", clearance_token="ct")
        with patch(
            "src.data.sync.api_client._get_tokens_via_oauth",
            new=AsyncMock(return_value=fake_tokens),
        ) as mock_oauth:
            await get_tokens_for_player("SpartanC")

        _, _, redirect, _ = mock_oauth.call_args.args
        assert redirect == "https://custom.redirect"

        for key in [
            "SPNKR_OAUTH_REFRESH_TOKEN_SPARTANC",
            "SPNKR_AZURE_CLIENT_ID",
            "SPNKR_AZURE_CLIENT_SECRET",
            "SPNKR_AZURE_REDIRECT_URI",
        ]:
            monkeypatch.delenv(key, raising=False)

    @pytest.mark.asyncio
    async def test_gamertag_spaces_normalized(self, monkeypatch):
        """Le gamertag est normalisé avant de chercher l'env var."""
        from src.data.sync.api_client import Tokens, get_tokens_for_player

        monkeypatch.setattr("src.data.sync.api_client._load_dotenv_if_present", lambda: None)
        # Gamertag "Mon GT" → clé "SPNKR_OAUTH_REFRESH_TOKEN_MON_GT"
        monkeypatch.setenv("SPNKR_OAUTH_REFRESH_TOKEN_MON_GT", "player_token")
        monkeypatch.setenv("SPNKR_AZURE_CLIENT_ID", "cid")
        monkeypatch.setenv("SPNKR_AZURE_CLIENT_SECRET", "csecret")

        fake_tokens = Tokens(spartan_token="st", clearance_token="ct")
        with patch(
            "src.data.sync.api_client._get_tokens_via_oauth",
            new=AsyncMock(return_value=fake_tokens),
        ):
            result = await get_tokens_for_player("Mon GT")

        assert result == fake_tokens

        for key in [
            "SPNKR_OAUTH_REFRESH_TOKEN_MON_GT",
            "SPNKR_AZURE_CLIENT_ID",
            "SPNKR_AZURE_CLIENT_SECRET",
        ]:
            monkeypatch.delenv(key, raising=False)


# =============================================================================
# engine.py — sync_career_rank skip silencieux + _save_career_rank spartan_id
# =============================================================================


duckdb = pytest.importorskip("duckdb")


class TestSyncCareerRankSkip:
    """sync_career_rank retourne None + warning si token joueur absent."""

    @pytest.fixture
    def engine(self, tmp_path):
        """Engine minimal avec DB temporaire."""
        from src.data.sync.engine import DuckDBSyncEngine

        db_path = tmp_path / "player" / "stats.duckdb"
        db_path.parent.mkdir(parents=True, exist_ok=True)

        meta_dir = tmp_path / "warehouse"
        meta_dir.mkdir(parents=True, exist_ok=True)
        duckdb.connect(str(meta_dir / "metadata.duckdb")).close()

        return DuckDBSyncEngine(
            player_db_path=db_path,
            xuid="2535423456789",
            gamertag="TestPlayer",
            metadata_db_path=meta_dir / "metadata.duckdb",
        )

    @pytest.mark.asyncio
    async def test_skips_when_no_player_token(self, engine, caplog, monkeypatch):
        """sync_career_rank retourne None et log un warning si token absent."""
        import logging

        with patch(
            "src.data.sync.engine.get_tokens_for_player",
            new=AsyncMock(return_value=None),
        ):
            with caplog.at_level(logging.WARNING):
                result = await engine.sync_career_rank()

        assert result is None
        # Un warning doit mentionner le gamertag
        assert any("TestPlayer" in r.message for r in caplog.records)
        engine.close()

    @pytest.mark.asyncio
    async def test_skips_warning_mentions_env_key(self, engine, caplog, monkeypatch):
        """Le warning inclut le nom de la variable d'env à configurer."""
        import logging

        with patch(
            "src.data.sync.engine.get_tokens_for_player",
            new=AsyncMock(return_value=None),
        ):
            with caplog.at_level(logging.WARNING):
                await engine.sync_career_rank()

        # Le nom de la clé env doit apparaître dans les logs
        all_messages = " ".join(r.message for r in caplog.records)
        assert "SPNKR_OAUTH_REFRESH_TOKEN_TESTPLAYER" in all_messages
        engine.close()

    @pytest.mark.asyncio
    async def test_proceeds_when_token_present(self, engine):
        """sync_career_rank appelle l'API si token joueur présent."""
        from src.data.sync.api_client import Tokens
        from src.data.sync.models import CareerRankData

        fake_tokens = Tokens(spartan_token="st", clearance_token="ct")
        fake_career = CareerRankData(
            xuid="2535423456789",
            current_rank=50,
            current_rank_name="Silver 5",
            current_rank_tier="Silver",
            current_xp=5000,
            xp_for_next_rank=10000,
            xp_total=100000,
        )

        mock_client_ctx = MagicMock()
        mock_client_ctx.__aenter__ = AsyncMock(return_value=mock_client_ctx)
        mock_client_ctx.__aexit__ = AsyncMock(return_value=False)
        mock_client_ctx.get_career_rank_progression = AsyncMock(return_value=fake_career)

        with (
            patch(
                "src.data.sync.engine.get_tokens_for_player",
                new=AsyncMock(return_value=fake_tokens),
            ),
            patch(
                "src.data.sync.engine.SPNKrAPIClient",
                return_value=mock_client_ctx,
            ),
        ):
            result = await engine.sync_career_rank()

        assert result is not None
        assert result.current_rank == 50
        engine.close()


class TestSaveCareerRankSpartanId:
    """_save_career_rank persiste spartan_id dans career_progression."""

    @pytest.fixture
    def engine(self, tmp_path):
        from src.data.sync.engine import DuckDBSyncEngine

        db_path = tmp_path / "player" / "stats.duckdb"
        db_path.parent.mkdir(parents=True, exist_ok=True)
        meta_dir = tmp_path / "warehouse"
        meta_dir.mkdir(parents=True, exist_ok=True)
        duckdb.connect(str(meta_dir / "metadata.duckdb")).close()

        return DuckDBSyncEngine(
            player_db_path=db_path,
            xuid="2535423456789",
            gamertag="TestPlayer",
            metadata_db_path=meta_dir / "metadata.duckdb",
        )

    def test_spartan_id_saved(self, engine):
        """spartan_id est inséré dans career_progression."""
        from src.data.sync.models import CareerRankData

        data = CareerRankData(
            xuid="2535423456789",
            current_rank=100,
            current_rank_name="Gold 5",
            current_rank_tier="Gold",
            current_xp=8000,
            xp_for_next_rank=12000,
            xp_total=300000,
            spartan_id="AB12",
        )

        engine._save_career_rank(data)
        conn = engine._get_connection()
        row = conn.execute(
            "SELECT spartan_id FROM career_progression WHERE xuid = ? LIMIT 1",
            ("2535423456789",),
        ).fetchone()

        assert row is not None
        assert row[0] == "AB12"
        engine.close()

    def test_spartan_id_none_saved(self, engine):
        """spartan_id = NULL si non fourni (champ optionnel)."""
        from src.data.sync.models import CareerRankData

        data = CareerRankData(
            xuid="2535423456789",
            current_rank=10,
            current_rank_name="Bronze 2",
            current_rank_tier="Bronze",
            current_xp=1000,
            xp_for_next_rank=5000,
            xp_total=10000,
            spartan_id=None,
        )

        engine._save_career_rank(data)
        conn = engine._get_connection()
        row = conn.execute(
            "SELECT spartan_id FROM career_progression WHERE xuid = ? LIMIT 1",
            ("2535423456789",),
        ).fetchone()

        assert row is not None
        assert row[0] is None
        engine.close()

    def test_get_career_rank_history_includes_spartan_id(self, engine):
        """get_career_rank_history retourne le champ spartan_id."""
        from src.data.sync.models import CareerRankData

        data = CareerRankData(
            xuid="2535423456789",
            current_rank=200,
            current_rank_name="Diamond 3",
            current_rank_tier="Diamond",
            current_xp=12000,
            xp_for_next_rank=15000,
            xp_total=700000,
            spartan_id="XY99",
        )

        engine._save_career_rank(data)
        history = engine.get_career_rank_history(limit=1)

        assert len(history) == 1
        assert "spartan_id" in history[0]
        assert history[0]["spartan_id"] == "XY99"
        engine.close()


# =============================================================================
# migrations.py — add_spartan_id_to_career_progression
# =============================================================================


class TestAddSpartanIdMigration:
    """Tests pour la migration add_spartan_id_to_career_progression."""

    def test_adds_column_when_missing(self, tmp_path):
        """La migration ajoute spartan_id si la colonne est absente."""
        from src.data.sync.migrations import add_spartan_id_to_career_progression

        conn = duckdb.connect(str(tmp_path / "test.duckdb"))
        # Table sans spartan_id (schéma legacy)
        conn.execute("""
            CREATE TABLE career_progression (
                id INTEGER PRIMARY KEY,
                xuid VARCHAR NOT NULL,
                rank INTEGER NOT NULL,
                adornment_path VARCHAR,
                recorded_at TIMESTAMP
            )
        """)

        add_spartan_id_to_career_progression(conn)

        # Vérifier que la colonne existe maintenant
        cols = {
            row[0]
            for row in conn.execute(
                "SELECT column_name FROM information_schema.columns "
                "WHERE table_name = 'career_progression'"
            ).fetchall()
        }
        assert "spartan_id" in cols
        conn.close()

    def test_idempotent_when_column_exists(self, tmp_path):
        """La migration est idempotente si la colonne existe déjà."""
        from src.data.sync.migrations import add_spartan_id_to_career_progression

        conn = duckdb.connect(str(tmp_path / "test_idem.duckdb"))
        conn.execute("""
            CREATE TABLE career_progression (
                id INTEGER PRIMARY KEY,
                xuid VARCHAR NOT NULL,
                rank INTEGER NOT NULL,
                spartan_id VARCHAR,
                recorded_at TIMESTAMP
            )
        """)

        # Appeler deux fois — ne doit pas lever d'exception
        add_spartan_id_to_career_progression(conn)
        add_spartan_id_to_career_progression(conn)

        cols = {
            row[0]
            for row in conn.execute(
                "SELECT column_name FROM information_schema.columns "
                "WHERE table_name = 'career_progression'"
            ).fetchall()
        }
        assert "spartan_id" in cols
        conn.close()

    def test_no_op_when_table_missing(self, tmp_path):
        """La migration ne lève pas d'erreur si la table n'existe pas."""
        from src.data.sync.migrations import add_spartan_id_to_career_progression

        conn = duckdb.connect(str(tmp_path / "empty.duckdb"))
        # Pas d'exception attendue
        add_spartan_id_to_career_progression(conn)
        conn.close()

    def test_existing_data_preserved(self, tmp_path):
        """La migration ne supprime pas les données existantes."""
        from src.data.sync.migrations import add_spartan_id_to_career_progression

        conn = duckdb.connect(str(tmp_path / "test_data.duckdb"))
        conn.execute("""
            CREATE TABLE career_progression (
                id INTEGER PRIMARY KEY,
                xuid VARCHAR NOT NULL,
                rank INTEGER NOT NULL
            )
        """)
        conn.execute("INSERT INTO career_progression VALUES (1, 'xuid123', 50)")

        add_spartan_id_to_career_progression(conn)

        row = conn.execute("SELECT xuid, rank FROM career_progression WHERE id = 1").fetchone()
        assert row == ("xuid123", 50)
        conn.close()


# =============================================================================
# profile_api_tokens.py — _normalize_gamertag_for_env + get_tokens avec gamertag
# =============================================================================


class TestNormalizeGamertag_TokensModule:
    """Tests pour _normalize_gamertag_for_env dans profile_api_tokens."""

    def test_simple(self):
        from src.ui.profile_api_tokens import _normalize_gamertag_for_env

        assert _normalize_gamertag_for_env("SpartanC") == "SPARTANC"

    def test_spaces(self):
        from src.ui.profile_api_tokens import _normalize_gamertag_for_env

        assert _normalize_gamertag_for_env("Mon GT 2") == "MON_GT_2"

    def test_special_chars(self):
        from src.ui.profile_api_tokens import _normalize_gamertag_for_env

        assert _normalize_gamertag_for_env("Tag#99") == "TAG_99"


class TestGetTokensWithGamertagParam:
    """get_tokens utilise le refresh token joueur si gamertag fourni."""

    @pytest.mark.asyncio
    async def test_gamertag_token_used_over_global(self, monkeypatch):
        """Le token joueur prend priorité sur le token global."""
        from src.ui.profile_api_tokens import get_tokens

        monkeypatch.setattr("src.ui.profile_api_tokens._load_dotenv_if_present", lambda: None)
        monkeypatch.setenv("SPNKR_OAUTH_REFRESH_TOKEN_SPARTANC", "player_refresh")
        monkeypatch.setenv("SPNKR_OAUTH_REFRESH_TOKEN", "global_refresh")
        monkeypatch.setenv("SPNKR_AZURE_CLIENT_ID", "cid")
        monkeypatch.setenv("SPNKR_AZURE_CLIENT_SECRET", "csecret")

        captured_refresh = {}

        async def fake_refresh_tokens(session, app, refresh_token):
            captured_refresh["token"] = refresh_token
            mock = MagicMock()
            mock.spartan_token.token = "st_player"
            mock.clearance_token.token = "ct_player"
            return mock

        # AzureApp et refresh_player_tokens sont importés localement (dans la fonction)
        # donc on patche directement dans le module spnkr
        with (
            patch("spnkr.AzureApp", MagicMock()),
            patch(
                "spnkr.refresh_player_tokens",
                side_effect=fake_refresh_tokens,
            ),
        ):
            session = MagicMock()
            st, ct = await get_tokens(
                session,
                spartan_token=None,
                clearance_token=None,
                timeout_seconds=5,
                gamertag="SpartanC",
            )

        assert captured_refresh.get("token") == "player_refresh"

        for key in [
            "SPNKR_OAUTH_REFRESH_TOKEN_SPARTANC",
            "SPNKR_OAUTH_REFRESH_TOKEN",
            "SPNKR_AZURE_CLIENT_ID",
            "SPNKR_AZURE_CLIENT_SECRET",
        ]:
            monkeypatch.delenv(key, raising=False)

    @pytest.mark.asyncio
    async def test_falls_back_to_global_when_no_player_token(self, monkeypatch):
        """Si le token joueur est absent, utilise le token global."""
        from src.ui.profile_api_tokens import get_tokens

        monkeypatch.setattr("src.ui.profile_api_tokens._load_dotenv_if_present", lambda: None)
        monkeypatch.delenv("SPNKR_OAUTH_REFRESH_TOKEN_SPARTANC", raising=False)
        monkeypatch.setenv("SPNKR_OAUTH_REFRESH_TOKEN", "global_refresh")
        monkeypatch.setenv("SPNKR_AZURE_CLIENT_ID", "cid")
        monkeypatch.setenv("SPNKR_AZURE_CLIENT_SECRET", "csecret")

        captured_refresh = {}

        async def fake_refresh_tokens(session, app, refresh_token):
            captured_refresh["token"] = refresh_token
            mock = MagicMock()
            mock.spartan_token.token = "st_global"
            mock.clearance_token.token = "ct_global"
            return mock

        with (
            patch("spnkr.AzureApp", MagicMock()),
            patch(
                "spnkr.refresh_player_tokens",
                side_effect=fake_refresh_tokens,
            ),
        ):
            session = MagicMock()
            await get_tokens(
                session,
                spartan_token=None,
                clearance_token=None,
                timeout_seconds=5,
                gamertag="SpartanC",
            )

        assert captured_refresh.get("token") == "global_refresh"

        for key in [
            "SPNKR_OAUTH_REFRESH_TOKEN",
            "SPNKR_AZURE_CLIENT_ID",
            "SPNKR_AZURE_CLIENT_SECRET",
        ]:
            monkeypatch.delenv(key, raising=False)

    @pytest.mark.asyncio
    async def test_without_gamertag_uses_global(self, monkeypatch):
        """Sans gamertag, comportement inchangé : token global uniquement."""
        from src.ui.profile_api_tokens import get_tokens

        monkeypatch.setattr("src.ui.profile_api_tokens._load_dotenv_if_present", lambda: None)
        monkeypatch.setenv("SPNKR_OAUTH_REFRESH_TOKEN", "global_only")
        monkeypatch.setenv("SPNKR_AZURE_CLIENT_ID", "cid")
        monkeypatch.setenv("SPNKR_AZURE_CLIENT_SECRET", "csecret")

        captured_refresh = {}

        async def fake_refresh_tokens(session, app, refresh_token):
            captured_refresh["token"] = refresh_token
            mock = MagicMock()
            mock.spartan_token.token = "st"
            mock.clearance_token.token = "ct"
            return mock

        with (
            patch("spnkr.AzureApp", MagicMock()),
            patch(
                "spnkr.refresh_player_tokens",
                side_effect=fake_refresh_tokens,
            ),
        ):
            session = MagicMock()
            await get_tokens(
                session,
                spartan_token=None,
                clearance_token=None,
                timeout_seconds=5,
                # pas de gamertag
            )

        assert captured_refresh.get("token") == "global_only"

        for key in [
            "SPNKR_OAUTH_REFRESH_TOKEN",
            "SPNKR_AZURE_CLIENT_ID",
            "SPNKR_AZURE_CLIENT_SECRET",
        ]:
            monkeypatch.delenv(key, raising=False)
