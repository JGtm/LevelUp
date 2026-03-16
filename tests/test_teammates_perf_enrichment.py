"""Tests pour l'enrichissement performance_score dans la page escouade (bug #6).

Couvre :
- load_performance_scores_from_player_db (_teammates_perf_queries)
- TeammatesService.enrich_with_performance_score
- _use_or_compute_performance (_teammates_trio_helpers)
- _merge_trio_dataframes — propagation de performance_score
"""

from __future__ import annotations

import polars as pl
import pytest

# ============================================================================
# _teammates_perf_queries.py — load_performance_scores_from_player_db
# ============================================================================


class TestLoadPerformanceScoresFromPlayerDb:
    def test_empty_match_ids_returns_empty(self, tmp_path):
        from src.data.services._teammates_perf_queries import (
            load_performance_scores_from_player_db,
        )

        result = load_performance_scores_from_player_db(str(tmp_path / "stats.duckdb"), [])
        assert result == {}

    def test_missing_file_returns_empty(self, tmp_path):
        from src.data.services._teammates_perf_queries import (
            load_performance_scores_from_player_db,
        )

        result = load_performance_scores_from_player_db(
            str(tmp_path / "nonexistent.duckdb"), ["m1"]
        )
        assert result == {}

    def test_reads_scores_excludes_null(self, tmp_path):
        import duckdb

        from src.data.services._teammates_perf_queries import (
            load_performance_scores_from_player_db,
        )

        db_path = str(tmp_path / "stats.duckdb")
        with duckdb.connect(db_path) as conn:
            conn.execute(
                "CREATE TABLE player_match_enrichment "
                "(match_id VARCHAR, performance_score FLOAT)"
            )
            conn.execute(
                "INSERT INTO player_match_enrichment VALUES "
                "('m1', 71.0), ('m2', 55.0), ('m3', NULL)"
            )

        result = load_performance_scores_from_player_db(db_path, ["m1", "m2", "m3", "m4"])
        assert result == {"m1": pytest.approx(71.0), "m2": pytest.approx(55.0)}
        # m3 NULL exclu, m4 absent du tout

    def test_missing_table_returns_empty(self, tmp_path):
        import duckdb

        from src.data.services._teammates_perf_queries import (
            load_performance_scores_from_player_db,
        )

        db_path = str(tmp_path / "stats.duckdb")
        with duckdb.connect(db_path) as conn:
            conn.execute("CREATE TABLE sync_meta (key VARCHAR, value VARCHAR)")

        result = load_performance_scores_from_player_db(db_path, ["m1"])
        assert result == {}


# ============================================================================
# TeammatesService.enrich_with_performance_score
# ============================================================================


class TestEnrichWithPerformanceScore:
    def test_empty_df_returns_unchanged(self, tmp_path):
        from src.data.services.teammates_service import TeammatesService

        df = pl.DataFrame(schema={"match_id": pl.Utf8, "kills": pl.Int64})
        result = TeammatesService.enrich_with_performance_score(
            df, "Player1", str(tmp_path / "stats.duckdb"), is_main=True
        )
        assert result.is_empty()

    def test_no_match_id_column_returns_unchanged(self, tmp_path):
        from src.data.services.teammates_service import TeammatesService

        df = pl.DataFrame({"kills": [5, 3]})
        result = TeammatesService.enrich_with_performance_score(
            df, "Player1", str(tmp_path / "stats.duckdb")
        )
        assert "performance_score" not in result.columns

    def test_is_main_uses_reference_db_path(self, tmp_path):
        import duckdb

        from src.data.services.teammates_service import TeammatesService

        db_path = str(tmp_path / "stats.duckdb")
        with duckdb.connect(db_path) as conn:
            conn.execute(
                "CREATE TABLE player_match_enrichment "
                "(match_id VARCHAR, performance_score FLOAT)"
            )
            conn.execute("INSERT INTO player_match_enrichment VALUES ('m1', 71.0)")

        df = pl.DataFrame({"match_id": ["m1", "m2"], "kills": [5, 3]})
        result = TeammatesService.enrich_with_performance_score(
            df, "AnyName", db_path, is_main=True
        )

        assert "performance_score" in result.columns
        row_m1 = result.filter(pl.col("match_id") == "m1")["performance_score"][0]
        assert row_m1 == pytest.approx(71.0)
        row_m2 = result.filter(pl.col("match_id") == "m2")["performance_score"][0]
        assert row_m2 is None

    def test_friend_resolves_player_subdir(self, tmp_path):
        import duckdb

        from src.data.services.teammates_service import TeammatesService

        # Structure : tmp_path/Madina/stats.duckdb (référence)
        #             tmp_path/Chocoboflor/stats.duckdb (coéquipier)
        (tmp_path / "Chocoboflor").mkdir()
        friend_db = str(tmp_path / "Chocoboflor" / "stats.duckdb")
        with duckdb.connect(friend_db) as conn:
            conn.execute(
                "CREATE TABLE player_match_enrichment "
                "(match_id VARCHAR, performance_score FLOAT)"
            )
            conn.execute("INSERT INTO player_match_enrichment VALUES ('m1', 87.0)")

        reference_db = str(tmp_path / "Madina" / "stats.duckdb")
        df = pl.DataFrame({"match_id": ["m1"], "kills": [10]})
        result = TeammatesService.enrich_with_performance_score(df, "Chocoboflor", reference_db)

        assert "performance_score" in result.columns
        assert result["performance_score"][0] == pytest.approx(87.0)

    def test_missing_friend_db_returns_unchanged(self, tmp_path):
        from src.data.services.teammates_service import TeammatesService

        reference_db = str(tmp_path / "Madina" / "stats.duckdb")
        df = pl.DataFrame({"match_id": ["m1"], "kills": [5]})
        result = TeammatesService.enrich_with_performance_score(df, "UnknownFriend", reference_db)

        assert "performance_score" not in result.columns
        assert result.equals(df)


