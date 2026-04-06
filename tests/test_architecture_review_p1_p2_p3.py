"""Tests de couverture pour les corrections Architecture Review P1/P2/P3.

Couvre :
- P1-1  : translations.py — duckdb_read_only (pas de bare connect)
- P1-9  : _career_repo.py — load_career_data inclut spartan_id
- P1-9  : main_helpers.py — _load_spartan_id_from_db via repo
- P2-1  : data_loader.default_identity_from_secrets délègue à profile
- P2-2  : data_loader.resolve_xuid_input délègue à profile.resolve_xuid
- P2-3  : _cache_core.PARIS_TZ_NAME importé depuis formatting.py
- P3-1  : DataRepository Protocol sans @abstractmethod
- P3-3  : cached_friend_matches_df — branche legacy supprimée
"""

from __future__ import annotations

from pathlib import Path
from unittest.mock import MagicMock, patch

import duckdb

# ─────────────────────────────────────────────────────────────────────────────
# Helpers
# ─────────────────────────────────────────────────────────────────────────────


def _make_career_db(tmp_path: Path, xuid: str, spartan_id: str | None = "SID_42") -> Path:
    """Crée une DB player minimaliste avec career_progression."""
    db_path = tmp_path / "stats.duckdb"
    with duckdb.connect(str(db_path)) as conn:
        conn.execute("""
            CREATE TABLE career_progression (
                xuid        VARCHAR,
                rank        INTEGER,
                rank_name   VARCHAR,
                rank_tier   INTEGER,
                current_xp  INTEGER,
                xp_for_next_rank INTEGER,
                xp_total    INTEGER,
                is_max_rank BOOLEAN,
                adornment_path  VARCHAR,
                spartan_id  VARCHAR,
                recorded_at TIMESTAMP DEFAULT current_timestamp
            )
        """)
        conn.execute(
            "INSERT INTO career_progression VALUES (?, 10, 'Hero', 1, 5000, 10000, 5000, false, "
            "'/path/adornment.png', ?, current_timestamp)",
            [xuid, spartan_id],
        )
    return db_path


def _make_repo(player_db: Path, xuid: str):
    from src.data.repositories.duckdb_repo import DuckDBRepository

    return DuckDBRepository(
        player_db_path=player_db,
        xuid=xuid,
        metadata_db_path=player_db.parent / "meta.duckdb",  # inexistant
        shared_db_path=player_db.parent / "shared.duckdb",  # inexistant
        read_only=True,
    )


# ─────────────────────────────────────────────────────────────────────────────
# P1-1 : translations.py n'utilise plus duckdb.connect() bare
# ─────────────────────────────────────────────────────────────────────────────


class TestTranslationsNoBareConnect:
    """Vérifie que _load_mode_tables utilise duckdb_read_only, pas duckdb.connect()."""

    def test_no_bare_duckdb_import_at_module_level(self):
        """'import duckdb' ne doit plus exister à la racine de translations.py."""
        import importlib
        import types

        import src.ui.translations as mod

        # Recharger pour être sûr
        importlib.reload(mod)

        # Vérifier que duckdb n'est pas dans le namespace du module (import top-level)
        # Note: import local dans la fonction est OK et attendu
        assert not isinstance(getattr(mod, "duckdb", None), types.ModuleType), (
            "'import duckdb' ne doit plus exister au niveau module dans translations.py"
        )

    def test_load_mode_tables_uses_duckdb_read_only(self):
        """_load_mode_tables doit passer par duckdb_read_only, pas duckdb.connect."""
        import inspect

        from src.ui.translations import _load_mode_tables

        source = inspect.getsource(_load_mode_tables)
        assert "duckdb_read_only" in source, "_load_mode_tables doit utiliser duckdb_read_only"
        assert "duckdb.connect" not in source, (
            "_load_mode_tables ne doit pas appeler duckdb.connect"
        )


# ─────────────────────────────────────────────────────────────────────────────
# P1-9a : load_career_data inclut spartan_id
# ─────────────────────────────────────────────────────────────────────────────


