"""Tests de contrats de performance Sprint 19.

Vérifie que les optimisations S19 sont correctement en place :
- Data path DuckDB → Polars zero-copy (tâche 19.1)
- Projection de colonnes (tâche 19.3)
- Cache invalidation unifiée (tâche 19.4)
- Scattergl conditionnel (tâche 19.5)
"""

from __future__ import annotations

from datetime import datetime, timezone
from pathlib import Path

import duckdb
import polars as pl
import pytest

# XUID utilisé pour tous les tests nécessitant un repo v6
SAMPLE_XUID = "xuid_sample_perf19"

# ─────────────────────────────────────────────────────────────────────
# Fixtures
# ─────────────────────────────────────────────────────────────────────


@pytest.fixture
def sample_duckdb(tmp_path):
    """Crée une structure v6 : player DB minimale + shared DB avec 20 matchs.

    Returns:
        Tuple (player_db_path, shared_db_path).
    """
    # Player DB minimale (enrichissements uniquement en v6)
    player_path = str(tmp_path / "player" / "stats.duckdb")
    Path(player_path).parent.mkdir(parents=True, exist_ok=True)
    conn = duckdb.connect(player_path)
    conn.close()

    # Shared DB : match_registry + match_participants + mv_player_matches
    shared_path = str(tmp_path / "shared" / "shared_matches_v2.duckdb")
    Path(shared_path).parent.mkdir(parents=True, exist_ok=True)
    conn = duckdb.connect(shared_path)

    conn.execute("""
        CREATE TABLE match_registry (
            match_id VARCHAR PRIMARY KEY,
            start_time TIMESTAMP WITH TIME ZONE,
            map_id VARCHAR, map_name VARCHAR,
            playlist_id VARCHAR, playlist_name VARCHAR,
            pair_id VARCHAR, pair_name VARCHAR,
            game_variant_id VARCHAR, game_variant_name VARCHAR,
            team_0_score INTEGER DEFAULT 0, team_1_score INTEGER DEFAULT 0,
            duration_seconds INTEGER DEFAULT 300,
            is_firefight BOOLEAN DEFAULT FALSE, is_ranked BOOLEAN DEFAULT FALSE
        )
    """)
    conn.execute("""
        CREATE TABLE match_participants (
            match_id VARCHAR, xuid VARCHAR,
            outcome INTEGER, team_id INTEGER,
            rank INTEGER DEFAULT 1,
            kills INTEGER, deaths INTEGER, assists INTEGER,
            score INTEGER DEFAULT 0,
            shots_fired INTEGER DEFAULT 0, shots_hit INTEGER DEFAULT 0,
            PRIMARY KEY (match_id, xuid)
        )
    """)

    for i in range(20):
        conn.execute(
            "INSERT INTO match_registry VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
            [
                f"match_{i:03d}",
                datetime(2025, 1, 1 + i, 12, 0, 0, tzinfo=timezone.utc),
                f"map_{i % 3}",
                f"Map {i % 3}",
                f"playlist_{i % 2}",
                f"Playlist {i % 2}",
                f"pair_{i % 4}",
                f"Pair {i % 4}",
                f"gv_{i % 2}",
                f"GameVariant {i % 2}",
                50 + i,
                45 + i,
                300 + i * 10,
                False,
                False,
            ],
        )
        conn.execute(
            "INSERT INTO match_participants VALUES (?,?,?,?,?,?,?,?,?,?,?)",
            [
                f"match_{i:03d}",
                SAMPLE_XUID,
                2 if i % 3 != 0 else 3,
                i % 2,
                i % 8 + 1,  # rank
                10 + i,
                5 + i % 3,
                3 + i % 2,
                100 + i * 10,
                200 + i,
                100 + i,
            ],
        )

    conn.execute("""
        CREATE VIEW mv_player_matches AS
        SELECT
            r.match_id, r.start_time,
            r.map_id, r.map_name, r.playlist_id, r.playlist_name,
            r.pair_id, r.pair_name, r.game_variant_id, r.game_variant_name,
            p.xuid, p.outcome, p.team_id,
            CASE WHEN p.deaths > 0
                THEN (CAST(p.kills AS DOUBLE) + CAST(p.assists AS DOUBLE) / 3.0)
                     / CAST(p.deaths AS DOUBLE)
                ELSE CAST(p.kills AS DOUBLE) + CAST(p.assists AS DOUBLE) / 3.0
            END AS kda,
            0 AS max_killing_spree, 0 AS headshot_kills,
            0.0 AS avg_life_seconds,
            CAST(COALESCE(r.duration_seconds, 0) AS DOUBLE) AS time_played_seconds,
            COALESCE(p.kills, 0) AS kills,
            COALESCE(p.deaths, 0) AS deaths,
            COALESCE(p.assists, 0) AS assists,
            CASE WHEN p.shots_fired > 0
                THEN CAST(p.shots_hit AS DOUBLE) * 100.0 / CAST(p.shots_fired AS DOUBLE)
                ELSE NULL END AS accuracy,
            CASE WHEN p.team_id = 0 THEN r.team_0_score ELSE r.team_1_score END AS my_team_score,
            CASE WHEN p.team_id = 0 THEN r.team_1_score ELSE r.team_0_score END AS enemy_team_score,
            NULL AS team_mmr, NULL AS enemy_mmr,
            CAST(p.score AS INTEGER) AS personal_score,
            COALESCE(r.is_firefight, FALSE) AS is_firefight,
            COALESCE(r.is_ranked, FALSE) AS is_ranked,
            NULL::VARCHAR AS map_name_fr,
            NULL::VARCHAR AS playlist_name_fr,
            NULL::VARCHAR AS pair_name_fr,
            NULL::VARCHAR AS game_variant_name_fr
        FROM match_registry r
        JOIN match_participants p ON r.match_id = p.match_id
    """)
    conn.close()

    return player_path, shared_path


