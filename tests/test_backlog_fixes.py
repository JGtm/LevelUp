"""Tests pour les correctifs du backlog (session 2026-03-19).

Couvre les fixes 1-5, 8, 10 et 11 identifiés dans .ai/BACKLOG.md.
Tests purement unitaires — pas de Streamlit, pas d'appels API.
"""

from __future__ import annotations

import json
from pathlib import Path

import duckdb
import polars as pl
import pytest

# ===========================================================================
# Fix 1 — map_name absent du schéma vide (ColumnNotFoundError)
# ===========================================================================


class TestMapNameInEmptySchema:
    """map_name doit être présent dans _FRIEND_DF_EMPTY_SCHEMA."""

    def test_friend_df_empty_schema_has_map_name(self):
        from src.ui.cache_filters import _FRIEND_DF_EMPTY_SCHEMA

        assert "map_name" in _FRIEND_DF_EMPTY_SCHEMA, (
            "map_name absent du schéma vide — ColumnNotFoundError lors du filtrage mode Amis"
        )

    def test_friend_df_empty_schema_map_name_is_utf8(self):
        from src.ui.cache_filters import _FRIEND_DF_EMPTY_SCHEMA

        assert _FRIEND_DF_EMPTY_SCHEMA["map_name"] == pl.Utf8


# ===========================================================================
# Fix 2 — Bots bid(33.0) affichés comme nom tronqué
# ===========================================================================


class TestBotNameResolution:
    """Les XUIDs de type bid(33.0) doivent résoudre vers un nom lisible."""

    def test_get_bot_name_returns_string(self):
        from src.config import get_bot_name

        result = get_bot_name("bid(33.0)")
        assert isinstance(result, str)
        assert len(result) > 0

    def test_get_bot_name_not_raw_xuid_prefix(self):
        from src.config import get_bot_name

        result = get_bot_name("bid(42.0)")
        # Le résultat ne doit pas être un fragment brut comme "bid(42."
        assert not result.startswith("bid("), (
            "get_bot_name ne doit pas retourner le préfixe brut du XUID"
        )

    def test_filter_encounter_xuids_excludes_ghosts(self):
        """filter_encounter_xuids doit exclure les joueurs fantômes."""
        from src.ui.pages.match_view_encounters_logic import filter_encounter_xuids

        players = [
            # joueur actif
            {"xuid": "111", "kills": 5, "deaths": 3, "assists": 1, "score": 200},
            # joueur fantôme (tout à 0)
            {"xuid": "222", "kills": 0, "deaths": 0, "assists": 0, "score": 0},
            # joueur actif
            {"xuid": "333", "kills": 0, "deaths": 1, "assists": 0, "score": 10},
        ]
        result = filter_encounter_xuids(players, self_xuid="999", friends_set=set())

        assert "222" not in result, "joueur fantôme ne doit pas apparaître dans les rencontres"
        assert "111" in result
        assert "333" in result


# ===========================================================================
# Fix 4 — Ratio KDA incohérent entre panneaux joueur et coéquipier
# ===========================================================================


class TestRatioKdaConsistency:
    """ratio doit être un alias de kda (API), pas recalculé depuis kills/deaths."""

    def test_finalize_polars_df_ratio_equals_kda(self):
        from src.data.repositories._match_queries_polars import _MatchQueriesPolarsMixin

        df = pl.DataFrame(
            {
                "kda": [1.5, 2.0, 0.0, None],
                "kills": [3, 6, 0, 0],
                "deaths": [2, 3, 1, 0],
                "assists": [0, 0, 0, 0],
            }
        )
        result = _MatchQueriesPolarsMixin._finalize_polars_df(df, columns=None)

        assert "ratio" in result.columns
        # ratio doit être identique à kda (API)
        assert result["ratio"].to_list() == result["kda"].to_list()

    def test_finalize_polars_df_ratio_null_when_kda_absent(self):
        from src.data.repositories._match_queries_polars import _MatchQueriesPolarsMixin

        df = pl.DataFrame({"kills": [3], "deaths": [2], "assists": [1]})
        result = _MatchQueriesPolarsMixin._finalize_polars_df(df, columns=None)

        assert "ratio" in result.columns
        assert result["ratio"][0] is None

    def test_ratio_not_recomputed_on_zero_deaths(self):
        """Cas critique : kda=2.0 avec deaths=0 — ratio ne doit PAS être +inf."""
        from src.data.repositories._match_queries_polars import _MatchQueriesPolarsMixin

        df = pl.DataFrame(
            {
                "kda": [2.0],
                "kills": [4],
                "deaths": [0],
                "assists": [0],
            }
        )
        result = _MatchQueriesPolarsMixin._finalize_polars_df(df, columns=None)
        assert result["ratio"][0] == pytest.approx(2.0)