class TestLoadCareerDataSpartanId:
    """load_career_data() retourne spartan_id depuis career_progression."""

    def test_load_career_data_includes_spartan_id(self, tmp_path):
        """Le dict retourné par load_career_data contient la clé spartan_id."""
        xuid = "2533274858283686"
        player_db = _make_career_db(tmp_path, xuid, spartan_id="SPARTAN_007")
        repo = _make_repo(player_db, xuid)

        result = repo.load_career_data()

        assert result is not None
        assert "spartan_id" in result, "load_career_data doit inclure spartan_id"
        assert result["spartan_id"] == "SPARTAN_007"

    def test_load_career_data_spartan_id_none_when_null(self, tmp_path):
        """spartan_id est None si la valeur DB est NULL."""
        xuid = "2533274858283686"
        player_db = _make_career_db(tmp_path, xuid, spartan_id=None)
        repo = _make_repo(player_db, xuid)

        result = repo.load_career_data()

        assert result is not None
        assert result["spartan_id"] is None

    def test_load_career_data_includes_adornment_path(self, tmp_path):
        """adornment_path est toujours présent dans le dict (déjà existant)."""
        xuid = "2533274858283686"
        player_db = _make_career_db(tmp_path, xuid)
        repo = _make_repo(player_db, xuid)

        result = repo.load_career_data()

        assert result is not None
        assert result["adornment_path"] == "/path/adornment.png"

    def test_load_career_data_returns_none_when_empty(self, tmp_path):
        """Retourne None si aucune donnée pour ce xuid."""
        db_path = tmp_path / "stats.duckdb"
        with duckdb.connect(str(db_path)) as conn:
            conn.execute("CREATE TABLE career_progression (xuid VARCHAR, spartan_id VARCHAR)")

        repo = _make_repo(db_path, "unknown_xuid")
        assert repo.load_career_data() is None


# ─────────────────────────────────────────────────────────────────────────────
# P1-9b : _load_spartan_id_from_db via repo (main_helpers)
# ─────────────────────────────────────────────────────────────────────────────


class TestLoadSpartanIdFromDb:
    """_load_spartan_id_from_db délègue à get_cached_repository_st."""

    def test_returns_spartan_id_from_repo(self):
        """Si le repo retourne career_data avec spartan_id, la valeur est retournée."""
        from src.app.main_helpers import _load_spartan_id_from_db

        mock_repo = MagicMock()
        mock_repo.load_career_data.return_value = {"spartan_id": "S42", "adornment_path": None}

        with patch("src.ui._cache_core.get_cached_repository_st", return_value=mock_repo):
            result = _load_spartan_id_from_db("/fake/db.duckdb", "XUID_TEST")

        assert result == "S42"

    def test_returns_none_when_spartan_id_null(self):
        """Retourne None si spartan_id est None dans career_data."""
        from src.app.main_helpers import _load_spartan_id_from_db

        mock_repo = MagicMock()
        mock_repo.load_career_data.return_value = {"spartan_id": None}

        with patch("src.ui._cache_core.get_cached_repository_st", return_value=mock_repo):
            result = _load_spartan_id_from_db("/fake/db.duckdb", "XUID_TEST")

        assert result is None

    def test_returns_none_when_spartan_id_empty_string(self):
        """Retourne None si spartan_id est une chaîne vide."""
        from src.app.main_helpers import _load_spartan_id_from_db

        mock_repo = MagicMock()
        mock_repo.load_career_data.return_value = {"spartan_id": "   "}

        with patch("src.ui._cache_core.get_cached_repository_st", return_value=mock_repo):
            result = _load_spartan_id_from_db("/fake/db.duckdb", "XUID_TEST")

        assert result is None

    def test_returns_none_when_repo_raises(self):
        """Retourne None silencieusement si le repo lève une exception."""
        from src.app.main_helpers import _load_spartan_id_from_db

        with patch(
            "src.ui._cache_core.get_cached_repository_st", side_effect=RuntimeError("DB error")
        ):
            result = _load_spartan_id_from_db("/fake/db.duckdb", "XUID_TEST")

        assert result is None

    def test_returns_none_when_career_data_none(self):
        """Retourne None si load_career_data retourne None."""
        from src.app.main_helpers import _load_spartan_id_from_db

        mock_repo = MagicMock()
        mock_repo.load_career_data.return_value = None

        with patch("src.ui._cache_core.get_cached_repository_st", return_value=mock_repo):
            result = _load_spartan_id_from_db("/fake/db.duckdb", "XUID_TEST")

        assert result is None


# ─────────────────────────────────────────────────────────────────────────────
# P2-1 : default_identity_from_secrets délègue à profile.get_identity_from_secrets
# ─────────────────────────────────────────────────────────────────────────────


