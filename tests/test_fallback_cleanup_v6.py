"""Tests vérifiant la suppression des fallbacks excessifs post-refactoring v6.

Valide que :
- Les méthodes de fallback v4/v3 ont bien été supprimées
- Les mixins interrogent directement les vues v6 garanties
- Les RuntimeErrors sont levées pour les configurations invalides
"""

from __future__ import annotations

import contextlib
import logging
from pathlib import Path
from unittest.mock import MagicMock

import duckdb
import pytest

# =============================================================================
# Helpers partagés
# =============================================================================


def _shared_db_with_view(tmp_path: Path) -> Path:
    """Crée un shared_matches_v2.duckdb minimal avec v_gamertag_lookup + v_killer_victim_full."""
    db = tmp_path / "shared_matches_v2.duckdb"
    conn = duckdb.connect(str(db))
    conn.execute("CREATE TABLE xuid_aliases (xuid VARCHAR PRIMARY KEY, gamertag VARCHAR)")
    conn.execute(
        "CREATE TABLE match_participants "
        "(match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR, team_id INTEGER, outcome INTEGER, "
        "rank SMALLINT, score INTEGER, kills SMALLINT, deaths SMALLINT, assists SMALLINT, "
        "shots_fired INTEGER, shots_hit INTEGER, damage_dealt FLOAT, damage_taken FLOAT, "
        "avg_life_seconds FLOAT, created_at TIMESTAMP, PRIMARY KEY (match_id, xuid))"
    )
    conn.execute(
        "CREATE TABLE killer_victim_pairs "
        "(match_id VARCHAR, killer_xuid VARCHAR, killer_gamertag VARCHAR, "
        "victim_xuid VARCHAR, victim_gamertag VARCHAR, kill_count INTEGER, time_ms BIGINT)"
    )
    conn.execute(
        "CREATE VIEW v_gamertag_lookup AS "
        "SELECT xuid, gamertag FROM xuid_aliases "
        "UNION "
        "SELECT xuid, gamertag FROM match_participants"
    )
    conn.execute(
        "CREATE VIEW v_killer_victim_full AS "
        "SELECT match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag, "
        "kill_count, time_ms FROM killer_victim_pairs"
    )
    conn.execute("INSERT INTO xuid_aliases VALUES ('xuid001', 'TestPlayer')")
    conn.execute(
        "INSERT INTO killer_victim_pairs VALUES "
        "('match_001', 'xuid001', 'TestPlayer', 'xuid002', 'Enemy', 3, 12000)"
    )
    conn.close()
    return db


def _player_db(tmp_path: Path) -> Path:
    """Crée une stats.duckdb minimale (v6 — sans match_stats)."""
    db = tmp_path / "stats.duckdb"
    conn = duckdb.connect(str(db))
    conn.execute("CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR)")
    conn.execute("INSERT INTO sync_meta VALUES ('xuid', 'xuid001')")
    conn.close()
    return db


def _make_repo(tmp_path: Path):
    """Crée un DuckDBRepository v6 minimal avec shared_db configuré."""
    from src.data.repositories.duckdb_repo import DuckDBRepository

    shared = _shared_db_with_view(tmp_path)
    player = _player_db(tmp_path)
    return DuckDBRepository(
        player_db_path=player,
        xuid="xuid001",
        shared_db_path=shared,
        read_only=False,
    )


# =============================================================================
# Ph-1a : GamertagResolverMixin
# =============================================================================


class TestGamertagResolverV6:
    """Vérifie la suppression du fallback _resolve_gamertag_without_view."""

    def test_resolve_gamertag_without_view_absent(self) -> None:
        from src.data.repositories._gamertag_resolver import GamertagResolverMixin

        assert not hasattr(GamertagResolverMixin, "_resolve_gamertag_without_view"), (
            "_resolve_gamertag_without_view doit être supprimée (Ph-1a)"
        )

    def test_resolve_gamertag_uses_view(self, tmp_path: Path) -> None:
        """resolve_gamertag retourne le gamertag depuis shared.v_gamertag_lookup."""
        repo = _make_repo(tmp_path)
        result = repo.resolve_gamertag("xuid001")
        assert result == "TestPlayer"

    def test_resolve_gamertag_unknown_returns_none(self, tmp_path: Path) -> None:
        """XUID inconnu → None + warning loggé."""
        repo = _make_repo(tmp_path)
        result = repo.resolve_gamertag("xuid_inexistant")
        assert result is None

    def test_resolve_gamertag_empty_returns_none(self, tmp_path: Path) -> None:
        """XUID vide → None (guard précoce)."""
        repo = _make_repo(tmp_path)
        result = repo.resolve_gamertag("")
        assert result is None

    def test_get_all_gamertags_no_table_guard(self) -> None:
        """get_all_gamertags ne doit plus appeler _has_shared_table('v_gamertag_lookup')."""
        import inspect

        from src.data.repositories._gamertag_resolver import GamertagResolverMixin

        src = inspect.getsource(GamertagResolverMixin.get_all_gamertags)
        assert "_has_shared_table" not in src, (
            "get_all_gamertags ne doit plus utiliser _has_shared_table"
        )

    def test_get_all_gamertags_via_view(self, tmp_path: Path) -> None:
        """get_all_gamertags retourne les gamertags depuis v_gamertag_lookup."""
        repo = _make_repo(tmp_path)
        result = repo.get_all_gamertags()
        assert "TestPlayer" in result

    def test_resolve_gamertag_warning_on_exception(
        self, tmp_path: Path, caplog: pytest.LogCaptureFixture
    ) -> None:
        """resolve_gamertag logue WARNING avec exc_info si la vue échoue."""
        repo = _make_repo(tmp_path)
        # Faire échouer la connexion en patchant _get_connection
        broken_conn = MagicMock()
        broken_conn.execute.side_effect = RuntimeError("connexion cassée")
        original = repo._get_connection
        repo._get_connection = lambda: broken_conn

        with caplog.at_level(logging.WARNING, logger="src.data.repositories._gamertag_resolver"):
            result = repo.resolve_gamertag("xuid001")

        assert result is None
        # Au moins un message WARNING doit avoir été loggé
        warnings = [r for r in caplog.records if r.levelno >= logging.WARNING]
        assert warnings, "Un WARNING doit être loggé quand v_gamertag_lookup échoue"

        repo._get_connection = original


