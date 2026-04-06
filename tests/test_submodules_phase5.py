"""Tests dédiés pour les sous-modules Phase 5 (préfixés _).

Sous-modules testés :
- src/analysis/_performance_relative.py
- src/analysis/_performance_session.py
- src/visualization/_antagonist_kv.py
- src/visualization/_antagonist_duels.py
- src/ai/_rag_models.py
- src/ai/_rag_github.py
- src/ai/_rag_chunker.py
- src/visualization/_maps_outcome_bullet.py  (Axe E1 — split maps_outcome >500L)
- src/visualization/_heatmap_squad.py        (Axe E2 — split friends_impact_heatmap >500L)
- src/visualization/_timeseries_helpers.py   (Axe D4 — helpers timeseries centralisés)
"""

from __future__ import annotations

import logging
from pathlib import Path

import polars as pl

# =============================================================================
# Tests _performance_relative.py
# =============================================================================


class TestPerformanceRelative:
    """Tests pour le sous-module _performance_relative."""

    def test_normalize_df_polars(self):
        """Vérifie que _normalize_df accepte un DataFrame Polars."""
        from src.analysis._performance_relative import _normalize_df

        df = pl.DataFrame({"a": [1, 2, 3]})
        result = _normalize_df(df)
        assert isinstance(result, pl.DataFrame)

    def test_normalize_df_already_polars(self):
        """Vérifie que _normalize_df laisse un Polars DataFrame tel quel."""
        from src.analysis._performance_relative import _normalize_df

        df = pl.DataFrame({"x": [1, 2]})
        result = _normalize_df(df)
        assert isinstance(result, pl.DataFrame)
        assert result.shape == (2, 1)

    def test_percentile_rank_basic(self):
        """Vérifie le calcul de percentile rank."""
        from src.analysis._performance_relative import _percentile_rank

        series = pl.Series([10, 20, 30, 40, 50])
        rank = _percentile_rank(30.0, series)
        assert 0.0 <= rank <= 100.0

    def test_percentile_rank_min(self):
        """Valeur inférieure à toutes → percentile faible."""
        from src.analysis._performance_relative import _percentile_rank

        series = pl.Series([10, 20, 30])
        rank = _percentile_rank(5.0, series)
        assert rank <= 20.0

    def test_percentile_rank_max(self):
        """Valeur supérieure à toutes → percentile élevé."""
        from src.analysis._performance_relative import _percentile_rank

        series = pl.Series([10, 20, 30])
        rank = _percentile_rank(35.0, series)
        assert rank >= 80.0

    def test_percentile_rank_inverse(self):
        """Vérifie que le percentile inversé fonctionne (critère plus bas = meilleur)."""
        from src.analysis._performance_relative import (
            _percentile_rank,
            _percentile_rank_inverse,
        )

        series = pl.Series([10, 20, 30, 40, 50])
        normal = _percentile_rank(10.0, series)
        inverse = _percentile_rank_inverse(10.0, series)
        # Pour une valeur basse, l'inverse devrait être plus élevé
        assert inverse > normal

    def test_safe_col_existing(self):
        """_safe_col retourne la colonne si elle existe."""
        from src.analysis._performance_relative import _safe_col

        df = pl.DataFrame({"kills": [5, 10]})
        expr = _safe_col(df, "kills", 0.0)
        assert expr is not None

    def test_compute_performance_series_empty_df(self):
        """compute_performance_series sur DataFrame vide → Series nulle."""
        from src.analysis._performance_relative import compute_performance_series

        df = pl.DataFrame({"kills": [], "deaths": [], "assists": []})
        result = compute_performance_series(df)
        assert len(result) == 0

    def test_compute_relative_performance_score_missing_data(self):
        """Score None si données essentielles absentes."""
        from src.analysis._performance_relative import (
            compute_relative_performance_score,
        )

        row = {"kills": None, "deaths": None}
        history = pl.DataFrame()
        result = compute_relative_performance_score(row, history)
        assert result is None

    def test_has_logger(self):
        """Le module _performance_relative possède un logger."""
        import src.analysis._performance_relative as mod

        assert hasattr(mod, "logger")
        assert isinstance(mod.logger, logging.Logger)