# ============================================================================
# _use_or_compute_performance
# ============================================================================


def _make_stats_df(match_ids: list[str], perf_scores: list[float | None] | None = None):
    """Construit un DataFrame de stats minimal pour les tests."""
    n = len(match_ids)
    data: dict = {
        "start_time": pl.Series([None] * n, dtype=pl.Datetime),
        "kills": [5] * n,
        "deaths": [2] * n,
        "assists": [1] * n,
        "ratio": [2.5] * n,
        "accuracy": [45.0] * n,
        "average_life_seconds": [60.0] * n,
        "time_played_seconds": [600.0] * n,
    }
    if perf_scores is not None:
        data["performance_score"] = pl.Series(perf_scores, dtype=pl.Float64)
    return pl.DataFrame(data)


class TestUseOrComputePerformance:
    def test_uses_stored_score_when_present(self):
        from src.ui.pages._teammates_trio_helpers import _use_or_compute_performance

        df = _make_stats_df(["m1", "m2"], perf_scores=[71.0, 55.0])
        result = _use_or_compute_performance(df)

        assert "performance" in result.columns
        assert result["performance"].to_list() == [71.0, 55.0]

    def test_all_null_performance_score_triggers_fallback(self):
        from src.ui.pages._teammates_trio_helpers import _use_or_compute_performance

        df = _make_stats_df(["m1", "m2"], perf_scores=[None, None])
        result = _use_or_compute_performance(df)

        # Fallback : colonne "performance" toujours créée
        assert "performance" in result.columns

    def test_fallback_without_performance_score_column(self):
        from src.ui.pages._teammates_trio_helpers import _use_or_compute_performance

        df = _make_stats_df(["m1", "m2"])
        result = _use_or_compute_performance(df)

        assert "performance" in result.columns

    def test_partial_null_uses_stored(self):
        """Si au moins une valeur est non-null, on utilise le stocké."""
        from src.ui.pages._teammates_trio_helpers import _use_or_compute_performance

        df = _make_stats_df(["m1", "m2"], perf_scores=[None, 65.0])
        result = _use_or_compute_performance(df)

        assert result["performance"].to_list() == [None, 65.0]


# ============================================================================
# _merge_trio_dataframes — propagation de performance_score
# ============================================================================


def _make_trio_df(match_ids: list[str], with_perf: bool = False) -> pl.DataFrame:
    """DataFrame complet pour le merge trio."""
    n = len(match_ids)
    data: dict = {
        "match_id": match_ids,
        "start_time": pl.Series([None] * n, dtype=pl.Datetime),
        "kills": [5] * n,
        "deaths": [2] * n,
        "assists": [1] * n,
        "accuracy": [45.0] * n,
        "ratio": [2.5] * n,
        "average_life_seconds": [60.0] * n,
        "time_played_seconds": [600.0] * n,
    }
    if with_perf:
        data["performance_score"] = [71.0] * n
    return pl.DataFrame(data)


class TestMergeTrioDataframesPerformanceScore:
    def test_propagated_for_all_when_present(self):
        from src.ui.pages._teammates_trio_helpers import _merge_trio_dataframes

        me = _make_trio_df(["m1", "m2"], with_perf=True)
        f1 = _make_trio_df(["m1", "m2"], with_perf=True)
        f2 = _make_trio_df(["m1", "m2"], with_perf=True)

        merged = _merge_trio_dataframes(me, f1, f2)

        assert not merged.is_empty()
        assert "performance_score" in merged.columns
        assert "f1_performance_score" in merged.columns
        assert "f2_performance_score" in merged.columns

    def test_merge_works_without_performance_score(self):
        from src.ui.pages._teammates_trio_helpers import _merge_trio_dataframes

        me = _make_trio_df(["m1", "m2"])
        f1 = _make_trio_df(["m1", "m2"])
        f2 = _make_trio_df(["m1", "m2"])

        merged = _merge_trio_dataframes(me, f1, f2)

        assert not merged.is_empty()
        assert "performance_score" not in merged.columns

    def test_partial_presence_handled(self):
        """me a performance_score, f1 non, f2 oui — merge ne doit pas échouer."""
        from src.ui.pages._teammates_trio_helpers import _merge_trio_dataframes

        me = _make_trio_df(["m1", "m2"], with_perf=True)
        f1 = _make_trio_df(["m1", "m2"], with_perf=False)
        f2 = _make_trio_df(["m1", "m2"], with_perf=True)

        merged = _merge_trio_dataframes(me, f1, f2)

        assert not merged.is_empty()
        assert "performance_score" in merged.columns
        assert "f1_performance_score" not in merged.columns
        assert "f2_performance_score" in merged.columns

    def test_extract_player_df_includes_performance_score(self):
        """_extract_player_df transmet performance_score au DataFrame joueur."""
        from src.ui.pages._teammates_trio_helpers import (
            _extract_player_df,
            _merge_trio_dataframes,
        )

        me = _make_trio_df(["m1"], with_perf=True)
        f1 = _make_trio_df(["m1"], with_perf=True)
        f2 = _make_trio_df(["m1"], with_perf=True)

        merged = _merge_trio_dataframes(me, f1, f2)
        d_self = _extract_player_df(merged, None)
        d_f1 = _extract_player_df(merged, "f1")

        assert "performance_score" in d_self.columns
        assert d_self["performance_score"][0] == pytest.approx(71.0)
        assert "performance_score" in d_f1.columns
        assert d_f1["performance_score"][0] == pytest.approx(71.0)
