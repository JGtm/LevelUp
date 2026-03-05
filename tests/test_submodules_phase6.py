"""Tests dédiés pour les sous-modules Phase 6 (préfixés _).

Sous-modules testés :
- src/data/domain/_refdata_personal_scores.py
- src/data/repositories/_gamertag_resolver.py
- src/ui/_cache_sessions.py
- src/ui/_cache_loading.py
- src/app/_filters_apply.py
- src/ui/pages/_session_compare_history.py
"""

from __future__ import annotations

import logging
from unittest.mock import MagicMock

import polars as pl

# =============================================================================
# Tests _refdata_personal_scores.py
# =============================================================================


class TestRefdataPersonalScores:
    """Tests pour le sous-module _refdata_personal_scores."""

    def test_enum_killed_player(self):
        """PersonalScoreNameId.KILLED_PLAYER existe et a une valeur int."""
        from src.data.domain._refdata_personal_scores import PersonalScoreNameId

        assert PersonalScoreNameId.KILLED_PLAYER == 1024030246
        assert isinstance(PersonalScoreNameId.KILLED_PLAYER, int)

    def test_enum_betrayed_negative_points(self):
        """BETRAYED_PLAYER vaut -100 points."""
        from src.data.domain._refdata_personal_scores import (
            PersonalScoreNameId,
            get_personal_score_points,
        )

        pts = get_personal_score_points(PersonalScoreNameId.BETRAYED_PLAYER)
        assert pts == -100

    def test_display_name_known(self):
        """Nom affiché pour un type connu."""
        from src.data.domain._refdata_personal_scores import (
            PersonalScoreNameId,
            get_personal_score_display_name,
        )

        name = get_personal_score_display_name(PersonalScoreNameId.KILLED_PLAYER)
        assert isinstance(name, str)
        assert name != "Score"  # Pas le fallback

    def test_display_name_unknown(self):
        """Nom affiché pour un type inconnu → fallback 'Score'."""
        from src.data.domain._refdata_personal_scores import (
            get_personal_score_display_name,
        )

        name = get_personal_score_display_name(999999999)
        assert name == "Score"

    def test_technical_id_known(self):
        """ID technique pour KILLED_PLAYER."""
        from src.data.domain._refdata_personal_scores import (
            PersonalScoreNameId,
            get_personal_score_technical_id,
        )

        tech_id = get_personal_score_technical_id(PersonalScoreNameId.KILLED_PLAYER)
        assert tech_id == "killed_player"

    def test_technical_id_unknown_not_enum(self):
        """ID technique pour une valeur qui n'est pas dans l'enum → 'score'."""
        from src.data.domain._refdata_personal_scores import (
            get_personal_score_technical_id,
        )

        tech_id = get_personal_score_technical_id(999999999)
        assert tech_id == "score"

    def test_technical_id_accepts_int(self):
        """get_personal_score_technical_id accepte un int brut."""
        from src.data.domain._refdata_personal_scores import (
            get_personal_score_technical_id,
        )

        # Passer la valeur int directement, pas l'enum
        tech_id = get_personal_score_technical_id(1024030246)
        assert tech_id == "killed_player"

    def test_points_known(self):
        """Points pour FLAG_CAPTURED = 300."""
        from src.data.domain._refdata_personal_scores import (
            PersonalScoreNameId,
            get_personal_score_points,
        )

        pts = get_personal_score_points(PersonalScoreNameId.FLAG_CAPTURED)
        assert pts == 300

    def test_points_unknown(self):
        """Points pour type inconnu → 0."""
        from src.data.domain._refdata_personal_scores import (
            get_personal_score_points,
        )

        pts = get_personal_score_points(999999999)
        assert pts == 0

    def test_dicts_consistent(self):
        """PERSONAL_SCORE_TECHNICAL_IDS couvre tous les membres de l'enum."""
        from src.data.domain._refdata_personal_scores import (
            PERSONAL_SCORE_TECHNICAL_IDS,
            PersonalScoreNameId,
        )

        for member in PersonalScoreNameId:
            assert (
                int(member) in PERSONAL_SCORE_TECHNICAL_IDS
            ), f"{member.name} ({int(member)}) absent de PERSONAL_SCORE_TECHNICAL_IDS"

    def test_points_dict_consistent(self):
        """PERSONAL_SCORE_POINTS couvre tous les membres de l'enum."""
        from src.data.domain._refdata_personal_scores import (
            PERSONAL_SCORE_POINTS,
            PersonalScoreNameId,
        )

        for member in PersonalScoreNameId:
            assert (
                int(member) in PERSONAL_SCORE_POINTS
            ), f"{member.name} ({int(member)}) absent de PERSONAL_SCORE_POINTS"