# =============================================================================
# Tests _performance_session.py
# =============================================================================


class TestPerformanceSession:
    """Tests pour le sous-module _performance_session."""

    def test_score_component_dataclass(self):
        """ScoreComponent instanciable avec les bons champs."""
        from src.analysis._performance_session import ScoreComponent

        comp = ScoreComponent(
            key="test",
            label="Test",
            weight=1.0,
            compute=lambda _df: (0.5, {}),
        )
        assert comp.key == "test"
        assert comp.weight == 1.0

    def test_v1_empty_df(self):
        """V1 sur DataFrame vide → score None."""
        from src.analysis._performance_session import (
            compute_session_performance_score_v1,
        )

        df = pl.DataFrame(
            {
                "kills": pl.Series([], dtype=pl.Int64),
                "deaths": pl.Series([], dtype=pl.Int64),
            }
        )
        result = compute_session_performance_score_v1(df)
        assert isinstance(result, dict)
        assert result.get("score") is None

    def test_v2_empty_df(self):
        """V2 sur DataFrame vide → score None."""
        from src.analysis._performance_session import (
            compute_session_performance_score_v2,
        )

        df = pl.DataFrame(
            {
                "kills": pl.Series([], dtype=pl.Int64),
                "deaths": pl.Series([], dtype=pl.Int64),
            }
        )
        result = compute_session_performance_score_v2(df)
        assert isinstance(result, dict)
        assert result.get("score") is None

    def test_v1_returns_expected_keys(self):
        """V1 retourne les clés standard du résultat."""
        from src.analysis._performance_session import (
            compute_session_performance_score_v1,
        )

        df = pl.DataFrame(
            {
                "kills": [10, 15, 8],
                "deaths": [5, 7, 3],
                "assists": [3, 4, 2],
                "outcome": [2, 2, 3],
                "accuracy": [45.0, 50.0, 40.0],
                "time_played_seconds": [600, 700, 500],
                "average_life_seconds": [30.0, 35.0, 25.0],
            }
        )
        result = compute_session_performance_score_v1(df)
        assert "score" in result
        # V1 retourne un dict avec score et composantes

    def test_v2_returns_expected_keys(self):
        """V2 retourne les clés standard du résultat."""
        from src.analysis._performance_session import (
            compute_session_performance_score_v2,
        )

        df = pl.DataFrame(
            {
                "kills": [10, 15, 8],
                "deaths": [5, 7, 3],
                "assists": [3, 4, 2],
                "outcome": [2, 2, 3],
                "accuracy": [45.0, 50.0, 40.0],
                "time_played_seconds": [600, 700, 500],
                "average_life_seconds": [30.0, 35.0, 25.0],
            }
        )
        result = compute_session_performance_score_v2(df)
        assert "score" in result
        assert "version" in result


# =============================================================================
# Tests _antagonist_kv.py
# =============================================================================