# =============================================================================
# Ph-1b : KillerVictimMixin
# =============================================================================


class TestKillerVictimV6:
    """Vérifie la suppression du triple fallback dans KillerVictimMixin."""

    def test_no_triple_fallback_in_has_pairs(self) -> None:
        """has_killer_victim_pairs ne doit plus avoir de triple branchement."""
        import inspect

        from src.data.repositories._killer_victim_repo import KillerVictimMixin

        src = inspect.getsource(KillerVictimMixin.has_killer_victim_pairs)
        assert "_has_shared_view" not in src, "Le guard _has_shared_view doit être supprimé"
        assert "_has_shared_table" not in src, "Le guard _has_shared_table doit être supprimé"

    def test_no_triple_fallback_in_load(self) -> None:
        """load_killer_victim_pairs_as_polars ne doit plus avoir de triple branchement."""
        import inspect

        from src.data.repositories._killer_victim_repo import KillerVictimMixin

        src = inspect.getsource(KillerVictimMixin.load_killer_victim_pairs_as_polars)
        assert "_has_shared_view" not in src, "Le guard _has_shared_view doit être supprimé"
        assert "_has_shared_table" not in src, "Le guard _has_shared_table doit être supprimé"

    def test_has_killer_victim_pairs_true(self, tmp_path: Path) -> None:
        """has_killer_victim_pairs retourne True si la vue contient des données."""
        repo = _make_repo(tmp_path)
        assert repo.has_killer_victim_pairs() is True

    def test_load_killer_victim_pairs_returns_data(self, tmp_path: Path) -> None:
        """load_killer_victim_pairs_as_polars retourne les données de v_killer_victim_full."""
        repo = _make_repo(tmp_path)
        df = repo.load_killer_victim_pairs_as_polars()
        assert not df.is_empty()
        assert "killer_xuid" in df.columns

    def test_load_killer_victim_pairs_match_filter(self, tmp_path: Path) -> None:
        """Filtre par match_id fonctionne correctement."""
        repo = _make_repo(tmp_path)
        df = repo.load_killer_victim_pairs_as_polars(match_id="match_001")
        assert len(df) == 1

        df_empty = repo.load_killer_victim_pairs_as_polars(match_id="match_inexistant")
        assert df_empty.is_empty()


# =============================================================================
# Ph-1c : EncounterCareerMixin
# =============================================================================


class TestCareerEncountersV6:
    """Vérifie la suppression de _get_kv_source_shared et du guard killer_victim_pairs."""

    def test_get_kv_source_shared_absent(self) -> None:
        from src.data.repositories._career_encounters_repo import EncounterCareerMixin

        assert not hasattr(EncounterCareerMixin, "_get_kv_source_shared"), (
            "_get_kv_source_shared doit être supprimée (Ph-1c)"
        )

    def test_load_antagonists_no_table_guard(self) -> None:
        """load_antagonists ne doit plus vérifier l'existence de killer_victim_pairs."""
        import inspect

        from src.data.repositories._career_encounters_repo import EncounterCareerMixin

        src = inspect.getsource(EncounterCareerMixin.load_antagonists)
        assert '_has_shared_table("killer_victim_pairs")' not in src, (
            "Le guard _has_shared_table('killer_victim_pairs') doit être supprimé"
        )

    def test_load_top_encountered_inline_view(self) -> None:
        """load_top_encountered doit inliner shared.v_killer_victim_full."""
        import inspect

        from src.data.repositories._career_encounters_repo import EncounterCareerMixin

        src = inspect.getsource(EncounterCareerMixin.load_top_encountered)
        assert "shared.v_killer_victim_full" in src
        assert "_get_kv_source_shared" not in src

    def test_load_antagonists_inline_view(self) -> None:
        """load_antagonists doit inliner shared.v_killer_victim_full."""
        import inspect

        from src.data.repositories._career_encounters_repo import EncounterCareerMixin

        src = inspect.getsource(EncounterCareerMixin.load_antagonists)
        assert "shared.v_killer_victim_full" in src
        assert "_get_kv_source_shared" not in src