# =============================================================================
# Tests _gamertag_resolver.py
# =============================================================================


class TestGamertagResolver:
    """Tests pour le sous-module _gamertag_resolver."""

    def test_clean_gamertag_valid(self):
        """Gamertag valide retourné tel quel."""
        from src.data.repositories._gamertag_resolver import _clean_gamertag_static

        assert _clean_gamertag_static("PlayerOne") == "PlayerOne"

    def test_clean_gamertag_none(self):
        """None → None."""
        from src.data.repositories._gamertag_resolver import _clean_gamertag_static

        assert _clean_gamertag_static(None) is None

    def test_clean_gamertag_empty(self):
        """Chaîne vide → None."""
        from src.data.repositories._gamertag_resolver import _clean_gamertag_static

        assert _clean_gamertag_static("") is None

    def test_clean_gamertag_xuid_prefix(self):
        """'xuid(...)' → None."""
        from src.data.repositories._gamertag_resolver import _clean_gamertag_static

        assert _clean_gamertag_static("xuid(12345)") is None

    def test_clean_gamertag_question_mark(self):
        """'?' → None."""
        from src.data.repositories._gamertag_resolver import _clean_gamertag_static

        assert _clean_gamertag_static("?") is None

    def test_clean_gamertag_digits_only(self):
        """Que des chiffres → None."""
        from src.data.repositories._gamertag_resolver import _clean_gamertag_static

        assert _clean_gamertag_static("123456") is None

    def test_clean_gamertag_strips_whitespace(self):
        """Trim des espaces autour."""
        from src.data.repositories._gamertag_resolver import _clean_gamertag_static

        assert _clean_gamertag_static("  Player  ") == "Player"

    def test_clean_gamertag_strips_control_chars(self):
        """Suppression des caractères de contrôle."""
        from src.data.repositories._gamertag_resolver import _clean_gamertag_static

        result = _clean_gamertag_static("Play\x00er")
        assert result == "Player"

    def test_mixin_has_logger(self):
        """Le module _gamertag_resolver a un logger configuré."""
        import src.data.repositories._gamertag_resolver as mod

        assert hasattr(mod, "logger")
        assert isinstance(mod.logger, logging.Logger)


# =============================================================================
# Tests _cache_loading.py
# =============================================================================