class TestAntagonistKV:
    """Tests pour le sous-module _antagonist_kv."""

    def test_plot_killer_victim_stacked_bars_empty(self):
        """Stacked bars avec DataFrame vide → Figure valide."""
        from src.visualization._antagonist_kv import plot_killer_victim_stacked_bars

        df = pl.DataFrame(
            {
                "player_gamertag": pl.Series([], dtype=pl.Utf8),
                "kills": pl.Series([], dtype=pl.Int64),
                "deaths": pl.Series([], dtype=pl.Int64),
            }
        )
        fig = plot_killer_victim_stacked_bars(df, "match-123")
        assert fig is not None

    def test_plot_kd_timeseries_empty(self):
        """KD timeseries avec DataFrame vide → Figure valide."""
        from src.visualization._antagonist_kv import plot_kd_timeseries

        df = pl.DataFrame(
            {
                "match_id": pl.Series([], dtype=pl.Utf8),
                "kills": pl.Series([], dtype=pl.Int64),
                "deaths": pl.Series([], dtype=pl.Int64),
            }
        )
        fig = plot_kd_timeseries(df)
        assert fig is not None

    def test_plot_killer_victim_heatmap_empty(self):
        """Heatmap avec DataFrame vide → Figure valide."""
        from src.visualization._antagonist_kv import plot_killer_victim_heatmap

        df = pl.DataFrame(
            {
                "killer": pl.Series([], dtype=pl.Utf8),
                "victim": pl.Series([], dtype=pl.Utf8),
                "count": pl.Series([], dtype=pl.Int64),
            }
        )
        fig = plot_killer_victim_heatmap(df)
        assert fig is not None


# =============================================================================
# Tests _antagonist_duels.py
# =============================================================================


class TestAntagonistDuels:
    """Tests pour le sous-module _antagonist_duels."""

    def test_get_antagonist_chart_colors(self):
        """Les couleurs retournent un dict non-vide."""
        from src.visualization._antagonist_duels import get_antagonist_chart_colors

        colors = get_antagonist_chart_colors()
        assert isinstance(colors, dict)
        assert len(colors) > 0

    def test_create_kd_indicator(self):
        """create_kd_indicator retourne une Figure Plotly."""
        from src.visualization._antagonist_duels import create_kd_indicator

        fig = create_kd_indicator(10, 5)
        assert fig is not None
        # Vérifier que c'est un objet Figure
        import plotly.graph_objects as go

        assert isinstance(fig, go.Figure)

    def test_plot_nemesis_victim_summary(self):
        """Nemesis/victim summary retourne une Figure."""
        from src.visualization._antagonist_duels import plot_nemesis_victim_summary

        nemesis_data = {
            "gamertag": "Nemesis1",
            "kills": 20,
            "deaths": 10,
            "matches": 5,
        }
        victim_data = {
            "gamertag": "Victim1",
            "kills": 5,
            "deaths": 15,
            "matches": 5,
        }
        fig = plot_nemesis_victim_summary(nemesis_data, victim_data)
        assert fig is not None


# =============================================================================
# Tests _rag_models.py
# =============================================================================


class TestRAGModels:
    """Tests pour le sous-module _rag_models."""

    def test_rag_config_defaults(self):
        """RAGConfig a des valeurs par défaut sensées."""
        from src.ai._rag_models import RAGConfig

        config = RAGConfig()
        assert config.chunk_size == 1000
        assert config.chunk_overlap == 200
        assert config.top_k == 5
        assert config.persist_directory == "data/rag"
        assert "*.py" in config.include_patterns
        assert "__pycache__" in config.exclude_patterns

    def test_rag_config_custom(self):
        """RAGConfig accepte des valeurs personnalisées."""
        from src.ai._rag_models import RAGConfig

        config = RAGConfig(chunk_size=500, top_k=10)
        assert config.chunk_size == 500
        assert config.top_k == 10

    def test_document_from_file(self):
        """Document.from_file crée un document avec les bons métadonnées."""
        from src.ai._rag_models import Document

        doc = Document.from_file(Path("src/test.py"), "print('hello')")
        assert doc.content == "print('hello')"
        assert doc.metadata["source"] == str(Path("src/test.py"))
        assert doc.metadata["source_type"] == "file"
        assert doc.metadata["filename"] == "test.py"
        assert doc.metadata["extension"] == ".py"
        assert doc.id  # Non-vide

    def test_document_from_github(self):
        """Document.from_github crée un document GitHub."""
        from src.ai._rag_models import Document

        doc = Document.from_github(
            "https://github.com/user/repo",
            "README.md",
            "# Hello",
        )
        assert doc.metadata["source_type"] == "github"
        assert doc.metadata["repo_url"] == "https://github.com/user/repo"
        assert doc.metadata["file_path"] == "README.md"
        assert doc.id  # Non-vide

    def test_document_unique_ids(self):
        """Deux documents différents ont des IDs différents."""
        from src.ai._rag_models import Document

        doc1 = Document.from_file(Path("a.py"), "content1")
        doc2 = Document.from_file(Path("b.py"), "content2")
        assert doc1.id != doc2.id

    def test_search_result(self):
        """SearchResult stocke les champs correctement."""
        from src.ai._rag_models import SearchResult

        sr = SearchResult(
            content="test",
            source="src/test.py",
            score=0.95,
            metadata={"key": "value"},
        )
        assert sr.score == 0.95
        assert sr.source == "src/test.py"


