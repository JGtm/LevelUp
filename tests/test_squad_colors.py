"""Tests des fonctionnalités Escouade/Squad : palette Okabe-Ito, couleurs, i18n, graphiques 4 joueurs.

Vérifie les ajouts réalisés lors du sprint Escouade :
- OKABE_ITO_PALETTE (format, unicité, longueur)
- assign_player_colors utilise bien Okabe-Ito
- plot_trio_metric avec 4 joueurs (d_f3)
- plot_trio_metric avec colors_by_name explicite
- i18n : renommage page_teammates → Escouade/Squad
- i18n : tm_squad_header

Exécution :
    pytest tests/test_squad_colors.py -v
"""

from __future__ import annotations

import re

import numpy as np
import pandas as pd
import plotly.graph_objects as go
import pytest

# =============================================================================
# FIXTURES
# =============================================================================


@pytest.fixture
def sample_df() -> pd.DataFrame:
    """DataFrame minimal avec colonnes essentielles pour plot_trio_metric."""
    np.random.seed(0)
    n = 10
    return pd.DataFrame(
        {
            "match_id": [f"m{i}" for i in range(n)],
            "start_time": pd.date_range("2025-01-01", periods=n, freq="h"),
            "kills": np.random.randint(2, 15, n).astype(float),
            "deaths": np.random.randint(1, 10, n).astype(float),
            "assists": np.random.randint(0, 8, n).astype(float),
            "accuracy": np.random.uniform(20, 60, n),
        }
    )


# =============================================================================
# TESTS : OKABE_ITO_PALETTE
# =============================================================================


class TestOkabeItoPalette:
    """Vérifie la palette Okabe-Ito dans src/config.py."""

    def test_palette_importable(self) -> None:
        """OKABE_ITO_PALETTE s'importe depuis src.config."""
        from src.config import OKABE_ITO_PALETTE

        assert OKABE_ITO_PALETTE is not None

    def test_palette_length(self) -> None:
        """La palette contient au moins 5 couleurs (moi + 3 coéquipiers + marge)."""
        from src.config import OKABE_ITO_PALETTE

        assert len(OKABE_ITO_PALETTE) >= 5

    def test_palette_unique_colors(self) -> None:
        """Toutes les couleurs sont distinctes."""
        from src.config import OKABE_ITO_PALETTE

        assert len(OKABE_ITO_PALETTE) == len(set(OKABE_ITO_PALETTE))

    def test_palette_hex_format(self) -> None:
        """Toutes les couleurs sont au format #RRGGBB."""
        from src.config import OKABE_ITO_PALETTE

        hex_pattern = re.compile(r"^#[0-9A-Fa-f]{6}$")
        for color in OKABE_ITO_PALETTE:
            assert hex_pattern.match(color), f"Couleur invalide : {color!r}"


# =============================================================================
# TESTS : assign_player_colors
# =============================================================================


class TestAssignPlayerColors:
    """Vérifie que assign_player_colors utilise la palette Okabe-Ito."""

    def test_first_color_is_okabe(self) -> None:
        """Le premier joueur reçoit OKABE_ITO_PALETTE[0]."""

        import streamlit as st

        from src.app.helpers import assign_player_colors
        from src.config import OKABE_ITO_PALETTE

        # Simuler session_state vide pour forcer le recalcul
        st.session_state["_os_player_colors"] = {}
        colors = assign_player_colors(["P1", "P2", "P3"])
        assert colors["P1"] == OKABE_ITO_PALETTE[0]

    def test_second_color_is_okabe(self) -> None:
        """Le second joueur reçoit OKABE_ITO_PALETTE[1]."""
        import streamlit as st

        from src.app.helpers import assign_player_colors
        from src.config import OKABE_ITO_PALETTE

        st.session_state["_os_player_colors"] = {}
        colors = assign_player_colors(["P1", "P2"])
        assert colors["P2"] == OKABE_ITO_PALETTE[1]

    def test_four_players_get_distinct_colors(self) -> None:
        """Quatre joueurs reçoivent des couleurs distinctes."""
        import streamlit as st

        from src.app.helpers import assign_player_colors

        st.session_state["_os_player_colors"] = {}
        colors = assign_player_colors(["P1", "P2", "P3", "P4"])
        assert len(set(colors.values())) == 4, "Les 4 joueurs doivent avoir des couleurs distinctes"

    def test_returns_all_players(self) -> None:
        """assign_player_colors retourne une entrée pour chaque joueur."""
        import streamlit as st

        from src.app.helpers import assign_player_colors

        st.session_state["_os_player_colors"] = {}
        names = ["Alice", "Bob", "Charlie"]
        colors = assign_player_colors(names)
        for n in names:
            assert n in colors


# =============================================================================
# TESTS : plot_trio_metric (4 joueurs)
# =============================================================================