class TestCacheLoading:
    """Tests pour le sous-module _cache_loading."""

    def test_has_logger(self):
        """Le module _cache_loading possède un logger."""
        import src.ui._cache_loading as mod

        assert hasattr(mod, "logger")
        assert isinstance(mod.logger, logging.Logger)

    def test_match_columns_constant(self):
        """_MATCH_COLUMNS est une liste de noms de colonnes."""
        from src.ui._cache_loading import _MATCH_COLUMNS

        assert isinstance(_MATCH_COLUMNS, list)
        assert "match_id" in _MATCH_COLUMNS
        assert "kills" in _MATCH_COLUMNS
        assert "deaths" in _MATCH_COLUMNS
        assert len(_MATCH_COLUMNS) == 25

    def test_convert_timezone_empty_df(self):
        """_convert_timezone sur DataFrame vide ne plante pas."""
        from src.ui._cache_loading import _convert_timezone

        df = pl.DataFrame(
            {
                "start_time": pl.Series([], dtype=pl.Datetime("us", "UTC")),
            }
        )
        result = _convert_timezone(df)
        assert isinstance(result, pl.DataFrame)

    def test_matches_to_dataframe_empty(self):
        """_matches_to_dataframe avec liste vide."""
        from src.ui._cache_loading import _matches_to_dataframe

        result = _matches_to_dataframe([])
        assert isinstance(result, pl.DataFrame)
        assert result.is_empty()

    def test_matches_to_dataframe_mock(self):
        """_matches_to_dataframe avec un mock de MatchResult."""
        from src.ui._cache_loading import _matches_to_dataframe

        mock_match = MagicMock()
        mock_match.match_id = "test-id"
        mock_match.start_time = "2024-01-01T00:00:00Z"
        mock_match.map_id = "map1"
        mock_match.map_name = "Live Fire"
        mock_match.playlist_id = "pl1"
        mock_match.playlist_name = "Ranked"
        mock_match.map_mode_pair_id = "pair1"
        mock_match.map_mode_pair_name = "Slayer on Live Fire"
        mock_match.game_variant_id = "gv1"
        mock_match.game_variant_name = "Slayer"
        mock_match.outcome = 2
        mock_match.kda = 1.5
        mock_match.my_team_score = 50
        mock_match.enemy_team_score = 45
        mock_match.max_killing_spree = 5
        mock_match.headshot_kills = 3
        mock_match.average_life_seconds = 30.0
        mock_match.time_played_seconds = 600
        mock_match.kills = 10
        mock_match.deaths = 7
        mock_match.assists = 3
        mock_match.accuracy = 45.0
        mock_match.ratio = 1.43
        mock_match.team_mmr = 1200.0
        mock_match.enemy_mmr = 1190.0

        result = _matches_to_dataframe([mock_match])
        assert len(result) == 1
        assert result["match_id"][0] == "test-id"
        assert result["kills"][0] == 10

    def test_cached_functions_importable(self):
        """Les 3 fonctions publiques sont importables."""
        from src.ui._cache_loading import (
            cached_get_match_count_duckdb,
            cached_load_matches_paginated,
            cached_load_recent_matches,
        )

        assert callable(cached_load_recent_matches)
        assert callable(cached_load_matches_paginated)
        assert callable(cached_get_match_count_duckdb)


# =============================================================================
# Tests _cache_sessions.py
# =============================================================================


class TestCacheSessions:
    """Tests pour le sous-module _cache_sessions."""

    def test_has_logger(self):
        """Le module _cache_sessions possède un logger."""
        import src.ui._cache_sessions as mod

        assert hasattr(mod, "logger")
        assert isinstance(mod.logger, logging.Logger)

    def test_cached_compute_sessions_db_importable(self):
        """La fonction cached_compute_sessions_db est importable."""
        from src.ui._cache_sessions import cached_compute_sessions_db

        assert callable(cached_compute_sessions_db)


# =============================================================================
# Tests _filters_apply.py
# =============================================================================


class TestFiltersApply:
    """Tests pour le sous-module _filters_apply."""

    def test_has_logger(self):
        """Le module _filters_apply possède un logger."""
        import src.app._filters_apply as mod

        assert hasattr(mod, "logger")
        assert isinstance(mod.logger, logging.Logger)

    def test_apply_filters_importable(self):
        """apply_filters est importable depuis le sous-module."""
        from src.app._filters_apply import apply_filters

        assert callable(apply_filters)


# =============================================================================
# Tests _session_compare_history.py
# =============================================================================