# =============================================================================
# Tests _rag_chunker.py
# =============================================================================


class TestRAGChunker:
    """Tests pour le sous-module _rag_chunker."""

    def test_chunk_short_text(self):
        """Texte court → un seul chunk."""
        from src.ai._rag_chunker import TextChunker

        chunker = TextChunker(chunk_size=1000)
        result = chunker.chunk_text("Short text")
        assert result == ["Short text"]

    def test_chunk_long_text(self):
        """Texte long → plusieurs chunks."""
        from src.ai._rag_chunker import TextChunker

        chunker = TextChunker(chunk_size=50, overlap=10)
        text = "A" * 200
        result = chunker.chunk_text(text)
        assert len(result) > 1

    def test_chunk_text_with_newlines(self):
        """Découpage préfère les sauts de ligne."""
        from src.ai._rag_chunker import TextChunker

        chunker = TextChunker(chunk_size=50, overlap=10)
        text = "Ligne 1\n" * 20
        result = chunker.chunk_text(text)
        assert len(result) > 1
        # Chaque chunk contient du contenu (pas vide)
        assert all(c.strip() for c in result)

    def test_chunk_python_code(self):
        """Découpage Python sépare par fonction."""
        from src.ai._rag_chunker import TextChunker

        chunker = TextChunker(chunk_size=200, overlap=20)
        code = '''def func_a():
    """Docstring."""
    return 1

def func_b():
    """Docstring."""
    x = 2
    y = 3
    return x + y

class MyClass:
    def method(self):
        pass
'''
        result = chunker.chunk_code(code, language="python")
        assert len(result) >= 1
        # Chaque chunk contient du code
        assert all(c.strip() for c in result)

    def test_chunk_non_python(self):
        """Langage non-Python fallback sur chunk_text."""
        from src.ai._rag_chunker import TextChunker

        chunker = TextChunker(chunk_size=50, overlap=10)
        text = "Some JavaScript code " * 20
        result = chunker.chunk_code(text, language="javascript")
        assert len(result) >= 1

    def test_custom_chunk_size(self):
        """Taille de chunk personnalisée respectée."""
        from src.ai._rag_chunker import TextChunker

        chunker = TextChunker(chunk_size=30, overlap=5)
        assert chunker.chunk_size == 30
        assert chunker.overlap == 5


# =============================================================================
# Tests _rag_github.py
# =============================================================================


class TestRAGGitHub:
    """Tests pour le sous-module _rag_github."""

    def test_github_indexer_init(self):
        """GitHubIndexer s'initialise avec une URL repo."""
        from src.ai._rag_github import GitHubIndexer

        indexer = GitHubIndexer("https://github.com/user/repo")
        assert indexer is not None

    def test_github_include_extensions(self):
        """Extensions incluses par défaut cohérentes."""
        from src.ai._rag_github import GitHubIndexer

        assert ".py" in GitHubIndexer.INCLUDE_EXTENSIONS
        assert ".md" in GitHubIndexer.INCLUDE_EXTENSIONS

    def test_github_exclude_patterns(self):
        """Patterns exclus contiennent __pycache__."""
        from src.ai._rag_github import GitHubIndexer

        assert "__pycache__" in GitHubIndexer.EXCLUDE_PATTERNS

    def test_has_logger(self):
        """Le module _rag_github possède un logger."""
        import src.ai._rag_github as mod

        assert hasattr(mod, "logger")
        assert isinstance(mod.logger, logging.Logger)