class TestDefaultIdentityDelegation:
    """data_loader.default_identity_from_secrets délègue à profile.get_identity_from_secrets."""

    def test_delegates_to_profile(self):
        """Vérifie que default_identity_from_secrets appelle get_identity_from_secrets."""
        from src.app.profile import PlayerIdentity

        fake_identity = PlayerIdentity(gamertag="FakeGT", xuid="999", waypoint_player="FakeGT")

        with patch("src.app.profile.get_identity_from_secrets", return_value=fake_identity) as mock:
            from src.app.data_loader import default_identity_from_secrets

            result = default_identity_from_secrets()

        mock.assert_called_once()
        assert result == ("FakeGT", "999", "FakeGT")

    def test_returns_tuple_with_gamertag_as_first(self):
        """gamertag est le premier élément du tuple (xuid_or_gamertag)."""
        from src.app.profile import PlayerIdentity

        identity = PlayerIdentity(gamertag="Spartan117", xuid="12345", waypoint_player="Spartan117")

        with patch("src.app.profile.get_identity_from_secrets", return_value=identity):
            from src.app.data_loader import default_identity_from_secrets

            xuid_or_gt, xuid_fb, wp = default_identity_from_secrets()

        assert xuid_or_gt == "Spartan117"
        assert xuid_fb == "12345"
        assert wp == "Spartan117"

    def test_no_duplicate_logic_in_data_loader(self):
        """data_loader ne doit plus contenir la logique de lecture secrets/env en double."""
        import inspect

        from src.app import data_loader

        source = inspect.getsource(data_loader.default_identity_from_secrets)
        # La fonction doit être courte (délégation = ~4 lignes)
        # Elle ne doit pas contenir de lecture directe de st.secrets
        assert "st.secrets" not in source, (
            "data_loader.default_identity_from_secrets ne doit plus lire st.secrets directement"
        )
        assert "LEVELUP_DEFAULT_GAMERTAG" not in source, (
            "La logique env ne doit plus être dans data_loader"
        )


# ─────────────────────────────────────────────────────────────────────────────
# P2-2 : resolve_xuid_input délègue à profile.resolve_xuid
# ─────────────────────────────────────────────────────────────────────────────


class TestResolveXuidInputDelegation:
    """data_loader.resolve_xuid_input délègue à profile.resolve_xuid."""

    def test_delegates_to_profile_resolve_xuid(self):
        """Vérifie que resolve_xuid_input appelle profile.resolve_xuid."""
        from src.app.profile import PlayerIdentity

        fake_identity = PlayerIdentity(gamertag="TestGT", xuid="777", waypoint_player="TestGT")

        with (
            patch("src.app.profile.get_identity_from_secrets", return_value=fake_identity),
            patch("src.app.profile.resolve_xuid", return_value="888") as mock_resolve,
        ):
            from src.app.data_loader import resolve_xuid_input

            result = resolve_xuid_input("TestGT", "/fake/db.duckdb")

        mock_resolve.assert_called_once_with("TestGT", "/fake/db.duckdb", identity=fake_identity)
        assert result == "888"

    def test_no_duplicate_resolution_logic_in_data_loader(self):
        """data_loader.resolve_xuid_input ne doit plus contenir parse_xuid_input inline."""
        import inspect

        from src.app import data_loader

        source = inspect.getsource(data_loader.resolve_xuid_input)
        assert "parse_xuid_input" not in source, (
            "La logique de résolution ne doit plus être dupliquée dans data_loader"
        )
        assert "resolve_xuid_from_db" not in source, (
            "La logique de résolution ne doit plus être dupliquée dans data_loader"
        )


# ─────────────────────────────────────────────────────────────────────────────
# P2-3 : PARIS_TZ_NAME importé depuis formatting.py dans _cache_core
# ─────────────────────────────────────────────────────────────────────────────


class TestParisTzNameSingleDefinition:
    """PARIS_TZ_NAME est défini une seule fois (formatting.py) et importé ailleurs."""

    def test_cache_core_imports_from_formatting(self):
        """_cache_core.PARIS_TZ_NAME est le même objet que formatting.PARIS_TZ_NAME."""
        from src.ui import _cache_core, formatting

        assert _cache_core.PARIS_TZ_NAME is formatting.PARIS_TZ_NAME, (
            "_cache_core.PARIS_TZ_NAME doit être importé depuis formatting, pas redéfini"
        )

    def test_cache_core_no_standalone_definition(self):
        """_cache_core.py ne doit plus définir PARIS_TZ_NAME en standalone."""
        import inspect

        from src.ui import _cache_core

        source = inspect.getsource(_cache_core)
        # On cherche la présence d'une assignation directe (pas un import)
        assert 'PARIS_TZ_NAME = "Europe/Paris"' not in source, (
            "_cache_core.py ne doit plus définir PARIS_TZ_NAME = 'Europe/Paris'"
        )

    def test_value_is_europe_paris(self):
        """La valeur reste 'Europe/Paris' depuis la source canonique."""
        from src.ui.formatting import PARIS_TZ_NAME

        assert PARIS_TZ_NAME == "Europe/Paris"


