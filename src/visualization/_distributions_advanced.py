"""Graphiques avancés de distributions : corrélations et distributions temporelles.

Extrait de distributions.py pour respecter la limite de 500 lignes.
"""

from __future__ import annotations

import numpy as np
import plotly.graph_objects as go
import polars as pl

from src.config import HALO_COLORS, OUTCOME_CODES, PLOT_CONFIG
from src.ui.i18n.viz import viz_t
from src.visualization._compat import DataFrameLike, ensure_polars
from src.visualization._plot_options import PlotOptions
from src.visualization.theme import apply_halo_plot_style, get_legend_horizontal_bottom


def plot_correlation_scatter(  # noqa: PLR0913
    df: DataFrameLike,
    x_col: str,
    y_col: str,
    *,
    color_col: str | None = None,
    x_label: str | None = None,
    y_label: str | None = None,
    show_trendline: bool = True,
    opts: PlotOptions | None = None,
) -> go.Figure:
    """Scatter plot pour visualiser les corrélations.

    Args:
        df: DataFrame (Pandas ou Polars) avec les données.
        x_col: Colonne pour l'axe X.
        y_col: Colonne pour l'axe Y.
        color_col: Colonne pour colorer les points (optionnel).
        title: Titre optionnel.
        x_label: Label axe X (défaut: nom colonne).
        y_label: Label axe Y (défaut: nom colonne).
        show_trendline: Afficher la ligne de tendance.

    Returns:
        Figure Plotly avec scatter plot.
    """
    _opts = opts if opts is not None else PlotOptions()
    lang = _opts.lang
    df = ensure_polars(df)

    colors = HALO_COLORS.as_dict()
    d = df.drop_nulls(subset=[x_col, y_col])

    if d.is_empty():
        fig = go.Figure()
        fig.update_layout(height=PLOT_CONFIG.default_height)
        return apply_halo_plot_style(fig)

    x_series = d.get_column(x_col).cast(pl.Float64, strict=False)
    y_series = d.get_column(y_col).cast(pl.Float64, strict=False)

    # Couleur des points
    if color_col and color_col in d.columns:
        color_values = d.get_column(color_col).to_list()

        # Si c'est outcome, mapper vers des couleurs
        if color_col == "outcome":
            color_map = {
                OUTCOME_CODES.WIN: colors["green"],
                OUTCOME_CODES.LOSS: colors["red"],
                OUTCOME_CODES.TIE: colors["amber"],
                OUTCOME_CODES.NO_FINISH: colors["violet"],
            }
            marker_colors = [color_map.get(v, colors["slate"]) for v in color_values]
        else:
            marker_colors = colors["cyan"]
    else:
        marker_colors = colors["cyan"]

    x_np = x_series.to_numpy()
    y_np = y_series.to_numpy()

    fig = go.Figure()

    fig.add_trace(
        go.Scatter(
            x=x_np,
            y=y_np,
            mode="markers",
            marker={
                "color": marker_colors,
                "size": 8,
                "opacity": 0.6,
            },
            hovertemplate=(
                f"{x_label or x_col}: %{{x:.2f}}<br>{y_label or y_col}: %{{y:.2f}}<extra></extra>"
            ),
        )
    )

    # Ligne de tendance
    if show_trendline and len(x_np) > 2:
        valid = ~(np.isnan(x_np) | np.isnan(y_np))
        if valid.sum() > 2:
            x_valid = x_np[valid]
            y_valid = y_np[valid]

            # Vérifier que les données ont une variance suffisante
            x_std = np.std(x_valid)
            y_std = np.std(y_valid)

            # Skipper si variance nulle (tous les points alignés verticalement/horizontalement)
            if x_std > 1e-10 and y_std > 1e-10:
                try:
                    # Régression linéaire simple avec suppression des warnings NumPy
                    with np.errstate(divide="ignore", invalid="ignore"):
                        m, b = np.polyfit(x_valid, y_valid, 1)

                    x_range = np.linspace(x_valid.min(), x_valid.max(), 50)
                    y_trend = m * x_range + b

                    # Calcul R²
                    y_pred = m * x_valid + b
                    ss_res = np.sum((y_valid - y_pred) ** 2)
                    ss_tot = np.sum((y_valid - y_valid.mean()) ** 2)
                    r_squared = 1 - (ss_res / ss_tot) if ss_tot > 0 else 0

                    # Vérifier que le résultat est valide
                    if np.isfinite(m) and np.isfinite(b) and np.isfinite(r_squared):
                        fig.add_trace(
                            go.Scatter(
                                x=x_range,
                                y=y_trend,
                                mode="lines",
                                name=viz_t("trace_trend_r2", lang, r2=r_squared),
                                line={"color": colors["amber"], "width": 2, "dash": "dash"},
                                hoverinfo="skip",
                            )
                        )
                except (np.linalg.LinAlgError, ValueError):
                    # Skipper silencieusement si la régression échoue
                    pass

    fig.update_layout(
        height=PLOT_CONFIG.default_height,
        margin={"l": 40, "r": 20, "t": 30, "b": 40},
        showlegend=show_trendline,
    )
    fig.update_xaxes(title_text=x_label or x_col)
    fig.update_yaxes(title_text=y_label or y_col)

    return apply_halo_plot_style(fig, height=PLOT_CONFIG.default_height)