# =============================================================================
# Tests Axe E1 — _maps_outcome_bullet.py (split depuis maps_outcome.py >500L)
# =============================================================================


class TestMapsOutcomeBulletSubmodule:
    """Tests de non-régression pour le sous-module _maps_outcome_bullet (Axe E1)."""

    def test_importable_directly(self):
        """Le module _maps_outcome_bullet est importable directement."""
        from src.visualization import _maps_outcome_bullet  # noqa: F401

    def test_plot_map_winrate_bullet_reexported(self):
        """plot_map_winrate_bullet est réexportée depuis maps_outcome (interface publique préservée)."""
        from src.visualization.maps_outcome import plot_map_winrate_bullet

        assert callable(plot_map_winrate_bullet)

    def test_sort_by_map_order_reexported(self):
        """_sort_by_map_order est réexportée depuis maps_outcome (dépendance interne préservée)."""
        from src.visualization.maps_outcome import _sort_by_map_order

        assert callable(_sort_by_map_order)

    def test_sort_by_map_order_stable_on_known_order(self):
        """_sort_by_map_order trie selon map_order (descending=True par défaut → newest en haut)."""
        import polars as pl

        from src.visualization._maps_outcome_bullet import _sort_by_map_order

        df = pl.DataFrame(
            {"map_name": ["Deadlock", "Aquarius", "Recharge"], "win_rate": [0.4, 0.6, 0.5]}
        )
        # map_order = [oldest=Aquarius, ..., newest=Deadlock]
        # descending=True → Deadlock (pos 2) en haut, Aquarius (pos 0) en bas
        result = _sort_by_map_order(df, ["Aquarius", "Recharge", "Deadlock"])
        assert result["map_name"].to_list() == ["Deadlock", "Recharge", "Aquarius"]

    def test_sort_by_map_order_ascending(self):
        """_sort_by_map_order avec descending=False trie oldest d'abord."""
        import polars as pl

        from src.visualization._maps_outcome_bullet import _sort_by_map_order

        df = pl.DataFrame(
            {"map_name": ["Deadlock", "Aquarius", "Recharge"], "win_rate": [0.4, 0.6, 0.5]}
        )
        result = _sort_by_map_order(df, ["Aquarius", "Recharge", "Deadlock"], descending=False)
        assert result["map_name"].to_list() == ["Aquarius", "Recharge", "Deadlock"]

    def test_sort_by_map_order_unknown_maps_at_top(self):
        """Les cartes absentes de map_order reçoivent la position max (default)."""
        import polars as pl

        from src.visualization._maps_outcome_bullet import _sort_by_map_order

        df = pl.DataFrame({"map_name": ["Bazaar", "Aquarius"], "win_rate": [0.3, 0.7]})
        # descending=True → Bazaar (default pos = 2) en haut, Aquarius (pos 0) en bas
        result = _sort_by_map_order(df, ["Aquarius", "Deadlock"])
        maps = result["map_name"].to_list()
        assert maps[0] == "Bazaar"  # inconnu → position max → en haut avec descending
        assert "Aquarius" in maps

    def test_plot_winrate_bullet_returns_none_on_empty(self):
        """plot_map_winrate_bullet retourne None si aucune carte commune."""
        import polars as pl

        from src.visualization._maps_outcome_bullet import plot_map_winrate_bullet

        empty = pl.DataFrame(
            {
                "map_name": pl.Series([], dtype=pl.Utf8),
                "win_rate": pl.Series([], dtype=pl.Float64),
                "matches": pl.Series([], dtype=pl.Int64),
            }
        )
        result = plot_map_winrate_bullet(empty, empty)
        assert result is None


