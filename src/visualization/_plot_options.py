"""Options de rendu et thème des graphiques Halo — couche d'abstraction V3.

Architecture 4 couches :
    1. Data        : pl.DataFrame depuis DuckDB
    2. Chart Model : ChartData / SquadRecordSet (V1)
    3. Options     : PlotOptions + ChartTheme (ce module — V3)
    4. Render      : render_chart_or_info dans chart_utils (V3)

Utilisation typique :
    opts = PlotOptions.from_session()
    fig = plot_mon_graphe(data, opts=opts)
"""

from __future__ import annotations

from dataclasses import dataclass, field

from src.config import HALO_COLORS, OKABE_ITO_PALETTE, THEME_COLORS


@dataclass(frozen=True)
class ChartTheme:
    """Thème visuel centralisé pour tous les graphiques Plotly.

    Remplace les constantes éparpillées :
        - PERFORMANCE_COLORS (_perf_cumulative.py)
        - OBJECTIVE_COLORS   (objective_charts_extra.py)
        - LEAD_MY_BG / LEAD_ENEMY_BG (_team_dominance_helpers.py)
        - avg_color inline (trio.py)
        - zeroline inline (trio.py, timeseries_combat.py)
        - _PATTERN_* (_squad_record_shapes.py)
    """

    # ------------------------------------------------------------------
    # Fond, texte, grille
    # ------------------------------------------------------------------
    bg_plot: str = field(default_factory=lambda: THEME_COLORS.bg_plot_rgba(1.0))
    font_color: str = field(default_factory=lambda: THEME_COLORS.text_primary)
    grid_color: str = "rgba(255,255,255,0.07)"
    border_color: str = field(default_factory=lambda: THEME_COLORS.border)

    # ------------------------------------------------------------------
    # Joueur principal et palette escouade
    # ------------------------------------------------------------------
    solo_color: str = field(default_factory=lambda: HALO_COLORS.cyan)
    squad_palette: tuple[str, ...] = field(default_factory=lambda: tuple(OKABE_ITO_PALETTE))

    # ------------------------------------------------------------------
    # Sémantique performance (Okabe-Ito — accessibilité daltonisme)
    # ------------------------------------------------------------------
    color_positive: str = field(default_factory=lambda: HALO_COLORS.green)
    color_negative: str = field(default_factory=lambda: HALO_COLORS.red)
    color_kills: str = "#009E73"  # Vert bleuté Okabe-Ito
    color_deaths: str = "#D55E00"  # Vermillon Okabe-Ito
    color_kd: str = "#E69F00"  # Orange Okabe-Ito
    color_baseline: str = "#999999"  # Gris neutre
    color_rolling: str = "#CC79A7"  # Rose mauve Okabe-Ito
    color_trend_up: str = "#009E73"  # Okabe-Ito green
    color_trend_down: str = "#D55E00"  # Okabe-Ito vermillon

    # ------------------------------------------------------------------
    # Références visuelles (moyennes, annotations)
    # ------------------------------------------------------------------
    avg_color: str = "rgba(255,255,255,0.55)"
    avg_dash: str = "dot"
    baseline_color: str = "rgba(255,255,255,0.35)"
    annotation_color: str = "#E69F00"  # unifie #ffaa00 / #E69F00

    # ------------------------------------------------------------------
    # Axe zéro (ligne gras blanc)
    # ------------------------------------------------------------------
    zero_line_color: str = "rgba(255,255,255,0.75)"
    zero_line_width: int = 2

    # ------------------------------------------------------------------
    # Records — barres fantômes hachurées (_squad_record_shapes.py)
    # ------------------------------------------------------------------
    record_pattern_shape: str = "/"
    record_pattern_opacity: float = 0.55
    record_pattern_size: int = 8
    record_pattern_solidity: float = 0.35
    record_border_width: float = 2.0

    # ------------------------------------------------------------------
    # Zones win / loss (team_dominance)
    # ------------------------------------------------------------------
    zone_win_bg: str = "rgba(0,158,115,0.22)"
    zone_loss_bg: str = "rgba(213,94,0,0.22)"


# Singleton — import direct pour éviter de reconstruire le dataclass partout
DEFAULT_THEME: ChartTheme = ChartTheme()


@dataclass
class PlotOptions:
    """Options de rendu d'un graphique, passées aux fonctions plot_*.

    Remplace les paramètres style éparpillés (height, lang, smooth…)
    par un objet unique typé, compatible avec PLR0913.
    """

    lang: str = "fr"
    smooth: bool = False
    height_px: int = 360
    show_records: bool = True
    is_negative: bool = False
    theme: ChartTheme = field(default_factory=lambda: DEFAULT_THEME)

    @classmethod
    def from_session(cls) -> PlotOptions:
        """Construit les options depuis l'état de la session Streamlit."""
        from src.ui.i18n import get_lang

        return cls(lang=get_lang())


@dataclass
class KdCiData:
    """Données pour les traces IC de la courbe K/D cumulée."""

    x: list
    y_cumul: list
    y_upper: list
    y_lower: list
    y_match: list


@dataclass
class EwmaData:
    """Données pour les traces EWMA K/D."""

    x: list
    y_kd: list
    y_ewma: list
    regression_data: dict | None