# ─────────────────────────────────────────────────────────────────────
# 19.1 — Data path DuckDB → Polars zero-copy
# ─────────────────────────────────────────────────────────────────────


class TestZeroCopyArrowPath:
    """Vérifie que load_matches_as_polars retourne un DataFrame Polars directement."""

    def test_load_matches_as_polars_returns_dataframe(self, sample_duckdb):
        """La méthode retourne un pl.DataFrame, pas une list[MatchRow]."""
        from src.data.repositories.duckdb_repo import DuckDBRepository

        player_db, shared_db = sample_duckdb
        repo = DuckDBRepository(
            player_db, xuid=SAMPLE_XUID, shared_db_path=shared_db, read_only=True
        )
        try:
            df = repo.load_matches_as_polars(include_firefight=True)
            assert isinstance(df, pl.DataFrame)
            assert df.height == 20
        finally:
            repo.close()

    def test_load_matches_as_polars_has_ratio(self, sample_duckdb):
        """ratio est un alias de kda (valeur API directe, pas recalculée)."""
        from src.data.repositories.duckdb_repo import DuckDBRepository

        player_db, shared_db = sample_duckdb
        repo = DuckDBRepository(
            player_db, xuid=SAMPLE_XUID, shared_db_path=shared_db, read_only=True
        )
        try:
            df = repo.load_matches_as_polars()
            assert "ratio" in df.columns
            assert "kda" in df.columns
            # ratio doit être identique à kda — même source (Fix 4 backlog)
            first = df.row(0, named=True)
            assert first["ratio"] == first["kda"]
        finally:
            repo.close()

    def test_load_matches_as_polars_columns_aligned_with_legacy(self, sample_duckdb):
        """Les colonnes essentielles sont identiques au chemin legacy."""
        from src.data.repositories.duckdb_repo import DuckDBRepository

        player_db, shared_db = sample_duckdb
        repo = DuckDBRepository(
            player_db, xuid=SAMPLE_XUID, shared_db_path=shared_db, read_only=True
        )
        try:
            df = repo.load_matches_as_polars()
            # Vérifier les colonnes essentielles
            essential = {
                "match_id",
                "start_time",
                "map_id",
                "map_name",
                "playlist_id",
                "playlist_name",
                "pair_id",
                "pair_name",
                "outcome",
                "kda",
                "kills",
                "deaths",
                "assists",
                "average_life_seconds",
                "time_played_seconds",
                "ratio",
            }
            assert essential.issubset(set(df.columns))
        finally:
            repo.close()

    def test_avg_life_seconds_renamed(self, sample_duckdb):
        """avg_life_seconds est renommé en average_life_seconds pour compat."""
        from src.data.repositories.duckdb_repo import DuckDBRepository

        player_db, shared_db = sample_duckdb
        repo = DuckDBRepository(
            player_db, xuid=SAMPLE_XUID, shared_db_path=shared_db, read_only=True
        )
        try:
            df = repo.load_matches_as_polars()
            assert "average_life_seconds" in df.columns
            assert "avg_life_seconds" not in df.columns
        finally:
            repo.close()

    def test_kills_deaths_coalesced_to_zero(self, sample_duckdb):
        """kills et deaths ne contiennent pas de NULL (COALESCE appliqué)."""
        from src.data.repositories.duckdb_repo import DuckDBRepository

        player_db, shared_db = sample_duckdb
        repo = DuckDBRepository(
            player_db, xuid=SAMPLE_XUID, shared_db_path=shared_db, read_only=True
        )
        try:
            df = repo.load_matches_as_polars()
            assert df["kills"].null_count() == 0
            assert df["deaths"].null_count() == 0
            assert df["assists"].null_count() == 0
        finally:
            repo.close()