class TestSessionCompareHistory:
    """Tests pour le sous-module _session_compare_history."""

    def test_render_session_history_table_importable(self):
        """render_session_history_table est importable."""
        from src.ui.pages._session_compare_history import (
            render_session_history_table,
        )

        assert callable(render_session_history_table)

    def test_build_history_dataframe_empty(self):
        """_build_history_dataframe avec DataFrame vide."""
        from src.ui.pages._session_compare_history import _build_history_dataframe

        df_sess = pl.DataFrame(
            {
                "match_id": pl.Series([], dtype=pl.Utf8),
                "kills": pl.Series([], dtype=pl.Int64),
                "deaths": pl.Series([], dtype=pl.Int64),
            }
        )
        result = _build_history_dataframe(df_sess)
        assert isinstance(result, tuple)
        assert len(result) == 2


# =============================================================================
# Tests d'intégrité des re-exports (façades)
# =============================================================================


class TestFacadeReExports:
    """Vérifie que les façades re-exportent correctement les sous-modules."""

    def test_refdata_reexports_personal_scores(self):
        """refdata.py re-exporte PersonalScoreNameId depuis le sous-module."""
        from src.data.domain._refdata_personal_scores import (
            PersonalScoreNameId as DirectImport,
        )
        from src.data.domain.refdata import PersonalScoreNameId

        assert PersonalScoreNameId is DirectImport

    def test_cache_filters_reexports_loading(self):
        """cache_filters.py re-exporte cached_load_recent_matches."""
        from src.ui._cache_loading import (
            cached_load_recent_matches as DirectImport,
        )
        from src.ui.cache_filters import cached_load_recent_matches

        assert cached_load_recent_matches is DirectImport

    def test_cache_filters_reexports_sessions(self):
        """cache_filters.py re-exporte cached_compute_sessions_db."""
        from src.ui._cache_sessions import (
            cached_compute_sessions_db as DirectImport,
        )
        from src.ui.cache_filters import cached_compute_sessions_db

        assert cached_compute_sessions_db is DirectImport

    def test_filters_render_reexports_apply(self):
        """filters_render.py re-exporte apply_filters."""
        from src.app._filters_apply import apply_filters as DirectImport
        from src.app.filters_render import apply_filters

        assert apply_filters is DirectImport

    def test_antagonist_charts_reexports_kv(self):
        """antagonist_charts.py re-exporte plot_killer_victim_stacked_bars."""
        from src.visualization._antagonist_kv import (
            plot_killer_victim_stacked_bars as DirectImport,
        )
        from src.visualization.antagonist_charts import (
            plot_killer_victim_stacked_bars,
        )

        assert plot_killer_victim_stacked_bars is DirectImport

    def test_antagonist_charts_reexports_duels(self):
        """antagonist_charts.py re-exporte plot_duel_history."""
        from src.visualization._antagonist_duels import (
            plot_duel_history as DirectImport,
        )
        from src.visualization.antagonist_charts import plot_duel_history

        assert plot_duel_history is DirectImport

    def test_rag_reexports_models(self):
        """rag.py re-exporte Document et RAGConfig."""
        from src.ai._rag_models import Document as DirectDoc
        from src.ai._rag_models import RAGConfig as DirectConfig
        from src.ai.rag import Document, RAGConfig

        assert Document is DirectDoc
        assert RAGConfig is DirectConfig

    def test_rag_reexports_chunker(self):
        """rag.py re-exporte TextChunker."""
        from src.ai._rag_chunker import TextChunker as DirectImport
        from src.ai.rag import TextChunker

        assert TextChunker is DirectImport

    def test_performance_reexports_relative(self):
        """performance_score.py re-exporte compute_performance_series."""
        from src.analysis._performance_relative import (
            compute_performance_series as DirectImport,
        )
        from src.analysis.performance_score import compute_performance_series

        assert compute_performance_series is DirectImport

    def test_performance_reexports_session(self):
        """performance_score.py re-exporte compute_session_performance_score_v2."""
        from src.analysis._performance_session import (
            compute_session_performance_score_v2 as DirectImport,
        )
        from src.analysis.performance_score import (
            compute_session_performance_score_v2,
        )

        assert compute_session_performance_score_v2 is DirectImport