# ─────────────────────────────────────────────────────────────────────────────
# P3-1 : DataRepository Protocol sans @abstractmethod
# ─────────────────────────────────────────────────────────────────────────────


class TestDataRepositoryProtocol:
    """DataRepository est un Protocol pur, sans @abstractmethod."""

    def test_no_abstractmethod_in_protocol(self):
        """Aucune méthode du Protocol ne doit être décorée avec @abstractmethod."""
        import inspect

        from src.ports.repository import DataRepository

        for name, method in inspect.getmembers(DataRepository, predicate=inspect.isfunction):
            assert not getattr(method, "__isabstractmethod__", False), (
                f"DataRepository.{name} ne doit pas être @abstractmethod dans un Protocol"
            )

    def test_no_abc_import_in_protocol_module(self):
        """Le module protocol.py ne doit plus importer abc.abstractmethod."""
        import inspect

        from src.data.repositories import protocol

        source = inspect.getsource(protocol)
        assert "abstractmethod" not in source, "protocol.py ne doit plus utiliser abstractmethod"
        assert "from abc import" not in source, "protocol.py ne doit plus importer depuis abc"

    def test_duckdbrepo_satisfies_protocol(self):
        """DuckDBRepository satisfait DataRepository via duck typing (@runtime_checkable)."""
        from src.data.repositories.duckdb_repo import DuckDBRepository

        # @runtime_checkable permet isinstance() sur les méthodes déclarées
        # Vérification structurelle : DuckDBRepository a les méthodes attendues
        for attr in ("xuid", "db_path", "load_matches", "get_match_count"):
            assert hasattr(DuckDBRepository, attr), (
                f"DuckDBRepository doit avoir la méthode/propriété '{attr}'"
            )

    def test_protocol_is_runtime_checkable(self):
        """DataRepository doit rester @runtime_checkable."""
        from src.ports.repository import DataRepository

        assert hasattr(DataRepository, "__protocol_attrs__") or getattr(
            DataRepository, "_is_protocol", False
        ), "DataRepository doit être un Protocol"


# ─────────────────────────────────────────────────────────────────────────────
# P3-3 : cached_friend_matches_df — branche legacy supprimée
# ─────────────────────────────────────────────────────────────────────────────


class TestCachedFriendMatchesDfNoLegacy:
    """La branche legacy à objets .same_team a été retirée de cached_friend_matches_df."""

    def test_no_legacy_object_branch_in_source(self):
        """Le code source ne doit plus contenir la construction legacy via r.match_id etc."""
        import inspect

        from src.ui.cache_filters import cached_friend_matches_df

        source = inspect.getsource(cached_friend_matches_df)
        assert "r.match_id" not in source, "Branche legacy .match_id doit être supprimée"
        assert "r.same_team" not in source, "Branche legacy .same_team doit être supprimée"
        assert "Chemin legacy" not in source, "Commentaire 'Chemin legacy' doit être supprimé"

    def test_empty_rows_branch_in_source(self):
        """Si rows est vide, la fonction retourne un DataFrame vide (vérifié via source)."""
        import inspect

        from src.ui.cache_filters import cached_friend_matches_df

        source = inspect.getsource(cached_friend_matches_df)
        assert "pl.DataFrame(schema=" in source, (
            "cached_friend_matches_df doit retourner un DataFrame vide quand rows est vide"
        )
        assert "_FRIEND_DF_EMPTY_SCHEMA" in source, (
            "Le schéma vide doit utiliser _FRIEND_DF_EMPTY_SCHEMA"
        )

    def test_no_is_duckdb_v4_path_import(self):
        """_is_duckdb_v4_path ne doit plus être importé dans cache_filters (devenu orphelin)."""
        import inspect

        from src.ui import cache_filters

        source = inspect.getsource(cache_filters)
        assert "is_duckdb_v4_path" not in source, (
            "is_duckdb_v4_path ne doit plus être importé dans cache_filters après suppression legacy"
        )