def plot_first_event_distribution(
    first_kills: dict[str, int | None],
    first_deaths: dict[str, int | None],
    *,
    lang: str = "fr",
) -> go.Figure:
    """Graphique de distribution des timestamps du premier kill/death.

    Args:
        first_kills: Dict {match_id: time_ms} pour le premier kill.
        first_deaths: Dict {match_id: time_ms} pour la première mort.
        lang: Langue d'affichage.

    Returns:
        Figure Plotly avec histogrammes superposés.
    """
    colors = HALO_COLORS.as_dict()

    # Convertir en secondes et filtrer les None
    kills_sec = [t / 1000 for t in first_kills.values() if t is not None and t > 0]
    deaths_sec = [t / 1000 for t in first_deaths.values() if t is not None and t > 0]

    if not kills_sec and not deaths_sec:
        fig = go.Figure()
        fig.update_layout(height=PLOT_CONFIG.default_height)
        return apply_halo_plot_style(fig)

    fig = go.Figure()

    if kills_sec:
        fig.add_trace(
            go.Histogram(
                x=kills_sec,
                name=viz_t("trace_first_kill", lang),
                marker_color=colors["green"],
                opacity=0.7,
                nbinsx=20,
                hovertemplate=viz_t("hover_first_event", lang),
            )
        )

    if deaths_sec:
        fig.add_trace(
            go.Histogram(
                x=deaths_sec,
                name=viz_t("trace_first_death", lang),
                marker_color=colors["red"],
                opacity=0.6,
                nbinsx=20,
                hovertemplate=viz_t("hover_first_event", lang),
            )
        )

    # Ajouter des lignes verticales pour les moyennes
    if kills_sec:
        avg_kill = sum(kills_sec) / len(kills_sec)
        fig.add_vline(
            x=avg_kill,
            line_dash="dash",
            line_color=colors["green"],
            annotation_text=viz_t("annot_avg_kill", lang, val=f"{avg_kill:.0f}"),
            annotation_position="top",
        )

    if deaths_sec:
        avg_death = sum(deaths_sec) / len(deaths_sec)
        fig.add_vline(
            x=avg_death,
            line_dash="dash",
            line_color=colors["red"],
            annotation_text=viz_t("annot_avg_death", lang, val=f"{avg_death:.0f}"),
            annotation_position="bottom",
        )

    # Ajouter des lignes verticales pour les médianes
    if kills_sec:
        median_kill = float(np.median(kills_sec))
        fig.add_vline(
            x=median_kill,
            line_dash="dot",
            line_color="#ffaa00",
            annotation_text=viz_t("annot_med_kill", lang, val=f"{median_kill:.0f}"),
            annotation_position="top right",
            annotation_font_color="#ffaa00",
        )

    if deaths_sec:
        median_death = float(np.median(deaths_sec))
        fig.add_vline(
            x=median_death,
            line_dash="dot",
            line_color="#ffaa00",
            annotation_text=viz_t("annot_med_death", lang, val=f"{median_death:.0f}"),
            annotation_position="bottom right",
            annotation_font_color="#ffaa00",
        )

    fig.update_layout(
        barmode="overlay",
        height=PLOT_CONFIG.default_height,
        margin={"l": 40, "r": 20, "t": 30, "b": 40},
        legend=get_legend_horizontal_bottom(),
    )
    fig.update_xaxes(title_text=viz_t("axis_time_seconds", lang))
    fig.update_yaxes(title_text=viz_t("axis_match_count", lang))

    return apply_halo_plot_style(fig, height=PLOT_CONFIG.default_height)