# ─────────────────────────────────────────────────────────────────────
# 19.3 — Projection de colonnes
# ─────────────────────────────────────────────────────────────────────


class TestColumnProjection:
    """Vérifie que la projection de colonnes fonctionne."""

    def test_projection_reduces_columns(self, sample_duckdb):
        """Passer columns= réduit le nombre de colonnes retournées."""
        from src.data.repositories.duckdb_repo import DuckDBRepository

        player_db, shared_db = sample_duckdb
        repo = DuckDBRepository(
            player_db, xuid=SAMPLE_XUID, shared_db_path=shared_db, read_only=True
        )
        try:
            df_full = repo.load_matches_as_polars()
            df_proj = repo.load_matches_as_polars(columns=["match_id", "kills", "deaths", "ratio"])
            assert df_proj.columns == ["match_id", "kills", "deaths", "ratio"]
            assert df_proj.height == df_full.height
        finally:
            repo.close()

    def test_projection_ignores_unknown_columns(self, sample_duckdb):
        """Les colonnes inconnues sont ignorées silencieusement."""
        from src.data.repositories.duckdb_repo import DuckDBRepository

        player_db, shared_db = sample_duckdb
        repo = DuckDBRepository(
            player_db, xuid=SAMPLE_XUID, shared_db_path=shared_db, read_only=True
        )
        try:
            df = repo.load_matches_as_polars(columns=["match_id", "nonexistent_col"])
            assert "match_id" in df.columns
            assert "nonexistent_col" not in df.columns
        finally:
            repo.close()

    def test_column_constants_defined(self):
        """Les constantes COLUMNS_COMMON et COLUMNS_COMPUTED sont définies."""
        from src.ui.cache_loaders import COLUMNS_COMMON, COLUMNS_COMPUTED

        assert isinstance(COLUMNS_COMMON, list)
        assert isinstance(COLUMNS_COMPUTED, list)
        assert "match_id" in COLUMNS_COMMON
        assert "kills" in COLUMNS_COMMON
        # ratio est calculé dans load_matches_as_polars (couche repo) → COLUMNS_COMMON
        assert "ratio" in COLUMNS_COMMON
        assert "game_variant_name" in COLUMNS_COMMON  # utilisé dans match_view.py
        assert "date" in COLUMNS_COMPUTED


# ─────────────────────────────────────────────────────────────────────
# 19.4 — Cache invalidation unifiée
# ─────────────────────────────────────────────────────────────────────


class TestCacheInvalidation:
    """Vérifie que l'invalidation cache est cohérente."""

    def test_db_cache_key_returns_tuple(self, sample_duckdb):
        """db_cache_key retourne un tuple non-vide."""
        from src.ui.cache_loaders import db_cache_key

        player_db, _shared_db = sample_duckdb
        key = db_cache_key(player_db)
        assert key is not None
        assert isinstance(key, tuple)
        assert len(key) >= 2  # v5 : 4-tuple (mtime_player, size_player, mtime_shared, size_shared)
        assert key[0] > 0  # mtime_ns player DB

    def test_db_cache_key_none_for_missing(self):
        """db_cache_key retourne None pour un fichier inexistant."""
        from src.ui.cache_loaders import db_cache_key

        key = db_cache_key("/nonexistent/path.duckdb")
        assert key is None

    def test_state_delegates_to_cache_loaders(self, sample_duckdb):
        """db_cache_key retourne un tuple cohérent."""
        from src.ui.cache_loaders import db_cache_key

        player_db, _shared_db = sample_duckdb
        key = db_cache_key(player_db)
        assert key is not None
        assert isinstance(key, tuple)


# ─────────────────────────────────────────────────────────────────────
# 19.5 — Scattergl conditionnel
# ─────────────────────────────────────────────────────────────────────