class TestPlotTrioMetric4Players:
    """Vérifie les nouvelles fonctionnalités de plot_trio_metric."""

    def test_4_players_returns_figure(self, sample_df: pd.DataFrame) -> None:
        """plot_trio_metric avec d_f3 retourne une figure valide."""
        from src.visualization.trio import plot_trio_metric

        fig = plot_trio_metric(
            sample_df,
            sample_df.copy(),
            sample_df.copy(),
            d_f3=sample_df.copy(),
            metric="kills",
            names=("P1", "P2", "P3", "P4"),
            title="Test 4 joueurs",
            y_title="Kills",
        )
        assert isinstance(fig, go.Figure)

    def test_4_players_has_more_traces(self, sample_df: pd.DataFrame) -> None:
        """Avec d_f3, le graphique contient plus de traces qu'avec 3 joueurs."""
        from src.visualization.trio import plot_trio_metric

        fig3 = plot_trio_metric(
            sample_df,
            sample_df.copy(),
            sample_df.copy(),
            metric="kills",
            names=("P1", "P2", "P3"),
            title="3 joueurs",
            y_title="Kills",
        )
        fig4 = plot_trio_metric(
            sample_df,
            sample_df.copy(),
            sample_df.copy(),
            d_f3=sample_df.copy(),
            metric="kills",
            names=("P1", "P2", "P3", "P4"),
            title="4 joueurs",
            y_title="Kills",
        )
        # 4 joueurs → 2 traces par joueur (bar + ligne) + 1 moyenne = 9 traces
        # 3 joueurs → 7 traces. Le 4-joueurs doit avoir plus que le 3-joueurs.
        assert len(fig4.data) > len(fig3.data)

    def test_4_players_bar_count(self, sample_df: pd.DataFrame) -> None:
        """Avec 4 joueurs, il y a exactement 4 traces Bar."""
        from src.visualization.trio import plot_trio_metric

        fig = plot_trio_metric(
            sample_df,
            sample_df.copy(),
            sample_df.copy(),
            d_f3=sample_df.copy(),
            metric="kills",
            names=("P1", "P2", "P3", "P4"),
            title="Test",
            y_title="Kills",
        )
        bars = [t for t in fig.data if isinstance(t, go.Bar)]
        assert len(bars) == 4

    def test_colors_by_name_applied(self, sample_df: pd.DataFrame) -> None:
        """colors_by_name est bien appliqué aux barres."""
        from src.visualization.trio import plot_trio_metric

        custom_colors = {"P1": "#FF0000", "P2": "#00FF00", "P3": "#0000FF"}
        fig = plot_trio_metric(
            sample_df,
            sample_df.copy(),
            sample_df.copy(),
            metric="kills",
            names=("P1", "P2", "P3"),
            title="Test couleurs",
            y_title="Kills",
            colors_by_name=custom_colors,
        )
        bar_colors = [t.marker.color for t in fig.data if isinstance(t, go.Bar)]

        # marker.color peut être un tuple/liste quand toutes les barres d'une trace ont la même couleur
        def _first_color(c):
            return c[0] if isinstance(c, (tuple, list)) else c

        flat = [_first_color(c) for c in bar_colors]
        assert "#FF0000" in flat, "La couleur de P1 doit être #FF0000"
        assert "#00FF00" in flat, "La couleur de P2 doit être #00FF00"

    def test_colors_by_name_with_4_players(self, sample_df: pd.DataFrame) -> None:
        """colors_by_name fonctionne avec 4 joueurs."""
        from src.visualization.trio import plot_trio_metric

        custom_colors = {
            "P1": "#FF0000",
            "P2": "#00FF00",
            "P3": "#0000FF",
            "P4": "#FFFF00",
        }
        fig = plot_trio_metric(
            sample_df,
            sample_df.copy(),
            sample_df.copy(),
            d_f3=sample_df.copy(),
            metric="kills",
            names=("P1", "P2", "P3", "P4"),
            title="Test 4 couleurs",
            y_title="Kills",
            colors_by_name=custom_colors,
        )
        bar_colors = [t.marker.color for t in fig.data if isinstance(t, go.Bar)]

        # marker.color peut être un tuple/liste quand toutes les barres d'une trace ont la même couleur
        def _first_color(c):
            return c[0] if isinstance(c, (tuple, list)) else c

        flat = [_first_color(c) for c in bar_colors]
        assert "#FF0000" in flat
        assert "#FFFF00" in flat

    def test_d_f3_none_same_as_3_players(self, sample_df: pd.DataFrame) -> None:
        """Sans d_f3 (None), le comportement est identique à 3 joueurs."""
        from src.visualization.trio import plot_trio_metric

        fig = plot_trio_metric(
            sample_df,
            sample_df.copy(),
            sample_df.copy(),
            d_f3=None,
            metric="kills",
            names=("P1", "P2", "P3"),
            title="3 joueurs (d_f3=None)",
            y_title="Kills",
        )
        bars = [t for t in fig.data if isinstance(t, go.Bar)]
        assert len(bars) == 3


# =============================================================================
# TESTS : i18n Escouade/Squad
# =============================================================================


class TestSquadI18n:
    """Vérifie les traductions pour le renommage Teammates → Escouade/Squad."""

    def test_page_teammates_fr(self) -> None:
        """La clé page_teammates retourne 'Escouade' en français."""
        from src.ui.i18n import t

        assert t("page_teammates", lang="fr") == "Escouade"

    def test_page_teammates_en(self) -> None:
        """La clé page_teammates retourne 'Squad' en anglais."""
        from src.ui.i18n import t

        assert t("page_teammates", lang="en") == "Squad"

    def test_tm_squad_header_fr(self) -> None:
        """tm_squad_header contient 'Escouade' en français."""
        from src.ui.i18n import t

        result = t("tm_squad_header", names="Alice + Bob", lang="fr")
        assert "Escouade" in result
        assert "Alice + Bob" in result

    def test_tm_squad_header_en(self) -> None:
        """tm_squad_header contient 'Squad' en anglais."""
        from src.ui.i18n import t

        result = t("tm_squad_header", names="Alice + Bob", lang="en")
        assert "Squad" in result
        assert "Alice + Bob" in result

    def test_tm_trio_header_still_exists(self) -> None:
        """tm_trio_header est conservé pour rétrocompatibilité."""
        from src.ui.i18n import t

        result = t("tm_trio_header", names="A + B", lang="fr")
        assert result  # Non vide