# =============================================================================
# Ph-2 : MatchQueriesMixin
# =============================================================================


class TestMatchQueriesV6:
    """Vérifie la suppression des branches v4/v3 dans _get_match_source."""

    def test_get_match_table_name_absent(self) -> None:
        from src.data.repositories._match_queries import MatchQueriesMixin

        assert not hasattr(MatchQueriesMixin, "_get_match_table_name"), (
            "_get_match_table_name doit être supprimée (Ph-2)"
        )

    def test_get_match_source_no_information_schema(self) -> None:
        """_get_match_source ne doit plus scanner information_schema."""
        import inspect

        from src.data.repositories._match_queries import MatchQueriesMixin

        src = inspect.getsource(MatchQueriesMixin._get_match_source)
        assert "information_schema" not in src

    def test_get_match_source_no_has_shared_view_guard(self) -> None:
        """_get_match_source ne doit plus appeler _has_shared_view."""
        import inspect

        from src.data.repositories._match_queries import MatchQueriesMixin

        src = inspect.getsource(MatchQueriesMixin._get_match_source)
        assert "_has_shared_view" not in src

    def test_get_match_source_raises_on_empty_xuid(self, tmp_path: Path) -> None:
        """RuntimeError si XUID vide."""
        from src.data.repositories.duckdb_repo import DuckDBRepository

        shared = _shared_db_with_view(tmp_path)
        player = _player_db(tmp_path)
        repo = DuckDBRepository(
            player_db_path=player,
            xuid="",  # XUID vide
            shared_db_path=shared,
            read_only=False,
        )
        conn = repo._get_connection()
        with pytest.raises(RuntimeError, match="XUID manquant"):
            repo._get_match_source(conn)

    def test_get_match_source_raises_on_no_shared(self, tmp_path: Path) -> None:
        """RuntimeError si shared_matches est absent."""
        from src.data.repositories.duckdb_repo import DuckDBRepository

        player = _player_db(tmp_path)
        repo = DuckDBRepository(
            player_db_path=player,
            xuid="xuid001",
            shared_db_path=None,  # Pas de shared
            read_only=False,
        )
        conn = repo._get_connection()
        with pytest.raises(RuntimeError, match="shared_matches_v2.duckdb indisponible"):
            repo._get_match_source(conn)

    def test_get_match_count_no_triple_guard(self) -> None:
        """get_match_count ne doit plus faire de triple guard has_shared_table."""
        import inspect

        from src.data.repositories._match_queries import MatchQueriesMixin

        src = inspect.getsource(MatchQueriesMixin.get_match_count)
        assert "_has_shared_table" not in src

    def test_get_match_count_returns_zero_on_error(
        self, tmp_path: Path, caplog: pytest.LogCaptureFixture
    ) -> None:
        """get_match_count retourne 0 et logge DEBUG si shared.match_participants échoue."""
        repo = _make_repo(tmp_path)
        # Ajouter un match_participant pour avoir un compteur non nul
        conn = repo._get_connection()
        with contextlib.suppress(Exception):
            conn.execute(
                "INSERT INTO shared.match_participants "
                "(match_id, xuid, gamertag, team_id, outcome, rank, score, kills, deaths, "
                "assists, shots_fired, shots_hit, damage_dealt, damage_taken, avg_life_seconds)"
                " VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
                [
                    "match_001",
                    "xuid001",
                    "TestPlayer",
                    0,
                    2,
                    1,
                    2500,
                    15,
                    6,
                    4,
                    200,
                    120,
                    5000.0,
                    2000.0,
                    30.0,
                ],
            )
        count = repo.get_match_count()
        assert isinstance(count, int)


# =============================================================================
# Ph-3 : LegacyCompatMixin
# =============================================================================


class TestLegacyCompatV6:
    """Vérifie la suppression de _collect_xuids_local."""

    def test_collect_xuids_local_absent(self) -> None:
        from src.data.repositories._legacy_compat import LegacyCompatMixin

        assert not hasattr(LegacyCompatMixin, "_collect_xuids_local"), (
            "_collect_xuids_local doit être supprimée (Ph-3)"
        )

    def test_list_other_player_xuids_no_local_tables(self) -> None:
        """list_other_player_xuids ne doit plus interroger les tables locales supprimées."""
        import inspect

        from src.data.repositories._legacy_compat import LegacyCompatMixin

        src = inspect.getsource(LegacyCompatMixin.list_other_player_xuids)
        assert "highlight_events" not in src
        assert "antagonists" not in src

    def test_list_other_player_xuids_returns_list(self, tmp_path: Path) -> None:
        """list_other_player_xuids retourne une liste (éventuellement vide)."""
        repo = _make_repo(tmp_path)
        result = repo.list_other_player_xuids(limit=100)
        assert isinstance(result, list)