# ===========================================================================
# Fix 5 — Joueurs fantômes dans les encounters (pas le scoreboard)
# ===========================================================================


class TestGhostPlayerFiltering:
    """Les joueurs fantômes sont exclus des encounters, pas du scoreboard."""

    def test_filter_encounter_xuids_all_ghosts_returns_empty(self):
        from src.ui.pages.match_view_encounters_logic import filter_encounter_xuids

        players = [
            {"xuid": "aaa", "kills": 0, "deaths": 0, "assists": 0, "score": 0},
            {"xuid": "bbb", "kills": 0, "deaths": 0, "assists": 0, "score": 0},
        ]
        result = filter_encounter_xuids(players, self_xuid="zzz", friends_set=set())
        assert "aaa" not in result
        assert "bbb" not in result

    def test_filter_encounter_xuids_partial_zero_not_ghost(self):
        """Un joueur avec kills=0 mais deaths=1 n'est PAS un fantôme."""
        from src.ui.pages.match_view_encounters_logic import filter_encounter_xuids

        players = [
            {"xuid": "aaa", "kills": 0, "deaths": 1, "assists": 0, "score": 0},
        ]
        result = filter_encounter_xuids(players, self_xuid="zzz", friends_set=set())
        assert "aaa" in result

    def test_filter_encounter_xuids_none_values_count_as_zero(self):
        """None est équivalent à 0 pour la détection fantôme."""
        from src.ui.pages.match_view_encounters_logic import filter_encounter_xuids

        players = [
            {"xuid": "aaa", "kills": None, "deaths": None, "assists": None, "score": None},
        ]
        result = filter_encounter_xuids(players, self_xuid="zzz", friends_set=set())
        # None = 0 sur toutes les stats → joueur fantôme → exclu
        assert "aaa" not in result


# ===========================================================================
# Fix 8 — performance_score utilisé depuis la colonne quand disponible
# ===========================================================================