# =============================================================================
# Tests Axe E2 — _heatmap_squad.py (split depuis friends_impact_heatmap.py >500L)
# =============================================================================


class TestHeatmapSquadSubmodule:
    """Tests de non-régression pour le sous-module _heatmap_squad (Axe E2)."""

    def test_importable_directly(self):
        """Le module _heatmap_squad est importable directement."""
        from src.visualization import _heatmap_squad  # noqa: F401

    def test_plot_squad_map_heatmap_reexported(self):
        """plot_squad_map_heatmap est réexportée depuis friends_impact_heatmap (interface publique préservée)."""
        from src.visualization.friends_impact_heatmap import plot_squad_map_heatmap

        assert callable(plot_squad_map_heatmap)

    def test_plot_squad_map_heatmap_returns_none_on_empty_series(self):
        """plot_squad_map_heatmap retourne None si series vide."""
        from src.visualization._heatmap_squad import plot_squad_map_heatmap

        result = plot_squad_map_heatmap([])
        assert result is None

    def test_top_maps_by_frequency_basic(self):
        """_top_maps_by_frequency retourne les cartes les plus jouées."""
        import polars as pl

        from src.visualization._heatmap_squad import _top_maps_by_frequency

        bd = pl.DataFrame(
            {
                "map_name": ["Aquarius", "Aquarius", "Recharge", "Deadlock"],
                "matches": [10, 10, 5, 2],
            }
        )
        result = _top_maps_by_frequency(bd)
        assert result[0] == "Aquarius"  # la plus jouée en premier

    def test_discrete_perf_colorscale_valid_format(self):
        """_discrete_perf_colorscale retourne une liste de paires [float, str]."""
        from src.visualization._heatmap_squad import _discrete_perf_colorscale

        cs = _discrete_perf_colorscale()
        assert isinstance(cs, list)
        assert len(cs) >= 2
        for entry in cs:
            assert len(entry) == 2
            assert isinstance(entry[0], float)
            assert isinstance(entry[1], str)


# =============================================================================
# Tests Axe D4 — _timeseries_helpers.py (helpers centralisés)
# =============================================================================


class TestTimeseriesHelpersSubmodule:
    """Tests de non-régression pour _timeseries_helpers (Axe D4)."""

    def test_importable_directly(self):
        """Le module _timeseries_helpers est importable directement."""
        from src.visualization import _timeseries_helpers  # noqa: F401

    def test_rolling_mean_importable(self):
        """_rolling_mean est accessible depuis _timeseries_helpers."""
        from src.visualization._timeseries_helpers import _rolling_mean

        assert callable(_rolling_mean)

    def test_normalize_df_importable(self):
        """_normalize_df est accessible depuis _timeseries_helpers."""
        from src.visualization._timeseries_helpers import _normalize_df

        assert callable(_normalize_df)

    def test_rolling_mean_polars_series(self):
        """_rolling_mean retourne une Series Polars de même longueur."""
        import polars as pl

        from src.visualization._timeseries_helpers import _rolling_mean

        s = pl.Series([1.0, 2.0, 3.0, 4.0, 5.0])
        result = _rolling_mean(s, window=3)
        assert isinstance(result, pl.Series)
        assert len(result) == len(s)

    def test_normalize_df_passthrough_polars(self):
        """_normalize_df accepte un pl.DataFrame et le retourne sans erreur."""
        import polars as pl

        from src.visualization._timeseries_helpers import _normalize_df

        df = pl.DataFrame({"x": [1, 2, 3], "y": [4.0, 5.0, 6.0]})
        result = _normalize_df(df)
        assert isinstance(result, pl.DataFrame)
        assert result.shape == (3, 2)