class TestSmartScatter:
    """Vérifie que smart_scatter bascule en WebGL au-delà du seuil."""

    def test_small_data_returns_scatter(self):
        """Avec peu de points, retourne go.Scatter (SVG)."""
        import plotly.graph_objects as go

        from src.visualization._compat import smart_scatter

        trace = smart_scatter(x=list(range(100)), y=list(range(100)), mode="lines")
        assert isinstance(trace, go.Scatter)

    def test_large_data_returns_scattergl(self):
        """Avec beaucoup de points (>=500), retourne go.Scattergl (WebGL)."""
        import plotly.graph_objects as go

        from src.visualization._compat import smart_scatter

        large_x = list(range(1000))
        large_y = list(range(1000))
        trace = smart_scatter(x=large_x, y=large_y, mode="lines")
        assert isinstance(trace, go.Scattergl)

    def test_threshold_boundary(self):
        """Exactement au seuil (500), bascule en Scattergl."""
        import plotly.graph_objects as go

        from src.visualization._compat import smart_scatter

        trace = smart_scatter(x=list(range(500)), y=list(range(500)), mode="lines")
        assert isinstance(trace, go.Scattergl)

    def test_below_threshold(self):
        """Juste en dessous du seuil (499), reste en Scatter."""
        import plotly.graph_objects as go

        from src.visualization._compat import smart_scatter

        trace = smart_scatter(x=list(range(499)), y=list(range(499)), mode="lines")
        assert isinstance(trace, go.Scatter)

    def test_smart_scatter_preserves_kwargs(self):
        """Les kwargs sont passés intacts au constructeur."""
        from src.visualization._compat import smart_scatter

        trace = smart_scatter(
            x=[1, 2, 3],
            y=[4, 5, 6],
            mode="lines+markers",
            name="test",
            line={"width": 2, "color": "red"},
        )
        assert trace.name == "test"
        assert trace.mode == "lines+markers"

    def test_timeseries_uses_smart_scatter(self):
        """Vérifie que timeseries.py utilise smart_scatter au lieu de go.Scatter."""
        import inspect

        from src.visualization import timeseries

        source = inspect.getsource(timeseries)

        # smart_scatter est utilisé
        assert "smart_scatter" in source
        # go.Scatter direct n'est plus utilisé (sauf dans les imports)
        lines = source.split("\n")
        scatter_direct = [
            line for line in lines if "go.Scatter(" in line and "smart_scatter" not in line
        ]
        assert (
            len(scatter_direct) == 0
        ), f"go.Scatter() encore utilisé directement : {scatter_direct}"


# ─────────────────────────────────────────────────────────────────────
# Enrichissement DataFrame
# ─────────────────────────────────────────────────────────────────────


class TestEnrichMatchesDf:
    """Vérifie la fonction _enrich_matches_df extraite en S19."""

    def test_enrich_adds_computed_columns(self, sample_duckdb):
        """_enrich_matches_df ajoute date, kills_per_min, etc."""
        from src.data.repositories.duckdb_repo import DuckDBRepository
        from src.ui.cache_loaders import _enrich_matches_df

        player_db, shared_db = sample_duckdb
        repo = DuckDBRepository(
            player_db, xuid=SAMPLE_XUID, shared_db_path=shared_db, read_only=True
        )
        try:
            df_raw = repo.load_matches_as_polars()
            df_enriched = _enrich_matches_df(df_raw)

            assert "date" in df_enriched.columns
            assert "kills_per_min" in df_enriched.columns
            assert "deaths_per_min" in df_enriched.columns
            assert "assists_per_min" in df_enriched.columns
        finally:
            repo.close()

    def test_enrich_empty_df(self):
        """_enrich_matches_df gère un DataFrame vide."""
        from src.ui.cache_loaders import _enrich_matches_df

        df = pl.DataFrame()
        result = _enrich_matches_df(df)
        assert result.is_empty()

    def test_enrich_timezone_conversion(self, sample_duckdb):
        """start_time est converti en timezone Paris (naïf)."""
        from src.data.repositories.duckdb_repo import DuckDBRepository
        from src.ui.cache_loaders import _enrich_matches_df

        player_db, shared_db = sample_duckdb
        repo = DuckDBRepository(
            player_db, xuid=SAMPLE_XUID, shared_db_path=shared_db, read_only=True
        )
        try:
            df = repo.load_matches_as_polars()
            df_enriched = _enrich_matches_df(df)

            # Le type doit être Datetime naïf (sans timezone)
            st_dtype = df_enriched.schema["start_time"]
            assert st_dtype == pl.Datetime("us") or "UTC" not in str(st_dtype)
        finally:
            repo.close()