class TestMapBreakdownPerformanceScore:
    """compute_map_breakdown doit prioritairement lire performance_score depuis la colonne."""

    def _make_df_with_perf(self, n: int = 20) -> pl.DataFrame:
        import random

        random.seed(42)
        return pl.DataFrame(
            {
                "match_id": [f"m{i}" for i in range(n)],
                "map_name": ["MapA"] * (n // 2) + ["MapB"] * (n // 2),
                "outcome": [2, 3] * (n // 2),  # WIN=2, LOSS=3
                "kills": [random.randint(3, 15) for _ in range(n)],
                "deaths": [random.randint(1, 10) for _ in range(n)],
                "assists": [random.randint(0, 5) for _ in range(n)],
                "kda": [random.uniform(0.5, 3.0) for _ in range(n)],
                "accuracy": [random.uniform(20, 60) for _ in range(n)],
                "time_played_seconds": [600.0] * n,
                "personal_score": [random.randint(1000, 5000) for _ in range(n)],
                "damage_dealt": [random.uniform(1000, 8000) for _ in range(n)],
                "rank": [None] * n,
                "team_mmr": [None] * n,
                "enemy_mmr": [None] * n,
                "performance_score": [random.uniform(30.0, 70.0) for _ in range(n)],
            }
        )

    def test_breakdown_uses_perf_column_when_present(self):
        from src.analysis.maps import compute_map_breakdown

        df = self._make_df_with_perf(30)
        result = compute_map_breakdown(df)

        assert not result.is_empty()
        assert "performance_avg" in result.columns
        # Tous les performance_avg doivent être non-None (car colonne présente et non vide)
        non_null = result["performance_avg"].drop_nulls()
        assert len(non_null) == len(result), (
            "performance_avg doit être non-null quand performance_score est présent dans le DataFrame"
        )

    def test_breakdown_performance_avg_range_from_column(self):
        """Les valeurs de performance_avg doivent rester dans la plage de la colonne source."""
        from src.analysis.maps import compute_map_breakdown

        df = self._make_df_with_perf(40)
        result = compute_map_breakdown(df)

        performance_avgs = result["performance_avg"].drop_nulls().to_list()
        source_min = df["performance_score"].min()
        source_max = df["performance_score"].max()
        for val in performance_avgs:
            assert source_min <= val <= source_max, (
                f"performance_avg={val} hors plage [{source_min}, {source_max}]"
            )


# ===========================================================================
# Fix 10 — performance_score dans COLUMNS_COMMON
# ===========================================================================


class TestColumnsCommonHasPerfScore:
    """performance_score doit être dans COLUMNS_COMMON pour être projeté."""

    def test_columns_common_has_performance_score(self):
        from src.ui._cache_core import COLUMNS_COMMON

        assert "performance_score" in COLUMNS_COMMON, (
            "performance_score absent de COLUMNS_COMMON — "
            "la colonne sera ignorée lors de la projection"
        )


# ===========================================================================
# Fix 11 — Fan-out enrichissement des coéquipiers
# ===========================================================================


class TestFanoutGetOtherPlayers:
    """_get_other_registered_players doit lire db_profiles.json et filtrer le joueur courant."""

    def _make_fake_engine(self, tmp_path: Path, gamertag: str, xuid: str) -> object:
        """Crée un objet minimal simulant l'interface du mixin."""
        from src.data.sync._engine_fanout import FanoutEnrichmentMixin

        player_db = tmp_path / "players" / gamertag / "stats.duckdb"
        player_db.parent.mkdir(parents=True, exist_ok=True)
        player_db.touch()

        profiles = {
            "version": "2.1",
            "profiles": {
                gamertag: {"db_path": f"players/{gamertag}/stats.duckdb", "xuid": xuid},
                "OtherPlayer": {
                    "db_path": "players/OtherPlayer/stats.duckdb",
                    "xuid": "9999999",
                },
            },
        }
        (tmp_path / "db_profiles.json").write_text(
            json.dumps(profiles), encoding="utf-8"
        )

        class FakeEngine(FanoutEnrichmentMixin):
            def __init__(self):
                self._player_db_path = player_db
                self._xuid = xuid
                self._gamertag = gamertag

        return FakeEngine()

    def test_excludes_current_player_by_xuid(self, tmp_path):
        engine = self._make_fake_engine(tmp_path, "MainPlayer", "1234567")
        others = engine._get_other_registered_players()
        xuids = [p["xuid"] for p in others]
        assert "1234567" not in xuids

    def test_includes_other_player(self, tmp_path):
        engine = self._make_fake_engine(tmp_path, "MainPlayer", "1234567")
        others = engine._get_other_registered_players()
        assert any(p["gamertag"] == "OtherPlayer" for p in others)

    def test_returns_empty_when_no_profiles_file(self, tmp_path):
        from src.data.sync._engine_fanout import FanoutEnrichmentMixin

        player_db = tmp_path / "stats.duckdb"
        player_db.touch()

        class FakeEngine(FanoutEnrichmentMixin):
            def __init__(self):
                self._player_db_path = player_db
                self._xuid = "1234567"
                self._gamertag = "MainPlayer"

        engine = FakeEngine()
        others = engine._get_other_registered_players()
        assert others == []

    def test_db_path_resolved_from_project_root(self, tmp_path):
        engine = self._make_fake_engine(tmp_path, "MainPlayer", "1234567")
        others = engine._get_other_registered_players()
        assert len(others) == 1
        # Le db_path retourné doit être un Path absolu sous tmp_path
        assert others[0]["db_path"].is_absolute()
        assert "OtherPlayer" in str(others[0]["db_path"])


class TestFanoutFindCommonMatchIds:
    """_find_common_match_ids doit retrouver les matchs partagés dans shared (read-only)."""

    def _create_shared_db(self, path: Path, xuid: str, match_ids: list[str]) -> None:
        """Crée un shared_matches minimal avec les match_participants demandés."""
        conn = duckdb.connect(str(path))
        conn.execute(
            "CREATE TABLE IF NOT EXISTS match_participants "
            "(match_id VARCHAR, xuid VARCHAR)"
        )
        for mid in match_ids:
            conn.execute(
                "INSERT INTO match_participants VALUES (?, ?)", [mid, xuid]
            )
        conn.close()

    def test_returns_common_ids(self, tmp_path):
        from src.data.sync._engine_fanout import FanoutEnrichmentMixin

        shared = tmp_path / "shared_matches_v2.duckdb"
        player_xuid = "TARGET_XUID"
        self._create_shared_db(shared, player_xuid, ["match_A", "match_B", "match_C"])

        class FakeEngine(FanoutEnrichmentMixin):
            pass

        engine = FakeEngine()
        result = engine._find_common_match_ids(
            player_xuid, shared, ["match_A", "match_B", "match_X"]
        )
        assert sorted(result) == ["match_A", "match_B"]

    def test_returns_empty_when_no_match(self, tmp_path):
        from src.data.sync._engine_fanout import FanoutEnrichmentMixin

        shared = tmp_path / "shared_matches_v2.duckdb"
        self._create_shared_db(shared, "OTHER_XUID", ["match_A"])

        class FakeEngine(FanoutEnrichmentMixin):
            pass

        engine = FakeEngine()
        result = engine._find_common_match_ids(
            "TARGET_XUID", shared, ["match_A"]
        )
        assert result == []

    def test_returns_empty_on_missing_shared(self, tmp_path):
        from src.data.sync._engine_fanout import FanoutEnrichmentMixin

        class FakeEngine(FanoutEnrichmentMixin):
            pass

        engine = FakeEngine()
        missing = tmp_path / "nonexistent.duckdb"
        result = engine._find_common_match_ids("any_xuid", missing, ["match_A"])
        assert result == []


class TestFanoutEnrichOtherRegisteredPlayers:
    """_enrich_other_registered_players ne doit pas lever d'exception si les DBs sont absentes."""

    def test_no_error_when_player_db_missing(self, tmp_path):
        """Si la DB d'un joueur n'existe pas, le fan-out est ignoré silencieusement."""
        from src.data.sync._engine_fanout import FanoutEnrichmentMixin

        player_db = tmp_path / "players" / "Main" / "stats.duckdb"
        player_db.parent.mkdir(parents=True)
        player_db.touch()

        profiles = {
            "version": "2.1",
            "profiles": {
                "Main": {"db_path": "players/Main/stats.duckdb", "xuid": "111"},
                "Ghost": {"db_path": "players/Ghost/stats.duckdb", "xuid": "222"},
            },
        }
        (tmp_path / "db_profiles.json").write_text(json.dumps(profiles), encoding="utf-8")

        class FakeEngine(FanoutEnrichmentMixin):
            def __init__(self):
                self._player_db_path = player_db
                self._xuid = "111"
                self._gamertag = "Main"

        engine = FakeEngine()
        # Ne doit pas lever d'exception même si Ghost/stats.duckdb est absent
        engine._enrich_other_registered_players(["match_1", "match_2"])

    def test_no_error_when_new_match_ids_empty(self, tmp_path):
        from src.data.sync._engine_fanout import FanoutEnrichmentMixin

        class FakeEngine(FanoutEnrichmentMixin):
            def __init__(self):
                self._player_db_path = tmp_path / "stats.duckdb"
                self._xuid = "111"
                self._gamertag = "Main"

        engine = FakeEngine()
        # Liste vide → retour immédiat, pas d'erreur
        engine._enrich_other_registered_players([])

    def test_no_error_with_malformed_profiles(self, tmp_path):
        from src.data.sync._engine_fanout import FanoutEnrichmentMixin

        player_db = tmp_path / "stats.duckdb"
        player_db.touch()
        (tmp_path / "db_profiles.json").write_text("NOT_JSON", encoding="utf-8")

        class FakeEngine(FanoutEnrichmentMixin):
            def __init__(self):
                self._player_db_path = player_db
                self._xuid = "111"
                self._gamertag = "Main"

        engine = FakeEngine()
        engine._enrich_other_registered_players(["match_1"])


# ===========================================================================
# Fix 3 — Ordre stable dans friends_impact_heatmap
# ===========================================================================


class TestImpactMatrixOrder:
    """unique() doit utiliser maintain_order=True pour éviter une matrice non déterministe."""

    def test_impact_matrix_match_ids_stable(self):
        """Le premier match_id dans impact_matrix doit toujours être le plus ancien."""
        # Vérification indirecte : on s'assure que le code appelle maintain_order=True
        import inspect

        from src.visualization import friends_impact_heatmap

        source = inspect.getsource(friends_impact_heatmap)
        assert "maintain_order=True" in source, (
            "friends_impact_heatmap.py doit utiliser unique(maintain_order=True) "
            "pour garantir un ordre déterministe de la matrice"
        )


# ===========================================================================
# Fix 5 (suite) — _is_ghost_player : clés absentes = données partielles, pas fantôme
# ===========================================================================


class TestIsGhostPlayerMissingKeys:
    """Un dict sans les clés stat n'est PAS un fantôme (données partielles, pas API 0)."""

    def test_dict_without_stat_keys_is_not_ghost(self):
        """Cas critique du bug : un dict sans kills/deaths/assists/score ne doit pas être filtré."""
        from src.ui.pages.match_view_encounters_logic import _is_ghost_player

        # Dict type scoreboard allégé (pas de champs stat)
        p = {"xuid": "aaa", "gamertag": "PlayerX", "team_id": 0}
        assert not _is_ghost_player(p), (
            "Un joueur sans champs kills/deaths/assists/score ne doit PAS être classé fantôme"
        )

    def test_dict_with_only_some_keys_not_ghost(self):
        """Un dict avec kills mais sans deaths n'est PAS un fantôme."""
        from src.ui.pages.match_view_encounters_logic import _is_ghost_player

        p = {"xuid": "aaa", "kills": 0, "deaths": 0}  # manque assists et score
        assert not _is_ghost_player(p)

    def test_filter_encounter_xuids_keeps_players_without_stat_keys(self):
        """filter_encounter_xuids doit inclure un joueur sans champs stat (données partielles)."""
        from src.ui.pages.match_view_encounters_logic import filter_encounter_xuids

        players = [
            {"xuid": "aaa", "gamertag": "PlayerX"},  # pas de champs stat → pas fantôme
            {"xuid": "bbb", "kills": 0, "deaths": 0, "assists": 0, "score": 0},  # fantôme
        ]
        result = filter_encounter_xuids(players, self_xuid="zzz", friends_set=set())
        assert "aaa" in result, "joueur sans champs stat doit être inclus (données partielles)"
        assert "bbb" not in result, "joueur tout à 0 doit être exclu (fantôme)"


# ===========================================================================
# Fix 1 (suite) — load_friend_match_details retourne map_name dans les lignes
# ===========================================================================


class TestLoadFriendMatchDetailsMapName:
    """map_name doit figurer dans les rows retournées par load_friend_match_details."""

    def test_map_name_column_present_in_result(self, tmp_path):
        """La colonne map_name doit être présente même si vide."""
        from datetime import datetime, timezone

        import duckdb

        from src.data.repositories.duckdb_repo import DuckDBRepository

        shared_path = tmp_path / "shared.duckdb"
        player_path = tmp_path / "stats.duckdb"

        # Shared DB
        conn = duckdb.connect(str(shared_path))
        conn.execute("""
            CREATE TABLE match_registry (
                match_id VARCHAR PRIMARY KEY, start_time TIMESTAMP,
                playlist_name VARCHAR, pair_name VARCHAR, map_name VARCHAR
            )
        """)
        conn.execute("""
            CREATE TABLE match_participants (
                match_id VARCHAR, xuid VARCHAR, team_id INTEGER,
                outcome INTEGER, kills INTEGER, deaths INTEGER, score INTEGER
            )
        """)
        ts = datetime(2024, 6, 1, 12, 0, tzinfo=timezone.utc)
        conn.execute("INSERT INTO match_registry VALUES (?, ?, ?, ?, ?)",
                     ["m1", ts, "Ranked", "RSSLAYER", "Bazaar"])
        conn.execute("INSERT INTO match_participants VALUES (?, ?, ?, ?, ?, ?, ?)",
                     ["m1", "0001", 0, 2, 10, 5, 1000])
        conn.execute("INSERT INTO match_participants VALUES (?, ?, ?, ?, ?, ?, ?)",
                     ["m1", "0002", 1, 3, 8, 3, 800])
        conn.close()

        # Player DB
        pconn = duckdb.connect(str(player_path))
        pconn.execute("CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR, updated_at TIMESTAMP)")
        pconn.close()

        repo = DuckDBRepository(player_db_path=player_path, xuid="0001",
                                shared_db_path=shared_path, read_only=False)
        dfr = repo.load_friend_match_details("0002", ["m1"])

        assert not dfr.is_empty(), "load_friend_match_details doit retourner des données"
        assert "map_name" in dfr.columns, "map_name doit figurer dans les colonnes retournées"
        assert dfr["map_name"][0] == "Bazaar"
